package handlers

import (
	"easyllm/internal/models"
	"easyllm/internal/storage"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CursorHandler manages Cursor accounts
type CursorHandler struct{ storage *storage.CursorStorage }

func NewCursorHandler(s *storage.CursorStorage) *CursorHandler { return &CursorHandler{storage: s} }

func (h *CursorHandler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/cursor")
	g.GET("/accounts", h.List)
	g.POST("/accounts", h.Add)
	g.PUT("/accounts/:id", h.Update)
	g.DELETE("/accounts/:id", h.Delete)
	g.DELETE("/accounts", h.DeleteMany)
	g.POST("/accounts/:id/activate", h.Activate)
	g.POST("/import", h.Import)
}

func (h *CursorHandler) List(c *gin.Context) {
	list, err := h.storage.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *CursorHandler) Add(c *gin.Context) {
	var a models.CursorAccount
	if err := c.ShouldBindJSON(&a); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	a = normalizeCursorAccountInput(a)
	if err := validateCursorAccountInput(a); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	a.Active = false
	a.CreatedAt = time.Now()
	a.UpdatedAt = time.Now()
	if err := h.storage.Save(&a); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *CursorHandler) Update(c *gin.Context) {
	existing, err := h.storage.Get(c.Param("id"))
	if err != nil {
		status := http.StatusInternalServerError
		code := "STORAGE_ERROR"
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
			code = "NOT_FOUND"
		}
		c.JSON(status, models.APIError{Error: err.Error(), Code: code})
		return
	}
	var a models.CursorAccount
	if err := c.ShouldBindJSON(&a); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	a = normalizeCursorAccountInput(a)
	if err := validateCursorAccountInput(a); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	a = mergeCursorAccountUpdate(existing, a)
	a.ID = c.Param("id")
	a.CreatedAt = existing.CreatedAt
	a.UpdatedAt = time.Now()
	if err := h.storage.Save(&a); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *CursorHandler) Delete(c *gin.Context) {
	if err := h.storage.Delete(c.Param("id")); err != nil {
		status := http.StatusInternalServerError
		code := "STORAGE_ERROR"
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
			code = "NOT_FOUND"
		}
		c.JSON(status, models.APIError{Error: err.Error(), Code: code})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *CursorHandler) DeleteMany(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	if err := h.storage.DeleteMany(req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *CursorHandler) Activate(c *gin.Context) {
	if err := h.storage.SetActive(c.Param("id")); err != nil {
		status := http.StatusInternalServerError
		code := "STORAGE_ERROR"
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
			code = "NOT_FOUND"
		}
		c.JSON(status, models.APIError{Error: err.Error(), Code: code})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *CursorHandler) Import(c *gin.Context) {
	var accounts []models.CursorAccount
	if err := c.ShouldBindJSON(&accounts); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	imported := 0
	for i := range accounts {
		if accounts[i].ID == "" {
			accounts[i].ID = uuid.New().String()
		}
		accounts[i].CreatedAt = time.Now()
		accounts[i].UpdatedAt = time.Now()
		if err := h.storage.Save(&accounts[i]); err == nil {
			imported++
		}
	}
	c.JSON(http.StatusOK, gin.H{"imported": imported})
}

// AntigravityHandler manages Antigravity accounts
type AntigravityHandler struct{ storage *storage.AntigravityStorage }

func NewAntigravityHandler(s *storage.AntigravityStorage) *AntigravityHandler {
	return &AntigravityHandler{storage: s}
}

func (h *AntigravityHandler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/antigravity")
	g.GET("/accounts", h.List)
	g.POST("/accounts", h.Add)
	g.PUT("/accounts/:id", h.Update)
	g.DELETE("/accounts/:id", h.Delete)
	g.DELETE("/accounts", h.DeleteMany)
	g.POST("/accounts/:id/activate", h.Activate)
}

func (h *AntigravityHandler) List(c *gin.Context) {
	list, err := h.storage.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *AntigravityHandler) Add(c *gin.Context) {
	var a models.AntigravityAccount
	if err := c.ShouldBindJSON(&a); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	a = normalizeAntigravityAccountInput(a)
	if err := validateAntigravityAccountInput(a); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	a.Active = false
	a.CreatedAt = time.Now()
	a.UpdatedAt = time.Now()
	if err := h.storage.Save(&a); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *AntigravityHandler) Update(c *gin.Context) {
	existing, err := h.storage.Get(c.Param("id"))
	if err != nil {
		status := http.StatusInternalServerError
		code := "STORAGE_ERROR"
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
			code = "NOT_FOUND"
		}
		c.JSON(status, models.APIError{Error: err.Error(), Code: code})
		return
	}
	var a models.AntigravityAccount
	if err := c.ShouldBindJSON(&a); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	a = normalizeAntigravityAccountInput(a)
	if err := validateAntigravityAccountInput(a); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	a = mergeAntigravityAccountUpdate(existing, a)
	a.ID = c.Param("id")
	a.CreatedAt = existing.CreatedAt
	a.UpdatedAt = time.Now()
	if err := h.storage.Save(&a); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, a)
}

func (h *AntigravityHandler) Delete(c *gin.Context) {
	if err := h.storage.Delete(c.Param("id")); err != nil {
		status := http.StatusInternalServerError
		code := "STORAGE_ERROR"
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
			code = "NOT_FOUND"
		}
		c.JSON(status, models.APIError{Error: err.Error(), Code: code})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *AntigravityHandler) DeleteMany(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	if err := h.storage.DeleteMany(req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *AntigravityHandler) Activate(c *gin.Context) {
	if err := h.storage.SetActive(c.Param("id")); err != nil {
		status := http.StatusInternalServerError
		code := "STORAGE_ERROR"
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
			code = "NOT_FOUND"
		}
		c.JSON(status, models.APIError{Error: err.Error(), Code: code})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func normalizeCursorAccountInput(a models.CursorAccount) models.CursorAccount {
	a.Email = strings.TrimSpace(a.Email)
	a.AccessToken = strings.TrimSpace(a.AccessToken)
	if a.Name != nil {
		trimmed := strings.TrimSpace(*a.Name)
		if trimmed == "" {
			a.Name = nil
		} else {
			a.Name = &trimmed
		}
	}
	if a.TagName != nil {
		trimmed := strings.TrimSpace(*a.TagName)
		if trimmed == "" {
			a.TagName = nil
		} else {
			a.TagName = &trimmed
		}
	}
	return a
}

func validateCursorAccountInput(a models.CursorAccount) error {
	if a.Email == "" {
		return errors.New("email is required")
	}
	if a.AccessToken == "" {
		return errors.New("access_token is required")
	}
	return nil
}

func mergeCursorAccountUpdate(existing *models.CursorAccount, incoming models.CursorAccount) models.CursorAccount {
	incoming.CookieToken = existing.CookieToken
	incoming.Plan = existing.Plan
	incoming.Active = existing.Active
	return incoming
}

func normalizeAntigravityAccountInput(a models.AntigravityAccount) models.AntigravityAccount {
	a.Email = strings.TrimSpace(a.Email)
	a.AccessToken = strings.TrimSpace(a.AccessToken)
	if a.Name != nil {
		trimmed := strings.TrimSpace(*a.Name)
		if trimmed == "" {
			a.Name = nil
		} else {
			a.Name = &trimmed
		}
	}
	if a.TagName != nil {
		trimmed := strings.TrimSpace(*a.TagName)
		if trimmed == "" {
			a.TagName = nil
		} else {
			a.TagName = &trimmed
		}
	}
	return a
}

func validateAntigravityAccountInput(a models.AntigravityAccount) error {
	if a.Email == "" {
		return errors.New("email is required")
	}
	if a.AccessToken == "" {
		return errors.New("access_token is required")
	}
	return nil
}

func mergeAntigravityAccountUpdate(existing *models.AntigravityAccount, incoming models.AntigravityAccount) models.AntigravityAccount {
	incoming.Active = existing.Active
	incoming.Plan = existing.Plan
	incoming.Quota = existing.Quota
	incoming.UsedQuota = existing.UsedQuota
	return incoming
}
