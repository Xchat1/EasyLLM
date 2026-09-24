package server

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type gzipWriter struct {
	gin.ResponseWriter
	writer io.Writer
}

func (g *gzipWriter) Write(data []byte) (int, error) {
	return g.writer.Write(data)
}

func (g *gzipWriter) WriteString(s string) (int, error) {
	return g.writer.Write([]byte(s))
}

// Flush 实现 http.Flusher：先把 gzip 缓冲区里的数据刷到底层 writer，
// 再把 Flush 透传到底层 writer。没有它，流式 handler 的 Flush 只会刷
// 底层 writer，而 gzip 缓冲区的数据要等 handler 结束（gz.Close）才吐出，
// SSE/流式响应会被迫退化成“结束时一次性返回”，客户端还可能超时。
func (g *gzipWriter) Flush() {
	if gz, ok := g.writer.(*gzip.Writer); ok {
		_ = gz.Flush()
	}
	if flusher, ok := g.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// GzipMiddleware compresses responses with gzip to optimize transfer speed and performance.
func GzipMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// SSE streams must not be gzip-compressed (breaks real-time delivery).
		if strings.Contains(c.GetHeader("Accept"), "text/event-stream") {
			c.Next()
			return
		}
		if c.Request.URL.Path == "/v1/responses" {
			c.Next()
			return
		}
		// Only compress if client accepts gzip and it's not a WebSocket upgrade request
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}
		if strings.Contains(strings.ToLower(c.GetHeader("Connection")), "upgrade") || c.GetHeader("Sec-WebSocket-Key") != "" {
			c.Next()
			return
		}

		c.Header("Content-Encoding", "gzip")
		c.Header("Vary", "Accept-Encoding")

		gz, err := gzip.NewWriterLevel(c.Writer, gzip.BestSpeed)
		if err != nil {
			c.Next()
			return
		}
		defer gz.Close()

		c.Writer = &gzipWriter{ResponseWriter: c.Writer, writer: gz}
		c.Next()
	}
}
