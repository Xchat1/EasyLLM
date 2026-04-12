package handlers

import (
	"net/http"

	"easyllm/internal/models"
	"easyllm/internal/storage"

	"github.com/gin-gonic/gin"
)

// AugmentHandler 占位实现，避免本裁剪树缺失 Augment 源码时无法编译；接口返回 501。
type AugmentHandler struct {
	store *storage.AugmentStorage
}

// NewAugmentHandler creates a stub Augment handler.
func NewAugmentHandler(s *storage.AugmentStorage) *AugmentHandler {
	return &AugmentHandler{store: s}
}

// RegisterRoutes registers augment API routes (stub).
func (h *AugmentHandler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/augment")
	g.GET("/tokens", h.notImplemented)
}

func (h *AugmentHandler) notImplemented(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, models.APIError{
		Error: "Augment 模块在此构建中未启用",
		Code:  "NOT_IMPLEMENTED",
	})
}

// ImportSession legacy stub.
func (h *AugmentHandler) ImportSession(c *gin.Context) {
	h.notImplemented(c)
}

// ImportSessions legacy stub.
func (h *AugmentHandler) ImportSessions(c *gin.Context) {
	h.notImplemented(c)
}
