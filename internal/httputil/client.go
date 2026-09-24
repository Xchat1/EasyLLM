package httputil

import (
	"context"
	"easyllm/config"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// NewChatGPTClient returns an HTTP client tuned for chatgpt.com upstream calls.
func NewChatGPTClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &http.Client{
		Transport: NewChatGPTTransport(),
		Timeout:   timeout,
	}
}

// NewChatGPTStreamingClient is like NewChatGPTClient but disables compression for SSE streams.
//
// The overall client Timeout is intentionally disabled (0): http.Client.Timeout spans
// the entire response body read, so any SSE stream living longer than the timeout
// would be aborted mid-stream ("Client.Timeout exceeded while reading body").
// Streams are terminated via the request context instead; ResponseHeaderTimeout
// (set on the transport by NewChatGPTTransport) still guards against hung upstreams.
// The timeout parameter is kept for API compatibility and no longer bounds the request.
func NewChatGPTStreamingClient(timeout time.Duration) *http.Client {
	transport := NewChatGPTTransport()
	transport.DisableCompression = true
	return &http.Client{
		Transport: transport,
		// Timeout: 0 — disabled for long-lived SSE streams, see above.
	}
}

// NewChatGPTTransport builds a transport that prefers IPv4 and honors app proxy settings.
func NewChatGPTTransport() *http.Transport {
	dialer := &net.Dialer{
		Timeout:   15 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	transport := &http.Transport{
		DialContext:           DialPreferIPv4(dialer),
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   12 * time.Second,
		ResponseHeaderTimeout: 120 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	if proxyURL := proxyURLFromConfig(); proxyURL != nil {
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	return transport
}

func proxyURLFromConfig() *url.URL {
	cfg := config.Get()
	if cfg == nil || !cfg.Proxy.Enabled || strings.TrimSpace(cfg.Proxy.Host) == "" {
		return nil
	}
	proxyURLStr := fmt.Sprintf("http://%s:%d", cfg.Proxy.Host, cfg.Proxy.Port)
	if cfg.Proxy.Username != "" {
		proxyURLStr = fmt.Sprintf("http://%s:%s@%s:%d",
			url.QueryEscape(cfg.Proxy.Username),
			url.QueryEscape(cfg.Proxy.Password),
			cfg.Proxy.Host, cfg.Proxy.Port)
	}
	u, err := url.Parse(proxyURLStr)
	if err != nil {
		return nil
	}
	return u
}

// DialPreferIPv4 dials IPv4 addresses first to avoid unstable IPv6 resets.
func DialPreferIPv4(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	if dialer == nil {
		dialer = &net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}
	}
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return dialer.DialContext(ctx, network, address)
		}

		ips, lookupErr := dialer.Resolver.LookupIP(ctx, "ip4", host)
		if lookupErr == nil && len(ips) > 0 {
			var lastErr error
			for _, ip := range ips {
				conn, dialErr := dialer.DialContext(ctx, "tcp4", net.JoinHostPort(ip.String(), port))
				if dialErr == nil {
					return conn, nil
				}
				lastErr = dialErr
			}
			if lastErr != nil && !IsTransientNetworkError(lastErr) {
				return nil, lastErr
			}
		}

		return dialer.DialContext(ctx, "tcp", address)
	}
}

// IsTransientNetworkError reports whether the error is worth retrying.
func IsTransientNetworkError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.EOF) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary()) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection reset by peer") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "no route to host") ||
		strings.Contains(msg, "network is unreachable") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "tls handshake timeout")
}

// DoWithRetries executes an HTTP request with retries on transient network failures.
func DoWithRetries(client *http.Client, req *http.Request, maxAttempts int) (*http.Response, error) {
	if client == nil {
		return nil, fmt.Errorf("http client is nil")
	}
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := req.Context().Err(); err != nil {
			return nil, err
		}
		if attempt > 0 {
			if req.Body != nil && req.GetBody == nil {
				break
			}
			timer := time.NewTimer(time.Duration(attempt) * 400 * time.Millisecond)
			select {
			case <-timer.C:
			case <-req.Context().Done():
				timer.Stop()
				return nil, req.Context().Err()
			}
		}

		attemptReq := req.Clone(req.Context())
		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			attemptReq.Body = body
		}
		resp, err := client.Do(attemptReq)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if ctxErr := req.Context().Err(); ctxErr != nil {
			return nil, ctxErr
		}
		if !IsTransientNetworkError(err) {
			break
		}
	}

	return nil, lastErr
}
