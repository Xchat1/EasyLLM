package server

import (
	"context"
	"easyllm/config"
	"easyllm/internal/handlers"
	"easyllm/internal/models"
	"easyllm/internal/proxy"
	"easyllm/internal/storage"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	ginStatic "github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

// App holds all application dependencies
type App struct {
	cfg          *config.Config
	auth         *handlers.AuthHandler
	openai       *handlers.OpenAIHandler
	antigravity  *handlers.AntigravityHandler
	cursor       *handlers.CursorHandler
	settings     *handlers.SettingsHandler
	codexProxy   *proxy.CodexProxy
	openaiStore  *storage.OpenAIStorage
	relayHandler *proxy.RelayHandler
	router       *gin.Engine
	client       *http.Client
}

// New creates a new App with all dependencies initialized
func New(cfg *config.Config) (*App, error) {
	if err := storage.InitDB(cfg); err != nil {
		return nil, fmt.Errorf("failed to init database: %w", err)
	}

	db := storage.GetDB()
	dataDir := cfg.App.DataDir
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}
	_ = os.Chmod(dataDir, 0700)

	// Initialize storages
	openaiStore := storage.NewOpenAIStorage(db)
	codexStore := storage.NewCodexStorage(db)
	antigravityStore := storage.NewAntigravityStorage(db)
	cursorStore := storage.NewCursorStorage(db)
	// Load persisted settings into config
	loadPersistedSettings(cfg)

	// Initialize Codex proxy (pool includes both dedicated CodexAccounts and OpenAI OAuth accounts with proxy enabled)
	strategy := "round_robin"
	if s, ok := storage.GetSetting("proxy_strategy"); ok && s != "" {
		strategy = s
	}
	codexProxy := proxy.InitProxy(codexStore, openaiStore, strategy)
	if v, ok := storage.GetSetting("proxy_pool_enabled"); ok && v == "false" {
		codexProxy.SetEnabled(false)
	}

	// Build handlers
	app := &App{
		cfg:          cfg,
		auth:         handlers.NewAuthHandler(),
		openai:       handlers.NewOpenAIHandler(openaiStore, codexStore),
		antigravity:  handlers.NewAntigravityHandler(antigravityStore),
		cursor:       handlers.NewCursorHandler(cursorStore),
		settings:     handlers.NewSettingsHandler(),
		codexProxy:   codexProxy,
		openaiStore:  openaiStore,
		relayHandler: proxy.NewRelayHandler(nil),
		client: &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				ResponseHeaderTimeout: 120 * time.Second,
				IdleConnTimeout:       90 * time.Second,
			},
		},
	}

	// Initialize default password if configured
	if err := app.auth.InitializeDefaultPassword(); err != nil {
		return nil, fmt.Errorf("failed to initialize default password: %w", err)
	}

	app.setupRouter()
	return app, nil
}

func (a *App) setupRouter() {
	if !a.cfg.App.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	// Allow moderately large multipart uploads for batch token JSON imports.
	r.MaxMultipartMemory = 100 << 20 // 100 MiB
	r.Use(conditionalLogger(a.cfg))
	r.Use(gin.Recovery())
	r.Use(GzipMiddleware())
	r.Use(ipBlacklistMiddleware(a.cfg))
	r.Use(noStoreAPIMiddleware())
	r.Use(webUICacheMiddleware())

	r.Use(cors.New(localCORSConfig()))

	// Serve embedded web UI
	r.Use(ginStatic.Serve("/", ginStatic.LocalFile("./web/dist", false)))

	// API routes
	api := r.Group("/api/v1")

	// Public auth routes (login/setup/check — no token needed)
	a.auth.RegisterRoutes(api)

	// 公开：API 服务状态（侧栏轮询用，不鉴权，避免 401 导致反复跳登录）
	api.GET("/api-server/status", a.settings.GetAPIServerStatus)

	// The native menu bar uses loopback; remote clients must authenticate.
	api.GET("/antigravity/summary", summaryAccessMiddleware(), a.antigravity.GetSummary)
	api.POST("/antigravity/summary/refresh", summaryAccessMiddleware(), a.antigravity.RefreshSummary)

	// Protected routes require a valid login JWT.
	protected := r.Group("/api/v1")
	protected.Use(handlers.AuthMiddleware())

	a.auth.RegisterProtectedRoutes(protected)
	a.openai.RegisterRoutes(protected)
	a.antigravity.RegisterRoutes(protected)
	a.cursor.RegisterRoutes(protected)
	a.settings.RegisterRoutes(protected)

	// Legacy API endpoint (compatible with original ATM API)
	legacy := r.Group("/api")
	legacy.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, models.HealthResponse{
			Status:  "ok",
			Version: models.AppVersion,
			Port:    a.cfg.Server.Port,
		})
	})

	// Legacy pool status endpoint (compatible with original ATM API).
	// Emails are masked to prevent unauthenticated enumeration of account emails.
	// If authentication is enabled and request is remote, verify authentication.
	r.GET("/pool/status", func(c *gin.Context) {
		if handlers.IsAuthEnabled() && !isLoopbackRemoteAddr(c.Request.RemoteAddr) {
			handlers.AuthMiddleware()(c)
			if c.IsAborted() {
				return
			}
		}
		if a.codexProxy == nil {
			c.JSON(http.StatusOK, gin.H{
				"total_accounts":   0,
				"enabled_accounts": 0,
				"total_requests":   0,
				"accounts":         []any{},
			})
			return
		}
		status := a.codexProxy.GetPoolStatus()
		if status != nil && len(status.Accounts) > 0 {
			maskedAccounts := make([]models.CodexAccount, len(status.Accounts))
			for i, acct := range status.Accounts {
				maskedAccounts[i] = acct
				maskedAccounts[i].Email = maskEmail(acct.Email)
			}
			statusCopy := *status
			statusCopy.Accounts = maskedAccounts
			c.JSON(http.StatusOK, &statusCopy)
			return
		}
		c.JSON(http.StatusOK, status)
	})

	// OpenAI-compatible proxy (v1/*) — wildcard catches all /v1/… paths.
	// /v1/responses and /v1/models are dispatched inside proxyV1Request to avoid Gin route conflicts.
	r.Any("/v1/*path", a.proxyV1Request)

	// ChatGPT-native Codex path — used by Codex CLI when chatgpt_base_url points here.
	// CLI appends "codex/*" to chatgpt_base_url, resulting in /backend-api/codex/* paths.
	r.Any("/backend-api/codex/*path", a.proxyCodexRequest)
	r.Any("/backend-api/codex", a.proxyCodexRequest)

	// Relay config API (protected)
	relayConfig := protected.Group("/relay")
	{
		relayConfig.GET("/config", a.relayHandler.HandleGetRelayConfig)
		relayConfig.PUT("/config", a.relayHandler.HandleUpdateRelayConfig)
		relayConfig.POST("/sessions/clear", a.relayHandler.HandleClearRelaySessions)
		relayConfig.GET("/sessions/stats", a.relayHandler.HandleGetRelaySessionStats)
		relayConfig.GET("/usage", a.relayHandler.HandleGetRelayUsage)
		relayConfig.DELETE("/usage/history", a.relayHandler.HandleClearRelayHistory)
		relayConfig.GET("/logs", a.relayHandler.HandleGetRelayLogs)
		relayConfig.GET("/logs/stream", a.relayHandler.HandleStreamRelayLogs)
		relayConfig.DELETE("/logs", a.relayHandler.HandleClearRelayLogs)
		relayConfig.POST("/inject-codex", a.relayHandler.HandleInjectCodexConfig)
	}

	// SPA fallback - serve index.html for all unmatched routes
	r.NoRoute(func(c *gin.Context) {
		c.File("./web/dist/index.html")
	})

	a.router = r
}

func (a *App) proxyCodexRequest(c *gin.Context) {
	if a.codexProxy == nil || !a.codexProxy.IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"message": "Codex proxy is not enabled",
				"type":    "service_unavailable",
			},
		})
		return
	}

	if !allowLocalProxyFallback(c) {
		return
	}

	// API key authentication: if proxy_api_key is set, require it.
	// Exception: skip the check if the request token matches a managed account
	// (passthrough mode for local Codex CLI routing through the proxy).
	if requiredKey, ok := storage.GetSetting("proxy_api_key"); ok && requiredKey != "" {
		token := extractBearerToken(c.GetHeader("Authorization"))
		isPassthrough := a.codexProxy != nil && a.codexProxy.IsKnownToken(token)
		if token != requiredKey && !isPassthrough {
			rejectUnauthorized(c)
			return
		}
	}

	// Detect WebSocket upgrade and route to WS proxy
	if isWebSocketUpgrade(c.Request) {
		a.codexProxy.ProxyWebSocket(c.Writer, c.Request)
		return
	}

	a.codexProxy.ProxyRequest(c.Writer, c.Request)
}

func (a *App) proxyV1Request(c *gin.Context) {
	path := c.Param("path")

	// Relay: /v1/responses (Responses API → Chat Completions translation).
	// When Codex Local Access is active, requests carrying proxy_api_key must hit the
	// OAuth proxy pool instead of Relay (which may point at a different upstream).
	if path == "/responses" && c.Request.Method == http.MethodPost {
		if a.shouldRouteV1ResponsesToCodexProxy(c) {
			if !a.authorizeProxyRequest(c, a.codexProxy.IsKnownToken) {
				return
			}
			a.codexProxy.ProxyRequest(c.Writer, c.Request)
			return
		}
		if !a.authorizeProxyRequest(c, nil) {
			return
		}
		a.relayHandler.HandleRelayResponses(c)
		return
	}
	// Relay: /v1/models (proxied from upstream)
	if path == "/models" {
		if !a.authorizeProxyRequest(c, nil) {
			return
		}
		a.relayHandler.HandleRelayModels(c)
		return
	}

	if !a.authorizeProxyRequest(c, a.isActiveAPIAccountToken) {
		return
	}

	// If explicitly configured to keep legacy behavior, route /v1/* into the Codex proxy.
	// This keeps backward compatibility for users who intentionally proxy /v1/* to chatgpt.com Codex backend.
	if mode, ok := storage.GetSetting("v1_proxy_mode"); ok && mode == "codex" {
		if a.codexProxy == nil || !a.codexProxy.IsEnabled() {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": gin.H{
					"message": "Codex proxy is not enabled",
					"type":    "service_unavailable",
				},
			})
			return
		}
		a.codexProxy.ProxyRequest(c.Writer, c.Request)
		return
	}

	// If there's no active API account, fall back to OAuth pool for the classic
	// /v1/chat/completions path (so curl can work with OAuth-only setups).
	if c.Request.URL.Path == "/v1/chat/completions" && a.codexProxy != nil && a.codexProxy.IsEnabled() {
		// Only use this fallback when no active API account is configured.
		if a.openaiStore != nil {
			if active, err := a.openaiStore.GetCodexActive(); err != nil || active == nil || active.AccountType != models.OpenAIAccountTypeAPI {
				a.codexProxy.ProxyChatCompletions(c.Writer, c.Request)
				return
			}
		}
	}

	// Default: forward to active API account base_url with its api_key.
	if a.openaiStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"message": "OpenAI storage not initialized",
				"type":    "service_unavailable",
			},
		})
		return
	}

	active, err := a.openaiStore.GetCodexActive()
	if err != nil || active == nil || active.AccountType != models.OpenAIAccountTypeAPI {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"message": "No active API account configured. Add an API account in the OpenAI page and switch it to active (Codex).",
				"type":    "no_active_api_account",
				"code":    "503",
			},
		})
		return
	}

	baseURL := ""
	if active.BaseURL != nil {
		baseURL = strings.TrimSpace(*active.BaseURL)
	}
	apiKey := ""
	if active.APIKey != nil {
		apiKey = strings.TrimSpace(*active.APIKey)
	}
	if baseURL == "" || apiKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"message": "Active API account is missing base_url or api_key.",
				"type":    "invalid_upstream_config",
				"code":    "503",
			},
		})
		return
	}

	upstreamURL := buildAPIAccountUpstreamURL(baseURL, c.Request.URL.Path)
	if q := c.Request.URL.RawQuery; q != "" {
		upstreamURL += "?" + q
	}

	req, err := http.NewRequest(c.Request.Method, upstreamURL, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"message": "Failed to create upstream request", "type": "internal_error"},
		})
		return
	}

	// Copy headers except upstream auth/Host; apply provider-specific API auth below.
	for k, vals := range c.Request.Header {
		lk := strings.ToLower(k)
		if lk == "authorization" || lk == "api-key" || lk == "host" || lk == "content-length" {
			continue
		}
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	applyAPIAccountAuthHeaders(req, active.ModelProvider, apiKey)
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := a.httpClient().Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": gin.H{"message": "Upstream request failed: " + err.Error(), "type": "upstream_error"},
		})
		return
	}
	defer resp.Body.Close()

	for k, vals := range resp.Header {
		lk := strings.ToLower(k)
		if lk == "transfer-encoding" || lk == "connection" || lk == "keep-alive" || lk == "content-length" {
			continue
		}
		for _, v := range vals {
			c.Writer.Header().Add(k, v)
		}
	}
	c.Writer.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(c.Writer, resp.Body)
}

func (a *App) isActiveAPIAccountToken(token string) bool {
	token = strings.TrimSpace(token)
	if token == "" || a.openaiStore == nil {
		return false
	}
	active, err := a.openaiStore.GetCodexActive()
	if err != nil || active == nil || active.AccountType != models.OpenAIAccountTypeAPI {
		return false
	}
	if active.APIKey == nil {
		return false
	}
	return token == strings.TrimSpace(*active.APIKey)
}

func (a *App) httpClient() *http.Client {
	return a.client
}

func applyAPIAccountAuthHeaders(req *http.Request, _ *string, apiKey string) {
	req.Header.Set("Authorization", "Bearer "+apiKey)
}

func buildAPIAccountUpstreamURL(baseURL, requestPath string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	path := "/" + strings.TrimLeft(requestPath, "/")
	if strings.HasSuffix(base, "/v1") && strings.HasPrefix(path, "/v1/") {
		path = strings.TrimPrefix(path, "/v1")
	}
	return base + path
}

func isWebSocketUpgrade(r *http.Request) bool {
	for _, v := range r.Header["Connection"] {
		for _, part := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(part), "upgrade") {
				for _, u := range r.Header["Upgrade"] {
					if strings.EqualFold(strings.TrimSpace(u), "websocket") {
						return true
					}
				}
			}
		}
	}
	return false
}

func allowLocalProxyFallback(c *gin.Context) bool {
	if requiredKey, ok := storage.GetSetting("proxy_api_key"); ok && strings.TrimSpace(requiredKey) != "" {
		return true
	}
	if isLoopbackRemoteAddr(c.Request.RemoteAddr) {
		return true
	}
	c.JSON(http.StatusUnauthorized, gin.H{
		"error": gin.H{
			"message": "Non-local proxy access requires proxy_api_key to be configured.",
			"type":    "proxy_api_key_required",
		},
	})
	return false
}

func (a *App) authorizeProxyRequest(c *gin.Context, allowToken func(string) bool) bool {
	if !allowLocalProxyFallback(c) {
		return false
	}
	requiredKey, ok := storage.GetSetting("proxy_api_key")
	requiredKey = strings.TrimSpace(requiredKey)
	if !ok || requiredKey == "" {
		return true
	}
	token := strings.TrimSpace(extractBearerToken(c.GetHeader("Authorization")))
	if token == requiredKey || (allowToken != nil && allowToken(token)) {
		return true
	}
	rejectUnauthorized(c)
	return false
}

func localCORSConfig() cors.Config {
	return cors.Config{
		AllowOriginFunc:  isAllowedBrowserOrigin,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length", "Content-Disposition"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}
}

func isAllowedBrowserOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	host := u.Hostname()
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func summaryAccessMiddleware() gin.HandlerFunc {
	authenticate := handlers.AuthMiddleware()
	return func(c *gin.Context) {
		if isLoopbackRemoteAddr(c.Request.RemoteAddr) {
			c.Next()
			return
		}
		authenticate(c)
	}
}

func isLoopbackRemoteAddr(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(strings.TrimSpace(host))
	return ip != nil && ip.IsLoopback()
}

// Run starts the HTTP server with graceful shutdown
func (a *App) Run() error {
	addr := fmt.Sprintf("%s:%d", a.cfg.Server.Host, a.cfg.Server.Port)

	srv := &http.Server{
		Addr:         addr,
		Handler:      a.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // no timeout — streaming endpoints (SSE) can run for minutes
	}

	// Periodic idle memory scrubber: return unused heap pages to macOS kernel
	memTicker := time.NewTicker(2 * time.Minute)
	defer memTicker.Stop()
	go func() {
		for range memTicker.C {
			debug.FreeOSMemory()
		}
	}()

	// Start server in goroutine
	go func() {
		log.Printf("EasyLLM server started on http://%s", addr)
		log.Printf("Web UI: http://localhost:%d", a.cfg.Server.Port)
		log.Printf("API:    http://localhost:%d/api/v1", a.cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return err
	}
	if err := storage.CloseDB(); err != nil {
		log.Printf("Warning: failed to close database: %v", err)
	}
	return nil
}

// loadPersistedSettings loads settings from DB into config
func loadPersistedSettings(cfg *config.Config) {
	settings := storage.GetAllSettings()

	if v, ok := settings["proxy_enabled"]; ok {
		cfg.Proxy.Enabled = v == "true"
	}
	if v, ok := settings["proxy_host"]; ok && v != "" {
		cfg.Proxy.Host = v
	}
	if v, ok := settings["proxy_port"]; ok && v != "" {
		if port := parseInt(v); port > 0 {
			cfg.Proxy.Port = port
		}
	}
	if v, ok := settings["proxy_username"]; ok {
		cfg.Proxy.Username = v
	}
	if v, ok := settings["proxy_password"]; ok {
		cfg.Proxy.Password = v
	}

	if v, ok := settings["ip_blacklist_enabled"]; ok {
		cfg.IPBlacklist.Enabled = v == "true"
	}
	if v, ok := settings["ip_blacklist"]; ok && v != "" {
		ips := strings.Split(v, ",")
		cleaned := make([]string, 0, len(ips))
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if ip != "" {
				cleaned = append(cleaned, ip)
			}
		}
		cfg.IPBlacklist.IPs = cleaned
	}
}

func maskEmail(email string) string {
	email = strings.TrimSpace(email)
	if email == "" {
		return ""
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		if len(email) <= 2 {
			return "***"
		}
		return email[:1] + "***" + email[len(email)-1:]
	}
	user, domain := parts[0], parts[1]
	if len(user) <= 1 {
		return "*@" + domain
	}
	if len(user) == 2 {
		return string(user[0]) + "*@" + domain
	}
	return string(user[0]) + "***" + string(user[len(user)-1]) + "@" + domain
}

func parseInt(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}

func conditionalLogger(_ *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func noStoreAPIMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/pool/status") {
			c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}
		c.Next()
	}
}

func webUICacheMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
			if path == "/" || strings.HasSuffix(path, ".html") || !strings.Contains(path[strings.LastIndex(path, "/")+1:], ".") {
				c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
				c.Header("Pragma", "no-cache")
				c.Header("Expires", "0")
			}
		}
		c.Next()
	}
}

func ipBlacklistMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		currentCfg := cfg
		if currentCfg == nil {
			currentCfg = config.Get()
		}
		if currentCfg == nil || !currentCfg.IPBlacklist.Enabled || len(currentCfg.IPBlacklist.IPs) == 0 {
			c.Next()
			return
		}

		clientIP := c.ClientIP()
		for _, blockedIP := range currentCfg.IPBlacklist.IPs {
			if blockedIP == clientIP {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": gin.H{
						"message": "Your IP has been blocked",
						"type":    "forbidden",
						"code":    "403",
					},
				})
				return
			}
		}
		c.Next()
	}
}

func (a *App) shouldRouteV1ResponsesToCodexProxy(c *gin.Context) bool {
	if a.codexProxy == nil || !a.codexProxy.IsEnabled() {
		return false
	}
	enabled, ok := storage.GetSetting("codex_local_access_enabled")
	if !ok || enabled != "true" {
		return false
	}
	mode, ok := storage.GetSetting("v1_proxy_mode")
	if !ok || mode != "codex" {
		return false
	}
	requiredKey, hasKey := storage.GetSetting("proxy_api_key")
	requiredKey = strings.TrimSpace(requiredKey)
	if hasKey && requiredKey != "" {
		token := extractBearerToken(c.GetHeader("Authorization"))
		return token == requiredKey || a.codexProxy.IsKnownToken(token)
	}
	return isLoopbackRemoteAddr(c.Request.RemoteAddr)
}

// extractBearerToken extracts the token from an Authorization: Bearer <token> header.
// Case-insensitive per RFC 7235.
func extractBearerToken(auth string) string {
	if len(auth) > 7 && strings.EqualFold(auth[:7], "bearer ") {
		return auth[7:]
	}
	return ""
}

// rejectUnauthorized writes a standard 401 JSON response.
func rejectUnauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, gin.H{
		"error": gin.H{
			"message": "Invalid API key",
			"type":    "invalid_api_key",
			"code":    "401",
		},
	})
}
