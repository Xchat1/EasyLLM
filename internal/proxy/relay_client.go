package proxy

import (
	"easyllm/config"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// newRelayTransport builds the shared transport for relay upstream requests
// with optional proxy support.
func newRelayTransport() *http.Transport {
	cfg := config.Get()
	transport := &http.Transport{
		DisableCompression:    true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 120 * time.Second,
	}

	if cfg != nil && cfg.Proxy.Enabled && cfg.Proxy.Host != "" {
		proxyURLStr := fmt.Sprintf("http://%s:%d", cfg.Proxy.Host, cfg.Proxy.Port)
		if cfg.Proxy.Username != "" {
			proxyURLStr = fmt.Sprintf("http://%s:%s@%s:%d",
				url.QueryEscape(cfg.Proxy.Username),
				url.QueryEscape(cfg.Proxy.Password),
				cfg.Proxy.Host, cfg.Proxy.Port)
		}
		if u, err := url.Parse(proxyURLStr); err == nil {
			transport.Proxy = http.ProxyURL(u)
		}
	}
	return transport
}

// NewRelayHTTPClient returns an HTTP client for upstream relay requests with optional proxy support.
func NewRelayHTTPClient() *http.Client {
	return &http.Client{
		Timeout:   300 * time.Second,
		Transport: newRelayTransport(),
	}
}

// NewRelayStreamingHTTPClient returns an HTTP client dedicated to upstream relay
// SSE streams. The overall client Timeout is disabled (0): http.Client.Timeout spans
// the entire response body read, so a stream living longer than the timeout would be
// aborted mid-stream. Streams are terminated via the request context instead;
// ResponseHeaderTimeout still guards against hung upstreams.
func NewRelayStreamingHTTPClient() *http.Client {
	return &http.Client{
		Transport: newRelayTransport(),
		// Timeout: 0 — disabled for long-lived SSE streams, see above.
	}
}

// ResolveRelayAPIKey returns the upstream API key: server config takes priority,
// then Authorization / custom auth header from the incoming Codex request.
func ResolveRelayAPIKey(config *RelayConfig, headers http.Header) string {
	if config != nil && config.APIKey != "" {
		return config.APIKey
	}
	if headers == nil {
		return ""
	}
	authHeader := "Authorization"
	authPrefix := "Bearer "
	if config != nil {
		if config.AuthHeader != "" {
			authHeader = config.AuthHeader
		}
		if config.AuthValuePrefix != "" {
			authPrefix = config.AuthValuePrefix
		}
	}
	if v := headers.Get(authHeader); v != "" {
		if authPrefix != "" && len(v) >= len(authPrefix) && v[:len(authPrefix)] == authPrefix {
			return v[len(authPrefix):]
		}
		return v
	}
	if authHeader != "Authorization" {
		if v := headers.Get("Authorization"); v != "" {
			if len(v) > 7 && strings.EqualFold(v[:7], "bearer ") {
				return v[7:]
			}
			return v
		}
	}
	return ""
}

func applyAuthHeader(req *http.Request, apiKey string, authHeader, authValuePrefix string) {
	if apiKey == "" {
		return
	}
	if authHeader == "" {
		authHeader = "Authorization"
	}
	// Custom headers (e.g. api-key) default to no prefix; Authorization defaults to Bearer.
	if authHeader == "Authorization" && authValuePrefix == "" {
		authValuePrefix = "Bearer "
	}
	req.Header.Set(authHeader, authValuePrefix+apiKey)
}
