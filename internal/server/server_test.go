package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"easyllm/internal/models"
	"easyllm/internal/storage"

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
