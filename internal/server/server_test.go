package server

import (
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
