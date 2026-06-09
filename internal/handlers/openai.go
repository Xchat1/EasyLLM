package handlers

import (
	"bytes"
	"crypto/rand"
	"easyllm/config"
	"easyllm/internal/models"
	openaiplatform "easyllm/internal/openai"
	"easyllm/internal/proxy"
	"easyllm/internal/storage"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OpenAIHandler struct {
	storage               *storage.OpenAIStorage
	codexStorage          *storage.CodexStorage
	mu                    sync.Mutex
	oauthSessions         map[string]*openaiOAuthSession
	oauthCallbackServer   *http.Server
	oauthCallbackListener net.Listener
	oauthCallbackStarted  bool
}

type openaiOAuthSession struct {
	State              string
	CodeVerifier       string
	RedirectURI        string
	CreatedAt          time.Time
	AuthorizationCode  string
	LastError          string
	CallbackReceivedAt *time.Time
}

const (
	maxImportMultipartMemory       = 100 << 20 // 100 MiB
	openaiOAuthSessionTTL          = 10 * time.Minute
	defaultOpenAIOAuthRedirectURI  = "http://localhost:1455/auth/callback"
	defaultOpenAIOAuthCallbackBase = "http://localhost:1455"
	defaultOpenAIOAuthCallbackAddr = "127.0.0.1:1455"

	codexLocalAccessEnabledKey              = "codex_local_access_enabled"
	codexLocalAccessPortKey                 = "codex_local_access_port"
	codexLocalAccessAccountIDsKey           = "codex_local_access_account_ids"
	codexLocalAccessRestrictFreeAccountsKey = "codex_local_access_restrict_free_accounts"
	codexLocalAccessRoutingStrategyKey      = "codex_local_access_routing_strategy"
	codexLocalAccessCreatedAtKey            = "codex_local_access_created_at"
	codexLocalAccessUpdatedAtKey            = "codex_local_access_updated_at"

	codexAPIAccountPostRestartReapplyDelay = 1500 * time.Millisecond
	codexAPIAccountGuardReapplyAttempts    = 15
	codexAPIAccountGuardReapplyInterval    = 1 * time.Second
)

var (
	errOpenAIOAuthSessionNotFound = errors.New("oauth session not found")
	errOpenAIOAuthSessionExpired  = errors.New("oauth session expired")
)

func NewOpenAIHandler(s *storage.OpenAIStorage, cs *storage.CodexStorage) *OpenAIHandler {
	h := &OpenAIHandler{
		storage:       s,
		codexStorage:  cs,
		oauthSessions: make(map[string]*openaiOAuthSession),
	}
	go h.cleanExpiredOAuthSessions()
	return h
}

func sanitizeOpenAIAccountForResponse(account *models.OpenAIAccount) models.OpenAIAccount {
	if account == nil {
		return models.OpenAIAccount{}
	}
	out := *account
	if out.Plan == nil {
		out.Plan = openAIAccountTokenPlan(account)
	}
	out.AccessToken = nil
	out.RefreshToken = nil
	out.IDToken = nil
	out.OpenAIAuthJSON = nil
	out.APIKey = nil
	return out
}

func sanitizeOpenAIAccountsForResponse(accounts []models.OpenAIAccount) []models.OpenAIAccount {
	out := make([]models.OpenAIAccount, 0, len(accounts))
	for i := range accounts {
		out = append(out, sanitizeOpenAIAccountForResponse(&accounts[i]))
	}
	return out
}

func openAIAccountTokenPlan(account *models.OpenAIAccount) *string {
	if account == nil {
		return nil
	}
	for _, token := range []string{derefStr(account.IDToken), derefStr(account.AccessToken)} {
		if userInfo := openaiplatform.ParseIDToken(token); userInfo != nil {
			if plan := normalizedOpenAIPlanPtr(userInfo.PlanType); plan != nil {
				return plan
			}
		}
	}
	return nil
}

func sanitizeCodexAccountForResponse(account *models.CodexAccount) models.CodexAccount {
	if account == nil {
		return models.CodexAccount{}
	}
	out := *account
	out.AccessToken = ""
	return out
}

func sanitizeCodexAccountsForResponse(accounts []*models.CodexAccount) []models.CodexAccount {
	out := make([]models.CodexAccount, 0, len(accounts))
	for i := range accounts {
		out = append(out, sanitizeCodexAccountForResponse(accounts[i]))
	}
	return out
}

// cleanExpiredOAuthSessions periodically removes OAuth sessions older than 10 minutes.
func (h *OpenAIHandler) cleanExpiredOAuthSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		h.mu.Lock()
		for id, sess := range h.oauthSessions {
			if time.Since(sess.CreatedAt) > openaiOAuthSessionTTL {
				delete(h.oauthSessions, id)
			}
		}
		h.mu.Unlock()
	}
}

func (h *OpenAIHandler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/openai")

	// Account management (both OAuth and API)
	g.GET("/accounts", h.ListAccounts)
	g.POST("/accounts", h.AddAccount)
	g.PUT("/accounts/:id", h.UpdateAccount)
	g.DELETE("/accounts/:id", h.DeleteAccount)
	g.DELETE("/accounts", h.DeleteAccounts)
	g.POST("/accounts/:id/switch", h.SwitchAccount)
	g.POST("/accounts/:id/refresh-token", h.RefreshAccountToken)
	g.POST("/accounts/refresh-all", h.RefreshAllTokens)
	g.POST("/accounts/:id/toggle-proxy", h.ToggleProxy)    // 单账号：加入/移出 /v1/* 代理池
	g.POST("/accounts/toggle-proxy-all", h.ToggleProxyAll) // 一键：全部 OAuth 账号加入/移出代理池

	// Batch import: token JSON files (no API call needed, parse directly)
	g.POST("/import/token-files", h.ImportByTokenFiles)       // upload multiple JSON files
	g.POST("/import/auto-files", h.ImportByAutoFiles)         // 上传单个/多个 JSON，自动识别格式导入
	g.POST("/import/refresh-tokens", h.ImportByRefreshTokens) // legacy: refresh_token list
	g.POST("/import/cpa", h.ImportCPA)                        // CPA / *-cpa.json 单文件或多文件
	g.POST("/import/from-export", h.ImportFromExport)         // re-import from exported backup JSON (no API calls)
	g.GET("/export", h.ExportAccounts)

	// OAuth flow
	g.POST("/oauth/generate-url", h.GenerateOAuthURL)
	g.GET("/oauth/sessions/:id", h.GetOAuthSession)
	g.DELETE("/oauth/sessions/:id", h.CancelOAuthSession)
	g.POST("/oauth/exchange-code", h.ExchangeCode)

	// API account management
	g.POST("/api-accounts", h.AddAPIAccount)
	g.PUT("/api-accounts/:id", h.UpdateAPIAccount)
	g.POST("/api-accounts/:id/switch", h.SwitchAPIAccount)
	g.POST("/api-accounts/:id/test", h.TestAPIAccount)

	// Codex proxy pool
	g.GET("/codex/accounts", h.ListCodexAccounts)
	g.POST("/codex/accounts", h.AddCodexAccount)
	g.PUT("/codex/accounts/:id", h.UpdateCodexAccount)
	g.DELETE("/codex/accounts/:id", h.DeleteCodexAccount)
	g.POST("/codex/accounts/:id/toggle", h.ToggleCodexAccount)
	g.GET("/codex/pool", h.GetCodexPoolStatus)
	g.POST("/codex/pool/refresh", h.RefreshCodexPool)

	// Quota check
	g.POST("/accounts/fetch-quotas", h.FetchQuotas)

	// Service config (proxy pool switch, API key, stats)
	g.GET("/service-config", h.GetServiceConfig)
	g.PUT("/service-config", h.UpdateServiceConfig)
	g.POST("/service-config/activate-codex", h.ActivateCodexAPIService)
	g.GET("/local-access", h.GetCodexLocalAccess)
	g.POST("/local-access/activate", h.ActivateCodexLocalAccess)
	g.POST("/local-access/deactivate", h.DeactivateCodexLocalAccess)
	g.PUT("/local-access/enabled", h.SetCodexLocalAccessEnabled)
	g.PUT("/local-access/accounts", h.SaveCodexLocalAccessAccounts)
	g.DELETE("/local-access/accounts/:id", h.RemoveCodexLocalAccessAccount)
	g.PUT("/local-access/port", h.UpdateCodexLocalAccessPort)
	g.PUT("/local-access/routing", h.UpdateCodexLocalAccessRoutingStrategy)
	g.POST("/local-access/rotate-key", h.RotateCodexLocalAccessAPIKey)
	g.DELETE("/local-access/stats", h.ClearCodexLocalAccessStats)
	g.POST("/local-access/kill-port", h.KillCodexLocalAccessPort)

}

func (h *OpenAIHandler) ListAccounts(c *gin.Context) {
	accounts, err := h.storage.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, sanitizeOpenAIAccountsForResponse(accounts))
}

func (h *OpenAIHandler) AddAccount(c *gin.Context) {
	var account models.OpenAIAccount
	if err := c.ShouldBindJSON(&account); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	if account.ID == "" {
		account.ID = uuid.New().String()
	}
	if account.AccountType == "" {
		account.AccountType = models.OpenAIAccountTypeOAuth
	}
	account.Email = strings.TrimSpace(account.Email)
	if account.Email == "" {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "email is required", Code: "INVALID_REQUEST"})
		return
	}
	if account.Status == "" {
		account.Status = "active"
	}
	account.CreatedAt = time.Now()
	account.UpdatedAt = time.Now()
	defaultJoined := h.applyDefaultAPIServiceMembership(&account)
	if err := h.storage.Save(&account); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	if defaultJoined {
		if err := h.saveDefaultAPIServiceMembership(&account); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
			return
		}
		refreshCodexProxyPool()
	}
	c.JSON(http.StatusOK, sanitizeOpenAIAccountForResponse(&account))
}

func (h *OpenAIHandler) UpdateAccount(c *gin.Context) {
	id := c.Param("id")
	existing, err := h.storage.Get(id)
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
	var account models.OpenAIAccount
	if err := c.ShouldBindJSON(&account); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	account.Email = strings.TrimSpace(account.Email)
	if account.Email == "" {
		account.Email = existing.Email
	}
	account = mergeOpenAIAccountUpdate(existing, account)
	account.ID = id
	account.AccountType = existing.AccountType
	account.CreatedAt = existing.CreatedAt
	account.UpdatedAt = time.Now()
	if err := h.storage.Save(&account); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, sanitizeOpenAIAccountForResponse(&account))
}

func (h *OpenAIHandler) DeleteAccount(c *gin.Context) {
	id := c.Param("id")
	if err := h.storage.Delete(id); err != nil {
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

func (h *OpenAIHandler) DeleteAccounts(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "ids cannot be empty", Code: "INVALID_REQUEST"})
		return
	}
	res := storage.GetDB().Where("id IN ?", req.IDs).Delete(&models.OpenAIAccount{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: res.Error.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "rows_affected": res.RowsAffected})
}

// SwitchAccount switches to an OAuth account (writes ~/.codex/auth.json)
func (h *OpenAIHandler) SwitchAccount(c *gin.Context) {
	account, err := h.storage.Get(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIError{Error: "Account not found", Code: "NOT_FOUND"})
		return
	}

	if account.AccountType == models.OpenAIAccountTypeAPI {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "Use /api-accounts/:id/switch for API type accounts", Code: "WRONG_TYPE"})
		return
	}

	accessToken := ""
	if account.AccessToken != nil {
		accessToken = *account.AccessToken
	}
	refreshToken := ""
	if account.RefreshToken != nil {
		refreshToken = *account.RefreshToken
	}
	idToken := ""
	if account.IDToken != nil {
		idToken = *account.IDToken
	}

	if err := openaiplatform.SwitchCodexOAuthAccount(accessToken, refreshToken, idToken, account.ChatGPTAccountID, localProxyOriginFromRequest(c)); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "SWITCH_ERROR"})
		return
	}

	now := time.Now()
	account.LastUsedAt = &now
	if err := h.storage.Save(account); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	if err := h.storage.SetCodexActive(account.ID); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Switched to " + account.Email})
}

// SwitchAPIAccount switches to an API key account (writes ~/.codex/config.toml)
func (h *OpenAIHandler) SwitchAPIAccount(c *gin.Context) {
	account, err := h.storage.Get(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIError{Error: "Account not found", Code: "NOT_FOUND"})
		return
	}

	if account.AccountType != models.OpenAIAccountTypeAPI {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "Use /accounts/:id/switch for OAuth accounts", Code: "WRONG_TYPE"})
		return
	}

	provider := derefStr(account.ModelProvider)
	model := derefStr(account.Model)
	baseURL := derefStr(account.BaseURL)
	apiKey := derefStr(account.APIKey)

	writeCodexAPIAccountConfig := func() error {
		return openaiplatform.SwitchCodexAPIAccount(provider, model, baseURL, apiKey, account.WireAPI, account.ModelReasoningEffort, localProxyOriginFromRequest(c))
	}

	if err := writeCodexAPIAccountConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "SWITCH_ERROR"})
		return
	}

	now := time.Now()
	account.LastUsedAt = &now
	if err := h.storage.Save(account); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	if err := h.storage.SetCodexActive(account.ID); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}

	providerModelInfo := fmt.Sprintf("API provider %s (%s)", derefStr(account.ModelProvider), derefStr(account.Model))

	launchResult, err := openaiplatform.RestartCodexApp()
	if err != nil {
		// Non-fatal: config was written, but Codex restart failed
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Switched to " + providerModelInfo, "restart_error": err.Error()})
		return
	}

	// Codex Desktop may recreate its codex_local_access provider while starting.
	// Write the selected API account again so the final on-disk config remains active.
	time.Sleep(codexAPIAccountPostRestartReapplyDelay)
	if err := writeCodexAPIAccountConfig(); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Switched to " + providerModelInfo, "launch": launchResult, "config_reapply_error": err.Error()})
		return
	}
	h.scheduleCodexAPIAccountConfigGuard(account.ID, writeCodexAPIAccountConfig, codexAPIAccountGuardReapplyAttempts, codexAPIAccountGuardReapplyInterval)

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Switched to " + providerModelInfo, "launch": launchResult, "config_reapplied": true, "config_guard_seconds": int(codexAPIAccountGuardReapplyAttempts)})
}

func (h *OpenAIHandler) scheduleCodexAPIAccountConfigGuard(accountID string, write func() error, attempts int, interval time.Duration) {
	if accountID == "" || write == nil || attempts <= 0 || interval <= 0 {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				_, _ = fmt.Fprintf(os.Stderr, "easyllm: panic recovered in Codex API account config guard goroutine: %v\n", r)
			}
		}()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for i := 0; i < attempts; i++ {
			<-ticker.C
			if !h.isCodexActiveAccount(accountID) {
				return
			}
			if err := write(); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "easyllm: failed to reapply Codex API account config: %v\n", err)
			}
		}
	}()
}

func (h *OpenAIHandler) isCodexActiveAccount(accountID string) bool {
	account, err := h.storage.Get(accountID)
	return err == nil && account != nil && account.IsCodexActive
}

// RefreshAccountToken refreshes the OAuth token for a single account
func (h *OpenAIHandler) RefreshAccountToken(c *gin.Context) {
	account, err := h.storage.Get(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIError{Error: "Account not found", Code: "NOT_FOUND"})
		return
	}

	if account.AccountType == models.OpenAIAccountTypeAPI {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "API accounts do not support token refresh", Code: "NOT_SUPPORTED"})
		return
	}
	if account.RefreshToken == nil || *account.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "No refresh token available", Code: "NO_REFRESH_TOKEN"})
		return
	}

	if err := h.refreshOAuthAccountTokens(account); err != nil {
		if isReauthRequiredError(err) {
			c.JSON(http.StatusConflict, models.APIError{Error: err.Error(), Code: "REAUTH_REQUIRED"})
			return
		}
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "REFRESH_ERROR"})
		return
	}
	refreshCodexProxyPool()
	c.JSON(http.StatusOK, sanitizeOpenAIAccountForResponse(account))
}

// RefreshAllTokens refreshes tokens for all OAuth accounts concurrently
func (h *OpenAIHandler) RefreshAllTokens(c *gin.Context) {
	accounts, err := h.storage.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}

	type result struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Success bool   `json:"success"`
		Skipped bool   `json:"skipped,omitempty"`
		Error   string `json:"error,omitempty"`
	}

	var oauthAccounts []models.OpenAIAccount
	for _, a := range accounts {
		if a.AccountType == models.OpenAIAccountTypeOAuth {
			oauthAccounts = append(oauthAccounts, a)
		}
	}

	results := make([]result, 0, len(oauthAccounts))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 3)

	for _, a := range oauthAccounts {
		if a.RefreshToken == nil || *a.RefreshToken == "" {
			mu.Lock()
			results = append(results, result{ID: a.ID, Email: a.Email, Success: false, Skipped: true, Error: "no refresh token"})
			mu.Unlock()
			continue
		}

		wg.Add(1)
		go func(acc models.OpenAIAccount) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := h.refreshOAuthAccountTokens(&acc); err != nil {
				mu.Lock()
				results = append(results, result{ID: acc.ID, Email: acc.Email, Success: false, Error: err.Error()})
				mu.Unlock()
				return
			}

			mu.Lock()
			results = append(results, result{ID: acc.ID, Email: acc.Email, Success: true})
			mu.Unlock()
		}(a)
	}

	wg.Wait()

	success := 0
	skipped := 0
	for _, r := range results {
		if r.Success {
			success++
		} else if r.Skipped {
			skipped++
		}
	}
	if success > 0 {
		refreshCodexProxyPool()
	}
	c.JSON(http.StatusOK, gin.H{
		"total":   len(results),
		"success": success,
		"skipped": skipped,
		"failed":  len(results) - success - skipped,
		"results": results,
	})
}

// tokenFileData is the structure of each token JSON file in the auth/ directory
type tokenFileData struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	AccountID    string `json:"account_id"`
	LastRefresh  string `json:"last_refresh"`
	Email        string `json:"email"`
	Type         string `json:"type"`
	Expired      string `json:"expired"`
	PlanType     string `json:"plan_type"`
}

type tokenImportResult struct {
	Filename string `json:"filename"`
	Format   string `json:"format,omitempty"` // 扫描目录自动识别：token / cpa / easyllm-export
	Success  bool   `json:"success"`
	Email    string `json:"email,omitempty"`
	Skipped  bool   `json:"skipped,omitempty"`
	Error    string `json:"error,omitempty"`
}

func parseTokenFileEntries(raw []byte) ([]tokenFileData, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty token file")
	}

	if trimmed[0] == '[' {
		var entries []tokenFileData
		if err := json.Unmarshal(trimmed, &entries); err != nil {
			return nil, err
		}
		if len(entries) == 0 {
			return nil, fmt.Errorf("no token entries found")
		}
		return entries, nil
	}

	dec := json.NewDecoder(bytes.NewReader(trimmed))
	var entries []tokenFileData
	for {
		var entry tokenFileData
		if err := dec.Decode(&entry); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no token entries found")
	}
	return entries, nil
}

// parseCPAFileEntries 解析 CPA 导出 JSON（单对象、数组或 NDJSON），格式如 *-cpa.json / *.codex.cpa.json。
func parseCPAFileEntries(raw []byte) ([]tokenFileData, error) {
	entries, err := parseTokenFileEntries(raw)
	if err != nil {
		return nil, err
	}
	for i := range entries {
		if err := validateCPAEntry(&entries[i]); err != nil {
			return nil, fmt.Errorf("entry %d: %w", i+1, err)
		}
	}
	return entries, nil
}

func validateCPAEntry(data *tokenFileData) error {
	if data == nil {
		return fmt.Errorf("empty entry")
	}
	if strings.TrimSpace(data.AccessToken) == "" &&
		strings.TrimSpace(data.RefreshToken) == "" &&
		strings.TrimSpace(data.IDToken) == "" {
		return fmt.Errorf("missing access_token, refresh_token and id_token")
	}
	return nil
}

// importTokenFileData converts a tokenFileData into an OpenAIAccount and saves it
// Returns (account, skipped, error)
func (h *OpenAIHandler) importSingleTokenFile(data *tokenFileData, existingAccounts *[]models.OpenAIAccount) (*models.OpenAIAccount, bool, error) {
	if data.Email == "" && data.IDToken != "" {
		// Try to parse email from id_token
		if userInfo := openaiplatform.ParseIDToken(data.IDToken); userInfo != nil && userInfo.Email != nil {
			data.Email = strings.TrimSpace(*userInfo.Email)
		}
	}
	if data.Email == "" {
		return nil, false, fmt.Errorf("no email found in token file")
	}

	now := time.Now()
	var expiresAt *time.Time
	if data.Expired != "" {
		if t, err := time.Parse(time.RFC3339, data.Expired); err == nil {
			expiresAt = &t
		}
	}

	account := &models.OpenAIAccount{
		ID:           uuid.New().String(),
		Email:        data.Email,
		AccountType:  models.OpenAIAccountTypeOAuth,
		AccessToken:  sPtr(data.AccessToken),
		RefreshToken: sPtr(data.RefreshToken),
		ExpiresAt:    expiresAt,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if data.IDToken != "" {
		account.IDToken = sPtr(data.IDToken)
		// Parse extra fields from id_token
		if userInfo := openaiplatform.ParseIDToken(data.IDToken); userInfo != nil {
			account.ChatGPTAccountID = userInfo.ChatGPTAccountID
			account.ChatGPTUserID = userInfo.ChatGPTUserID
			account.OrganizationID = userInfo.OrganizationID
			account.Plan = normalizedOpenAIPlanPtr(userInfo.PlanType)
		}
		if j := openaiplatform.ExtractOpenAIAuthJSON(data.IDToken); j != "" {
			account.OpenAIAuthJSON = sPtr(j)
		}
	}
	if account.Plan == nil && strings.TrimSpace(data.PlanType) != "" {
		account.Plan = normalizedOpenAIPlanPtr(sPtr(data.PlanType))
	}
	if data.AccountID != "" && account.ChatGPTAccountID == nil {
		account.ChatGPTAccountID = sPtr(data.AccountID)
	}

	return h.upsertImportedOAuthAccount(account, existingAccounts)
}

// ImportByTokenFiles handles uploading multiple token JSON files at once (multipart form)
func (h *OpenAIHandler) ImportByTokenFiles(c *gin.Context) {
	// Parse multipart explicitly so behavior stays consistent across router setups.
	if err := c.Request.ParseMultipartForm(maxImportMultipartMemory); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "invalid multipart form: " + err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	form := c.Request.MultipartForm
	if form == nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "invalid multipart form", Code: "INVALID_REQUEST"})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "no files uploaded", Code: "NO_FILES"})
		return
	}

	existingMu := sync.Mutex{}
	existingAccounts, _ := h.storage.List()
	resultsMu := sync.Mutex{}
	results := make([]tokenImportResult, 0, len(files))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10) // Higher concurrency - no network calls needed

	for i, fh := range files {
		wg.Add(1)
		go func(idx int, fileHeader *multipart.FileHeader) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			f, err := fileHeader.Open()
			if err != nil {
				resultsMu.Lock()
				results = append(results, tokenImportResult{Filename: fileHeader.Filename, Success: false, Error: "open error: " + err.Error()})
				resultsMu.Unlock()
				return
			}
			defer f.Close()

			raw, err := io.ReadAll(f)
			if err != nil {
				resultsMu.Lock()
				results = append(results, tokenImportResult{Filename: fileHeader.Filename, Success: false, Error: "read error: " + err.Error()})
				resultsMu.Unlock()
				return
			}

			entries, err := parseTokenFileEntries(raw)
			if err != nil {
				resultsMu.Lock()
				results = append(results, tokenImportResult{Filename: fileHeader.Filename, Success: false, Error: "parse error: " + err.Error()})
				resultsMu.Unlock()
				return
			}

			fileResults := make([]tokenImportResult, 0, len(entries))
			for _, data := range entries {
				entry := data
				existingMu.Lock()
				account, skipped, err := h.importSingleTokenFile(&entry, &existingAccounts)
				existingMu.Unlock()

				if err != nil {
					fileResults = append(fileResults, tokenImportResult{Filename: fileHeader.Filename, Success: false, Skipped: skipped, Error: err.Error(), Email: entry.Email})
					continue
				}
				fileResults = append(fileResults, tokenImportResult{Filename: fileHeader.Filename, Success: true, Email: account.Email})
			}

			resultsMu.Lock()
			results = append(results, fileResults...)
			resultsMu.Unlock()
		}(i, fh)
	}

	wg.Wait()

	successCount := 0
	skippedCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		} else if r.Skipped {
			skippedCount++
		}
	}
	if successCount > 0 {
		refreshCodexProxyPool()
	}

	c.JSON(http.StatusOK, gin.H{
		"total":   len(results),
		"success": successCount,
		"skipped": skippedCount,
		"failed":  len(results) - successCount - skippedCount,
		"results": results,
	})
}

// ImportCPABytes 解析 CPA JSON 并导入 OAuth 账号（单对象、数组或 NDJSON）。
func (h *OpenAIHandler) ImportCPABytes(data []byte) (gin.H, error) {
	entries, err := parseCPAFileEntries(data)
	if err != nil {
		return nil, err
	}
	return h.importCPAEntries(entries, "cpa.json")
}

func (h *OpenAIHandler) importCPAEntries(entries []tokenFileData, filename string) (gin.H, error) {
	existingAccounts, _ := h.storage.List()
	results := make([]tokenImportResult, 0, len(entries))
	for _, data := range entries {
		entry := data
		account, skipped, err := h.importSingleTokenFile(&entry, &existingAccounts)
		if err != nil {
			label := entry.Email
			if label == "" {
				label = filename
			}
			results = append(results, tokenImportResult{Filename: filename, Success: false, Skipped: skipped, Error: err.Error(), Email: label})
			continue
		}
		results = append(results, tokenImportResult{Filename: filename, Success: true, Email: account.Email})
	}
	successCount, skippedCount := 0, 0
	for _, r := range results {
		if r.Success {
			successCount++
		} else if r.Skipped {
			skippedCount++
		}
	}
	if successCount > 0 {
		refreshCodexProxyPool()
	}
	return gin.H{
		"total":   len(results),
		"success": successCount,
		"skipped": skippedCount,
		"failed":  len(results) - successCount - skippedCount,
		"results": results,
	}, nil
}

// ImportCPA 支持 multipart 多文件上传，或请求体为单个/多个 CPA JSON 对象。
func (h *OpenAIHandler) ImportCPA(c *gin.Context) {
	if strings.Contains(strings.ToLower(c.GetHeader("Content-Type")), "multipart/form-data") {
		h.importCPAFromMultipart(c)
		return
	}
	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "读取请求体失败: " + err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	out, err := h.ImportCPABytes(data)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "无效的 CPA 数据: " + err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *OpenAIHandler) importCPAFromMultipart(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(maxImportMultipartMemory); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "invalid multipart form: " + err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	form := c.Request.MultipartForm
	if form == nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "invalid multipart form", Code: "INVALID_REQUEST"})
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "no files uploaded", Code: "NO_FILES"})
		return
	}

	existingMu := sync.Mutex{}
	existingAccounts, _ := h.storage.List()
	resultsMu := sync.Mutex{}
	results := make([]tokenImportResult, 0)
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)

	for _, fh := range files {
		wg.Add(1)
		go func(fileHeader *multipart.FileHeader) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			f, err := fileHeader.Open()
			if err != nil {
				resultsMu.Lock()
				results = append(results, tokenImportResult{Filename: fileHeader.Filename, Success: false, Error: "open error: " + err.Error()})
				resultsMu.Unlock()
				return
			}
			defer f.Close()

			raw, err := io.ReadAll(f)
			if err != nil {
				resultsMu.Lock()
				results = append(results, tokenImportResult{Filename: fileHeader.Filename, Success: false, Error: "read error: " + err.Error()})
				resultsMu.Unlock()
				return
			}

			entries, err := parseCPAFileEntries(raw)
			if err != nil {
				resultsMu.Lock()
				results = append(results, tokenImportResult{Filename: fileHeader.Filename, Success: false, Error: "parse error: " + err.Error()})
				resultsMu.Unlock()
				return
			}

			fileResults := make([]tokenImportResult, 0, len(entries))
			for _, data := range entries {
				entry := data
				existingMu.Lock()
				account, skipped, err := h.importSingleTokenFile(&entry, &existingAccounts)
				existingMu.Unlock()
				if err != nil {
					email := entry.Email
					if email == "" {
						email = fileHeader.Filename
					}
					fileResults = append(fileResults, tokenImportResult{Filename: fileHeader.Filename, Success: false, Skipped: skipped, Error: err.Error(), Email: email})
					continue
				}
				fileResults = append(fileResults, tokenImportResult{Filename: fileHeader.Filename, Success: true, Email: account.Email})
			}
			resultsMu.Lock()
			results = append(results, fileResults...)
			resultsMu.Unlock()
		}(fh)
	}
	wg.Wait()

	successCount, skippedCount := 0, 0
	for _, r := range results {
		if r.Success {
			successCount++
		} else if r.Skipped {
			skippedCount++
		}
	}
	if successCount > 0 {
		refreshCodexProxyPool()
	}
	c.JSON(http.StatusOK, gin.H{
		"total":   len(results),
		"success": successCount,
		"skipped": skippedCount,
		"failed":  len(results) - successCount - skippedCount,
		"results": results,
	})
}

func findMatchingOAuthAccountIndex(existingAccounts []models.OpenAIAccount, incoming *models.OpenAIAccount) int {
	if incoming == nil {
		return -1
	}

	targetID := strings.TrimSpace(derefStr(incoming.ChatGPTAccountID))
	targetOrgID := strings.TrimSpace(derefStr(incoming.OrganizationID))
	if targetID != "" {
		emptyOrgIdx := -1
		firstScopedIdx := -1
		firstScopedOrgID := ""
		hasMultipleScopedOrgs := false

		for i := range existingAccounts {
			existing := existingAccounts[i]
			if existing.AccountType != models.OpenAIAccountTypeOAuth {
				continue
			}
			if strings.TrimSpace(derefStr(existing.ChatGPTAccountID)) == targetID {
				existingOrgID := strings.TrimSpace(derefStr(existing.OrganizationID))
				if targetOrgID != "" {
					if existingOrgID == targetOrgID {
						return i
					}
					if existingOrgID == "" && emptyOrgIdx < 0 {
						emptyOrgIdx = i
					}
					continue
				}

				if existingOrgID == "" {
					return i
				}
				if firstScopedIdx < 0 {
					firstScopedIdx = i
					firstScopedOrgID = existingOrgID
				} else if existingOrgID != firstScopedOrgID {
					hasMultipleScopedOrgs = true
				}
			}
		}

		if targetOrgID != "" {
			return emptyOrgIdx
		}
		if firstScopedIdx >= 0 && !hasMultipleScopedOrgs {
			return firstScopedIdx
		}
		return -1
	}

	targetEmail := strings.TrimSpace(incoming.Email)
	if targetEmail == "" {
		return -1
	}
	for i := range existingAccounts {
		existing := existingAccounts[i]
		if existing.AccountType != models.OpenAIAccountTypeOAuth {
			continue
		}
		if !strings.EqualFold(existing.Email, targetEmail) {
			continue
		}
		return i
	}

	return -1
}

func findMatchingAPIAccountIndex(existingAccounts []models.OpenAIAccount, incoming *models.OpenAIAccount) int {
	targetProvider := strings.ToLower(strings.TrimSpace(derefStr(incoming.ModelProvider)))
	targetModel := strings.ToLower(strings.TrimSpace(derefStr(incoming.Model)))
	targetBaseURL := normalizeBaseURL(derefStr(incoming.BaseURL))
	targetWireAPI := strings.ToLower(strings.TrimSpace(derefStr(incoming.WireAPI)))
	targetEmail := strings.ToLower(strings.TrimSpace(incoming.Email))

	for i := range existingAccounts {
		existing := existingAccounts[i]
		if existing.AccountType != models.OpenAIAccountTypeAPI {
			continue
		}

		if strings.ToLower(strings.TrimSpace(derefStr(existing.ModelProvider))) == targetProvider &&
			strings.ToLower(strings.TrimSpace(derefStr(existing.Model))) == targetModel &&
			normalizeBaseURL(derefStr(existing.BaseURL)) == targetBaseURL &&
			strings.ToLower(strings.TrimSpace(derefStr(existing.WireAPI))) == targetWireAPI {
			return i
		}

		if targetEmail != "" && strings.EqualFold(strings.TrimSpace(existing.Email), targetEmail) &&
			targetBaseURL != "" && normalizeBaseURL(derefStr(existing.BaseURL)) == targetBaseURL {
			return i
		}
	}

	return -1
}

func (h *OpenAIHandler) upsertImportedOAuthAccount(incoming *models.OpenAIAccount, existingAccounts *[]models.OpenAIAccount) (*models.OpenAIAccount, bool, error) {
	if incoming == nil {
		return nil, false, fmt.Errorf("incoming oauth account is nil")
	}
	now := time.Now()
	incomingStatus := incoming.Status
	if incomingStatus == "" {
		incomingStatus = "active"
	}

	if idx := findMatchingOAuthAccountIndex(*existingAccounts, incoming); idx >= 0 {
		existing := &(*existingAccounts)[idx]
		if incoming.Email != "" {
			existing.Email = incoming.Email
		}
		existing.AccountType = models.OpenAIAccountTypeOAuth
		existing.Status = incomingStatus
		if incoming.AccessToken != nil && *incoming.AccessToken != "" {
			existing.AccessToken = incoming.AccessToken
		}
		if incoming.RefreshToken != nil && *incoming.RefreshToken != "" {
			existing.RefreshToken = incoming.RefreshToken
		}
		if incoming.IDToken != nil && *incoming.IDToken != "" {
			existing.IDToken = incoming.IDToken
		}
		if incoming.ExpiresAt != nil {
			existing.ExpiresAt = incoming.ExpiresAt
		}
		if incoming.ChatGPTAccountID != nil && *incoming.ChatGPTAccountID != "" {
			existing.ChatGPTAccountID = incoming.ChatGPTAccountID
		}
		if incoming.ChatGPTUserID != nil && *incoming.ChatGPTUserID != "" {
			existing.ChatGPTUserID = incoming.ChatGPTUserID
		}
		if incoming.OrganizationID != nil && *incoming.OrganizationID != "" {
			existing.OrganizationID = incoming.OrganizationID
		}
		if incoming.OpenAIAuthJSON != nil && *incoming.OpenAIAuthJSON != "" {
			existing.OpenAIAuthJSON = incoming.OpenAIAuthJSON
		}
		if incoming.Plan != nil && *incoming.Plan != "" {
			existing.Plan = incoming.Plan
		}
		if incoming.ProxyEnabled {
			existing.ProxyEnabled = true
		}
		defaultJoined := h.applyDefaultAPIServiceMembership(existing)
		existing.UpdatedAt = now
		if err := h.storage.Save(existing); err != nil {
			return nil, false, err
		}
		if defaultJoined {
			if err := h.saveDefaultAPIServiceMembership(existing); err != nil {
				return nil, false, err
			}
		}
		return existing, false, nil
	}

	if incoming.ID == "" {
		incoming.ID = uuid.New().String()
	}
	if incoming.CreatedAt.IsZero() {
		incoming.CreatedAt = now
	}
	incoming.Status = incomingStatus
	incoming.UpdatedAt = now
	defaultJoined := h.applyDefaultAPIServiceMembership(incoming)
	if err := h.storage.Save(incoming); err != nil {
		return nil, false, err
	}
	if defaultJoined {
		if err := h.saveDefaultAPIServiceMembership(incoming); err != nil {
			return nil, false, err
		}
	}
	*existingAccounts = append(*existingAccounts, *incoming)
	return incoming, false, nil
}

func (h *OpenAIHandler) upsertImportedAPIAccount(incoming *models.OpenAIAccount, existingAccounts *[]models.OpenAIAccount) (*models.OpenAIAccount, bool, error) {
	if incoming == nil {
		return nil, false, fmt.Errorf("incoming api account is nil")
	}
	now := time.Now()

	if idx := findMatchingAPIAccountIndex(*existingAccounts, incoming); idx >= 0 {
		existing := &(*existingAccounts)[idx]
		if incoming.Email != "" {
			existing.Email = incoming.Email
		}
		existing.AccountType = models.OpenAIAccountTypeAPI
		if incoming.ModelProvider != nil && *incoming.ModelProvider != "" {
			existing.ModelProvider = incoming.ModelProvider
		}
		if incoming.Model != nil && *incoming.Model != "" {
			existing.Model = incoming.Model
		}
		if incoming.ModelReasoningEffort != nil {
			existing.ModelReasoningEffort = incoming.ModelReasoningEffort
		}
		if incoming.WireAPI != nil && *incoming.WireAPI != "" {
			existing.WireAPI = incoming.WireAPI
		}
		if incoming.BaseURL != nil && *incoming.BaseURL != "" {
			existing.BaseURL = incoming.BaseURL
		}
		if incoming.APIKey != nil && *incoming.APIKey != "" {
			existing.APIKey = incoming.APIKey
		}
		existing.ProxyEnabled = incoming.ProxyEnabled
		existing.UpdatedAt = now
		if err := h.storage.Save(existing); err != nil {
			return nil, false, err
		}
		return existing, false, nil
	}

	if incoming.ID == "" {
		incoming.ID = uuid.New().String()
	}
	if incoming.CreatedAt.IsZero() {
		incoming.CreatedAt = now
	}
	incoming.UpdatedAt = now
	if err := h.storage.Save(incoming); err != nil {
		return nil, false, err
	}
	*existingAccounts = append(*existingAccounts, *incoming)
	return incoming, false, nil
}

func normalizeBaseURL(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	return strings.TrimRight(normalized, "/")
}

// ImportByRefreshTokens is the CORE batch import feature:
// Takes a list of refresh_tokens, exchanges each for access_token+id_token,
// extracts email from id_token, and saves the account.
func (h *OpenAIHandler) ImportByRefreshTokens(c *gin.Context) {
	var req struct {
		RefreshTokens []string `json:"refresh_tokens"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}

	if len(req.RefreshTokens) == 0 {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "refresh_tokens cannot be empty", Code: "EMPTY_INPUT"})
		return
	}
	if len(req.RefreshTokens) > 100 {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "Too many tokens (max 100)", Code: "TOO_MANY"})
		return
	}

	type importResult struct {
		Index        int                   `json:"index"`
		Success      bool                  `json:"success"`
		Email        string                `json:"email,omitempty"`
		Account      *models.OpenAIAccount `json:"account,omitempty"`
		Error        string                `json:"error,omitempty"`
		TokenPreview string                `json:"token_preview"`
	}

	// Preload existing accounts for duplicate detection
	existingAccounts, _ := h.storage.List()
	existingMu := sync.Mutex{}

	results := make([]importResult, len(req.RefreshTokens))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 3)

	for i, rt := range req.RefreshTokens {
		rt = strings.TrimSpace(rt)
		preview := maskOpenAIToken(rt)

		if rt == "" {
			results[i] = importResult{Index: i, Success: false, Error: "empty token", TokenPreview: preview}
			continue
		}

		wg.Add(1)
		go func(idx int, refreshToken, tokenPreview string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Step 1: Call OpenAI token API to get access_token + id_token
			tokenResp, err := openaiplatform.RefreshToken(refreshToken)
			if err != nil {
				results[idx] = importResult{Index: idx, Success: false, Error: err.Error(), TokenPreview: tokenPreview}
				return
			}

			// Step 2: Parse id_token to get email and account info
			var email string
			var chatgptAccountID, chatgptUserID, orgID *string
			var plan *string
			var openaiAuthJSON string

			if tokenResp.IDToken != "" {
				if userInfo := openaiplatform.ParseIDToken(tokenResp.IDToken); userInfo != nil {
					if userInfo.Email != nil {
						email = strings.TrimSpace(*userInfo.Email)
					}
					chatgptAccountID = userInfo.ChatGPTAccountID
					chatgptUserID = userInfo.ChatGPTUserID
					orgID = userInfo.OrganizationID
					plan = normalizedOpenAIPlanPtr(userInfo.PlanType)
				}
				openaiAuthJSON = openaiplatform.ExtractOpenAIAuthJSON(tokenResp.IDToken)
			}

			if email == "" {
				results[idx] = importResult{Index: idx, Success: false, Error: "Failed to get email from id_token", TokenPreview: tokenPreview}
				return
			}

			// Step 3: Upsert by account id/email
			existingMu.Lock()
			finalEmail := email
			// Step 4: Build and upsert account
			now := time.Now()
			var expiresAt *time.Time
			if tokenResp.ExpiresIn > 0 {
				t := now.Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
				expiresAt = &t
			}
			refreshValue := refreshToken
			if tokenResp.RefreshToken != "" {
				refreshValue = tokenResp.RefreshToken
			}

			account := &models.OpenAIAccount{
				Email:            finalEmail,
				AccountType:      models.OpenAIAccountTypeOAuth,
				AccessToken:      sPtr(tokenResp.AccessToken),
				RefreshToken:     sPtr(refreshValue),
				ExpiresAt:        expiresAt,
				ChatGPTAccountID: chatgptAccountID,
				ChatGPTUserID:    chatgptUserID,
				OrganizationID:   orgID,
				Plan:             plan,
				CreatedAt:        now,
				UpdatedAt:        now,
			}
			if tokenResp.IDToken != "" {
				account.IDToken = sPtr(tokenResp.IDToken)
			}
			if openaiAuthJSON != "" {
				account.OpenAIAuthJSON = sPtr(openaiAuthJSON)
			}

			account, _, err = h.upsertImportedOAuthAccount(account, &existingAccounts)
			existingMu.Unlock()
			if err != nil {
				results[idx] = importResult{
					Index:        idx,
					Success:      false,
					Error:        err.Error(),
					Email:        finalEmail,
					TokenPreview: tokenPreview,
				}
				return
			}

			results[idx] = importResult{
				Index:        idx,
				Success:      true,
				Email:        finalEmail,
				Account:      account,
				TokenPreview: tokenPreview,
			}
		}(i, rt, preview)
	}

	wg.Wait()

	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}
	if successCount > 0 {
		refreshCodexProxyPool()
	}

	c.JSON(http.StatusOK, gin.H{
		"total":      len(req.RefreshTokens),
		"successful": successCount,
		"failed":     len(req.RefreshTokens) - successCount,
		"results":    results,
	})
}

// ImportOAuthAccountByRefreshToken 用单个 refresh_token 向 OpenAI 换票并 upsert 一条 OAuth 账号（供 CLI / 脚本）.
func (h *OpenAIHandler) ImportOAuthAccountByRefreshToken(refreshToken string) (*models.OpenAIAccount, error) {
	rt := strings.TrimSpace(refreshToken)
	if rt == "" {
		return nil, fmt.Errorf("empty refresh_token")
	}
	tokenResp, err := openaiplatform.RefreshToken(rt)
	if err != nil {
		return nil, err
	}
	var email string
	var chatgptAccountID, chatgptUserID, orgID *string
	var plan *string
	var openaiAuthJSON string
	if tokenResp.IDToken != "" {
		if userInfo := openaiplatform.ParseIDToken(tokenResp.IDToken); userInfo != nil {
			if userInfo.Email != nil {
				email = strings.TrimSpace(*userInfo.Email)
			}
			chatgptAccountID = userInfo.ChatGPTAccountID
			chatgptUserID = userInfo.ChatGPTUserID
			orgID = userInfo.OrganizationID
			plan = normalizedOpenAIPlanPtr(userInfo.PlanType)
		}
		openaiAuthJSON = openaiplatform.ExtractOpenAIAuthJSON(tokenResp.IDToken)
	}
	if email == "" {
		return nil, fmt.Errorf("failed to get email from id_token")
	}
	now := time.Now()
	var expiresAt *time.Time
	if tokenResp.ExpiresIn > 0 {
		t := now.Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		expiresAt = &t
	}
	refreshValue := rt
	if tokenResp.RefreshToken != "" {
		refreshValue = tokenResp.RefreshToken
	}
	account := &models.OpenAIAccount{
		Email:            email,
		AccountType:      models.OpenAIAccountTypeOAuth,
		AccessToken:      sPtr(tokenResp.AccessToken),
		RefreshToken:     sPtr(refreshValue),
		ExpiresAt:        expiresAt,
		ChatGPTAccountID: chatgptAccountID,
		ChatGPTUserID:    chatgptUserID,
		OrganizationID:   orgID,
		Plan:             plan,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if tokenResp.IDToken != "" {
		account.IDToken = sPtr(tokenResp.IDToken)
	}
	if openaiAuthJSON != "" {
		account.OpenAIAuthJSON = sPtr(openaiAuthJSON)
	}
	existingAccounts, err := h.storage.List()
	if err != nil {
		return nil, err
	}
	out, _, err := h.upsertImportedOAuthAccount(account, &existingAccounts)
	if err == nil && out != nil {
		refreshCodexProxyPool()
	}
	return out, err
}

// GenerateOAuthURL generates an OpenAI OAuth authorization URL
func (h *OpenAIHandler) GenerateOAuthURL(c *gin.Context) {
	var req struct {
		RedirectURI *string `json:"redirect_uri"`
	}
	c.ShouldBindJSON(&req)

	redirectURI := defaultOpenAIOAuthRedirectURI
	if req.RedirectURI != nil && *req.RedirectURI != "" {
		redirectURI = *req.RedirectURI
	}

	state := openaiplatform.GenerateState()
	codeVerifier := openaiplatform.GenerateCodeVerifier()
	codeChallenge := openaiplatform.GenerateCodeChallenge(codeVerifier)
	sessionID := uuid.New().String()
	authURL := openaiplatform.BuildAuthorizationURL(state, codeChallenge, redirectURI)

	h.mu.Lock()
	h.oauthSessions[sessionID] = &openaiOAuthSession{
		State:        state,
		CodeVerifier: codeVerifier,
		RedirectURI:  redirectURI,
		CreatedAt:    time.Now(),
	}
	h.mu.Unlock()

	autoCallbackEnabled := false
	autoCallbackError := ""
	if supportsAutoOpenAIOAuthCallback(redirectURI) {
		if err := h.ensureOAuthCallbackServer(); err != nil {
			autoCallbackError = fmt.Sprintf("自动回调不可用: %v", err)
		} else {
			autoCallbackEnabled = true
		}
	}

	payload := gin.H{
		"auth_url":              authURL,
		"session_id":            sessionID,
		"auto_callback_enabled": autoCallbackEnabled,
	}
	if autoCallbackError != "" {
		payload["auto_callback_error"] = autoCallbackError
	}
	c.JSON(http.StatusOK, payload)
}

func (h *OpenAIHandler) GetOAuthSession(c *gin.Context) {
	sessionID := c.Param("id")
	session, err := h.getOAuthSession(sessionID)
	if err != nil {
		status := http.StatusNotFound
		code := "SESSION_NOT_FOUND"
		message := "Session not found or expired"
		if errors.Is(err, errOpenAIOAuthSessionExpired) {
			status = http.StatusGone
			code = "SESSION_EXPIRED"
			message = "Session expired"
		}
		c.JSON(status, models.APIError{Error: message, Code: code})
		return
	}

	status := "pending"
	if session.LastError != "" {
		status = "error"
	} else if session.AuthorizationCode != "" {
		status = "callback_received"
	}

	expiresIn := int64(time.Until(session.CreatedAt.Add(openaiOAuthSessionTTL)).Seconds())
	if expiresIn < 0 {
		expiresIn = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id":                sessionID,
		"status":                    status,
		"authorization_code_ready":  session.AuthorizationCode != "",
		"expires_in_seconds":        expiresIn,
		"error":                     session.LastError,
		"callback_received_at_unix": timePtrUnix(session.CallbackReceivedAt),
	})
}

func (h *OpenAIHandler) CancelOAuthSession(c *gin.Context) {
	if !h.deleteOAuthSession(c.Param("id")) {
		c.JSON(http.StatusNotFound, models.APIError{Error: "Session not found or expired", Code: "SESSION_NOT_FOUND"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ExchangeCode completes OAuth login with the authorization code
func (h *OpenAIHandler) ExchangeCode(c *gin.Context) {
	var req struct {
		SessionID   string  `json:"session_id"`
		Code        string  `json:"code"`
		CallbackURL string  `json:"callback_url"`
		RedirectURI *string `json:"redirect_uri"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}

	session, err := h.getOAuthSession(req.SessionID)
	if err != nil {
		status := http.StatusBadRequest
		code := "SESSION_NOT_FOUND"
		message := "Session not found or expired"
		if errors.Is(err, errOpenAIOAuthSessionExpired) {
			status = http.StatusGone
			code = "SESSION_EXPIRED"
			message = "Session expired"
		}
		c.JSON(status, models.APIError{Error: message, Code: code})
		return
	}

	redirectURI := session.RedirectURI
	if req.RedirectURI != nil && *req.RedirectURI != "" {
		redirectURI = *req.RedirectURI
	}

	code, err := resolveOpenAIOAuthCode(session, req.Code, req.CallbackURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}

	tokenResp, err := openaiplatform.ExchangeCode(code, session.CodeVerifier, redirectURI)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "EXCHANGE_ERROR"})
		return
	}

	var email string
	var chatgptAccountID, chatgptUserID, orgID *string
	var plan *string
	var openaiAuthJSON string

	if tokenResp.IDToken != "" {
		if userInfo := openaiplatform.ParseIDToken(tokenResp.IDToken); userInfo != nil {
			if userInfo.Email != nil {
				email = strings.TrimSpace(*userInfo.Email)
			}
			chatgptAccountID = userInfo.ChatGPTAccountID
			chatgptUserID = userInfo.ChatGPTUserID
			orgID = userInfo.OrganizationID
			plan = normalizedOpenAIPlanPtr(userInfo.PlanType)
		}
		openaiAuthJSON = openaiplatform.ExtractOpenAIAuthJSON(tokenResp.IDToken)
	}

	if email == "" {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: "Failed to get email from token", Code: "NO_EMAIL"})
		return
	}

	now := time.Now()
	var expiresAt *time.Time
	if tokenResp.ExpiresIn > 0 {
		t := now.Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		expiresAt = &t
	}

	account := &models.OpenAIAccount{
		ID:               uuid.New().String(),
		Email:            email,
		AccountType:      models.OpenAIAccountTypeOAuth,
		AccessToken:      sPtr(tokenResp.AccessToken),
		RefreshToken:     sPtr(tokenResp.RefreshToken),
		IDToken:          sPtr(tokenResp.IDToken),
		ExpiresAt:        expiresAt,
		ChatGPTAccountID: chatgptAccountID,
		ChatGPTUserID:    chatgptUserID,
		OrganizationID:   orgID,
		Plan:             plan,
		ProxyEnabled:     true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if openaiAuthJSON != "" {
		account.OpenAIAuthJSON = sPtr(openaiAuthJSON)
	}

	existingAccounts, _ := h.storage.List()
	account, _, err = h.upsertImportedOAuthAccount(account, &existingAccounts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}

	if p := proxy.GetProxy(); p != nil {
		p.Refresh()
	}

	h.deleteOAuthSession(req.SessionID)
	c.JSON(http.StatusOK, gin.H{
		"account":            account,
		"proxy_enabled":      account.ProxyEnabled,
		"auto_joined_proxy":  account.ProxyEnabled,
		"authorization_mode": "auto",
	})
}

func supportsAutoOpenAIOAuthCallback(redirectURI string) bool {
	parsed, err := url.Parse(strings.TrimSpace(redirectURI))
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		return false
	}
	if parsed.Port() != "1455" {
		return false
	}
	return parsed.Path == "/auth/callback"
}

func (h *OpenAIHandler) ensureOAuthCallbackServer() error {
	h.mu.Lock()
	if h.oauthCallbackStarted {
		h.mu.Unlock()
		return nil
	}

	listener, err := net.Listen("tcp", defaultOpenAIOAuthCallbackAddr)
	if err != nil {
		h.mu.Unlock()
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/callback", h.handleOAuthCallback)
	mux.HandleFunc("/cancel", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("cancelled"))
	})

	server := &http.Server{Handler: mux}
	h.oauthCallbackListener = listener
	h.oauthCallbackServer = server
	h.oauthCallbackStarted = true
	h.mu.Unlock()

	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			h.mu.Lock()
			h.oauthCallbackStarted = false
			h.oauthCallbackServer = nil
			h.oauthCallbackListener = nil
			h.mu.Unlock()
		}
	}()

	return nil
}

func (h *OpenAIHandler) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	state := strings.TrimSpace(r.URL.Query().Get("state"))
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	oauthErr := strings.TrimSpace(r.URL.Query().Get("error"))
	oauthErrDescription := strings.TrimSpace(r.URL.Query().Get("error_description"))

	if state == "" {
		writeOpenAIOAuthCallbackHTML(w, http.StatusBadRequest, "OAuth failed", "Missing state in callback.")
		return
	}

	if err := h.recordOAuthCallback(state, code, oauthErr, oauthErrDescription); err != nil {
		status := http.StatusBadRequest
		title := "OAuth failed"
		message := err.Error()
		if errors.Is(err, errOpenAIOAuthSessionExpired) {
			status = http.StatusGone
			message = "This authorization session has expired. Please retry from EasyLLM."
		} else if errors.Is(err, errOpenAIOAuthSessionNotFound) {
			message = "This authorization session was not found. Please retry from EasyLLM."
		}
		writeOpenAIOAuthCallbackHTML(w, status, title, message)
		return
	}

	if oauthErr != "" {
		writeOpenAIOAuthCallbackHTML(w, http.StatusBadRequest, "OAuth cancelled", formatOpenAIOAuthError(oauthErr, oauthErrDescription))
		return
	}

	writeOpenAIOAuthCallbackHTML(w, http.StatusOK, "Authorization complete", "You can close this tab and return to EasyLLM.")
}

func writeOpenAIOAuthCallbackHTML(w http.ResponseWriter, status int, title, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s</title>
  <style>
    body { margin: 0; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: #0f172a; color: #e2e8f0; display: flex; align-items: center; justify-content: center; min-height: 100vh; }
    .card { width: min(92vw, 520px); background: rgba(15, 23, 42, 0.96); border: 1px solid #334155; border-radius: 20px; padding: 32px; box-shadow: 0 24px 60px rgba(15, 23, 42, 0.35); }
    h1 { margin: 0 0 12px; font-size: 28px; }
    p { margin: 0; line-height: 1.6; color: #cbd5e1; }
  </style>
</head>
<body>
  <div class="card">
    <h1>%s</h1>
    <p>%s</p>
  </div>
</body>
</html>`, html.EscapeString(title), html.EscapeString(title), html.EscapeString(message))
}

func (h *OpenAIHandler) recordOAuthCallback(state, code, oauthErr, oauthErrDescription string) error {
	now := time.Now()

	h.mu.Lock()
	defer h.mu.Unlock()

	for id, session := range h.oauthSessions {
		if session.State != state {
			continue
		}
		if now.Sub(session.CreatedAt) > openaiOAuthSessionTTL {
			delete(h.oauthSessions, id)
			return errOpenAIOAuthSessionExpired
		}
		session.CallbackReceivedAt = &now
		if oauthErr != "" {
			session.AuthorizationCode = ""
			session.LastError = formatOpenAIOAuthError(oauthErr, oauthErrDescription)
			return nil
		}
		if code == "" {
			return fmt.Errorf("missing authorization code")
		}
		session.AuthorizationCode = code
		session.LastError = ""
		return nil
	}

	return errOpenAIOAuthSessionNotFound
}

func (h *OpenAIHandler) getOAuthSession(sessionID string) (openaiOAuthSession, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	session, ok := h.oauthSessions[sessionID]
	if !ok {
		return openaiOAuthSession{}, errOpenAIOAuthSessionNotFound
	}
	if time.Since(session.CreatedAt) > openaiOAuthSessionTTL {
		delete(h.oauthSessions, sessionID)
		return openaiOAuthSession{}, errOpenAIOAuthSessionExpired
	}

	return *session, nil
}

func (h *OpenAIHandler) deleteOAuthSession(sessionID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.oauthSessions[sessionID]; !ok {
		return false
	}
	delete(h.oauthSessions, sessionID)
	return true
}

func resolveOpenAIOAuthCode(session openaiOAuthSession, rawCode, rawCallbackURL string) (string, error) {
	if code := strings.TrimSpace(rawCode); code != "" {
		return code, nil
	}
	if callbackURL := strings.TrimSpace(rawCallbackURL); callbackURL != "" {
		return extractOpenAIOAuthCodeFromCallbackURL(session.State, callbackURL)
	}
	if session.LastError != "" {
		return "", errors.New(session.LastError)
	}
	if code := strings.TrimSpace(session.AuthorizationCode); code != "" {
		return code, nil
	}
	return "", fmt.Errorf("authorization not completed yet")
}

func extractOpenAIOAuthCodeFromCallbackURL(expectedState, raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("callback URL is required")
	}
	if !strings.Contains(trimmed, "=") && !strings.Contains(trimmed, "://") && !strings.HasPrefix(trimmed, "/") && !strings.HasPrefix(trimmed, "?") {
		return trimmed, nil
	}

	var parsed *url.URL
	var err error
	switch {
	case strings.HasPrefix(trimmed, "http://"), strings.HasPrefix(trimmed, "https://"):
		parsed, err = url.Parse(trimmed)
	case strings.HasPrefix(trimmed, "/"):
		parsed, err = url.Parse(defaultOpenAIOAuthCallbackBase + trimmed)
	default:
		parsed, err = url.Parse(defaultOpenAIOAuthRedirectURI + "?" + strings.TrimLeft(trimmed, "?"))
	}
	if err != nil {
		return "", fmt.Errorf("invalid callback URL: %w", err)
	}

	query := parsed.Query()
	if state := strings.TrimSpace(query.Get("state")); expectedState != "" && state != "" && state != expectedState {
		return "", fmt.Errorf("callback state mismatch")
	}
	if oauthErr := strings.TrimSpace(query.Get("error")); oauthErr != "" {
		return "", errors.New(formatOpenAIOAuthError(oauthErr, strings.TrimSpace(query.Get("error_description"))))
	}

	code := strings.TrimSpace(query.Get("code"))
	if code == "" {
		return "", fmt.Errorf("authorization code not found in callback URL")
	}
	return code, nil
}

func formatOpenAIOAuthError(code, description string) string {
	if description != "" {
		return fmt.Sprintf("%s: %s", code, description)
	}
	if code == "" {
		return "oauth failed"
	}
	return code
}

func timePtrUnix(v *time.Time) any {
	if v == nil {
		return nil
	}
	return v.Unix()
}

// AddAPIAccount adds an API key-based Codex configuration
func (h *OpenAIHandler) AddAPIAccount(c *gin.Context) {
	var req struct {
		ModelProvider        string  `json:"model_provider"`
		Model                string  `json:"model"`
		ModelReasoningEffort *string `json:"model_reasoning_effort"`
		WireAPI              *string `json:"wire_api"`
		BaseURL              string  `json:"base_url"`
		APIKey               string  `json:"api_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	req.ModelProvider = strings.TrimSpace(req.ModelProvider)
	req.Model = strings.TrimSpace(req.Model)
	req.BaseURL = strings.TrimSpace(req.BaseURL)
	req.APIKey = strings.TrimSpace(req.APIKey)
	if req.ModelProvider == "" || req.Model == "" || req.BaseURL == "" || req.APIKey == "" {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "model_provider, model, base_url and api_key are required", Code: "INVALID_REQUEST"})
		return
	}

	wireAPI := "responses"
	if req.WireAPI != nil && *req.WireAPI != "" {
		wireAPI = *req.WireAPI
	}
	email := req.ModelProvider
	if email == "" {
		email = "API Account"
	}

	now := time.Now()
	account := &models.OpenAIAccount{
		ID:                   uuid.New().String(),
		Email:                email,
		AccountType:          models.OpenAIAccountTypeAPI,
		ModelProvider:        sPtr(req.ModelProvider),
		Model:                sPtr(req.Model),
		ModelReasoningEffort: req.ModelReasoningEffort,
		WireAPI:              sPtr(wireAPI),
		BaseURL:              sPtr(req.BaseURL),
		APIKey:               sPtr(req.APIKey),
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if err := h.storage.Save(account); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, sanitizeOpenAIAccountForResponse(account))
}

// UpdateAPIAccount updates an API account's configuration
func (h *OpenAIHandler) UpdateAPIAccount(c *gin.Context) {
	id := c.Param("id")
	existing, err := h.storage.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIError{Error: "Account not found", Code: "NOT_FOUND"})
		return
	}

	var req struct {
		ModelProvider        string  `json:"model_provider"`
		Model                string  `json:"model"`
		ModelReasoningEffort *string `json:"model_reasoning_effort"`
		WireAPI              *string `json:"wire_api"`
		BaseURL              string  `json:"base_url"`
		APIKey               string  `json:"api_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	req.ModelProvider = strings.TrimSpace(req.ModelProvider)
	req.Model = strings.TrimSpace(req.Model)
	req.BaseURL = strings.TrimSpace(req.BaseURL)
	req.APIKey = strings.TrimSpace(req.APIKey)
	if req.ModelProvider == "" || req.Model == "" || req.BaseURL == "" {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "model_provider, model and base_url are required", Code: "INVALID_REQUEST"})
		return
	}

	existing.Email = req.ModelProvider
	if existing.Email == "" {
		existing.Email = "API Account"
	}
	existing.ModelProvider = sPtr(req.ModelProvider)
	existing.Model = sPtr(req.Model)
	existing.ModelReasoningEffort = req.ModelReasoningEffort
	if req.WireAPI != nil && strings.TrimSpace(*req.WireAPI) != "" {
		existing.WireAPI = sPtr(strings.TrimSpace(*req.WireAPI))
	}
	existing.BaseURL = sPtr(req.BaseURL)
	if req.APIKey != "" {
		existing.APIKey = sPtr(req.APIKey)
	}
	existing.UpdatedAt = time.Now()

	if err := h.storage.Save(existing); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, sanitizeOpenAIAccountForResponse(existing))
}

// TestAPIAccount sends a minimal request to verify an API account's key is valid.
func (h *OpenAIHandler) TestAPIAccount(c *gin.Context) {
	id := c.Param("id")
	account, err := h.storage.Get(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIError{Error: "Account not found", Code: "NOT_FOUND"})
		return
	}
	if account.AccountType != models.OpenAIAccountTypeAPI {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "Not an API account", Code: "INVALID_REQUEST"})
		return
	}

	baseURL := derefStr(account.BaseURL)
	apiKey := derefStr(account.APIKey)
	model := derefStr(account.Model)
	if baseURL == "" || apiKey == "" {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "Account missing base_url or api_key", Code: "INVALID_REQUEST"})
		return
	}
	if model == "" {
		model = "gpt-4o-mini"
	}

	wireAPI := strings.ToLower(strings.TrimSpace(derefStr(account.WireAPI)))
	if wireAPI == "" {
		wireAPI = "responses"
	}
	requestPath := "/v1/responses"
	testReq := map[string]interface{}{
		"model":             model,
		"input":             "hi",
		"max_output_tokens": 1,
		"stream":            false,
	}
	if wireAPI == "chat_completions" || wireAPI == "chat-completions" || wireAPI == "chat" {
		requestPath = "/v1/chat/completions"
		testReq = map[string]interface{}{
			"model":      model,
			"messages":   []map[string]interface{}{{"role": "user", "content": "hi"}},
			"max_tokens": 1,
			"stream":     false,
		}
	}
	body, _ := json.Marshal(testReq)

	upstreamURL := buildAPIAccountTestURL(baseURL, requestPath)
	req, err := http.NewRequest("POST", upstreamURL, bytes.NewReader(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: "Failed to create request", Code: "INTERNAL_ERROR"})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	start := time.Now()
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"id": id, "success": false, "error": err.Error(), "latency_ms": latency})
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	result := gin.H{"id": id, "success": resp.StatusCode >= 200 && resp.StatusCode < 300, "http_status": resp.StatusCode, "latency_ms": latency}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result["error"] = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	c.JSON(http.StatusOK, result)
}

func buildAPIAccountTestURL(baseURL, requestPath string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	path := "/" + strings.TrimLeft(requestPath, "/")
	if strings.HasSuffix(base, "/v1") && strings.HasPrefix(path, "/v1/") {
		path = strings.TrimPrefix(path, "/v1")
	}
	return base + path
}

// ---- Codex Pool handlers ----

func (h *OpenAIHandler) ListCodexAccounts(c *gin.Context) {
	accounts, err := h.codexStorage.LoadAllAccounts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, sanitizeCodexAccountsForResponse(accounts))
}

func (h *OpenAIHandler) AddCodexAccount(c *gin.Context) {
	var account models.CodexAccount
	if err := c.ShouldBindJSON(&account); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	if account.ID == "" {
		account.ID = uuid.New().String()
	}
	account.Email = strings.TrimSpace(account.Email)
	account.AccessToken = strings.TrimSpace(account.AccessToken)
	if account.Email == "" || account.AccessToken == "" {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "email and access_token are required", Code: "INVALID_REQUEST"})
		return
	}
	account.CreatedAt = time.Now()
	account.UpdatedAt = time.Now()
	account.Enabled = true
	if err := h.codexStorage.SaveAccount(&account); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, sanitizeCodexAccountForResponse(&account))
}

func (h *OpenAIHandler) UpdateCodexAccount(c *gin.Context) {
	id := c.Param("id")
	existing, err := h.codexStorage.GetAccount(id)
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
	var account models.CodexAccount
	if err := c.ShouldBindJSON(&account); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	account.Email = strings.TrimSpace(account.Email)
	account.AccessToken = strings.TrimSpace(account.AccessToken)
	if account.AccessToken == "" {
		account.AccessToken = existing.AccessToken
	}
	if account.Email == "" || account.AccessToken == "" {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "email and access_token are required", Code: "INVALID_REQUEST"})
		return
	}
	account.ID = id
	account.CreatedAt = existing.CreatedAt
	account.RequestCount = existing.RequestCount
	account.LastUsedAt = existing.LastUsedAt
	account.UpdatedAt = time.Now()
	if err := h.codexStorage.SaveAccount(&account); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, sanitizeCodexAccountForResponse(&account))
}

func (h *OpenAIHandler) DeleteCodexAccount(c *gin.Context) {
	if err := h.codexStorage.DeleteAccount(c.Param("id")); err != nil {
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

func (h *OpenAIHandler) ToggleCodexAccount(c *gin.Context) {
	id := c.Param("id")
	account, err := h.codexStorage.GetAccount(id)
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
	account.Enabled = !account.Enabled
	if err := h.codexStorage.SaveAccount(account); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"enabled": account.Enabled})
}

func (h *OpenAIHandler) GetCodexPoolStatus(c *gin.Context) {
	accounts, _ := h.codexStorage.LoadAllAccounts()
	enabled := 0
	var totalRequests int64
	for _, a := range accounts {
		if a.Enabled {
			enabled++
		}
		totalRequests += a.RequestCount
	}
	accts := make([]models.CodexAccount, len(accounts))
	for i, a := range accounts {
		accts[i] = *a
		accts[i].AccessToken = ""
	}
	c.JSON(http.StatusOK, models.CodexPoolStatus{
		TotalAccounts:   len(accounts),
		EnabledAccounts: enabled,
		TotalRequests:   totalRequests,
		Accounts:        accts,
	})
}

func (h *OpenAIHandler) RefreshCodexPool(c *gin.Context) {
	if p := proxy.GetProxy(); p != nil {
		p.Refresh()
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Pool refreshed"})
}

// ToggleProxy toggles proxy_enabled for an OpenAI OAuth account.
// When enabled=true, the account's access_token joins the /v1/* proxy pool.
func (h *OpenAIHandler) ToggleProxy(c *gin.Context) {
	id := c.Param("id")
	enabled, err := h.storage.ToggleProxy(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIError{Error: "Account not found", Code: "NOT_FOUND"})
		return
	}

	// Immediately refresh proxy pool so the change takes effect without restart
	if p := proxy.GetProxy(); p != nil {
		p.Refresh()
	}

	c.JSON(http.StatusOK, gin.H{"proxy_enabled": enabled})
}

// ToggleProxyAll sets proxy_enabled for all OAuth accounts (one-click pool on/off).
// Body: { "enabled": true } or { "enabled": false }. /v1/chat/completions 轮询池一键开关.
func (h *OpenAIHandler) ToggleProxyAll(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "enabled (bool) required", Code: "INVALID_REQUEST"})
		return
	}
	count, err := h.storage.SetProxyAll(req.Enabled)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "INTERNAL_ERROR"})
		return
	}
	if p := proxy.GetProxy(); p != nil {
		p.Refresh()
	}
	c.JSON(http.StatusOK, gin.H{"enabled": req.Enabled, "updated_count": count})
}

// ---- Quota ----

// exportedOAuthAccount 是导出文件中 oauth_accounts 每条记录的格式。
// 导出时与 Codex / 常见 token JSON 一致（account_id、expired、disabled、last_refresh、type）；
// 导入仍兼容旧字段 chatgpt_account_id、expires_at。
type exportedOAuthAccount struct {
	ID               string `json:"id,omitempty"`
	AccessToken      string `json:"access_token"`
	AccountID        string `json:"account_id,omitempty"`
	Disabled         bool   `json:"disabled"`
	Email            string `json:"email"`
	Expired          string `json:"expired,omitempty"`
	IDToken          string `json:"id_token"`
	LastRefresh      string `json:"last_refresh,omitempty"`
	Plan             string `json:"plan,omitempty"`
	PlanType         string `json:"plan_type,omitempty"`
	RefreshToken     string `json:"refresh_token"`
	Type             string `json:"type"`
	ChatGPTAccountID string `json:"chatgpt_account_id,omitempty"`
	ExpiresAt        string `json:"expires_at,omitempty"`
	Status           string `json:"status,omitempty"`
}

func exportedOAuthExpiresSource(a exportedOAuthAccount) string {
	if strings.TrimSpace(a.Expired) != "" {
		return strings.TrimSpace(a.Expired)
	}
	return strings.TrimSpace(a.ExpiresAt)
}

func exportedOAuthChatGPTAccountID(a exportedOAuthAccount) string {
	if strings.TrimSpace(a.ChatGPTAccountID) != "" {
		return strings.TrimSpace(a.ChatGPTAccountID)
	}
	return strings.TrimSpace(a.AccountID)
}

func exportedOAuthPlan(a exportedOAuthAccount) *string {
	for _, value := range []string{a.Plan, a.PlanType} {
		if normalized := openaiplatform.NormalizePlanType(value); normalized != "" {
			return sPtr(normalized)
		}
	}
	if userInfo := openaiplatform.ParseIDToken(a.IDToken); userInfo != nil {
		if plan := normalizedOpenAIPlanPtr(userInfo.PlanType); plan != nil {
			return plan
		}
	}
	return nil
}

func parseExportedOAuthTime(s string) (*time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, false
	}
	layouts := []string{time.RFC3339, time.RFC3339Nano}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return &t, true
		}
	}
	return nil, false
}

// exportedAPIAccount 是导出文件中 api_accounts 数组里每条记录的格式
type exportedAPIAccount struct {
	ModelProvider        string `json:"model_provider"`
	Model                string `json:"model"`
	BaseURL              string `json:"base_url"`
	APIKey               string `json:"api_key"`
	WireAPI              string `json:"wire_api"`
	ModelReasoningEffort string `json:"model_reasoning_effort"`
	ProxyEnabled         bool   `json:"proxy_enabled"`
}

// ExportAccounts returns the latest persisted account snapshot from the backend.
// This ensures exports always reflect refreshed tokens / quota-triggered token updates
// that were already saved to the database.
func (h *OpenAIHandler) ExportAccounts(c *gin.Context) {
	accounts, err := h.storage.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}

	resp := struct {
		ExportedAt    string                             `json:"exported_at"`
		Usage         string                             `json:"_usage"`
		OAuthAccounts []exportedOAuthAccount             `json:"oauth_accounts"`
		APIAccounts   []exportedAPIAccount               `json:"api_accounts"`
		LocalAccess   *models.CodexLocalAccessCollection `json:"local_access,omitempty"`
	}{
		ExportedAt:  time.Now().UTC().Format(time.RFC3339),
		Usage:       "恢复时：在「批量导入 → 从备份导入」中上传此文件即可一键恢复所有账号，无需任何 API 调用。oauth_accounts 每条为 Codex 风格字段（account_id、expired、disabled、last_refresh、type 等）。仍兼容旧版 chatgpt_account_id / expires_at。请妥善保管此文件。",
		LocalAccess: h.codexLocalAccessState(c).Collection,
	}

	for _, a := range accounts {
		if a.AccountType == models.OpenAIAccountTypeAPI {
			resp.APIAccounts = append(resp.APIAccounts, exportedAPIAccount{
				ModelProvider:        derefStr(a.ModelProvider),
				Model:                derefStr(a.Model),
				BaseURL:              derefStr(a.BaseURL),
				APIKey:               derefStr(a.APIKey),
				WireAPI:              derefStr(a.WireAPI),
				ModelReasoningEffort: derefStr(a.ModelReasoningEffort),
				ProxyEnabled:         a.ProxyEnabled,
			})
			continue
		}

		var expiredStr, lastRefreshStr string
		if a.ExpiresAt != nil {
			expiredStr = a.ExpiresAt.In(time.Local).Format(time.RFC3339)
		}
		lastRefreshStr = a.UpdatedAt.In(time.Local).Format(time.RFC3339)

		resp.OAuthAccounts = append(resp.OAuthAccounts, exportedOAuthAccount{
			ID:           a.ID,
			AccessToken:  derefStr(a.AccessToken),
			AccountID:    derefStr(a.ChatGPTAccountID),
			Disabled:     a.Status != "" && a.Status != "active",
			Email:        a.Email,
			Expired:      expiredStr,
			IDToken:      derefStr(a.IDToken),
			LastRefresh:  lastRefreshStr,
			Plan:         derefStr(a.Plan),
			PlanType:     derefStr(a.Plan),
			RefreshToken: derefStr(a.RefreshToken),
			Type:         "codex",
			Status:       a.Status,
		})
	}

	c.JSON(http.StatusOK, resp)
}

// ImportFromExport 直接从导出的备份 JSON 中重新导入所有账号，无需任何 OpenAI API 调用。
// 支持导出文件中的 oauth_accounts 和 api_accounts 两类账号。
func (h *OpenAIHandler) ImportFromExport(c *gin.Context) {
	var payload easyLLMExportPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "无效的请求体: " + err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	if len(payload.OAuthAccounts) == 0 && len(payload.APIAccounts) == 0 && payload.LocalAccess == nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "备份文件中没有账号数据", Code: "EMPTY_INPUT"})
		return
	}
	existingAccounts, _ := h.storage.List()
	out, err := h.applyEasyLLMExportPayload(&payload, &existingAccounts)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	c.JSON(http.StatusOK, out)
}

// FetchQuotas checks the quota for OAuth accounts by calling the ChatGPT
// Codex Responses API (POST /codex/responses) and reading x-codex-* headers.
// Returns percentage-based 5h/7d quota data. Results are persisted to the database.
// Accepts optional {"ids": ["id1","id2"]} to query only specific accounts.
func (h *OpenAIHandler) FetchQuotas(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids"`
	}
	_ = c.ShouldBindJSON(&req)

	accounts, err := h.storage.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}

	type quotaResult struct {
		ID       string `json:"id"`
		Email    string `json:"email"`
		Success  bool   `json:"success"`
		Verified bool   `json:"verified,omitempty"`
		Total    int64  `json:"total,omitempty"`
		Used     int64  `json:"used,omitempty"`
		Reset    string `json:"reset,omitempty"`
		Error    string `json:"error,omitempty"`

		// New percentage-based fields
		Quota5hUsedPercent   *float64 `json:"quota_5h_used_percent,omitempty"`
		Quota5hResetSeconds  *int64   `json:"quota_5h_reset_seconds,omitempty"`
		Quota5hWindowMinutes *int64   `json:"quota_5h_window_minutes,omitempty"`
		Quota7dUsedPercent   *float64 `json:"quota_7d_used_percent,omitempty"`
		Quota7dResetSeconds  *int64   `json:"quota_7d_reset_seconds,omitempty"`
		Quota7dWindowMinutes *int64   `json:"quota_7d_window_minutes,omitempty"`
		IsForbidden          bool     `json:"is_forbidden,omitempty"`
		HTTPStatus           int      `json:"http_status,omitempty"`
	}

	idSet := make(map[string]bool, len(req.IDs))
	for _, id := range req.IDs {
		idSet[id] = true
	}

	var oauthAccounts []models.OpenAIAccount
	for _, a := range accounts {
		if a.AccountType == models.OpenAIAccountTypeOAuth && (derefStr(a.AccessToken) != "" || derefStr(a.RefreshToken) != "") {
			if len(idSet) > 0 && !idSet[a.ID] {
				continue
			}
			oauthAccounts = append(oauthAccounts, a)
		}
	}

	results := make([]quotaResult, len(oauthAccounts))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)

	for i, acc := range oauthAccounts {
		wg.Add(1)
		go func(idx int, account models.OpenAIAccount) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			chatgptID := ""
			if account.ChatGPTAccountID != nil {
				chatgptID = *account.ChatGPTAccountID
			}

			accessToken := derefStr(account.AccessToken)
			if accessToken == "" && derefStr(account.RefreshToken) != "" {
				if err := h.refreshOAuthAccountTokens(&account); err != nil {
					if saveErr := h.persistQuotaFailureState(&account, err.Error(), false); saveErr != nil {
						results[idx] = quotaResult{ID: account.ID, Email: account.Email, Success: false, Error: fmt.Sprintf("persist quota failed: %v", saveErr)}
						return
					}
					results[idx] = quotaResult{ID: account.ID, Email: account.Email, Success: false, Error: err.Error()}
					return
				}
				accessToken = derefStr(account.AccessToken)
				chatgptID = derefStr(account.ChatGPTAccountID)
			}

			info, err := openaiplatform.FetchQuota(accessToken, chatgptID)
			if err != nil && isQuotaUnauthorized(err) && derefStr(account.RefreshToken) != "" {
				if refreshErr := h.refreshOAuthAccountTokens(&account); refreshErr != nil {
					if saveErr := h.persistQuotaFailureState(&account, refreshErr.Error(), false); saveErr != nil {
						results[idx] = quotaResult{ID: account.ID, Email: account.Email, Success: false, Error: fmt.Sprintf("persist quota failed: %v", saveErr)}
						return
					}
					results[idx] = quotaResult{ID: account.ID, Email: account.Email, Success: false, Error: refreshErr.Error()}
					return
				}
				accessToken = derefStr(account.AccessToken)
				chatgptID = derefStr(account.ChatGPTAccountID)
				info, err = openaiplatform.FetchQuota(accessToken, chatgptID)
			}
			if err != nil {
				if saveErr := h.persistQuotaFailureState(&account, err.Error(), false); saveErr != nil {
					results[idx] = quotaResult{ID: account.ID, Email: account.Email, Success: false, Error: fmt.Sprintf("persist quota failed: %v", saveErr)}
					return
				}
				results[idx] = quotaResult{ID: account.ID, Email: account.Email, Success: false, Error: err.Error()}
				return
			}
			if info == nil {
				account.QuotaVerified = true
				account.QuotaError = nil
				http200 := 200
				account.QuotaHTTPStatus = &http200
				now := time.Now()
				account.QuotaUpdatedAt = &now
				if saveErr := h.storage.Save(&account); saveErr != nil {
					results[idx] = quotaResult{ID: account.ID, Email: account.Email, Success: false, Error: fmt.Sprintf("persist quota failed: %v", saveErr)}
					return
				}
				results[idx] = quotaResult{ID: account.ID, Email: account.Email, Success: true, Verified: true}
				return
			}

			if info.IsForbidden {
				account.QuotaIsForbidden = true
				account.QuotaVerified = false
				account.QuotaError = nil
				http403 := 403
				account.QuotaHTTPStatus = &http403
				now := time.Now()
				account.QuotaUpdatedAt = &now
				if err := h.storage.Save(&account); err != nil {
					results[idx] = quotaResult{ID: account.ID, Email: account.Email, Success: false, Error: fmt.Sprintf("persist quota failed: %v", err)}
					return
				}
				results[idx] = quotaResult{ID: account.ID, Email: account.Email, Success: true, IsForbidden: true}
				return
			}

			account.QuotaIsForbidden = false
			account.QuotaVerified = false
			account.QuotaError = nil
			http200 := 200
			account.QuotaHTTPStatus = &http200
			account.QuotaTotal = &info.Total
			account.QuotaUsed = &info.Used
			if info.ResetAt != "" {
				account.QuotaResetAt = &info.ResetAt
			}
			account.Quota5hUsedPercent = info.Codex5hUsedPercent
			account.Quota5hResetSeconds = info.Codex5hResetSeconds
			account.Quota5hWindowMinutes = info.Codex5hWindowMinutes
			account.Quota7dUsedPercent = info.Codex7dUsedPercent
			account.Quota7dResetSeconds = info.Codex7dResetSeconds
			account.Quota7dWindowMinutes = info.Codex7dWindowMinutes
			if plan := normalizedOpenAIPlanPtr(info.PlanType); plan != nil {
				account.Plan = plan
			}
			now := time.Now()
			account.QuotaUpdatedAt = &now
			if err := h.storage.Save(&account); err != nil {
				results[idx] = quotaResult{ID: account.ID, Email: account.Email, Success: false, Error: fmt.Sprintf("persist quota failed: %v", err)}
				return
			}

			results[idx] = quotaResult{
				ID:                   account.ID,
				Email:                account.Email,
				Success:              true,
				Total:                info.Total,
				Used:                 info.Used,
				Reset:                info.ResetAt,
				Quota5hUsedPercent:   info.Codex5hUsedPercent,
				Quota5hResetSeconds:  info.Codex5hResetSeconds,
				Quota5hWindowMinutes: info.Codex5hWindowMinutes,
				Quota7dUsedPercent:   info.Codex7dUsedPercent,
				Quota7dResetSeconds:  info.Codex7dResetSeconds,
				Quota7dWindowMinutes: info.Codex7dWindowMinutes,
			}
		}(i, acc)
	}

	wg.Wait()

	// Fill http_status from persisted account state so the frontend can display it
	for i, r := range results {
		if r.Success && !r.IsForbidden {
			r.HTTPStatus = 200
		} else if r.ID != "" {
			if acc, err := h.storage.Get(r.ID); err == nil && acc.QuotaHTTPStatus != nil {
				r.HTTPStatus = *acc.QuotaHTTPStatus
			}
		}
		results[i] = r
	}

	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total":   len(oauthAccounts),
		"success": successCount,
		"failed":  len(oauthAccounts) - successCount,
		"results": results,
	})
}

// ---- Service Config ----

// GetServiceConfig returns proxy pool status, masked API key state, strategy, and request stats.
func (h *OpenAIHandler) GetServiceConfig(c *gin.Context) {
	c.JSON(http.StatusOK, h.serviceConfigPayload(c))
}

func (h *OpenAIHandler) serviceConfigPayload(c *gin.Context) gin.H {
	p := proxy.GetProxy()
	enabled := false
	strategy := "round_robin"
	poolSize := 0
	totalReqs := int64(0)
	if p != nil {
		enabled = p.IsEnabled()
		strategy = p.GetStrategy()
		poolSize = p.PoolSize()
		totalReqs = p.TotalRequests()
	}

	apiKey, _ := storage.GetSetting("proxy_api_key")
	maskedKey := ""
	if apiKey != "" {
		if len(apiKey) > 8 {
			maskedKey = apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
		} else {
			maskedKey = "****"
		}
	}

	proxyCount, _ := h.storage.CountProxyEnabled()
	v1ProxyMode, _ := storage.GetSetting("v1_proxy_mode")
	codexAPIBaseURL := buildCodexAPIServiceBaseURL(c)

	return gin.H{
		"proxy_pool_enabled":    enabled,
		"strategy":              strategy,
		"pool_size":             poolSize,
		"proxy_enabled_count":   proxyCount,
		"total_requests":        totalReqs,
		"request_logs_retained": false,
		"api_key_set":           apiKey != "",
		"api_key":               apiKey,
		"api_key_masked":        maskedKey,
		"v1_proxy_mode":         v1ProxyMode,
		"codex_api_service":     enabled && v1ProxyMode == "codex",
		"codex_api_base_url":    codexAPIBaseURL,
		"codex_api_port_url":    strings.TrimRight(codexAPIBaseURL, "/") + "/responses",
	}
}

// UpdateServiceConfig updates proxy pool enabled, strategy, and API key.
func (h *OpenAIHandler) UpdateServiceConfig(c *gin.Context) {
	var req struct {
		ProxyPoolEnabled *bool   `json:"proxy_pool_enabled,omitempty"`
		Strategy         *string `json:"strategy,omitempty"`
		APIKey           *string `json:"api_key,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "Invalid request body", Code: "INVALID_REQUEST"})
		return
	}

	p := proxy.GetProxy()

	if req.ProxyPoolEnabled != nil && p != nil {
		p.SetEnabled(*req.ProxyPoolEnabled)
		storage.SaveSetting("proxy_pool_enabled", fmt.Sprintf("%v", *req.ProxyPoolEnabled))
	}
	if req.Strategy != nil && p != nil {
		if !isValidCodexProxyStrategy(*req.Strategy) {
			c.JSON(http.StatusBadRequest, models.APIError{Error: "invalid strategy", Code: "INVALID_REQUEST"})
			return
		}
		p.SetStrategy(*req.Strategy)
		storage.SaveSetting("proxy_strategy", *req.Strategy)
		storage.SaveSetting(codexLocalAccessRoutingStrategyKey, *req.Strategy)
	}
	if req.APIKey != nil {
		storage.SaveSetting("proxy_api_key", strings.TrimSpace(*req.APIKey))
	}

	h.GetServiceConfig(c)
}

// ActivateCodexAPIService enables the local proxy service and injects it into
// the user's ~/.codex config so Codex CLI can call EasyLLM directly.
func (h *OpenAIHandler) ActivateCodexAPIService(c *gin.Context) {
	apiKeyBefore, _ := storage.GetSetting("proxy_api_key")
	if err := h.activateCodexLocalAccess(c); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "LOCAL_ACCESS_ERROR"})
		return
	}
	launchResult, err := openaiplatform.RestartCodexApp()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "CODEX_LAUNCH_FAILED"})
		return
	}

	payload := h.serviceConfigPayload(c)
	payload["codex_config_injected"] = true
	payload["api_key_generated"] = strings.TrimSpace(apiKeyBefore) == ""
	payload["codex_app_started"] = launchResult.Started
	payload["codex_app_restarted"] = launchResult.Restarted
	payload["codex_app_was_running"] = launchResult.RunningBefore
	c.JSON(http.StatusOK, payload)
}

func (h *OpenAIHandler) GetCodexLocalAccess(c *gin.Context) {
	c.JSON(http.StatusOK, h.codexLocalAccessState(c))
}

func (h *OpenAIHandler) ActivateCodexLocalAccess(c *gin.Context) {
	if err := h.activateCodexLocalAccess(c); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "LOCAL_ACCESS_ERROR"})
		return
	}
	c.JSON(http.StatusOK, h.codexLocalAccessState(c))
}

func (h *OpenAIHandler) DeactivateCodexLocalAccess(c *gin.Context) {
	_ = storage.SaveSetting(codexLocalAccessEnabledKey, "false")
	_ = storage.SaveSetting("proxy_pool_enabled", "false")
	_ = storage.SaveSetting("v1_proxy_mode", "")
	if p := proxy.GetProxy(); p != nil {
		p.SetEnabled(false)
	}
	apiKey, _ := storage.GetSetting("proxy_api_key")
	_ = openaiplatform.RemoveCodexAPIService(apiKey)
	c.JSON(http.StatusOK, h.codexLocalAccessState(c))
}

func (h *OpenAIHandler) SetCodexLocalAccessEnabled(c *gin.Context) {
	var req struct {
		Enabled *bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "enabled is required", Code: "INVALID_REQUEST"})
		return
	}
	if *req.Enabled {
		if err := h.activateCodexLocalAccess(c); err != nil {
			c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "LOCAL_ACCESS_ERROR"})
			return
		}
		c.JSON(http.StatusOK, h.codexLocalAccessState(c))
		return
	}
	h.DeactivateCodexLocalAccess(c)
}

func (h *OpenAIHandler) SaveCodexLocalAccessAccounts(c *gin.Context) {
	var req struct {
		AccountIDs           []string `json:"account_ids"`
		RestrictFreeAccounts *bool    `json:"restrict_free_accounts"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "invalid request body", Code: "INVALID_REQUEST"})
		return
	}
	restrictFree := true
	if req.RestrictFreeAccounts != nil {
		restrictFree = *req.RestrictFreeAccounts
	}
	ids, err := h.filterCodexLocalAccessAccountIDs(req.AccountIDs, restrictFree)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	if err := saveCodexLocalAccessAccountIDs(ids); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "INTERNAL_ERROR"})
		return
	}
	_ = storage.SaveSetting(codexLocalAccessRestrictFreeAccountsKey, fmt.Sprintf("%v", restrictFree))
	_ = saveCodexLocalAccessTimestamp(false)
	if h.storage != nil && len(ids) > 0 {
		_, _ = h.storage.SetProxyForIDs(ids, true)
	}
	if p := proxy.GetProxy(); p != nil {
		p.Refresh()
	}
	c.JSON(http.StatusOK, h.codexLocalAccessState(c))
}

func (h *OpenAIHandler) RemoveCodexLocalAccessAccount(c *gin.Context) {
	target := strings.TrimSpace(c.Param("id"))
	ids := readCodexLocalAccessAccountIDs()
	next := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != target {
			next = append(next, id)
		}
	}
	if err := saveCodexLocalAccessAccountIDs(next); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "INTERNAL_ERROR"})
		return
	}
	_ = saveCodexLocalAccessTimestamp(false)
	if p := proxy.GetProxy(); p != nil {
		p.Refresh()
	}
	c.JSON(http.StatusOK, h.codexLocalAccessState(c))
}

func (h *OpenAIHandler) UpdateCodexLocalAccessPort(c *gin.Context) {
	var req struct {
		Port int `json:"port"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Port <= 0 || req.Port > 65535 {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "port must be between 1 and 65535", Code: "INVALID_REQUEST"})
		return
	}
	currentPort := config.Get().Server.Port
	if currentPort > 0 && req.Port != currentPort {
		c.JSON(http.StatusBadRequest, models.APIError{Error: fmt.Sprintf("port must match the running EasyLLM server port (%d)", currentPort), Code: "INVALID_REQUEST"})
		return
	}
	_ = storage.SaveSetting(codexLocalAccessPortKey, strconv.Itoa(req.Port))
	_ = saveCodexLocalAccessTimestamp(false)
	if readBoolSetting(codexLocalAccessEnabledKey, false) {
		apiKey, _ := storage.GetSetting("proxy_api_key")
		if apiKey != "" {
			if err := openaiplatform.SwitchCodexAPIService(buildCodexAPIServiceBaseURL(c), apiKey); err != nil {
				c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "CODEX_CONFIG_WRITE_FAILED"})
				return
			}
		}
	}
	c.JSON(http.StatusOK, h.codexLocalAccessState(c))
}

func (h *OpenAIHandler) UpdateCodexLocalAccessRoutingStrategy(c *gin.Context) {
	var req struct {
		Strategy string `json:"strategy"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "invalid request body", Code: "INVALID_REQUEST"})
		return
	}
	strategy := strings.TrimSpace(req.Strategy)
	if !isValidCodexProxyStrategy(strategy) {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "invalid strategy", Code: "INVALID_REQUEST"})
		return
	}
	_ = storage.SaveSetting(codexLocalAccessRoutingStrategyKey, strategy)
	_ = storage.SaveSetting("proxy_strategy", strategy)
	_ = saveCodexLocalAccessTimestamp(false)
	if p := proxy.GetProxy(); p != nil {
		p.SetStrategy(strategy)
		p.Refresh()
	}
	c.JSON(http.StatusOK, h.codexLocalAccessState(c))
}

func (h *OpenAIHandler) RotateCodexLocalAccessAPIKey(c *gin.Context) {
	apiKey, err := generateCodexAPIServiceKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "INTERNAL_ERROR"})
		return
	}
	if err := storage.SaveSetting("proxy_api_key", apiKey); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "INTERNAL_ERROR"})
		return
	}
	_ = saveCodexLocalAccessTimestamp(false)
	if readBoolSetting(codexLocalAccessEnabledKey, false) {
		if err := openaiplatform.SwitchCodexAPIService(buildCodexAPIServiceBaseURL(c), apiKey); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "CODEX_CONFIG_WRITE_FAILED"})
			return
		}
	}
	c.JSON(http.StatusOK, h.codexLocalAccessState(c))
}

func (h *OpenAIHandler) ClearCodexLocalAccessStats(c *gin.Context) {
	c.JSON(http.StatusOK, h.codexLocalAccessState(c))
}

func (h *OpenAIHandler) KillCodexLocalAccessPort(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"killed_count": 0,
		"state":        h.codexLocalAccessState(c),
	})
}

func (h *OpenAIHandler) activateCodexLocalAccess(c *gin.Context) error {
	p := proxy.GetProxy()
	if p == nil {
		return errors.New("Codex proxy is not initialized")
	}
	if h.storage == nil {
		return errors.New("OpenAI storage is not initialized")
	}

	ids := readCodexLocalAccessAccountIDs()
	accounts, err := h.storage.List()
	if err != nil {
		return err
	}
	restrictFree := readBoolSetting(codexLocalAccessRestrictFreeAccountsKey, true)
	if codexLocalAccessAccountIDsConfigured() {
		ids, err = h.filterCodexLocalAccessAccountIDs(ids, restrictFree)
		if err != nil {
			return err
		}
		if err := saveCodexLocalAccessAccountIDs(ids); err != nil {
			return err
		}
	} else {
		ids = []string{}
		for _, account := range accounts {
			if !isCodexLocalAccessEligibleAccount(account, restrictFree) {
				continue
			}
			ids = append(ids, account.ID)
		}
		if len(ids) > 0 {
			if err := saveCodexLocalAccessAccountIDs(ids); err != nil {
				return err
			}
		}
	}
	if len(ids) == 0 {
		return errors.New("没有 200 成功的 OAuth 账号，请先查询配额")
	}
	if h.storage != nil {
		_, _ = h.storage.SetProxyForIDs(ids, true)
	}

	strategy := readStringSetting(codexLocalAccessRoutingStrategyKey, readStringSetting("proxy_strategy", "auto"))
	if !isValidCodexProxyStrategy(strategy) {
		strategy = "auto"
	}
	p.SetStrategy(strategy)
	p.SetEnabled(true)
	p.Refresh()

	if p.PoolSize() == 0 {
		return errors.New("没有可用代理池账号，请检查账号 token 或订阅状态")
	}

	_ = storage.SaveSetting(codexLocalAccessEnabledKey, "true")
	_ = storage.SaveSetting(codexLocalAccessRoutingStrategyKey, strategy)
	_ = storage.SaveSetting("proxy_strategy", strategy)
	_ = storage.SaveSetting("proxy_pool_enabled", "true")
	_ = storage.SaveSetting("v1_proxy_mode", "codex")
	_ = saveCodexLocalAccessTimestamp(true)

	apiKey, _ := storage.GetSetting("proxy_api_key")
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		var err error
		apiKey, err = generateCodexAPIServiceKey()
		if err != nil {
			return err
		}
		if err := storage.SaveSetting("proxy_api_key", apiKey); err != nil {
			return err
		}
	}

	return openaiplatform.SwitchCodexAPIService(buildCodexAPIServiceBaseURL(c), apiKey)
}

func generateCodexAPIServiceKey() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to generate API key: %w", err)
	}
	return "easyllm_codex_" + strings.TrimRight(base64.RawURLEncoding.EncodeToString(buf), "="), nil
}

func localProxyOriginFromRequest(c *gin.Context) string {
	if c != nil && c.Request != nil {
		if host := strings.TrimSpace(c.Request.Host); host != "" {
			return openaiplatform.LocalProxyOrigin(host)
		}
	}
	return openaiplatform.LocalProxyOrigin("")
}

func buildCodexAPIServiceBaseURL(c *gin.Context) string {
	host := "localhost:8022"
	if c != nil && c.Request != nil {
		if requestHost := strings.TrimSpace(c.Request.Host); requestHost != "" {
			host = requestHost
		}
	}
	hostOnly, port, err := net.SplitHostPort(host)
	if err == nil {
		if hostOnly == "" || hostOnly == "0.0.0.0" || hostOnly == "::" || hostOnly == "[::]" || strings.EqualFold(hostOnly, "localhost") || net.ParseIP(hostOnly) != nil {
			hostOnly = "localhost"
		}
		if serverPort := config.Get().Server.Port; serverPort > 0 {
			port = strconv.Itoa(serverPort)
		}
		host = net.JoinHostPort(hostOnly, port)
	} else {
		hostOnly := strings.TrimSpace(host)
		if hostOnly == "" || hostOnly == "0.0.0.0" || hostOnly == "::" || hostOnly == "[::]" || strings.EqualFold(hostOnly, "localhost") || net.ParseIP(hostOnly) != nil {
			hostOnly = "localhost"
		}
		port := "8022"
		if serverPort := config.Get().Server.Port; serverPort > 0 {
			port = strconv.Itoa(serverPort)
		}
		host = net.JoinHostPort(hostOnly, port)
	}
	scheme := "http"
	if c != nil && c.Request != nil && c.Request.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/v1", scheme, host)
}

func (h *OpenAIHandler) codexLocalAccessState(c *gin.Context) models.CodexLocalAccessState {
	baseURL := buildCodexAPIServiceBaseURL(c)
	apiKey, _ := storage.GetSetting("proxy_api_key")
	collection := &models.CodexLocalAccessCollection{
		Enabled:              readBoolSetting(codexLocalAccessEnabledKey, false),
		Port:                 localAccessPortFromBaseURL(baseURL),
		APIKeyMasked:         maskAPIKey(apiKey),
		RoutingStrategy:      readStringSetting(codexLocalAccessRoutingStrategyKey, readStringSetting("proxy_strategy", "auto")),
		RestrictFreeAccounts: readBoolSetting(codexLocalAccessRestrictFreeAccountsKey, true),
		AccountIDs:           readCodexLocalAccessAccountIDs(),
		CreatedAt:            readStringSetting(codexLocalAccessCreatedAtKey, ""),
		UpdatedAt:            readStringSetting(codexLocalAccessUpdatedAtKey, ""),
	}
	if !isValidCodexProxyStrategy(collection.RoutingStrategy) {
		collection.RoutingStrategy = "auto"
	}
	running := false
	if p := proxy.GetProxy(); p != nil {
		running = p.IsEnabled() && collection.Enabled
	}
	return models.CodexLocalAccessState{
		Collection:  collection,
		Running:     running,
		BaseURL:     baseURL,
		APIPortURL:  strings.TrimRight(baseURL, "/") + "/responses",
		ModelIDs:    []string{"gpt-5-codex", "gpt-5-codex-mini", "gpt-5.4", "gpt-5.4-mini", "gpt-5.3-codex", "gpt-image-2"},
		MemberCount: len(collection.AccountIDs),
		Stats:       h.buildCodexLocalAccessStats(),
	}
}

func (h *OpenAIHandler) buildCodexLocalAccessStats() models.CodexLocalAccessStats {
	now := time.Now()
	return models.CodexLocalAccessStats{
		Daily:   h.buildCodexLocalAccessStatsWindow(now.Add(-24*time.Hour), now),
		Weekly:  h.buildCodexLocalAccessStatsWindow(now.Add(-7*24*time.Hour), now),
		Monthly: h.buildCodexLocalAccessStatsWindow(now.Add(-30*24*time.Hour), now),
	}
}

func (h *OpenAIHandler) buildCodexLocalAccessStatsWindow(since, now time.Time) models.CodexLocalAccessStatsWindow {
	return models.CodexLocalAccessStatsWindow{
		Since:     since.UTC().Format(time.RFC3339),
		UpdatedAt: now.UTC().Format(time.RFC3339),
		Accounts:  []models.CodexLocalAccessAccountStats{},
	}
}

func (h *OpenAIHandler) filterCodexLocalAccessAccountIDs(ids []string, restrictFree bool) ([]string, error) {
	if h.storage == nil {
		return nil, errors.New("OpenAI storage is not initialized")
	}
	accounts, err := h.storage.List()
	if err != nil {
		return nil, err
	}
	byID := make(map[string]models.OpenAIAccount, len(accounts))
	for _, account := range accounts {
		if account.AccountType == models.OpenAIAccountTypeOAuth {
			byID[account.ID] = account
		}
	}
	seen := map[string]bool{}
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		account, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("账号不存在: %s", id)
		}
		if !isCodexLocalAccessEligibleAccount(account, restrictFree) {
			continue
		}
		result = append(result, id)
		seen[id] = true
	}
	return result, nil
}

func isCodexLocalAccessEligibleAccount(account models.OpenAIAccount, restrictFree bool) bool {
	if account.AccountType != models.OpenAIAccountTypeOAuth {
		return false
	}
	if derefStr(account.AccessToken) == "" {
		return false
	}
	if account.QuotaIsForbidden {
		return false
	}
	if account.QuotaHTTPStatus == nil || *account.QuotaHTTPStatus != http.StatusOK {
		return false
	}
	if restrictFree && strings.EqualFold(strings.TrimSpace(derefStr(account.Plan)), "free") {
		return false
	}
	return true
}

func readCodexLocalAccessAccountIDs() []string {
	raw, ok := storage.GetSetting(codexLocalAccessAccountIDsKey)
	if !ok || strings.TrimSpace(raw) == "" {
		return []string{}
	}
	var ids []string
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return []string{}
	}
	result := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" && !seen[id] {
			result = append(result, id)
			seen[id] = true
		}
	}
	return result
}

func codexLocalAccessAccountIDsConfigured() bool {
	raw, ok := storage.GetSetting(codexLocalAccessAccountIDsKey)
	return ok && strings.TrimSpace(raw) != ""
}

func saveCodexLocalAccessAccountIDs(ids []string) error {
	data, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	return storage.SaveSetting(codexLocalAccessAccountIDsKey, string(data))
}

func appendCodexLocalAccessAccountID(id string) (bool, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}
	ids := readCodexLocalAccessAccountIDs()
	for _, existing := range ids {
		if existing == id {
			return false, nil
		}
	}
	ids = append(ids, id)
	return true, saveCodexLocalAccessAccountIDs(ids)
}

func refreshCodexProxyPool() {
	if p := proxy.GetProxy(); p != nil {
		p.Refresh()
	}
}

func isDefaultAPIServiceOAuthAccount(account *models.OpenAIAccount) bool {
	if account == nil {
		return false
	}
	if account.AccountType != "" && account.AccountType != models.OpenAIAccountTypeOAuth {
		return false
	}
	status := strings.ToLower(strings.TrimSpace(account.Status))
	if status != "" && status != "active" {
		return false
	}
	if strings.TrimSpace(derefStr(account.AccessToken)) == "" {
		return false
	}
	if account.ExpiresAt != nil && !account.ExpiresAt.After(time.Now()) {
		return false
	}
	return true
}

func shouldAddDefaultAPIServiceCollectionMember(account *models.OpenAIAccount) bool {
	if !isDefaultAPIServiceOAuthAccount(account) {
		return false
	}
	if readBoolSetting(codexLocalAccessRestrictFreeAccountsKey, true) &&
		strings.EqualFold(strings.TrimSpace(derefStr(account.Plan)), "free") {
		return false
	}
	return true
}

func (h *OpenAIHandler) applyDefaultAPIServiceMembership(account *models.OpenAIAccount) bool {
	if !isDefaultAPIServiceOAuthAccount(account) {
		return false
	}
	account.ProxyEnabled = true
	return true
}

func (h *OpenAIHandler) saveDefaultAPIServiceMembership(account *models.OpenAIAccount) error {
	if !shouldAddDefaultAPIServiceCollectionMember(account) {
		return nil
	}
	added, err := appendCodexLocalAccessAccountID(account.ID)
	if err != nil {
		return err
	}
	if added {
		return saveCodexLocalAccessTimestamp(false)
	}
	return nil
}

func saveCodexLocalAccessTimestamp(createIfMissing bool) error {
	now := time.Now().UTC().Format(time.RFC3339)
	if createIfMissing {
		if _, ok := storage.GetSetting(codexLocalAccessCreatedAtKey); !ok {
			if err := storage.SaveSetting(codexLocalAccessCreatedAtKey, now); err != nil {
				return err
			}
		}
	}
	return storage.SaveSetting(codexLocalAccessUpdatedAtKey, now)
}

func readStringSetting(key, fallback string) string {
	if value, ok := storage.GetSetting(key); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func readBoolSetting(key string, fallback bool) bool {
	value, ok := storage.GetSetting(key)
	if !ok {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return fallback
	}
}

func readIntSetting(key string, fallback int) int {
	value, ok := storage.GetSetting(key)
	if !ok {
		return fallback
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func localAccessPortFromBaseURL(baseURL string) int {
	u, err := url.Parse(baseURL)
	if err != nil {
		return readIntSetting(codexLocalAccessPortKey, 8022)
	}
	if p := u.Port(); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			return parsed
		}
	}
	return readIntSetting(codexLocalAccessPortKey, 8022)
}

func maskAPIKey(apiKey string) string {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return ""
	}
	if len(apiKey) <= 8 {
		return "****"
	}
	return apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
}

func isValidCodexProxyStrategy(strategy string) bool {
	switch strings.TrimSpace(strategy) {
	case "round_robin", "random", "least_used", "auto", "quota_high_first", "quota_low_first", "plan_high_first", "plan_low_first", "expiry_soon_first":
		return true
	default:
		return false
	}
}

// ---- helpers ----

func (h *OpenAIHandler) refreshOAuthAccountTokens(account *models.OpenAIAccount) error {
	if account == nil {
		return fmt.Errorf("account is nil")
	}
	if account.AccountType == models.OpenAIAccountTypeAPI {
		return fmt.Errorf("API accounts do not support token refresh")
	}
	if account.RefreshToken == nil || *account.RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	tokenResp, err := openaiplatform.RefreshToken(*account.RefreshToken)
	if err != nil {
		if isRefreshTokenReusedError(err) {
			h.markOAuthAccountReauthRequired(account)
			return fmt.Errorf("refresh_token 已轮换失效，需要重新 OAuth 登录或重新导入最新账号")
		}
		return err
	}

	account.Status = "active"
	account.AccessToken = sPtr(tokenResp.AccessToken)
	if tokenResp.RefreshToken != "" {
		account.RefreshToken = sPtr(tokenResp.RefreshToken)
	}
	if tokenResp.IDToken != "" {
		account.IDToken = sPtr(tokenResp.IDToken)
		if userInfo := openaiplatform.ParseIDToken(tokenResp.IDToken); userInfo != nil {
			if userInfo.Email != nil && *userInfo.Email != "" {
				account.Email = *userInfo.Email
			}
			account.ChatGPTAccountID = userInfo.ChatGPTAccountID
			account.ChatGPTUserID = userInfo.ChatGPTUserID
			account.OrganizationID = userInfo.OrganizationID
			if plan := normalizedOpenAIPlanPtr(userInfo.PlanType); plan != nil {
				account.Plan = plan
			}
		}
		if j := openaiplatform.ExtractOpenAIAuthJSON(tokenResp.IDToken); j != "" {
			account.OpenAIAuthJSON = sPtr(j)
		}
	}
	if tokenResp.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		account.ExpiresAt = &t
	}
	defaultJoined := h.applyDefaultAPIServiceMembership(account)
	account.UpdatedAt = time.Now()
	if err := h.storage.Save(account); err != nil {
		return fmt.Errorf("persist refreshed token failed: %w", err)
	}
	if defaultJoined {
		if err := h.saveDefaultAPIServiceMembership(account); err != nil {
			return fmt.Errorf("persist API service membership failed: %w", err)
		}
	}
	return nil
}

func (h *OpenAIHandler) persistQuotaFailureState(account *models.OpenAIAccount, message string, forbidden bool) error {
	if account == nil {
		return fmt.Errorf("account is nil")
	}
	account.QuotaVerified = false
	account.QuotaIsForbidden = forbidden
	account.QuotaError = &message
	account.Quota5hUsedPercent = nil
	account.Quota5hResetSeconds = nil
	account.Quota5hWindowMinutes = nil
	account.Quota7dUsedPercent = nil
	account.Quota7dResetSeconds = nil
	account.Quota7dWindowMinutes = nil
	account.QuotaTotal = nil
	account.QuotaUsed = nil
	account.QuotaResetAt = nil
	if code := extractHTTPStatusCode(message); code != nil {
		account.QuotaHTTPStatus = code
	}
	now := time.Now()
	account.QuotaUpdatedAt = &now
	return h.storage.Save(account)
}

func (h *OpenAIHandler) markOAuthAccountReauthRequired(account *models.OpenAIAccount) {
	if account == nil {
		return
	}
	account.Status = "reauth_required"
	reauthErr := "refresh_token 已轮换失效 (401)，需要重新 OAuth 登录或重新导入最新账号"
	account.QuotaError = &reauthErr
	http401 := 401
	account.QuotaHTTPStatus = &http401
	account.QuotaVerified = false
	account.Quota5hUsedPercent = nil
	account.Quota5hResetSeconds = nil
	account.Quota5hWindowMinutes = nil
	account.Quota7dUsedPercent = nil
	account.Quota7dResetSeconds = nil
	account.Quota7dWindowMinutes = nil
	account.QuotaTotal = nil
	account.QuotaUsed = nil
	account.QuotaResetAt = nil
	account.QuotaIsForbidden = false
	now := time.Now()
	account.UpdatedAt = now
	account.QuotaUpdatedAt = &now
	_ = h.storage.Save(account)
}

func sPtr(s string) *string { return &s }

func normalizedOpenAIPlanPtr(value *string) *string {
	if value == nil {
		return nil
	}
	if normalized := openaiplatform.NormalizePlanType(*value); normalized != "" {
		return sPtr(normalized)
	}
	return nil
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func maskOpenAIToken(s string) string {
	if len(s) < 12 {
		return "***"
	}
	return s[:6] + "..." + s[len(s)-4:]
}

func isQuotaUnauthorized(err error) bool {
	return err != nil && strings.Contains(err.Error(), "HTTP 401")
}

func extractHTTPStatusCode(message string) *int {
	m := regexp.MustCompile(`HTTP\s+(\d{3})`).FindStringSubmatch(message)
	if len(m) < 2 {
		return nil
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return nil
	}
	return &n
}

func isRefreshTokenReusedError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "refresh_token_reused")
}

func isReauthRequiredError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "需要重新 OAuth 登录或重新导入最新账号")
}

func mergeOpenAIAccountUpdate(existing *models.OpenAIAccount, incoming models.OpenAIAccount) models.OpenAIAccount {
	incoming.LastUsedAt = existing.LastUsedAt
	incoming.IsCodexActive = existing.IsCodexActive
	incoming.ProxyEnabled = existing.ProxyEnabled
	incoming.QuotaUsed = existing.QuotaUsed
	incoming.QuotaTotal = existing.QuotaTotal
	incoming.QuotaResetAt = existing.QuotaResetAt
	incoming.QuotaUpdatedAt = existing.QuotaUpdatedAt
	incoming.Quota5hUsedPercent = existing.Quota5hUsedPercent
	incoming.Quota5hResetSeconds = existing.Quota5hResetSeconds
	incoming.Quota5hWindowMinutes = existing.Quota5hWindowMinutes
	incoming.Quota7dUsedPercent = existing.Quota7dUsedPercent
	incoming.Quota7dResetSeconds = existing.Quota7dResetSeconds
	incoming.Quota7dWindowMinutes = existing.Quota7dWindowMinutes
	incoming.QuotaIsForbidden = existing.QuotaIsForbidden
	incoming.QuotaHTTPStatus = existing.QuotaHTTPStatus
	incoming.QuotaError = existing.QuotaError
	incoming.QuotaVerified = existing.QuotaVerified
	return incoming
}
