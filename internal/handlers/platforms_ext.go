// platforms_ext.go — platform-specific API handlers for Kiro, Gemini,
// GitHub Copilot, and Antigravity.
// These handlers align EasyLLM with the cockpit-tools-main logic.
package handlers

import (
	"context"
	"crypto/sha1"
	"easyllm/internal/models"
	"easyllm/internal/platforms/antigravity"
	"easyllm/internal/platforms/gemini"
	"easyllm/internal/platforms/github"
	"easyllm/internal/platforms/kiro"
	"easyllm/internal/storage"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ============================================================
// PlatformExtHandler — shared handler for platform-specific ops
// ============================================================

type PlatformExtHandler struct {
	cockpit *storage.CockpitStorage
	dataDir string
}

func NewPlatformExtHandler(cockpit *storage.CockpitStorage, dataDir string) *PlatformExtHandler {
	return &PlatformExtHandler{cockpit: cockpit, dataDir: dataDir}
}

func (h *PlatformExtHandler) RegisterRoutes(rg *gin.RouterGroup) {
	// Kiro
	kiroG := rg.Group("/cockpit/platforms/kiro")
	kiroG.POST("/accounts/:id/refresh", h.KiroRefreshAccount)
	kiroG.POST("/oauth/start", h.KiroOAuthStart)
	kiroG.POST("/oauth/complete", h.KiroOAuthComplete)
	kiroG.POST("/oauth/cancel", h.KiroOAuthCancel)

	// Gemini
	geminiG := rg.Group("/cockpit/platforms/gemini")
	geminiG.POST("/accounts/:id/refresh", h.GeminiRefreshAccount)
	geminiG.POST("/oauth/start", h.GeminiOAuthStart)
	geminiG.POST("/oauth/complete", h.GeminiOAuthComplete)
	geminiG.POST("/oauth/submit-callback", h.GeminiOAuthSubmitCallback)
	geminiG.POST("/oauth/cancel", h.GeminiOAuthCancel)

	// GitHub Copilot
	ghcpG := rg.Group("/cockpit/platforms/github-copilot")
	ghcpG.POST("/accounts/:id/refresh", h.GitHubCopilotRefreshAccount)
	ghcpG.POST("/oauth/start", h.GitHubCopilotOAuthStart)
	ghcpG.POST("/oauth/complete", h.GitHubCopilotOAuthComplete)
	ghcpG.POST("/oauth/cancel", h.GitHubCopilotOAuthCancel)
	ghcpG.POST("/accounts/import-token", h.GitHubCopilotImportToken)

	// Antigravity
	agG := rg.Group("/cockpit/platforms/antigravity")
	agG.POST("/oauth/start", h.AntigravityOAuthStart)
	agG.POST("/oauth/complete", h.AntigravityOAuthComplete)
	agG.POST("/oauth/submit-callback", h.AntigravityOAuthSubmitCallback)
	agG.POST("/oauth/cancel", h.AntigravityOAuthCancel)
	agG.POST("/accounts/:id/wakeup", h.AntigravityWakeup)
	agG.GET("/switch-history", h.AntigravityGetSwitchHistory)
	agG.DELETE("/switch-history", h.AntigravityClearSwitchHistory)

}

// ============================================================
// Kiro handlers
// ============================================================

// KiroRefreshAccount refreshes the Kiro account token and usage quota.
// POST /api/v1/cockpit/platforms/kiro/accounts/:id/refresh
func (h *PlatformExtHandler) KiroRefreshAccount(c *gin.Context) {
	account, ok := h.loadPlatformAccount(c, "kiro")
	if !ok {
		return
	}

	meta := parseMetadataJSON(account.MetadataJSON)

	accessToken := ""
	if account.AccessToken != nil {
		accessToken = *account.AccessToken
	}
	refreshToken := ""
	if account.RefreshToken != nil {
		refreshToken = *account.RefreshToken
	}
	profileArn := stringFromMeta(meta, "profile_arn", "profileArn", "arn")
	region := stringFromMeta(meta, "idc_region", "idcRegion", "region")

	result, err := kiro.RefreshAccount(accessToken, refreshToken, profileArn, region)
	if err != nil {
		c.JSON(http.StatusBadGateway, models.APIError{Error: err.Error(), Code: "KIRO_REFRESH_ERROR"})
		return
	}

	// Update account fields
	now := time.Now()
	if result.AccessToken != "" {
		account.AccessToken = &result.AccessToken
	}
	if result.RefreshToken != nil {
		account.RefreshToken = result.RefreshToken
	}
	if result.ExpiresAt != nil {
		t := time.Unix(*result.ExpiresAt, 0)
		account.QuotaResetAt = &t
	}
	if result.CreditsTotal != nil {
		account.QuotaLimit = result.CreditsTotal
		unit := "credits"
		account.QuotaUnit = &unit
	}
	if result.CreditsUsed != nil {
		account.QuotaUsed = result.CreditsUsed
	}
	if result.PlanName != nil {
		account.Plan = result.PlanName
	}
	if result.Status != nil {
		account.Status = *result.Status
	}
	if result.Email != "" {
		account.Email = result.Email
	}

	// Merge raw usage into MetadataJSON
	if result.RawUsage != nil {
		if meta == nil {
			meta = make(map[string]interface{})
		}
		meta["kiro_usage_raw"] = result.RawUsage
		if result.RawJSON != nil {
			for k, v := range result.RawJSON {
				meta[k] = v
			}
		}
		if raw, err := json.Marshal(meta); err == nil {
			s := string(raw)
			account.MetadataJSON = &s
		}
	}

	account.UpdatedAt = now
	if err := h.cockpit.SaveAccount(account); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "account": account})
}

// KiroOAuthStart initiates the Kiro OAuth flow.
// POST /api/v1/cockpit/platforms/kiro/oauth/start
func (h *PlatformExtHandler) KiroOAuthStart(c *gin.Context) {
	resp, err := kiro.StartOAuthFlow()
	if err != nil {
		c.JSON(http.StatusBadGateway, models.APIError{Error: err.Error(), Code: "KIRO_OAUTH_START_ERROR"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// KiroOAuthComplete completes the Kiro OAuth flow and creates an account.
// POST /api/v1/cockpit/platforms/kiro/oauth/complete
func (h *PlatformExtHandler) KiroOAuthComplete(c *gin.Context) {
	var req struct {
		LoginID    string `json:"login_id"`
		TimeoutSec int    `json:"timeout_sec"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	if req.TimeoutSec <= 0 {
		req.TimeoutSec = 300
	}

	result, err := kiro.CompleteOAuthFlow(req.LoginID, req.TimeoutSec)
	if err != nil {
		c.JSON(http.StatusBadGateway, models.APIError{Error: err.Error(), Code: "KIRO_OAUTH_COMPLETE_ERROR"})
		return
	}

	// Create account
	now := time.Now()
	account := &models.PlatformAccount{
		ID:        uuid.New().String(),
		Platform:  "kiro",
		Email:     result.Email,
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if result.AccessToken != "" {
		account.AccessToken = &result.AccessToken
	}
	if result.RefreshToken != nil {
		account.RefreshToken = result.RefreshToken
	}
	if result.ExpiresAt != nil {
		t := time.Unix(*result.ExpiresAt, 0)
		account.QuotaResetAt = &t
	}
	if result.CreditsTotal != nil {
		account.QuotaLimit = result.CreditsTotal
		unit := "credits"
		account.QuotaUnit = &unit
	}
	if result.CreditsUsed != nil {
		account.QuotaUsed = result.CreditsUsed
	}
	if result.PlanName != nil {
		account.Plan = result.PlanName
	}

	// Store metadata
	if result.RawJSON != nil {
		if raw, err := json.Marshal(result.RawJSON); err == nil {
			s := string(raw)
			account.MetadataJSON = &s
		}
	}

	if err := h.cockpit.SaveAccount(account); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"account": account,
		"message": "Kiro account added successfully",
	})
}

// KiroOAuthCancel cancels the pending Kiro OAuth flow.
// POST /api/v1/cockpit/platforms/kiro/oauth/cancel
func (h *PlatformExtHandler) KiroOAuthCancel(c *gin.Context) {
	var req struct {
		LoginID string `json:"login_id"`
	}
	_ = c.ShouldBindJSON(&req)

	if err := kiro.CancelOAuthFlow(req.LoginID); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "KIRO_OAUTH_CANCEL_ERROR"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "OAuth flow cancelled"})
}

// ============================================================
// Gemini handlers
// ============================================================

// GeminiRefreshAccount refreshes the Gemini account token and quota.
// POST /api/v1/cockpit/platforms/gemini/accounts/:id/refresh
func (h *PlatformExtHandler) GeminiRefreshAccount(c *gin.Context) {
	account, ok := h.loadPlatformAccount(c, "gemini")
	if !ok {
		return
	}

	meta := parseMetadataJSON(account.MetadataJSON)

	accessToken := ""
	if account.AccessToken != nil {
		accessToken = *account.AccessToken
	}
	refreshToken := ""
	if account.RefreshToken != nil {
		refreshToken = *account.RefreshToken
	}
	// expiry_date stored in metadata as milliseconds
	var expiryDateMs int64
	if v := int64FromMeta(meta, "expiry_date"); v > 0 {
		expiryDateMs = v
	}

	result, err := gemini.RefreshAccount(accessToken, refreshToken, expiryDateMs)
	if err != nil {
		c.JSON(http.StatusBadGateway, models.APIError{Error: err.Error(), Code: "GEMINI_REFRESH_ERROR"})
		return
	}

	now := time.Now()
	if result.Token.AccessToken != "" {
		account.AccessToken = &result.Token.AccessToken
	}
	if result.Token.RefreshToken != nil {
		account.RefreshToken = result.Token.RefreshToken
	}
	if result.UserInfo != nil {
		if result.UserInfo.Email != nil && *result.UserInfo.Email != "" {
			account.Email = *result.UserInfo.Email
		}
		if result.UserInfo.Name != nil && account.DisplayName == nil {
			account.DisplayName = result.UserInfo.Name
		}
	}
	if result.CodeAssist != nil {
		if result.CodeAssist.TierName != nil {
			account.Plan = result.CodeAssist.TierName
		}
	}

	// Store quota raw + expiry in metadata
	if meta == nil {
		meta = make(map[string]interface{})
	}
	if result.Token.ExpiryDate != nil {
		meta["expiry_date"] = *result.Token.ExpiryDate
	}
	if result.CodeAssist != nil && result.CodeAssist.ProjectID != nil {
		meta["project_id"] = *result.CodeAssist.ProjectID
	}
	if result.Quota != nil {
		meta["gemini_usage_raw"] = result.Quota.Raw
		// extract quota numbers from raw if available
		applyGeminiQuotaToAccount(account, result.Quota.Raw)
	}
	if result.QuotaError != nil {
		meta["quota_query_last_error"] = *result.QuotaError
	}
	if raw, err := json.Marshal(meta); err == nil {
		s := string(raw)
		account.MetadataJSON = &s
	}

	account.UpdatedAt = now
	if err := h.cockpit.SaveAccount(account); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "account": account})
}

// applyGeminiQuotaToAccount extracts quota numbers from the raw retrieveUserQuota response.
func applyGeminiQuotaToAccount(account *models.PlatformAccount, raw map[string]interface{}) {
	if raw == nil {
		return
	}
	// The quota response structure varies; try common paths
	if quotas, ok := raw["quotas"].([]interface{}); ok {
		for _, q := range quotas {
			if qm, ok := q.(map[string]interface{}); ok {
				if limit, ok := qm["limit"].(float64); ok {
					account.QuotaLimit = &limit
				}
				if used, ok := qm["usage"].(float64); ok {
					account.QuotaUsed = &used
				}
				unit := "requests"
				account.QuotaUnit = &unit
				break
			}
		}
	}
}

// GeminiOAuthStart initiates the Gemini OAuth flow.
// POST /api/v1/cockpit/platforms/gemini/oauth/start
func (h *PlatformExtHandler) GeminiOAuthStart(c *gin.Context) {
	resp, err := gemini.StartOAuthFlow()
	if err != nil {
		c.JSON(http.StatusBadGateway, models.APIError{Error: err.Error(), Code: "GEMINI_OAUTH_START_ERROR"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GeminiOAuthComplete completes the Gemini OAuth flow and creates an account.
// POST /api/v1/cockpit/platforms/gemini/oauth/complete
func (h *PlatformExtHandler) GeminiOAuthComplete(c *gin.Context) {
	var req struct {
		LoginID    string `json:"login_id"`
		TimeoutSec int    `json:"timeout_sec"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	if req.TimeoutSec <= 0 {
		req.TimeoutSec = 300
	}

	payload, err := gemini.CompleteOAuthFlow(req.LoginID, req.TimeoutSec)
	if err != nil {
		c.JSON(http.StatusBadGateway, models.APIError{Error: err.Error(), Code: "GEMINI_OAUTH_COMPLETE_ERROR"})
		return
	}

	account, err := h.saveGeminiOAuthAccount(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"account": account,
		"message": "Gemini account added successfully",
	})
}

// GeminiOAuthSubmitCallback accepts a manually pasted Google callback URL.
// POST /api/v1/cockpit/platforms/gemini/oauth/submit-callback
func (h *PlatformExtHandler) GeminiOAuthSubmitCallback(c *gin.Context) {
	var req struct {
		LoginID     string `json:"login_id"`
		CallbackURL string `json:"callback_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	if err := gemini.SubmitOAuthCallbackURL(req.LoginID, req.CallbackURL); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "GEMINI_OAUTH_CALLBACK_ERROR"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GeminiOAuthCancel cancels the pending Gemini OAuth flow.
// POST /api/v1/cockpit/platforms/gemini/oauth/cancel
func (h *PlatformExtHandler) GeminiOAuthCancel(c *gin.Context) {
	var req struct {
		LoginID string `json:"login_id"`
	}
	_ = c.ShouldBindJSON(&req)

	if err := gemini.CancelOAuthFlow(req.LoginID); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "GEMINI_OAUTH_CANCEL_ERROR"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "OAuth flow cancelled"})
}

// ============================================================
// GitHub Copilot handlers
// ============================================================

// GitHubCopilotRefreshAccount refreshes the Copilot short-lived token.
// POST /api/v1/cockpit/platforms/github-copilot/accounts/:id/refresh
func (h *PlatformExtHandler) GitHubCopilotRefreshAccount(c *gin.Context) {
	account, ok := h.loadPlatformAccount(c, "github-copilot")
	if !ok {
		return
	}

	meta := parseMetadataJSON(account.MetadataJSON)
	githubAccessToken := stringFromMeta(meta, "github_access_token")
	if githubAccessToken == "" && account.AccessToken != nil {
		githubAccessToken = *account.AccessToken
	}
	if githubAccessToken == "" {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "github_access_token not found in account", Code: "MISSING_TOKEN"})
		return
	}

	bundle, err := github.RefreshCopilotToken(githubAccessToken)
	if err != nil {
		c.JSON(http.StatusBadGateway, models.APIError{Error: err.Error(), Code: "GHCP_REFRESH_ERROR"})
		return
	}

	now := time.Now()
	if bundle.Plan != nil {
		account.Plan = bundle.Plan
	}
	if bundle.ExpiresAt != nil {
		t := time.Unix(*bundle.ExpiresAt, 0)
		account.QuotaResetAt = &t
	}

	// Store copilot token + quota in metadata
	if meta == nil {
		meta = make(map[string]interface{})
	}
	meta["copilot_token"] = bundle.Token
	if bundle.ChatEnabled != nil {
		meta["copilot_chat_enabled"] = *bundle.ChatEnabled
	}
	if bundle.ExpiresAt != nil {
		meta["copilot_expires_at"] = *bundle.ExpiresAt
	}
	if bundle.RefreshIn != nil {
		meta["copilot_refresh_in"] = *bundle.RefreshIn
	}
	if bundle.QuotaSnapshots != nil {
		meta["copilot_quota_snapshots"] = bundle.QuotaSnapshots
	}
	if bundle.QuotaResetDate != nil {
		meta["copilot_quota_reset_date"] = *bundle.QuotaResetDate
	}
	if bundle.LimitedUserQuotas != nil {
		meta["copilot_limited_user_quotas"] = bundle.LimitedUserQuotas
	}
	if bundle.LimitedUserResetDate != nil {
		meta["copilot_limited_user_reset_date"] = *bundle.LimitedUserResetDate
	}
	if raw, err := json.Marshal(meta); err == nil {
		s := string(raw)
		account.MetadataJSON = &s
	}

	account.UpdatedAt = now
	if err := h.cockpit.SaveAccount(account); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "account": account, "copilot_token": bundle.Token})
}

// GitHubCopilotOAuthStart initiates the GitHub device flow.
// POST /api/v1/cockpit/platforms/github-copilot/oauth/start
func (h *PlatformExtHandler) GitHubCopilotOAuthStart(c *gin.Context) {
	resp, err := github.RequestDeviceCode()
	if err != nil {
		c.JSON(http.StatusBadGateway, models.APIError{Error: err.Error(), Code: "GHCP_OAUTH_START_ERROR"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GitHubCopilotOAuthComplete polls for the device token and creates the account.
// POST /api/v1/cockpit/platforms/github-copilot/oauth/complete
func (h *PlatformExtHandler) GitHubCopilotOAuthComplete(c *gin.Context) {
	var req struct {
		DeviceCode string `json:"device_code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}

	tokenResp, err := github.PollDeviceToken(req.DeviceCode)
	if err != nil {
		c.JSON(http.StatusBadGateway, models.APIError{Error: err.Error(), Code: "GHCP_OAUTH_POLL_ERROR"})
		return
	}
	if tokenResp.Error != nil {
		msg := *tokenResp.Error
		if tokenResp.ErrorDescription != nil {
			msg += ": " + *tokenResp.ErrorDescription
		}
		switch *tokenResp.Error {
		case "authorization_pending", "slow_down":
			c.JSON(http.StatusAccepted, gin.H{"status": "pending", "error": *tokenResp.Error, "message": msg})
			return
		}
		c.JSON(http.StatusUnauthorized, models.APIError{Error: msg, Code: "GHCP_OAUTH_ERROR"})
		return
	}
	if tokenResp.AccessToken == nil {
		c.JSON(http.StatusAccepted, gin.H{"status": "pending"})
		return
	}

	payload, err := github.BuildPayloadFromGitHubToken(*tokenResp.AccessToken)
	if err != nil {
		c.JSON(http.StatusBadGateway, models.APIError{Error: err.Error(), Code: "GHCP_BUILD_PAYLOAD_ERROR"})
		return
	}
	account, err := h.saveGitHubCopilotAccount(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "complete", "payload": payload, "account": account})
}

// GitHubCopilotOAuthCancel is a no-op placeholder (device flow has no server-side cancel).
// POST /api/v1/cockpit/platforms/github-copilot/oauth/cancel
func (h *PlatformExtHandler) GitHubCopilotOAuthCancel(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GitHubCopilotImportToken builds an account payload from a raw GitHub access token.
// POST /api/v1/cockpit/platforms/github-copilot/accounts/import-token
func (h *PlatformExtHandler) GitHubCopilotImportToken(c *gin.Context) {
	var req struct {
		GitHubAccessToken string `json:"github_access_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}

	payload, err := github.BuildPayloadFromGitHubToken(req.GitHubAccessToken)
	if err != nil {
		c.JSON(http.StatusBadGateway, models.APIError{Error: err.Error(), Code: "GHCP_IMPORT_ERROR"})
		return
	}
	account, err := h.saveGitHubCopilotAccount(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "payload": payload, "account": account})
}

// ============================================================
// Antigravity handlers
// ============================================================

// AntigravityOAuthStart starts the Google OAuth flow used by Antigravity.
// POST /api/v1/cockpit/platforms/antigravity/oauth/start
func (h *PlatformExtHandler) AntigravityOAuthStart(c *gin.Context) {
	resp, err := antigravity.StartOAuthFlow()
	if err != nil {
		c.JSON(http.StatusBadGateway, models.APIError{Error: err.Error(), Code: "ANTIGRAVITY_OAUTH_START_ERROR"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// AntigravityOAuthComplete waits for the callback, exchanges the code, and upserts an account.
// POST /api/v1/cockpit/platforms/antigravity/oauth/complete
func (h *PlatformExtHandler) AntigravityOAuthComplete(c *gin.Context) {
	var req struct {
		LoginID    string `json:"login_id"`
		TimeoutSec int    `json:"timeout_sec"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	if req.TimeoutSec <= 0 {
		req.TimeoutSec = 600
	}

	payload, err := antigravity.CompleteOAuthFlow(req.LoginID, req.TimeoutSec)
	if err != nil {
		c.JSON(http.StatusBadGateway, models.APIError{Error: err.Error(), Code: "ANTIGRAVITY_OAUTH_COMPLETE_ERROR"})
		return
	}
	account, err := h.saveAntigravityOAuthAccount(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "account": account})
}

// AntigravityOAuthSubmitCallback accepts a manually pasted OAuth callback URL.
// POST /api/v1/cockpit/platforms/antigravity/oauth/submit-callback
func (h *PlatformExtHandler) AntigravityOAuthSubmitCallback(c *gin.Context) {
	var req struct {
		LoginID     string `json:"login_id"`
		CallbackURL string `json:"callback_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	if err := antigravity.SubmitOAuthCallbackURL(req.LoginID, req.CallbackURL); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "ANTIGRAVITY_OAUTH_CALLBACK_ERROR"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// AntigravityOAuthCancel cancels a pending OAuth flow.
// POST /api/v1/cockpit/platforms/antigravity/oauth/cancel
func (h *PlatformExtHandler) AntigravityOAuthCancel(c *gin.Context) {
	var req struct {
		LoginID string `json:"login_id"`
	}
	_ = c.ShouldBindJSON(&req)

	if err := antigravity.CancelOAuthFlow(req.LoginID); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "ANTIGRAVITY_OAUTH_CANCEL_ERROR"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// AntigravityWakeup executes a wakeup prompt for an Antigravity account.
// POST /api/v1/cockpit/platforms/antigravity/accounts/:id/wakeup
func (h *PlatformExtHandler) AntigravityWakeup(c *gin.Context) {
	account, ok := h.loadPlatformAccount(c, "antigravity")
	if !ok {
		return
	}

	var req struct {
		ProjectID       string `json:"project_id"`
		Model           string `json:"model"`
		Prompt          string `json:"prompt"`
		MaxOutputTokens uint32 `json:"max_output_tokens"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}

	accessToken := ""
	if account.AccessToken != nil {
		accessToken = *account.AccessToken
	}
	if accessToken == "" {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "account has no access_token", Code: "MISSING_TOKEN"})
		return
	}

	// project_id can come from request or account metadata
	if req.ProjectID == "" {
		meta := parseMetadataJSON(account.MetadataJSON)
		req.ProjectID = stringFromMeta(meta, "project_id", "projectId")
	}
	if req.ProjectID == "" {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "project_id is required", Code: "INVALID_REQUEST"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
	defer cancel()

	wakeupReq := antigravity.WakeupRequest{
		ProjectID:       req.ProjectID,
		Model:           req.Model,
		Prompt:          req.Prompt,
		MaxOutputTokens: req.MaxOutputTokens,
	}
	result, err := antigravity.ExecuteWakeup(ctx, accessToken, wakeupReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, models.APIError{Error: err.Error(), Code: "ANTIGRAVITY_WAKEUP_ERROR"})
		return
	}
	c.JSON(http.StatusOK, result)
}

// AntigravityGetSwitchHistory returns the account switch history.
// GET /api/v1/cockpit/platforms/antigravity/switch-history
func (h *PlatformExtHandler) AntigravityGetSwitchHistory(c *gin.Context) {
	items, err := antigravity.LoadHistory(h.dataDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, items)
}

// AntigravityClearSwitchHistory clears the account switch history.
// DELETE /api/v1/cockpit/platforms/antigravity/switch-history
func (h *PlatformExtHandler) AntigravityClearSwitchHistory(c *gin.Context) {
	if err := antigravity.ClearHistory(h.dataDir); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ============================================================
// shared helpers
// ============================================================

func (h *PlatformExtHandler) saveGitHubCopilotAccount(payload *github.AccountPayload) (*models.PlatformAccount, error) {
	if payload == nil {
		return nil, errors.New("empty github copilot payload")
	}
	key := payload.GitHubLogin
	if payload.GitHubID > 0 {
		key = stablePlatformKey("github_id", payload.GitHubID)
	}
	account, meta, err := h.platformAccountForOAuth("github-copilot", key)
	if err != nil {
		return nil, err
	}

	email := ""
	if payload.GitHubEmail != nil {
		email = strings.TrimSpace(*payload.GitHubEmail)
	}
	if email == "" {
		email = strings.TrimSpace(payload.GitHubLogin)
	}
	if email == "" {
		email = "github-copilot-" + stablePlatformHash(key)
	}
	account.Email = email
	if payload.GitHubName != nil && strings.TrimSpace(*payload.GitHubName) != "" {
		name := strings.TrimSpace(*payload.GitHubName)
		account.DisplayName = &name
	} else if strings.TrimSpace(payload.GitHubLogin) != "" {
		login := strings.TrimSpace(payload.GitHubLogin)
		account.DisplayName = &login
	}
	account.AccessToken = &payload.GitHubAccessToken
	account.Status = "active"
	if payload.CopilotBundle != nil && payload.CopilotBundle.Plan != nil {
		account.Plan = payload.CopilotBundle.Plan
	}

	meta["auth_source"] = "oauth"
	meta["github_login"] = payload.GitHubLogin
	meta["github_id"] = payload.GitHubID
	if payload.GitHubName != nil {
		meta["github_name"] = *payload.GitHubName
	}
	if payload.GitHubEmail != nil {
		meta["github_email"] = *payload.GitHubEmail
	}
	meta["github_access_token"] = payload.GitHubAccessToken
	if payload.CopilotBundle != nil {
		meta["copilot_token"] = payload.CopilotBundle.Token
		if payload.CopilotBundle.Plan != nil {
			meta["copilot_plan"] = *payload.CopilotBundle.Plan
		}
		if payload.CopilotBundle.ChatEnabled != nil {
			meta["copilot_chat_enabled"] = *payload.CopilotBundle.ChatEnabled
		}
		if payload.CopilotBundle.ExpiresAt != nil {
			meta["copilot_expires_at"] = *payload.CopilotBundle.ExpiresAt
			t := time.Unix(*payload.CopilotBundle.ExpiresAt, 0)
			account.QuotaResetAt = &t
		}
		if payload.CopilotBundle.RefreshIn != nil {
			meta["copilot_refresh_in"] = *payload.CopilotBundle.RefreshIn
		}
		if payload.CopilotBundle.QuotaSnapshots != nil {
			meta["copilot_quota_snapshots"] = payload.CopilotBundle.QuotaSnapshots
		}
		if payload.CopilotBundle.QuotaResetDate != nil {
			meta["copilot_quota_reset_date"] = *payload.CopilotBundle.QuotaResetDate
		}
		if payload.CopilotBundle.LimitedUserQuotas != nil {
			meta["copilot_limited_user_quotas"] = payload.CopilotBundle.LimitedUserQuotas
		}
		if payload.CopilotBundle.LimitedUserResetDate != nil {
			meta["copilot_limited_user_reset_date"] = *payload.CopilotBundle.LimitedUserResetDate
		}
	}
	writePlatformAccountMetadata(account, meta)
	return account, h.cockpit.SaveAccount(account)
}

func (h *PlatformExtHandler) saveGeminiOAuthAccount(payload *gemini.OAuthCompletePayload) (*models.PlatformAccount, error) {
	if payload == nil {
		return nil, errors.New("empty gemini oauth payload")
	}
	key := firstNonEmpty(payload.AuthID, payload.Email)
	account, meta, err := h.platformAccountForOAuth("gemini", key)
	if err != nil {
		return nil, err
	}
	account.Email = firstNonEmpty(payload.Email, "unknown@gmail.com")
	if payload.Name != nil && strings.TrimSpace(*payload.Name) != "" {
		name := strings.TrimSpace(*payload.Name)
		account.DisplayName = &name
	}
	if payload.AccessToken != "" {
		account.AccessToken = &payload.AccessToken
	}
	if payload.RefreshToken != nil && strings.TrimSpace(*payload.RefreshToken) != "" {
		account.RefreshToken = payload.RefreshToken
	}
	if payload.TierName != nil && strings.TrimSpace(*payload.TierName) != "" {
		account.Plan = payload.TierName
	}
	account.Status = firstNonEmpty(payload.Status, "active")

	meta["auth_source"] = "oauth"
	if payload.AuthID != "" {
		meta["auth_id"] = payload.AuthID
	}
	if payload.IDToken != nil {
		meta["id_token"] = *payload.IDToken
	}
	if payload.TokenType != nil {
		meta["token_type"] = *payload.TokenType
	}
	if payload.Scope != nil {
		meta["scope"] = *payload.Scope
	}
	if payload.ExpiryDate != nil {
		meta["expiry_date"] = *payload.ExpiryDate
	}
	if payload.ProjectID != nil {
		meta["project_id"] = *payload.ProjectID
	}
	if payload.TierID != nil {
		meta["tier_id"] = *payload.TierID
	}
	if payload.GeminiAuthRaw != nil {
		meta["gemini_auth_raw"] = payload.GeminiAuthRaw
	}
	writePlatformAccountMetadata(account, meta)
	return account, h.cockpit.SaveAccount(account)
}

func (h *PlatformExtHandler) saveAntigravityOAuthAccount(payload *antigravity.OAuthCompletePayload) (*models.PlatformAccount, error) {
	if payload == nil {
		return nil, errors.New("empty antigravity oauth payload")
	}
	key := firstNonEmpty(payload.AuthID, payload.Email)
	account, meta, err := h.platformAccountForOAuth("antigravity", key)
	if err != nil {
		return nil, err
	}
	account.Email = firstNonEmpty(payload.Email, "unknown@gmail.com")
	if payload.Name != nil && strings.TrimSpace(*payload.Name) != "" {
		name := strings.TrimSpace(*payload.Name)
		account.DisplayName = &name
	}
	account.AccessToken = &payload.AccessToken
	if payload.RefreshToken != "" {
		account.RefreshToken = &payload.RefreshToken
	}
	account.Status = "active"

	meta["auth_source"] = "oauth"
	if payload.AuthID != "" {
		meta["auth_id"] = payload.AuthID
	}
	meta["expires_in"] = payload.ExpiresIn
	meta["expiry_timestamp"] = payload.ExpiryTimestamp
	if payload.TokenType != "" {
		meta["token_type"] = payload.TokenType
	}
	if payload.ProjectID != nil {
		meta["project_id"] = *payload.ProjectID
	}
	if payload.SessionID != nil {
		meta["session_id"] = *payload.SessionID
	}
	writePlatformAccountMetadata(account, meta)
	return account, h.cockpit.SaveAccount(account)
}

func (h *PlatformExtHandler) platformAccountForOAuth(platform, key string) (*models.PlatformAccount, map[string]interface{}, error) {
	key = firstNonEmpty(key, uuid.New().String())
	id := stablePlatformAccountID(platform, key)
	account, err := h.cockpit.GetAccount(id)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, err
		}
		account = &models.PlatformAccount{
			ID:        id,
			Platform:  platform,
			Status:    "active",
			CreatedAt: time.Now(),
		}
	}
	account.Platform = platform
	meta := parseMetadataJSON(account.MetadataJSON)
	if meta == nil {
		meta = make(map[string]interface{})
	}
	meta["oauth_account_key"] = key
	meta["oauth_updated_at"] = time.Now().UTC().Format(time.RFC3339)
	return account, meta, nil
}

func writePlatformAccountMetadata(account *models.PlatformAccount, meta map[string]interface{}) {
	if raw, err := json.Marshal(meta); err == nil {
		text := string(raw)
		account.MetadataJSON = &text
	}
}

func stablePlatformAccountID(platform, key string) string {
	return platform + "-" + stablePlatformHash(key)
}

func stablePlatformHash(key string) string {
	sum := sha1.Sum([]byte(strings.TrimSpace(key)))
	return hex.EncodeToString(sum[:])[:16]
}

func stablePlatformKey(prefix string, value interface{}) string {
	raw, _ := json.Marshal(value)
	return prefix + ":" + string(raw)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if text := strings.TrimSpace(value); text != "" {
			return text
		}
	}
	return ""
}

func ptrString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func (h *PlatformExtHandler) loadPlatformAccount(c *gin.Context, platform string) (*models.PlatformAccount, bool) {
	id := c.Param("id")
	account, err := h.cockpit.GetAccount(id)
	if err != nil {
		status := http.StatusInternalServerError
		code := "STORAGE_ERROR"
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
			code = "NOT_FOUND"
		}
		c.JSON(status, models.APIError{Error: err.Error(), Code: code})
		return nil, false
	}
	if account.Platform != platform {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "platform mismatch", Code: "INVALID_REQUEST"})
		return nil, false
	}
	return account, true
}

func parseMetadataJSON(raw *string) map[string]interface{} {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil
	}
	var m map[string]interface{}
	_ = json.Unmarshal([]byte(*raw), &m)
	return m
}

func stringFromMeta(meta map[string]interface{}, keys ...string) string {
	if meta == nil {
		return ""
	}
	for _, k := range keys {
		if v, ok := meta[k].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func int64FromMeta(meta map[string]interface{}, keys ...string) int64 {
	if meta == nil {
		return 0
	}
	for _, k := range keys {
		if v, ok := meta[k]; ok {
			switch n := v.(type) {
			case float64:
				return int64(n)
			case int64:
				return n
			case json.Number:
				if i, err := n.Int64(); err == nil {
					return i
				}
			}
		}
	}
	return 0
}
