package httpapi

import (
	"io"
	"math/rand"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/google/uuid"

	"proxy-control-plane/internal/config"
	proxyexec "proxy-control-plane/internal/proxy"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func NewRouter(cfg *config.Config) (*gin.Engine, error) {
	executor := proxyexec.NewExecutor()

	r := gin.Default()

	r.Static("/static", filepath.Join(".", "web", "static"))
	r.LoadHTMLGlob(filepath.Join(".", "web", "templates", "*.html"))

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.GET("/health", healthCheck)

	api := r.Group("/api")
	{
		providers := api.Group("/providers")
		{
			providers.GET("", listProviders(cfg))
			providers.POST("", createProvider(cfg))
			providers.PUT("/:id", updateProvider(cfg))
			providers.DELETE("/:id", deleteProvider(cfg))
		}

		services := api.Group("/services")
		{
			services.GET("", listServices(cfg))
			services.POST("", createService(cfg))
			services.GET("/:name", getService(cfg))
			services.DELETE("/:name", deleteService(cfg))
			services.PUT("/:name/enabled", setServiceEnabled(cfg))
			services.POST("/:name/providers", addProviderToService(cfg))
			services.DELETE("/:name/providers/:providerId", removeProviderFromService(cfg))
			services.PUT("/:name/providers/:providerId/weight", updateProviderWeight(cfg))
			services.GET("/:name/usage", getServiceUsage(cfg))
		}

		api.GET("/assign", assignProxy(cfg))
	}

	r.Any("/proxy", proxyRequest(cfg, executor))

	return r, nil
}

// @Summary 健康检查
// @Description 检查服务是否正常运行
// @Tags 系统
// @Produce json
// @Success 200 {object} map[string]bool
// @Router /health [get]
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// @Summary 获取所有代理提供商
// @Description 获取所有代理提供商列表
// @Tags 代理提供商
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/providers [get]
func listProviders(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"providers": cfg.GetProviders()})
	}
}

// @Summary 创建代理提供商
// @Description 创建新的代理提供商
// @Tags 代理提供商
// @Accept json
// @Produce json
// @Param provider body config.Provider true "代理提供商信息"
// @Success 200 {object} config.Provider
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/providers [post]
func createProvider(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var p config.Provider
		if err := c.ShouldBindJSON(&p); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if p.ID == "" {
			p.ID = uuid.New().String()
		}
		cfg.SetProvider(p)
		if err := cfg.SaveProviders(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, p)
	}
}

// @Summary 更新代理提供商
// @Description 更新指定ID的代理提供商
// @Tags 代理提供商
// @Accept json
// @Produce json
// @Param id path string true "提供商ID"
// @Param provider body config.Provider true "代理提供商信息"
// @Success 200 {object} config.Provider
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/providers/{id} [put]
func updateProvider(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var p config.Provider
		if err := c.ShouldBindJSON(&p); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		p.ID = id
		cfg.SetProvider(p)
		if err := cfg.SaveProviders(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, p)
	}
}

// @Summary 删除代理提供商
// @Description 删除指定ID的代理提供商
// @Tags 代理提供商
// @Produce json
// @Param id path string true "提供商ID"
// @Success 200 {object} map[string]bool
// @Failure 500 {object} map[string]string
// @Router /api/providers/{id} [delete]
func deleteProvider(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		cfg.DeleteProvider(id)
		if err := cfg.SaveProviders(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// @Summary 获取所有服务
// @Description 获取所有服务列表及今日使用统计
// @Tags 服务管理
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/services [get]
func listServices(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		services := cfg.GetServices()
		todayUsage := cfg.GetTodayUsage("")
		c.JSON(http.StatusOK, gin.H{
			"services":    services,
			"today_usage": todayUsage,
		})
	}
}

// @Summary 创建服务
// @Description 创建新的服务
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param request body map[string]string true "服务名称"
// @Success 200 {object} config.Service
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/services [post]
func createService(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name string `json:"name" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if _, exists := cfg.GetService(req.Name); exists {
			c.JSON(http.StatusConflict, gin.H{"error": "service_already_exists"})
			return
		}

		svc := config.Service{
			Name:      req.Name,
			Enabled:   true,
			Providers: []config.ProviderWeight{},
		}
		cfg.SetService(svc)
		if err := cfg.SaveServices(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, svc)
	}
}

// @Summary 获取服务详情
// @Description 获取指定服务的详细信息和今日使用统计
// @Tags 服务管理
// @Produce json
// @Param name path string true "服务名称"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /api/services/{name} [get]
func getService(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		svc, ok := cfg.GetService(name)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "service_not_found"})
			return
		}

		todayUsage := cfg.GetTodayUsage(name)
		c.JSON(http.StatusOK, gin.H{
			"service":     svc,
			"today_usage": todayUsage,
		})
	}
}

// @Summary 删除服务
// @Description 删除指定服务
// @Tags 服务管理
// @Produce json
// @Param name path string true "服务名称"
// @Success 200 {object} map[string]bool
// @Failure 500 {object} map[string]string
// @Router /api/services/{name} [delete]
func deleteService(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		cfg.DeleteService(name)
		if err := cfg.SaveServices(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// @Summary 启用/禁用服务
// @Description 设置服务的启用状态
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param name path string true "服务名称"
// @Param request body map[string]bool true "启用状态"
// @Success 200 {object} config.Service
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/services/{name}/enabled [put]
func setServiceEnabled(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		svc, ok := cfg.GetService(name)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "service_not_found"})
			return
		}
		svc.Enabled = req.Enabled
		cfg.SetService(svc)
		if err := cfg.SaveServices(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, svc)
	}
}

// @Summary 为服务添加代理提供商
// @Description 为指定服务添加代理提供商并设置权重
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param name path string true "服务名称"
// @Param request body map[string]interface{} true "提供商ID和权重"
// @Success 200 {object} config.Service
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/services/{name}/providers [post]
func addProviderToService(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		var req struct {
			ProviderID string `json:"provider_id" binding:"required"`
			Weight     int    `json:"weight"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.Weight <= 0 {
			req.Weight = 1
		}

		if _, ok := cfg.GetProvider(req.ProviderID); !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "provider_not_found"})
			return
		}

		svc, ok := cfg.GetService(name)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "service_not_found"})
			return
		}

		for i, p := range svc.Providers {
			if p.ProviderID == req.ProviderID {
				svc.Providers[i].Weight = req.Weight
				cfg.SetService(svc)
				_ = cfg.SaveServices()
				c.JSON(http.StatusOK, svc)
				return
			}
		}

		svc.Providers = append(svc.Providers, config.ProviderWeight{
			ProviderID: req.ProviderID,
			Weight:     req.Weight,
		})
		cfg.SetService(svc)
		if err := cfg.SaveServices(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, svc)
	}
}

// @Summary 从服务移除代理提供商
// @Description 从指定服务移除代理提供商
// @Tags 服务管理
// @Produce json
// @Param name path string true "服务名称"
// @Param providerId path string true "提供商ID"
// @Success 200 {object} config.Service
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/services/{name}/providers/{providerId} [delete]
func removeProviderFromService(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		providerID := c.Param("providerId")

		svc, ok := cfg.GetService(name)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "service_not_found"})
			return
		}

		newProviders := make([]config.ProviderWeight, 0)
		for _, p := range svc.Providers {
			if p.ProviderID != providerID {
				newProviders = append(newProviders, p)
			}
		}
		svc.Providers = newProviders
		cfg.SetService(svc)
		if err := cfg.SaveServices(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, svc)
	}
}

// @Summary 更新代理提供商权重
// @Description 更新服务中指定代理提供商的权重
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param name path string true "服务名称"
// @Param providerId path string true "提供商ID"
// @Param request body map[string]int true "权重值"
// @Success 200 {object} config.Service
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/services/{name}/providers/{providerId}/weight [put]
func updateProviderWeight(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		providerID := c.Param("providerId")
		var req struct {
			Weight int `json:"weight" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		svc, ok := cfg.GetService(name)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "service_not_found"})
			return
		}

		found := false
		for i, p := range svc.Providers {
			if p.ProviderID == providerID {
				svc.Providers[i].Weight = req.Weight
				found = true
				break
			}
		}

		if !found {
			c.JSON(http.StatusNotFound, gin.H{"error": "provider_not_in_service"})
			return
		}

		cfg.SetService(svc)
		if err := cfg.SaveServices(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, svc)
	}
}

// @Summary 获取服务使用统计
// @Description 获取指定服务的每日使用统计
// @Tags 服务管理
// @Produce json
// @Param name path string true "服务名称"
// @Success 200 {object} map[string]interface{}
// @Router /api/services/{name}/usage [get]
func getServiceUsage(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		usage := cfg.GetDailyUsage(name)
		c.JSON(http.StatusOK, gin.H{"usage": usage})
	}
}

// @Summary 代理分配
// @Description 根据权重为服务分配代理提供商
// @Tags 代理
// @Produce json
// @Param service query string true "服务名称"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /api/assign [get]
func assignProxy(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceName := c.Query("service")
		if serviceName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "service parameter required"})
			return
		}

		svc, ok := cfg.GetService(serviceName)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "service_not_found"})
			return
		}

		if !svc.Enabled {
			c.JSON(http.StatusForbidden, gin.H{"error": "service_disabled"})
			return
		}

		if len(svc.Providers) == 0 {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no_providers_configured"})
			return
		}

		providerID := selectProviderByWeight(svc.Providers)
		provider, ok := cfg.GetProvider(providerID)
		if !ok {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "provider_not_found"})
			return
		}

		if !provider.Enabled {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "provider_disabled"})
			return
		}

		cfg.IncrementDailyUsage(serviceName, providerID)
		_ = cfg.SaveDailyUsage()

		proxyURL := buildProxyURL(provider)

		c.JSON(http.StatusOK, gin.H{
			"service":     serviceName,
			"provider_id": providerID,
			"provider":    provider.Name,
			"proxy_url":   proxyURL,
			"host":        provider.Host,
			"port":        provider.Port,
			"username":    provider.Username,
			"password":    provider.Password,
		})
	}
}

// @Summary 透明代理
// @Description 通过代理转发请求到目标URL
// @Tags 代理
// @Param X-Target-Url header string true "目标URL"
// @Param X-Service-Name header string false "服务名称"
// @Param X-Proxy-Country header string false "国家代码"
// @Param X-Proxy-Region header string false "区域代码"
// @Param X-Proxy-Session header string false "会话ID"
// @Param X-Proxy-Duration header string false "代理时长"
// @Param X-Proxy-Provider header string false "提供商ID"
// @Success 200 {string} string "目标服务器响应"
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 502 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /proxy [get]
// @Router /proxy [post]
// @Router /proxy [put]
// @Router /proxy [delete]
func proxyRequest(cfg *config.Config, executor *proxyexec.Executor) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceName := c.GetHeader("X-Service-Name")
		country := c.GetHeader("X-Proxy-Country")
		region := c.GetHeader("X-Proxy-Region")
		session := c.GetHeader("X-Proxy-Session")
		duration := c.GetHeader("X-Proxy-Duration")
		providerID := c.GetHeader("X-Proxy-Provider")
		targetURL := c.GetHeader("X-Target-Url")

		if targetURL == "" {
			targetURL = c.Query("url")
		}

		if targetURL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "X-Target-Url header or url query parameter required"})
			return
		}

		if serviceName != "" {
			svc, ok := cfg.GetService(serviceName)
			if ok && !svc.Enabled {
				c.JSON(http.StatusForbidden, gin.H{"error": "service_disabled"})
				return
			}
			cfg.IncrementServiceRequest(serviceName)
			_ = cfg.SaveServices()
		}

		var selectedProvider *config.Provider
		providers := cfg.GetProviders()

		for _, p := range providers {
			if !p.Enabled {
				continue
			}
			if providerID != "" && p.ID != providerID {
				continue
			}
			if country != "" {
				countrySupported := len(p.Countries) == 0
				for _, c := range p.Countries {
					if strings.EqualFold(c, country) {
						countrySupported = true
						break
					}
				}
				if !countrySupported {
					continue
				}
			}
			selectedProvider = &p
			break
		}

		if selectedProvider == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no_available_provider"})
			return
		}

		params := config.DynamicParams{
			Country:  country,
			Region:   region,
			Session:  session,
			Duration: duration,
		}

		resp, err := executor.Execute(*selectedProvider, params, targetURL, c.Request.Method, c.Request.Header, c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		defer resp.Body.Close()

		for k, v := range resp.Header {
			for _, val := range v {
				c.Writer.Header().Add(k, val)
			}
		}
		c.Writer.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(c.Writer, resp.Body)
	}
}

func selectProviderByWeight(providers []config.ProviderWeight) string {
	totalWeight := 0
	for _, p := range providers {
		totalWeight += p.Weight
	}

	r := rand.Intn(totalWeight)
	for _, p := range providers {
		r -= p.Weight
		if r < 0 {
			return p.ProviderID
		}
	}

	return providers[0].ProviderID
}

func buildProxyURL(p config.Provider) string {
	return ""
}
