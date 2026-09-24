package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"easyllm/internal/models"
	"easyllm/internal/proxy"
	"easyllm/internal/storage"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupProxyAccessTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.AppSettings{}); err != nil {
		t.Fatalf("migrate app settings: %v", err)
	}
	storage.DB = db
}

func TestSummaryAccessRequiresRemoteAuthentication(t *testing.T) {
	setupProxyAccessTestDB(t)
	if err := storage.SaveSetting("auth_enabled", "true"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { storage.DB.Where("key = ?", "auth_enabled").Delete(&models.AppSettings{}) })
	r := gin.New()
	r.GET("/summary", summaryAccessMiddleware(), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	for _, tc := range []struct {
		addr   string
		status int
	}{
		{"127.0.0.1:12345", http.StatusNoContent},
		{"[::1]:12345", http.StatusNoContent},
		{"192.0.2.1:12345", http.StatusUnauthorized},
	} {
		req := httptest.NewRequest(http.MethodGet, "/summary", nil)
		req.RemoteAddr = tc.addr
		req.Header.Set("X-Forwarded-For", "127.0.0.1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != tc.status {
			t.Fatalf("address %s: got %d, want %d", tc.addr, w.Code, tc.status)
		}
	}
}

func TestAllowLocalProxyFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupProxyAccessTestDB(t)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest("GET", "/v1/chat/completions", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	ctx.Request = req
	if !allowLocalProxyFallback(ctx) {
		t.Fatalf("expected loopback request to be allowed")
	}

	recorder = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(recorder)
	req = httptest.NewRequest("GET", "/v1/chat/completions", nil)
	req.RemoteAddr = "10.0.0.12:12345"
	ctx.Request = req
	if allowLocalProxyFallback(ctx) {
		t.Fatalf("expected remote request without proxy_api_key to be rejected")
	}
	if recorder.Code != 401 {
		t.Fatalf("expected status 401, got %d", recorder.Code)
	}

	if err := storage.SaveSetting("proxy_api_key", "secret"); err != nil {
		t.Fatalf("save proxy_api_key: %v", err)
	}

	recorder = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(recorder)
	req = httptest.NewRequest("GET", "/v1/chat/completions", nil)
	req.RemoteAddr = "10.0.0.12:12345"
	ctx.Request = req
	if !allowLocalProxyFallback(ctx) {
		t.Fatalf("expected remote request with proxy_api_key configured to be allowed")
	}
}

func TestAuthorizeProxyRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupProxyAccessTestDB(t)
	if err := storage.SaveSetting("proxy_api_key", "secret"); err != nil {
		t.Fatalf("save proxy_api_key: %v", err)
	}

	tests := []struct {
		name       string
		authHeader string
		allowToken func(string) bool
		want       bool
	}{
		{name: "missing token", want: false},
		{name: "wrong token", authHeader: "Bearer wrong", want: false},
		{name: "proxy key", authHeader: "Bearer secret", want: true},
		{
			name:       "managed token",
			authHeader: "Bearer managed",
			allowToken: func(token string) bool { return token == "managed" },
			want:       true,
		},
	}

	app := &App{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			req := httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			ctx.Request = req

			if got := app.authorizeProxyRequest(ctx, tc.allowToken); got != tc.want {
				t.Fatalf("authorizeProxyRequest() = %v, want %v", got, tc.want)
			}
			if !tc.want && recorder.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestAllowedBrowserOrigin(t *testing.T) {
	tests := []struct {
		origin string
		want   bool
	}{
		{origin: "http://localhost:8022", want: true},
		{origin: "https://LOCALHOST:5180", want: true},
		{origin: "http://127.0.0.1:5180", want: true},
		{origin: "http://[::1]:8022", want: true},
		{origin: "https://example.com", want: false},
		{origin: "http://localhost.example.com", want: false},
		{origin: "null", want: false},
		{origin: "file:///tmp/index.html", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.origin, func(t *testing.T) {
			if got := isAllowedBrowserOrigin(tc.origin); got != tc.want {
				t.Fatalf("isAllowedBrowserOrigin(%q) = %v, want %v", tc.origin, got, tc.want)
			}
		})
	}
}

func TestLocalCORSMiddlewareRejectsExternalOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handled := false
	router := gin.New()
	router.Use(cors.New(localCORSConfig()))
	router.POST("/api/v1/auth/setup", func(c *gin.Context) {
		handled = true
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/setup", nil)
	req.Host = "127.0.0.1:8022"
	req.Header.Set("Origin", "https://example.com")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
	if handled {
		t.Fatal("expected disallowed cross-origin request to be aborted before the handler")
	}
}

func TestApplyAPIAccountAuthHeaders(t *testing.T) {
	req, err := http.NewRequest("POST", "https://example.com/v1/chat/completions", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	applyAPIAccountAuthHeaders(req, strPtr("openai"), "secret-key")
	if got := req.Header.Get("Authorization"); got != "Bearer secret-key" {
		t.Fatalf("Authorization = %q, want Bearer secret-key", got)
	}
	if got := req.Header.Get("api-key"); got != "" {
		t.Fatalf("api-key = %q, want empty", got)
	}
}

func TestBuildAPIAccountUpstreamURL(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		requestPath string
		want        string
	}{
		{
			name:        "base includes v1",
			baseURL:     "https://api.example.com/v1",
			requestPath: "/v1/chat/completions",
			want:        "https://api.example.com/v1/chat/completions",
		},
		{
			name:        "base without v1",
			baseURL:     "https://api.example.com",
			requestPath: "/v1/chat/completions",
			want:        "https://api.example.com/v1/chat/completions",
		},
		{
			name:        "trailing slash",
			baseURL:     "https://api.example.com/v1/",
			requestPath: "v1/models",
			want:        "https://api.example.com/v1/models",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildAPIAccountUpstreamURL(tc.baseURL, tc.requestPath); got != tc.want {
				t.Fatalf("buildAPIAccountUpstreamURL() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestShouldRouteV1ResponsesToCodexProxy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupProxyAccessTestDB(t)

	if err := storage.SaveSetting("codex_local_access_enabled", "true"); err != nil {
		t.Fatalf("save codex_local_access_enabled: %v", err)
	}
	if err := storage.SaveSetting("v1_proxy_mode", "codex"); err != nil {
		t.Fatalf("save v1_proxy_mode: %v", err)
	}
	if err := storage.SaveSetting("proxy_api_key", "easyllm_codex_test"); err != nil {
		t.Fatalf("save proxy_api_key: %v", err)
	}

	app := &App{codexProxy: &proxy.CodexProxy{}}
	app.codexProxy.SetEnabled(true)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	req.Header.Set("Authorization", "Bearer easyllm_codex_test")
	req.RemoteAddr = "127.0.0.1:12345"
	ctx.Request = req
	if !app.shouldRouteV1ResponsesToCodexProxy(ctx) {
		t.Fatalf("expected matching proxy_api_key to route to codex proxy")
	}

	recorder = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(recorder)
	req = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	ctx.Request = req
	if app.shouldRouteV1ResponsesToCodexProxy(ctx) {
		t.Fatalf("expected unauthenticated loopback request not to route to codex proxy when proxy_api_key is set")
	}

	if err := storage.SaveSetting("proxy_api_key", ""); err != nil {
		t.Fatalf("clear proxy_api_key: %v", err)
	}
	recorder = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(recorder)
	req = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	ctx.Request = req
	if !app.shouldRouteV1ResponsesToCodexProxy(ctx) {
		t.Fatalf("expected loopback request to route to codex proxy when proxy_api_key is unset")
	}

	if err := storage.SaveSetting("v1_proxy_mode", ""); err != nil {
		t.Fatalf("clear v1_proxy_mode: %v", err)
	}
	recorder = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(recorder)
	req = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	req.Header.Set("Authorization", "Bearer easyllm_codex_test")
	req.RemoteAddr = "127.0.0.1:12345"
	ctx.Request = req
	if app.shouldRouteV1ResponsesToCodexProxy(ctx) {
		t.Fatalf("expected relay mode when v1_proxy_mode is not codex")
	}
}

func TestIsLoopbackRemoteAddr(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{addr: "127.0.0.1:8000", want: true},
		{addr: "[::1]:8000", want: true},
		{addr: "10.0.0.8:8000", want: false},
		{addr: "invalid-address", want: false},
	}

	for _, tc := range tests {
		if got := isLoopbackRemoteAddr(tc.addr); got != tc.want {
			t.Fatalf("isLoopbackRemoteAddr(%q) = %v, want %v", tc.addr, got, tc.want)
		}
	}
}

func TestIsWebSocketUpgradeAcceptsConnectionTokenList(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/backend-api/codex/ws", nil)
	req.Header.Set("Connection", "keep-alive, Upgrade")
	req.Header.Set("Upgrade", "websocket")

	if !isWebSocketUpgrade(req) {
		t.Fatalf("expected comma-separated Connection header to be treated as websocket upgrade")
	}
}

func strPtr(value string) *string {
	return &value
}
