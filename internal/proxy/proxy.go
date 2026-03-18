package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"proxy-control-plane/internal/config"
)

type Executor struct {
}

func NewExecutor() *Executor {
	return &Executor{}
}

func (e *Executor) createHTTPClient(provider config.Provider, params config.DynamicParams) (*http.Client, error) {
	transport := &http.Transport{}

	if provider.Type == "http" || provider.Type == "https" {
		username, password := provider.ResolveCredentials(params)
		proxyURL := fmt.Sprintf("http://%s:%s@%s:%d",
			url.QueryEscape(username),
			url.QueryEscape(password),
			provider.Host,
			provider.Port,
		)
		proxy, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("parse proxy url: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxy)
	} else if provider.Type == "socks5" {
		return nil, fmt.Errorf("socks5 proxy not implemented yet")
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}, nil
}

func (e *Executor) Execute(provider config.Provider, params config.DynamicParams, targetURL string, method string, headers http.Header, body io.ReadCloser) (*http.Response, error) {
	client, err := e.createHTTPClient(provider, params)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, targetURL, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	for k, v := range headers {
		req.Header[k] = v
	}

	req.Header.Del("X-Service-Name")
	req.Header.Del("X-Proxy-Country")
	req.Header.Del("X-Proxy-Provider")
	req.Header.Del("X-Proxy-Region")
	req.Header.Del("X-Proxy-Session")
	req.Header.Del("X-Proxy-Duration")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	return resp, nil
}
