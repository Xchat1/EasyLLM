package server

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestGzipMiddlewareImplementsFlusher(t *testing.T) {
	r := gin.New()
	r.Use(GzipMiddleware())

	flushed := false
	r.GET("/stream-test", func(c *gin.Context) {
		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
			return
		}
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = c.Writer.Write([]byte("data: hello\n\n"))
		flusher.Flush()
		flushed = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/stream-test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body: %s", w.Code, w.Body.String())
	}
	if !flushed {
		t.Fatalf("expected flusher.Flush() to be called successfully")
	}

	// Verify the body can be uncompressed
	gz, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gz.Close()

	decompressed, err := io.ReadAll(gz)
	if err != nil {
		t.Fatalf("failed to read decompressed body: %v", err)
	}
	if !strings.Contains(string(decompressed), "data: hello") {
		t.Fatalf("expected decompressed body to contain 'data: hello', got: %s", string(decompressed))
	}
}

func TestGzipMiddlewareSkipsTextEventStream(t *testing.T) {
	r := gin.New()
	r.Use(GzipMiddleware())

	r.GET("/sse", func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = c.Writer.Write([]byte("data: uncompressed\n\n"))
	})

	req := httptest.NewRequest(http.MethodGet, "/sse", nil)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Fatalf("expected Content-Encoding NOT to be gzip for Accept: text/event-stream")
	}
	if !strings.Contains(w.Body.String(), "data: uncompressed") {
		t.Fatalf("expected raw body, got: %s", w.Body.String())
	}
}
