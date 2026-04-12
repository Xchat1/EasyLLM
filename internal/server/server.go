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
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	ginStatic "github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

// App holds all application dependencies
type App struct {
	cfg            *config.Config
	auth           *handlers.AuthHandler
	openai         *handlers.OpenAIHandler
	cursor         *handlers.CursorHandler
	antigravity    *handlers.AntigravityHandler
	settings       *handlers.SettingsHandler
	cockpit        *handlers.CockpitHandler
	codexProxy     *proxy.CodexProxy
	sessionScanner *proxy.SessionScanner
	openaiStore    *storage.OpenAIStorage
	router         *gin.Engine
}

// New creates a new App with all dependencies initialized
func New(cfg *config.Config) (*App, error) {
	if err := storage.InitDB(cfg); err != nil {
		return nil, fmt.Errorf("failed to init database: %w", err)
	}

	db := storage.GetDB()
	dataDir := cfg.App.DataDir
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Initialize storages
	openaiStore := storage.NewOpenAIStorage(db)
	codexStore := storage.NewCodexStorage(db)
	cursorStore := storage.NewCursorStorage(db)
	antigravityStore := storage.NewAntigravityStorage(db)
	cockpitStore := storage.NewCockpitStorage(db)
	if err := cockpitStore.MigrateLegacyPlatformAccounts(); err != nil {
		return nil, fmt.Errorf("failed to migrate cockpit platform data: %w", err)
	}
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

	// Session scanner: imports Codex CLI session logs into dashboard
	sessionScanner := proxy.NewSessionScanner(codexStore)

	// Build handlers
	app := &App{
		cfg:            cfg,
		auth:           handlers.NewAuthHandler(),
		openai:         handlers.NewOpenAIHandler(openaiStore, codexStore),
		cursor:         handlers.NewCursorHandler(cursorStore),
		antigravity:    handlers.NewAntigravityHandler(antigravityStore),
		settings:       handlers.NewSettingsHandler(),
		cockpit:        handlers.NewCockpitHandler(cockpitStore, openaiStore, codexStore),
		codexProxy:     codexProxy,
		sessionScanner: sessionScanner,
		openaiStore:    openaiStore,
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
	// Allow large multipart uploads (batch import token JSON files).
	// Gin defaults to 32 MiB which easily triggers "multipart: message too large".
	// Setting a very large value effectively removes the limit (bounded by machine resources).
	r.MaxMultipartMemory = 8 << 30 // 8 GiB
	r.Use(conditionalLogger(a.cfg))
	r.Use(gin.Recovery())
	r.Use(ipBlacklistMiddleware(a.cfg))

	// CORS - allow all origins since this is a local tool
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length", "Content-Disposition"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	// Serve embedded web UI
	r.Use(ginStatic.Serve("/", ginStatic.LocalFile("./web/dist", false)))

	// API routes
	api := r.Group("/api/v1")

	// Public auth routes (login/setup/check — no token needed)
	a.auth.RegisterRoutes(api)

	// 公开：API 服务状态（侧栏轮询用，不鉴权，避免 401 导致反复跳登录）
	api.GET("/api-server/status", a.settings.GetAPIServerStatus)

	// Protected routes — require valid JWT when password is set
	protected := r.Group("/api/v1")
	protected.Use(handlers.AuthMiddleware())

	a.auth.RegisterProtectedRoutes(protected)
	a.openai.RegisterRoutes(protected)
	a.cursor.RegisterRoutes(protected)
	a.antigravity.RegisterRoutes(protected)
	a.settings.RegisterRoutes(protected)
	a.cockpit.RegisterRoutes(protected)

	// Legacy API endpoint (compatible with original ATM API)
	legacy := r.Group("/api")
	legacy.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, models.HealthResponse{
			Status:  "ok",
			Version: models.AppVersion,
			Port:    a.cfg.Server.Port,
		})
	})

	// Legacy pool status endpoint (compatible with original ATM API)
	r.GET("/pool/status", func(c *gin.Context) {
		if a.codexProxy == nil {
			c.JSON(http.StatusOK, gin.H{
				"total_accounts":   0,
				"enabled_accounts": 0,
				"total_requests":   0,
				"accounts":         []interface{}{},
			})
			return
		}
		status := a.codexProxy.GetPoolStatus()
		c.JSON(http.StatusOK, status)
	})

	// OpenAI-compatible proxy (v1/*)
	r.Any("/v1/*path", a.proxyV1Request)

	// ChatGPT-native Codex path — used by Codex CLI when chatgpt_base_url points here.
	// CLI appends "codex/*" to chatgpt_base_url, resulting in /backend-api/codex/* paths.
	r.Any("/backend-api/codex/*path", a.proxyCodexRequest)
	r.Any("/backend-api/codex", a.proxyCodexRequest)

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
	if !allowLocalProxyFallback(c) {
		return
	}

	// API key authentication: if proxy_api_key is set, require it (same as codex proxy).
	if requiredKey, ok := storage.GetSetting("proxy_api_key"); ok && requiredKey != "" {
		token := extractBearerToken(c.GetHeader("Authorization"))
		if token != requiredKey {
			rejectUnauthorized(c)
			return
		}
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

	upstreamURL := strings.TrimRight(baseURL, "/") + c.Request.URL.Path
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

	// Copy headers except Authorization/Host; set upstream Authorization to the API account key.
	for k, vals := range c.Request.Header {
		lk := strings.ToLower(k)
		if lk == "authorization" || lk == "host" || lk == "content-length" {
			continue
		}
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Do(req)
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

func isWebSocketUpgrade(r *http.Request) bool {
	for _, v := range r.Header["Connection"] {
		if strings.EqualFold(strings.TrimSpace(v), "upgrade") {
			for _, u := range r.Header["Upgrade"] {
				if strings.EqualFold(strings.TrimSpace(u), "websocket") {
					return true
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
		WriteTimeout: 120 * time.Second,
	}

	// Start session scanner
	if a.sessionScanner != nil {
		a.sessionScanner.Start()
	}

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
	if a.sessionScanner != nil {
		a.sessionScanner.Stop()
	}

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

	if v, ok := settings["log_enabled"]; ok {
		cfg.Log.Enabled = v == "true"
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

func parseInt(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}

func conditionalLogger(cfg *config.Config) gin.HandlerFunc {
	ginLogger := gin.Logger()
	return func(c *gin.Context) {
		if cfg.Log.Enabled {
			ginLogger(c)
			return
		}
		c.Next()
	}
}

func ipBlacklistMiddleware(cfg *config.Config) gin.HandlerFunc {
	// Pre-build the blocklist set once, not per-request.
	type blacklist struct {
		enabled bool
		ips     map[string]struct{}
	}
	bl := &blacklist{
		enabled: cfg.IPBlacklist.Enabled && len(cfg.IPBlacklist.IPs) > 0,
		ips:     make(map[string]struct{}, len(cfg.IPBlacklist.IPs)),
	}
	for _, ip := range cfg.IPBlacklist.IPs {
		bl.ips[ip] = struct{}{}
	}
	return func(c *gin.Context) {
		if !bl.enabled {
			c.Next()
			return
		}
		if _, ok := bl.ips[c.ClientIP()]; ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"message": "Your IP has been blocked",
					"type":    "forbidden",
					"code":    "403",
				},
			})
			return
		}
		c.Next()
	}
}

// extractBearerToken extracts the token from an Authorization: Bearer <token> header.
func extractBearerToken(auth string) string {
	if len(auth) > 7 && (auth[:7] == "Bearer " || auth[:7] == "bearer ") {
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
