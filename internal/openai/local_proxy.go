package openai

import (
	"easyllm/config"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// LocalProxyOrigin 返回 EasyLLM 本地代理根地址（无 /v1 后缀），供 Codex CLI 注入使用。
// hostOverride 通常来自 HTTP 请求的 Host，为空时回退到配置端口。
func LocalProxyOrigin(hostOverride string) string {
	host := strings.TrimSpace(hostOverride)
	if host == "" {
		cfg := config.Get()
		port := 8022
		if cfg != nil && cfg.Server.Port > 0 {
			port = cfg.Server.Port
		}
		return fmt.Sprintf("http://localhost:%d", port)
	}

	hostOnly, port, err := net.SplitHostPort(host)
	if err != nil {
		hostOnly = host
		port = ""
	}
	if hostOnly == "" || hostOnly == "0.0.0.0" || hostOnly == "::" || hostOnly == "[::]" ||
		strings.EqualFold(hostOnly, "localhost") || net.ParseIP(hostOnly) != nil {
		hostOnly = "localhost"
	}
	if port == "" {
		cfg := config.Get()
		port = "8022"
		if cfg != nil && cfg.Server.Port > 0 {
			port = strconv.Itoa(cfg.Server.Port)
		}
	}
	return fmt.Sprintf("http://%s", net.JoinHostPort(hostOnly, port))
}

// LocalProxyAPIBaseURL 返回 EasyLLM OpenAI 兼容 API 根地址（含 /v1）。
func LocalProxyAPIBaseURL(hostOverride string) string {
	return strings.TrimRight(LocalProxyOrigin(hostOverride), "/") + "/v1"
}
