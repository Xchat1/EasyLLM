package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"easyllm/internal/cursor"
	"easyllm/internal/models"
	"easyllm/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CursorHandler struct {
	storage *storage.CursorStorage
}

func NewCursorHandler(s *storage.CursorStorage) *CursorHandler {
	return &CursorHandler{storage: s}
}

func (h *CursorHandler) RegisterRoutes(rg *gin.RouterGroup) {
	cr := rg.Group("/cursor")
	{
		cr.GET("/accounts", h.ListAccounts)
		cr.POST("/accounts", h.CreateAccount)
		cr.GET("/accounts/:id", h.GetAccount)
		cr.PUT("/accounts/:id", h.UpdateAccount)
		cr.DELETE("/accounts/:id", h.DeleteAccount)
		cr.DELETE("/accounts", h.DeleteManyAccounts)
		cr.POST("/accounts/:id/activate", h.ActivateAccount)
		cr.POST("/accounts/:id/refresh", h.RefreshAccount)
		cr.POST("/accounts/refresh-all", h.RefreshAllAccounts)
		cr.POST("/accounts/:id/wakeup", h.WakeupAccount)

		cr.POST("/detect-local", h.DetectLocalAccount)
		cr.POST("/accounts/import", h.ImportAccounts)
		cr.GET("/accounts/export", h.ExportAccounts)
	}
}

func (h *CursorHandler) ListAccounts(c *gin.Context) {
	accounts, err := h.storage.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{
			Error: "获取 Cursor 账号列表失败: " + err.Error(),
			Code:  "STORAGE_ERROR",
		})
		return
	}
	c.JSON(http.StatusOK, accounts)
}

func (h *CursorHandler) GetAccount(c *gin.Context) {
	id := c.Param("id")
	account, err := h.storage.Get(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, models.APIError{Error: "账号不存在", Code: "NOT_FOUND"})
			return
		}
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, account)
}

type CreateCursorAccountInput struct {
	Email        string  `json:"email" binding:"required"`
	SessionToken string  `json:"session_token" binding:"required"`
	AccessToken  string  `json:"access_token,omitempty"`
	RefreshToken string  `json:"refresh_token,omitempty"`
	DisplayName  *string `json:"display_name,omitempty"`
	TagName      *string `json:"tag_name,omitempty"`
	TagColor     *string `json:"tag_color,omitempty"`
	Notes        *string `json:"notes,omitempty"`
}

func (h *CursorHandler) CreateAccount(c *gin.Context) {
	var in CreateCursorAccountInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{
			Error: "参数校验失败: " + err.Error(),
			Code:  "INVALID_INPUT",
		})
		return
	}

	email := strings.TrimSpace(in.Email)
	sessionToken := strings.TrimSpace(in.SessionToken)
	accessToken := strings.TrimSpace(in.AccessToken)
	if accessToken == "" && strings.HasPrefix(sessionToken, "eyJ") {
		accessToken = sessionToken
	}

	account := &models.CursorAccount{
		ID:                 uuid.New().String(),
		Email:              email,
		DisplayName:        in.DisplayName,
		SessionToken:       sessionToken,
		AccessToken:        accessToken,
		RefreshToken:       strings.TrimSpace(in.RefreshToken),
		MembershipType:     "pro",
		SubscriptionStatus: "active",
		Status:             "active",
		TagName:            in.TagName,
		TagColor:           in.TagColor,
		Notes:              in.Notes,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Fetch quota on creation
	_ = cursor.FetchAccountQuota(account)

	if err := h.storage.Save(account); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{
			Error: "保存 Cursor 账号失败: " + err.Error(),
			Code:  "STORAGE_ERROR",
		})
		return
	}

	c.JSON(http.StatusCreated, account)
}

type UpdateCursorAccountInput struct {
	DisplayName  *string `json:"display_name"`
	SessionToken *string `json:"session_token"`
	AccessToken  *string `json:"access_token"`
	RefreshToken *string `json:"refresh_token"`
	TagName      *string `json:"tag_name"`
	TagColor     *string `json:"tag_color"`
	Notes        *string `json:"notes"`
	Status       *string `json:"status"`
}

func (h *CursorHandler) UpdateAccount(c *gin.Context) {
	id := c.Param("id")
	account, err := h.storage.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIError{Error: "账号未找到", Code: "NOT_FOUND"})
		return
	}

	var in UpdateCursorAccountInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_INPUT"})
		return
	}

	if in.DisplayName != nil {
		account.DisplayName = in.DisplayName
	}
	if in.SessionToken != nil && strings.TrimSpace(*in.SessionToken) != "" {
		account.SessionToken = strings.TrimSpace(*in.SessionToken)
	}
	if in.AccessToken != nil {
		account.AccessToken = strings.TrimSpace(*in.AccessToken)
	}
	if in.RefreshToken != nil {
		account.RefreshToken = strings.TrimSpace(*in.RefreshToken)
	}
	if in.TagName != nil {
		account.TagName = in.TagName
	}
	if in.TagColor != nil {
		account.TagColor = in.TagColor
	}
	if in.Notes != nil {
		account.Notes = in.Notes
	}
	if in.Status != nil && strings.TrimSpace(*in.Status) != "" {
		account.Status = strings.TrimSpace(*in.Status)
	}

	if err := h.storage.Save(account); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}

	c.JSON(http.StatusOK, account)
}

func (h *CursorHandler) DeleteAccount(c *gin.Context) {
	id := c.Param("id")
	if err := h.storage.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "账号已删除"})
}

func (h *CursorHandler) DeleteManyAccounts(c *gin.Context) {
	var body struct {
		IDs []string `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_INPUT"})
		return
	}
	if len(body.IDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": true, "deleted": 0})
		return
	}
	if err := h.storage.DeleteMany(body.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "deleted": len(body.IDs)})
}

func (h *CursorHandler) ActivateAccount(c *gin.Context) {
	id := c.Param("id")
	account, err := h.storage.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIError{Error: "账号不存在", Code: "NOT_FOUND"})
		return
	}

	if err := h.storage.SetActive(id); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: "设置生效账号失败: " + err.Error(), Code: "STORAGE_ERROR"})
		return
	}

	// Sync to local Cursor IDE database if available
	syncedLocal := false
	if err := cursor.SyncToLocalCursor(account); err == nil {
		syncedLocal = true
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"message":      "账号已设为当前激活",
		"synced_local": syncedLocal,
	})
}

func (h *CursorHandler) RefreshAccount(c *gin.Context) {
	id := c.Param("id")
	account, err := h.storage.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIError{Error: "账号未找到", Code: "NOT_FOUND"})
		return
	}

	if err := cursor.FetchAccountQuota(account); err != nil {
		_ = h.storage.Save(account)
		c.JSON(http.StatusBadRequest, models.APIError{
			Error: "刷新 Cursor 配额失败: " + err.Error(),
			Code:  "REFRESH_FAILED",
		})
		return
	}

	if err := h.storage.Save(account); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}

	c.JSON(http.StatusOK, account)
}

func (h *CursorHandler) RefreshAllAccounts(c *gin.Context) {
	accounts, err := h.storage.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	updated := 0
	failed := 0

	for i := range accounts {
		wg.Add(1)
		go func(acc *models.CursorAccount) {
			defer wg.Done()
			if err := cursor.FetchAccountQuota(acc); err != nil {
				mu.Lock()
				failed++
				mu.Unlock()
			} else {
				mu.Lock()
				updated++
				mu.Unlock()
			}
			_ = h.storage.Save(acc)
		}(&accounts[i])
	}
	wg.Wait()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"total":   len(accounts),
		"updated": updated,
		"failed":  failed,
	})
}

func (h *CursorHandler) DetectLocalAccount(c *gin.Context) {
	detected, err := cursor.DetectLocalCursorAccount()
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{
			Error: err.Error(),
			Code:  "DETECT_FAILED",
		})
		return
	}

	// Check if already in database by email
	existing, _ := h.storage.GetByEmail(detected.Email)
	if existing != nil {
		existing.SessionToken = detected.SessionToken
		existing.AccessToken = detected.AccessToken
		existing.RefreshToken = detected.RefreshToken
		existing.MembershipType = detected.MembershipType
		existing.PlanRemaining = detected.PlanRemaining
		existing.PlanLimit = detected.PlanLimit
		existing.PlanUsed = detected.PlanUsed
		existing.PlanPercentage = detected.PlanPercentage
		existing.OnDemandUsedCents = detected.OnDemandUsedCents
		existing.BillingCycleStart = detected.BillingCycleStart
		existing.BillingCycleEnd = detected.BillingCycleEnd
		existing.Status = "active"
		now := time.Now()
		existing.LastRefreshAt = &now
		if err := h.storage.Save(existing); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"action":  "updated",
			"account": existing,
			"message": fmt.Sprintf("已成功同步本地 Cursor 账号: %s", existing.Email),
		})
		return
	}

	detected.Active = true
	if err := h.storage.Save(detected); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	_ = h.storage.SetActive(detected.ID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"action":  "created",
		"account": detected,
		"message": fmt.Sprintf("已成功导入本地 Cursor 账号: %s", detected.Email),
	})
}

func (h *CursorHandler) WakeupAccount(c *gin.Context) {
	id := c.Param("id")
	account, err := h.storage.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIError{Error: "账号未找到", Code: "NOT_FOUND"})
		return
	}

	res, err := cursor.Wakeup(account)
	_ = h.storage.Save(account)
	if err != nil {
		c.JSON(http.StatusBadRequest, res)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *CursorHandler) ImportAccounts(c *gin.Context) {
	var rawList []json.RawMessage
	if err := c.ShouldBindJSON(&rawList); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "JSON 格式必须为数组", Code: "INVALID_INPUT"})
		return
	}

	imported := 0
	for _, raw := range rawList {
		var full models.CursorAccount
		if err := json.Unmarshal(raw, &full); err == nil && full.Email != "" && full.SessionToken != "" {
			if full.ID == "" {
				full.ID = uuid.New().String()
			}
			full.Status = "active"
			_ = cursor.FetchAccountQuota(&full)
			_ = h.storage.Save(&full)
			imported++
			continue
		}

		var simple map[string]interface{}
		if err := json.Unmarshal(raw, &simple); err == nil {
			email, _ := simple["email"].(string)
			token, _ := simple["session_token"].(string)
			if token == "" {
				token, _ = simple["token"].(string)
			}
			if token == "" {
				token, _ = simple["sessionToken"].(string)
			}
			if token != "" {
				if email == "" {
					email = fmt.Sprintf("cursor-%s@user.com", uuid.New().String()[:6])
				}
				newAcc := &models.CursorAccount{
					ID:           uuid.New().String(),
					Email:        email,
					SessionToken: token,
					Status:       "active",
				}
				_ = cursor.FetchAccountQuota(newAcc)
				_ = h.storage.Save(newAcc)
				imported++
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "imported": imported})
}

func (h *CursorHandler) ExportAccounts(c *gin.Context) {
	accounts, err := h.storage.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.Header("Content-Disposition", "attachment; filename=cursor-accounts.json")
	c.JSON(http.StatusOK, accounts)
}
