package main

import (
	"log"
	"os"

	_ "proxy-control-plane/docs"
	"proxy-control-plane/internal/config"
	"proxy-control-plane/internal/httpapi"
)

// @title Proxy Control Plane API
// @version 1.0
// @description 代理控制平面 API - 管理多个代理服务，支持权重分配、使用统计等功能
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /
// @securityDefinitions.basic BasicAuth
func main() {
	providersPath := getenv("PROVIDERS_CONFIG", "providers.json")
	servicesPath := getenv("SERVICES_CONFIG", "services.json")
	usagePath := getenv("USAGE_CONFIG", "daily_usage.json")

	cfg, err := config.Load(providersPath, servicesPath, usagePath)
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	r, err := httpapi.NewRouter(cfg)
	if err != nil {
		log.Fatalf("init router failed: %v", err)
	}

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("proxy-control-plane listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server run failed: %v", err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
