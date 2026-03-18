package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type Provider struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Host      string   `json:"host"`
	Port      int      `json:"port"`
	Username  string   `json:"username"`
	Password  string   `json:"password"`
	Countries []string `json:"countries"`
	Enabled   bool     `json:"enabled"`
}

type DynamicParams struct {
	Country  string
	Region   string
	Session  string
	Duration string
}

func (p *Provider) ResolveCredentials(params DynamicParams) (username, password string) {
	username = replaceTemplateVars(p.Username, params)
	password = replaceTemplateVars(p.Password, params)
	return username, password
}

func replaceTemplateVars(template string, params DynamicParams) string {
	result := template
	if params.Country != "" {
		result = replaceAll(result, "{country}", params.Country)
	}
	if params.Region != "" {
		result = replaceAll(result, "{region}", params.Region)
	}
	if params.Session != "" {
		result = replaceAll(result, "{session}", params.Session)
	}
	if params.Duration != "" {
		result = replaceAll(result, "{duration}", params.Duration)
	}
	return result
}

func replaceAll(s, old, new string) string {
	result := ""
	for {
		idx := findSubstring(s, old)
		if idx == -1 {
			break
		}
		result += s[:idx] + new
		s = s[idx+len(old):]
	}
	return result + s
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

type ProviderWeight struct {
	ProviderID string `json:"provider_id"`
	Weight     int    `json:"weight"`
}

type Service struct {
	Name          string           `json:"name"`
	Enabled       bool             `json:"enabled"`
	Providers     []ProviderWeight `json:"providers"`
	RequestCount  int64            `json:"request_count"`
	LastRequestAt int64            `json:"last_request_at"`
}

type DailyUsage struct {
	Date       string `json:"date"`
	Service    string `json:"service"`
	ProviderID string `json:"provider_id"`
	Count      int64  `json:"count"`
}

type Config struct {
	providersPath string
	servicesPath  string
	usagePath     string
	providers     map[string]Provider
	services      map[string]Service
	dailyUsage    map[string]DailyUsage
	mu            sync.RWMutex
}

func Load(providersPath, servicesPath, usagePath string) (*Config, error) {
	cfg := &Config{
		providersPath: providersPath,
		servicesPath:  servicesPath,
		usagePath:     usagePath,
		providers:     make(map[string]Provider),
		services:      make(map[string]Service),
		dailyUsage:    make(map[string]DailyUsage),
	}

	if err := cfg.loadProviders(); err != nil {
		return nil, err
	}
	if err := cfg.loadServices(); err != nil {
		return nil, err
	}
	if err := cfg.loadDailyUsage(); err != nil {
		return nil, err
	}

	cfg.cleanupOldUsage()

	return cfg, nil
}

func (c *Config) loadProviders() error {
	if _, err := os.Stat(c.providersPath); os.IsNotExist(err) {
		return nil
	}
	data, err := os.ReadFile(c.providersPath)
	if err != nil {
		return fmt.Errorf("read providers: %w", err)
	}
	var providers []Provider
	if err := json.Unmarshal(data, &providers); err != nil {
		return fmt.Errorf("parse providers: %w", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, p := range providers {
		if p.ID == "" {
			continue
		}
		c.providers[p.ID] = p
	}
	return nil
}

func (c *Config) loadServices() error {
	if _, err := os.Stat(c.servicesPath); os.IsNotExist(err) {
		return nil
	}
	data, err := os.ReadFile(c.servicesPath)
	if err != nil {
		return fmt.Errorf("read services: %w", err)
	}
	var services []Service
	if err := json.Unmarshal(data, &services); err != nil {
		return fmt.Errorf("parse services: %w", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, s := range services {
		if s.Name == "" {
			continue
		}
		c.services[s.Name] = s
	}
	return nil
}

func (c *Config) loadDailyUsage() error {
	if _, err := os.Stat(c.usagePath); os.IsNotExist(err) {
		return nil
	}
	data, err := os.ReadFile(c.usagePath)
	if err != nil {
		return fmt.Errorf("read daily usage: %w", err)
	}
	var usage []DailyUsage
	if err := json.Unmarshal(data, &usage); err != nil {
		return fmt.Errorf("parse daily usage: %w", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, u := range usage {
		key := u.Date + "|" + u.Service + "|" + u.ProviderID
		c.dailyUsage[key] = u
	}
	return nil
}

func (c *Config) cleanupOldUsage() {
	c.mu.Lock()
	defer c.mu.Unlock()

	threshold := time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	for key, u := range c.dailyUsage {
		if u.Date < threshold {
			delete(c.dailyUsage, key)
		}
	}
}

func (c *Config) SaveProviders() error {
	c.mu.RLock()
	providers := make([]Provider, 0, len(c.providers))
	for _, p := range c.providers {
		providers = append(providers, p)
	}
	c.mu.RUnlock()

	data, err := json.MarshalIndent(providers, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal providers: %w", err)
	}
	if err := os.WriteFile(c.providersPath, data, 0644); err != nil {
		return fmt.Errorf("write providers: %w", err)
	}
	return nil
}

func (c *Config) SaveServices() error {
	c.mu.RLock()
	services := make([]Service, 0, len(c.services))
	for _, s := range c.services {
		services = append(services, s)
	}
	c.mu.RUnlock()

	data, err := json.MarshalIndent(services, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal services: %w", err)
	}
	if err := os.WriteFile(c.servicesPath, data, 0644); err != nil {
		return fmt.Errorf("write services: %w", err)
	}
	return nil
}

func (c *Config) SaveDailyUsage() error {
	c.mu.RLock()
	usage := make([]DailyUsage, 0, len(c.dailyUsage))
	for _, u := range c.dailyUsage {
		usage = append(usage, u)
	}
	c.mu.RUnlock()

	data, err := json.MarshalIndent(usage, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal daily usage: %w", err)
	}
	if err := os.WriteFile(c.usagePath, data, 0644); err != nil {
		return fmt.Errorf("write daily usage: %w", err)
	}
	return nil
}

func (c *Config) GetProviders() []Provider {
	c.mu.RLock()
	defer c.mu.RUnlock()
	providers := make([]Provider, 0, len(c.providers))
	for _, p := range c.providers {
		providers = append(providers, p)
	}
	return providers
}

func (c *Config) GetProvider(id string) (Provider, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	p, ok := c.providers[id]
	return p, ok
}

func (c *Config) SetProvider(p Provider) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.providers[p.ID] = p
}

func (c *Config) DeleteProvider(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.providers, id)
}

func (c *Config) GetServices() []Service {
	c.mu.RLock()
	defer c.mu.RUnlock()
	services := make([]Service, 0, len(c.services))
	for _, s := range c.services {
		services = append(services, s)
	}
	return services
}

func (c *Config) GetService(name string) (Service, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s, ok := c.services[name]
	return s, ok
}

func (c *Config) SetService(s Service) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.services[s.Name] = s
}

func (c *Config) DeleteService(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.services, name)
}

func (c *Config) IncrementServiceRequest(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s, ok := c.services[name]
	if !ok {
		s = Service{Name: name, Enabled: true}
	}
	s.RequestCount++
	s.LastRequestAt = getTimestamp()
	c.services[name] = s
}

func (c *Config) IncrementDailyUsage(serviceName, providerID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	date := time.Now().Format("2006-01-02")
	key := date + "|" + serviceName + "|" + providerID

	u, ok := c.dailyUsage[key]
	if !ok {
		u = DailyUsage{
			Date:       date,
			Service:    serviceName,
			ProviderID: providerID,
			Count:      0,
		}
	}
	u.Count++
	c.dailyUsage[key] = u
}

func (c *Config) GetDailyUsage(serviceName string) []DailyUsage {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]DailyUsage, 0)
	for _, u := range c.dailyUsage {
		if serviceName == "" || u.Service == serviceName {
			result = append(result, u)
		}
	}
	return result
}

func (c *Config) GetTodayUsage(serviceName string) map[string]int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	today := time.Now().Format("2006-01-02")
	result := make(map[string]int64)

	for _, u := range c.dailyUsage {
		if u.Date == today && (serviceName == "" || u.Service == serviceName) {
			result[u.ProviderID] = u.Count
		}
	}
	return result
}

func getTimestamp() int64 {
	return time.Now().Unix()
}
