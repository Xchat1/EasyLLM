package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"easyllm/internal/antigravity"
	"easyllm/internal/models"
	"easyllm/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AntigravityHandler struct {
	storage                 *storage.AntigravityStorage
	injectAccountToLocalIDE func(*models.AntigravityAccount) ([]string, error)
	injectAccountToAG2      func(*models.AntigravityAccount) ([]string, error)
	syncAccountToLocalCLI   func(*models.AntigravityAccount) error
}

func NewAntigravityHandler(s *storage.AntigravityStorage) *AntigravityHandler {
	return &AntigravityHandler{
		storage:                 s,
		injectAccountToLocalIDE: antigravity.InjectAccountToLocalIDE,
		injectAccountToAG2:      antigravity.InjectAccountToLocalAntigravity2,
		syncAccountToLocalCLI:   antigravity.SyncAccountToLocalCLI,
	}
}

func (h *AntigravityHandler) RegisterRoutes(rg *gin.RouterGroup) {
	ag := rg.Group("/antigravity")
	{
		ag.GET("/accounts", h.ListAccounts)
		ag.POST("/accounts", h.CreateAccount)
		ag.GET("/accounts/:id", h.GetAccount)
		ag.PUT("/accounts/:id", h.UpdateAccount)
		ag.DELETE("/accounts/:id", h.DeleteAccount)
		ag.DELETE("/accounts", h.DeleteManyAccounts)
		ag.POST("/accounts/:id/activate", h.ActivateAccount)
		ag.POST("/accounts/:id/refresh", h.RefreshAccount)
		ag.POST("/accounts/refresh-all", h.RefreshAllAccounts)
		ag.POST("/accounts/:id/wakeup", h.WakeupAccount)

		ag.POST("/oauth/start", h.StartOAuth)
		ag.POST("/oauth/complete", h.CompleteOAuth)
		ag.POST("/oauth/submit-callback", h.SubmitOAuthCallback)
		ag.POST("/oauth/cancel", h.CancelOAuth)

		ag.POST("/accounts/import", h.ImportAccounts)
		ag.GET("/accounts/export", h.ExportAccounts)
	}
}

func (h *AntigravityHandler) ListAccounts(c *gin.Context) {
	accounts, err := h.storage.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{
			Error: "Failed to list accounts: " + err.Error(),
			Code:  "STORAGE_ERROR",
		})
		return
	}
	c.JSON(http.StatusOK, accounts)
}

func (h *AntigravityHandler) GetAccount(c *gin.Context) {
	id := c.Param("id")
	account, err := h.storage.Get(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, models.APIError{
				Error: "Account not found",
				Code:  "NOT_FOUND",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.APIError{
			Error: err.Error(),
			Code:  "STORAGE_ERROR",
		})
		return
	}
	c.JSON(http.StatusOK, account)
}

type CreateAccountInput struct {
	Email        string  `json:"email" binding:"required"`
	RefreshToken string  `json:"refresh_token"`
	AccessToken  string  `json:"access_token"`
	DisplayName  *string `json:"display_name,omitempty"`
	TagName      *string `json:"tag_name,omitempty"`
	TagColor     *string `json:"tag_color,omitempty"`
	Notes        *string `json:"notes,omitempty"`
}

func (h *AntigravityHandler) CreateAccount(c *gin.Context) {
	var in CreateAccountInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{
			Error: "Invalid input: " + err.Error(),
			Code:  "INVALID_INPUT",
		})
		return
	}

	email := strings.TrimSpace(in.Email)
	refreshToken := strings.TrimSpace(in.RefreshToken)
	accessToken := strings.TrimSpace(in.AccessToken)

	if refreshToken == "" && accessToken == "" {
		c.JSON(http.StatusBadRequest, models.APIError{
			Error: "Either refresh_token or access_token must be provided",
			Code:  "INVALID_INPUT",
		})
		return
	}

	// If refresh token is provided, try to fetch access token
	now := time.Now().Unix()
	expiresIn := int64(3600)
	if refreshToken != "" && accessToken == "" {
		tRes, err := antigravity.RefreshAccessToken(refreshToken)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.APIError{
				Error: "Failed to exchange refresh token: " + err.Error(),
				Code:  "REFRESH_FAILED",
			})
			return
		}
		accessToken = tRes.AccessToken
		expiresIn = tRes.ExpiresIn
	}

	account := &models.AntigravityAccount{
		ID:              uuid.New().String(),
		Email:           email,
		DisplayName:     in.DisplayName,
		AccessToken:     accessToken,
		RefreshToken:    refreshToken,
		ExpiresIn:       expiresIn,
		ExpiryTimestamp: now + expiresIn,
		TagName:         in.TagName,
		TagColor:        in.TagColor,
		Notes:           in.Notes,
		Status:          "active",
	}

	// Try fetching quota & userinfo asynchronously or inline
	_ = antigravity.RefreshAccountQuota(account)

	if err := h.storage.Save(account); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{
			Error: "Failed to save account: " + err.Error(),
			Code:  "STORAGE_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, account)
}

type UpdateAccountInput struct {
	DisplayName *string `json:"display_name"`
	TagName     *string `json:"tag_name"`
	TagColor    *string `json:"tag_color"`
	Notes       *string `json:"notes"`
}

func (h *AntigravityHandler) UpdateAccount(c *gin.Context) {
	id := c.Param("id")
	account, err := h.storage.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIError{Error: "Account not found", Code: "NOT_FOUND"})
		return
	}

	var in UpdateAccountInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_INPUT"})
		return
	}

	if in.DisplayName != nil {
		account.DisplayName = in.DisplayName
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

	if err := h.storage.Save(account); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, account)
}

func (h *AntigravityHandler) DeleteAccount(c *gin.Context) {
	id := c.Param("id")
	if err := h.storage.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

type DeleteManyInput struct {
	IDs []string `json:"ids" binding:"required"`
}

func (h *AntigravityHandler) DeleteManyAccounts(c *gin.Context) {
	var in DeleteManyInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_INPUT"})
		return
	}
	if err := h.storage.DeleteMany(in.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "deleted": len(in.IDs)})
}

func (h *AntigravityHandler) ActivateAccount(c *gin.Context) {
	id := c.Param("id")
	account, err := h.storage.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIError{Error: "Account not found", Code: "NOT_FOUND"})
		return
	}

	// 1. Mark as active in DB
	if err := h.storage.SetActive(id); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	account.Active = true

	// 2. Sync the legacy IDE, Antigravity 2.0 desktop, and the agy CLI OAuth file.
	idePaths, ideErr := h.injectAccountToLocalIDE(account)
	antigravity2Paths, antigravity2Err := h.injectAccountToAG2(account)
	cliErr := h.syncAccountToLocalCLI(account)
	injectedPaths := append(append([]string{}, idePaths...), antigravity2Paths...)
	msg := fmt.Sprintf("已激活账号: %s", account.Email)
	if len(idePaths) > 0 {
		msg += fmt.Sprintf("，已同步 %d 个 Antigravity IDE 数据库", len(idePaths))
	}
	if ideErr != nil {
		msg += fmt.Sprintf("；IDE 同步失败: %s", ideErr.Error())
	}
	if len(antigravity2Paths) > 0 {
		msg += fmt.Sprintf("，已同步 %d 个 Antigravity 2.0 数据库", len(antigravity2Paths))
	}
	if antigravity2Err != nil {
		msg += fmt.Sprintf("；Antigravity 2.0 同步失败: %s", antigravity2Err.Error())
	} else if len(antigravity2Paths) == 0 {
		msg += "；未检测到 Antigravity 2.0 本地数据库"
	}
	if cliErr == nil {
		msg += "，已同步 agy CLI"
	} else {
		msg += fmt.Sprintf("；agy CLI 同步失败: %s", cliErr.Error())
	}
	cliSyncError := ""
	if cliErr != nil {
		cliSyncError = cliErr.Error()
	}
	antigravity2SyncError := ""
	if antigravity2Err != nil {
		antigravity2SyncError = antigravity2Err.Error()
	}
	syncWarning := ideErr != nil || antigravity2Err != nil || cliErr != nil

	c.JSON(http.StatusOK, gin.H{
		"success":                      true,
		"message":                      msg,
		"injected_paths":               injectedPaths,
		"ide_injected_paths":           idePaths,
		"antigravity_2_detected":       len(antigravity2Paths) > 0 || antigravity2Err != nil,
		"antigravity_2_synced":         len(antigravity2Paths) > 0 && antigravity2Err == nil,
		"antigravity_2_injected_paths": antigravity2Paths,
		"antigravity_2_sync_error":     antigravity2SyncError,
		"cli_synced":                   cliErr == nil,
		"cli_sync_error":               cliSyncError,
		"sync_warning":                 syncWarning,
		"account":                      account,
	})
}

func (h *AntigravityHandler) RefreshAccount(c *gin.Context) {
	id := c.Param("id")
	account, err := h.storage.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIError{Error: "Account not found", Code: "NOT_FOUND"})
		return
	}

	if err := antigravity.RefreshAccountQuota(account); err != nil {
		_ = h.storage.Save(account)
		c.JSON(http.StatusBadRequest, models.APIError{
			Error: "刷新配额失败: " + err.Error(),
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

func (h *AntigravityHandler) RefreshAllAccounts(c *gin.Context) {
	accounts, err := h.storage.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}

	successCount := 0
	failCount := 0
	for i := range accounts {
		if err := antigravity.RefreshAccountQuota(&accounts[i]); err != nil {
			failCount++
		} else {
			successCount++
		}
		_ = h.storage.Save(&accounts[i])
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"total":   len(accounts),
		"updated": successCount,
		"failed":  failCount,
	})
}

func (h *AntigravityHandler) WakeupAccount(c *gin.Context) {
	id := c.Param("id")
	account, err := h.storage.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIError{Error: "Account not found", Code: "NOT_FOUND"})
		return
	}

	var req models.AntigravityWakeupRequest
	_ = c.ShouldBindJSON(&req)

	res, err := antigravity.ExecuteWakeup(c.Request.Context(), account, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{
			Error: "唤醒调用失败: " + err.Error(),
			Code:  "WAKEUP_FAILED",
		})
		return
	}

	c.JSON(http.StatusOK, res)
}

// OAuth Handlers

func (h *AntigravityHandler) StartOAuth(c *gin.Context) {
	state, err := antigravity.StartOAuthFlow()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{
			Error: "无法启动 OAuth 监听: " + err.Error(),
			Code:  "OAUTH_START_FAILED",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"login_id":      state.LoginID,
		"auth_url":      state.AuthURL,
		"callback_url":  state.CallbackURL,
		"callback_port": state.CallbackPort,
	})
}

type CompleteOAuthInput struct {
	LoginID    string `json:"login_id" binding:"required"`
	TimeoutSec int    `json:"timeout_sec"`
}

func (h *AntigravityHandler) CompleteOAuth(c *gin.Context) {
	var in CompleteOAuthInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_INPUT"})
		return
	}

	token, userInfo, err := antigravity.CompleteOAuthFlow(in.LoginID, in.TimeoutSec)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{
			Error: err.Error(),
			Code:  "OAUTH_COMPLETE_FAILED",
		})
		return
	}

	now := time.Now().Unix()
	refreshToken := ""
	if token.RefreshToken != nil {
		refreshToken = *token.RefreshToken
	}

	// Check if account already exists by email
	existing, _ := h.storage.GetByEmail(userInfo.Email)
	var account *models.AntigravityAccount
	if existing != nil {
		account = existing
		account.AccessToken = token.AccessToken
		if refreshToken != "" {
			account.RefreshToken = refreshToken
		}
		account.ExpiresIn = token.ExpiresIn
		account.ExpiryTimestamp = now + token.ExpiresIn
		if token.IDToken != nil && *token.IDToken != "" {
			account.IDToken = token.IDToken
		}
	} else {
		displayName := userInfo.DisplayName()
		account = &models.AntigravityAccount{
			ID:              uuid.New().String(),
			Email:           userInfo.Email,
			DisplayName:     &displayName,
			Picture:         userInfo.Picture,
			AccessToken:     token.AccessToken,
			RefreshToken:    refreshToken,
			IDToken:         token.IDToken,
			ExpiresIn:       token.ExpiresIn,
			ExpiryTimestamp: now + token.ExpiresIn,
			Status:          "active",
		}
	}

	// Refresh quota immediately
	_ = antigravity.RefreshAccountQuota(account)
	_ = h.storage.Save(account)

	c.JSON(http.StatusOK, account)
}

type SubmitCallbackInput struct {
	LoginID     string `json:"login_id" binding:"required"`
	CallbackURL string `json:"callback_url" binding:"required"`
}

func (h *AntigravityHandler) SubmitOAuthCallback(c *gin.Context) {
	var in SubmitCallbackInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_INPUT"})
		return
	}
	if err := antigravity.SubmitManualCallbackURL(in.LoginID, in.CallbackURL); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_CALLBACK"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

type CancelOAuthInput struct {
	LoginID string `json:"login_id"`
}

func (h *AntigravityHandler) CancelOAuth(c *gin.Context) {
	var in CancelOAuthInput
	_ = c.ShouldBindJSON(&in)
	_ = antigravity.CancelOAuthFlow(in.LoginID)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Import and Export Handlers

func (h *AntigravityHandler) ImportAccounts(c *gin.Context) {
	var rawItems []json.RawMessage
	if err := c.ShouldBindJSON(&rawItems); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "Invalid JSON format: " + err.Error(), Code: "INVALID_JSON"})
		return
	}

	imported := 0
	for _, raw := range rawItems {
		var acc models.AntigravityAccount
		if err := json.Unmarshal(raw, &acc); err == nil && acc.Email != "" && (acc.RefreshToken != "" || acc.AccessToken != "") {
			if acc.ID == "" {
				acc.ID = uuid.New().String()
			}
			if acc.Status == "" {
				acc.Status = "active"
			}
			_ = antigravity.RefreshAccountQuota(&acc)
			_ = h.storage.Save(&acc)
			imported++
			continue
		}

		// Try parsing simple string or { "refresh_token": "..." } or { "email": "...", "refresh_token": "..." }
		var simple map[string]interface{}
		if err := json.Unmarshal(raw, &simple); err == nil {
			email, _ := simple["email"].(string)
			rf, _ := simple["refresh_token"].(string)
			at, _ := simple["access_token"].(string)
			if rf != "" || at != "" {
				if email == "" {
					email = fmt.Sprintf("imported-%s@example.com", uuid.New().String()[:6])
				}
				newAcc := &models.AntigravityAccount{
					ID:           uuid.New().String(),
					Email:        email,
					RefreshToken: rf,
					AccessToken:  at,
					Status:       "active",
				}
				_ = antigravity.RefreshAccountQuota(newAcc)
				_ = h.storage.Save(newAcc)
				imported++
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "imported": imported})
}

func (h *AntigravityHandler) ExportAccounts(c *gin.Context) {
	accounts, err := h.storage.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.Header("Content-Disposition", "attachment; filename=antigravity-accounts.json")
	c.JSON(http.StatusOK, accounts)
}

func formatResetTimeStr(resetTime string, percentage int) string {
	if percentage >= 100 {
		return "额度满额"
	}
	if resetTime == "" {
		return "实时周期"
	}
	t, err := time.Parse(time.RFC3339, resetTime)
	if err != nil {
		return resetTime
	}
	now := time.Now()
	diff := t.Sub(now)
	if diff <= 0 {
		return "已到重置点"
	}
	mins := int(diff.Minutes())
	hours := int(diff.Hours())
	timeStr := t.Local().Format("15:04")
	if mins < 60 {
		return fmt.Sprintf("%dm (%s)", mins, timeStr)
	}
	if hours < 24 {
		return fmt.Sprintf("%dh %dm (%s)", hours, mins%60, timeStr)
	}
	days := hours / 24
	remHours := hours % 24
	return fmt.Sprintf("%d天 %dh (%s)", days, remHours, timeStr)
}

func (h *AntigravityHandler) buildSummary(account *models.AntigravityAccount) models.AntigravitySummaryResponse {
	if account == nil {
		return models.AntigravitySummaryResponse{
			HasAccount:  false,
			StatusText:  "✦ EasyLLM",
			TooltipText: "EasyLLM: 暂无 Antigravity 账号",
		}
	}

	res := models.AntigravitySummaryResponse{
		HasAccount: true,
		AccountID:  account.ID,
		Email:      account.Email,
		Status:     account.Status,
	}

	if account.DisplayName != nil && *account.DisplayName != "" {
		res.DisplayName = *account.DisplayName
	} else {
		res.DisplayName = account.Email
	}

	if account.SubscriptionTier != nil {
		res.SubscriptionTier = *account.SubscriptionTier
	}

	if account.LastRefreshAt != nil {
		res.LastRefreshAt = account.LastRefreshAt.Local().Format("15:04:05")
	}

	if account.Status == "forbidden" {
		res.StatusText = "⚠️ 403受限"
		res.TooltipText = fmt.Sprintf("EasyLLM: 账号 %s 访问受限 (403)", res.DisplayName)
		return res
	}
	if account.Status == "expired" {
		res.StatusText = "⚠️ Token过期"
		res.TooltipText = fmt.Sprintf("EasyLLM: 账号 %s Token 已过期", res.DisplayName)
		return res
	}

	var modelsList []models.AntigravityQuotaModel
	if account.QuotaModelsJSON != nil && *account.QuotaModelsJSON != "" {
		_ = json.Unmarshal([]byte(*account.QuotaModelsJSON), &modelsList)
	}

	for _, m := range modelsList {
		switch m.Name {
		case "3p-5h", "claude:5h":
			pct := m.Percentage
			res.Claude5h = &pct
			res.Claude5hResetFormatted = formatResetTimeStr(m.ResetTime, pct)
		case "3p-weekly", "claude:weekly":
			pct := m.Percentage
			res.ClaudeWeekly = &pct
			res.ClaudeWeeklyResetFormatted = formatResetTimeStr(m.ResetTime, pct)
		case "gemini-5h", "gemini:5h":
			pct := m.Percentage
			res.Gemini5h = &pct
			res.Gemini5hResetFormatted = formatResetTimeStr(m.ResetTime, pct)
		case "gemini-weekly", "gemini:weekly":
			pct := m.Percentage
			res.GeminiWeekly = &pct
			res.GeminiWeeklyResetFormatted = formatResetTimeStr(m.ResetTime, pct)
		}
	}

	// Status text for menu bar, e.g. "✦ C:100% · G:74%"
	var parts []string
	warning := false
	if res.Claude5h != nil {
		parts = append(parts, fmt.Sprintf("C:%d%%", *res.Claude5h))
		if *res.Claude5h <= 20 {
			warning = true
		}
	}
	if res.Gemini5h != nil {
		parts = append(parts, fmt.Sprintf("G:%d%%", *res.Gemini5h))
		if *res.Gemini5h <= 20 {
			warning = true
		}
	}

	icon := "✦ "
	if warning {
		icon = "⚠️ "
	}
	if len(parts) > 0 {
		res.StatusText = icon + strings.Join(parts, " · ")
	} else {
		res.StatusText = icon + "额度就绪"
	}

	// Detailed Tooltip
	var ttLines []string
	tier := ""
	if res.SubscriptionTier != "" {
		tier = fmt.Sprintf(" (%s)", res.SubscriptionTier)
	}
	ttLines = append(ttLines, fmt.Sprintf("EasyLLM Antigravity - %s%s", res.DisplayName, tier))
	if res.Claude5h != nil {
		ttLines = append(ttLines, fmt.Sprintf("Claude (5h): %d%% [%s]", *res.Claude5h, res.Claude5hResetFormatted))
	}
	if res.ClaudeWeekly != nil {
		ttLines = append(ttLines, fmt.Sprintf("Claude (周): %d%%", *res.ClaudeWeekly))
	}
	if res.Gemini5h != nil {
		ttLines = append(ttLines, fmt.Sprintf("Gemini (5h): %d%% [%s]", *res.Gemini5h, res.Gemini5hResetFormatted))
	}
	if res.GeminiWeekly != nil {
		ttLines = append(ttLines, fmt.Sprintf("Gemini (周): %d%%", *res.GeminiWeekly))
	}
	if res.LastRefreshAt != "" {
		ttLines = append(ttLines, fmt.Sprintf("刷新时间: %s", res.LastRefreshAt))
	}
	res.TooltipText = strings.Join(ttLines, "\n")

	return res
}

func (h *AntigravityHandler) getTargetSummaryAccount() (*models.AntigravityAccount, error) {
	acc, err := h.storage.GetActive()
	if err == nil && acc != nil {
		return acc, nil
	}
	list, err := h.storage.List()
	if err != nil {
		return nil, err
	}
	if len(list) > 0 {
		return &list[0], nil
	}
	return nil, nil
}

func (h *AntigravityHandler) GetSummary(c *gin.Context) {
	account, err := h.getTargetSummaryAccount()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	summary := h.buildSummary(account)
	c.JSON(http.StatusOK, summary)
}

func (h *AntigravityHandler) RefreshSummary(c *gin.Context) {
	account, err := h.getTargetSummaryAccount()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	if account == nil {
		c.JSON(http.StatusOK, h.buildSummary(nil))
		return
	}
	if err := antigravity.RefreshAccountQuota(account); err != nil {
		_ = h.storage.Save(account)
		c.JSON(http.StatusOK, h.buildSummary(account))
		return
	}
	_ = h.storage.Save(account)
	c.JSON(http.StatusOK, h.buildSummary(account))
}
