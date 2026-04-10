package handlers

import (
	"easyllm/internal/models"
	openaiplatform "easyllm/internal/platforms/openai"
	"easyllm/internal/proxy"
	"easyllm/internal/storage"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OpenAIHandler struct {
	storage       *storage.OpenAIStorage
	codexStorage  *storage.CodexStorage
	mu            sync.Mutex
	oauthSessions map[string]*openaiOAuthSession
}

type openaiOAuthSession struct {
	State        string
	CodeVerifier string
	RedirectURI  string
	CreatedAt    time.Time
}

func NewOpenAIHandler(s *storage.OpenAIStorage, cs *storage.CodexStorage) *OpenAIHandler {
	h := &OpenAIHandler{
		storage:       s,
		codexStorage:  cs,
		oauthSessions: make(map[string]*openaiOAuthSession),
	}
	go h.cleanExpiredOAuthSessions()
	return h
}

// cleanExpiredOAuthSessions periodically removes OAuth sessions older than 10 minutes.
func (h *OpenAIHandler) cleanExpiredOAuthSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		h.mu.Lock()
		for id, sess := range h.oauthSessions {
			if time.Since(sess.CreatedAt) > 10*time.Minute {
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
	g.POST("/import/scan-dir", h.ImportByScanDir)             // scan local directory path
	g.POST("/import/refresh-tokens", h.ImportByRefreshTokens) // legacy: refresh_token list
	g.POST("/import/from-export", h.ImportFromExport)         // re-import from exported backup JSON (no API calls)
	g.POST("/import/sub2api", h.ImportFromSub2API)            // import from sub2api format
	g.GET("/export", h.ExportAccounts)

	// OAuth flow
	g.POST("/oauth/generate-url", h.GenerateOAuthURL)
	g.POST("/oauth/exchange-code", h.ExchangeCode)

	// API account management
	g.POST("/api-accounts", h.AddAPIAccount)
	g.PUT("/api-accounts/:id", h.UpdateAPIAccount)
	g.POST("/api-accounts/:id/switch", h.SwitchAPIAccount)

	// Codex proxy pool
	g.GET("/codex/accounts", h.ListCodexAccounts)
	g.POST("/codex/accounts", h.AddCodexAccount)
	g.PUT("/codex/accounts/:id", h.UpdateCodexAccount)
	g.DELETE("/codex/accounts/:id", h.DeleteCodexAccount)
	g.POST("/codex/accounts/:id/toggle", h.ToggleCodexAccount)
	g.GET("/codex/pool", h.GetCodexPoolStatus)
	g.POST("/codex/pool/refresh", h.RefreshCodexPool)
	g.GET("/codex/logs", h.GetCodexLogs)
	g.DELETE("/codex/logs", h.ClearCodexLogs)

	// Quota check
	g.POST("/accounts/fetch-quotas", h.FetchQuotas)

	// Service config (proxy pool switch, API key, stats)
	g.GET("/service-config", h.GetServiceConfig)
	g.PUT("/service-config", h.UpdateServiceConfig)

}

func (h *OpenAIHandler) ListAccounts(c *gin.Context) {
	accounts, err := h.storage.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, accounts)
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
	account.CreatedAt = time.Now()
	account.UpdatedAt = time.Now()
	if err := h.storage.Save(&account); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, account)
}

func (h *OpenAIHandler) UpdateAccount(c *gin.Context) {
	id := c.Param("id")
	var account models.OpenAIAccount
	if err := c.ShouldBindJSON(&account); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	account.ID = id
	account.UpdatedAt = time.Now()
	if err := h.storage.Save(&account); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, account)
}

func (h *OpenAIHandler) DeleteAccount(c *gin.Context) {
	id := c.Param("id")
	fmt.Printf("[OpenAI] Attempting to delete account ID: %s\n", id)

	db := storage.GetDB()
	res := db.Unscoped().Where("id = ?", id).Delete(&models.OpenAIAccount{})
	if res.Error != nil {
		fmt.Printf("[OpenAI] Delete error: %v\n", res.Error)
		c.JSON(http.StatusInternalServerError, models.APIError{Error: res.Error.Error(), Code: "STORAGE_ERROR"})
		return
	}

	fmt.Printf("[OpenAI] Delete success. Rows affected: %d\n", res.RowsAffected)
	c.JSON(http.StatusOK, gin.H{"success": true, "rows_affected": res.RowsAffected})
}

func (h *OpenAIHandler) DeleteAccounts(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}

	db := storage.GetDB()
	res := db.Unscoped().Where("id IN ?", req.IDs).Delete(&models.OpenAIAccount{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: res.Error.Error(), Code: "STORAGE_ERROR"})
		return
	}

	fmt.Printf("[OpenAI] Bulk delete success. Rows affected: %d\n", res.RowsAffected)
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

	if err := openaiplatform.SwitchCodexOAuthAccount(accessToken, refreshToken, idToken, account.ChatGPTAccountID); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "SWITCH_ERROR"})
		return
	}

	now := time.Now()
	account.LastUsedAt = &now
	h.storage.Save(account)
	h.storage.SetCodexActive(account.ID) // mark this account as currently active in ~/.codex/auth.json

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

	if err := openaiplatform.SwitchCodexAPIAccount(provider, model, baseURL, apiKey, account.WireAPI, account.ModelReasoningEffort); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "SWITCH_ERROR"})
		return
	}

	now := time.Now()
	account.LastUsedAt = &now
	h.storage.Save(account)
	h.storage.SetCodexActive(account.ID) // mark as active

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Switched to " + account.Email})
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
		fmt.Printf("[OpenAI] Refresh token failed for account id=%s email=%s: %v\n", account.ID, account.Email, err)
		if isReauthRequiredError(err) {
			c.JSON(http.StatusConflict, models.APIError{Error: err.Error(), Code: "REAUTH_REQUIRED"})
			return
		}
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "REFRESH_ERROR"})
		return
	}
	c.JSON(http.StatusOK, account)
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
				fmt.Printf("[OpenAI] Refresh all failed for account id=%s email=%s: %v\n", acc.ID, acc.Email, err)
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
		}
		if j := openaiplatform.ExtractOpenAIAuthJSON(data.IDToken); j != "" {
			account.OpenAIAuthJSON = sPtr(j)
		}
	}
	if data.AccountID != "" && account.ChatGPTAccountID == nil {
		account.ChatGPTAccountID = sPtr(data.AccountID)
	}

	return h.upsertImportedOAuthAccount(account, existingAccounts)
}

// ImportByTokenFiles handles uploading multiple token JSON files at once (multipart form)
func (h *OpenAIHandler) ImportByTokenFiles(c *gin.Context) {
	// Parse multipart explicitly with a large limit.
	// Even though Gin has Engine.MaxMultipartMemory, this keeps behavior consistent
	// across different router setups and avoids the default 32MiB constraint.
	if err := c.Request.ParseMultipartForm(8 << 30); err != nil { // 8GiB
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

	type result struct {
		Filename string `json:"filename"`
		Success  bool   `json:"success"`
		Email    string `json:"email,omitempty"`
		Skipped  bool   `json:"skipped,omitempty"`
		Error    string `json:"error,omitempty"`
	}

	results := make([]result, len(files))
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
				results[idx] = result{Filename: fileHeader.Filename, Success: false, Error: "open error: " + err.Error()}
				return
			}
			defer f.Close()

			var data tokenFileData
			if err := json.NewDecoder(f).Decode(&data); err != nil {
				results[idx] = result{Filename: fileHeader.Filename, Success: false, Error: "parse error: " + err.Error()}
				return
			}

			existingMu.Lock()
			account, skipped, err := h.importSingleTokenFile(&data, &existingAccounts)
			existingMu.Unlock()

			if err != nil {
				results[idx] = result{Filename: fileHeader.Filename, Success: false, Skipped: skipped, Error: err.Error(), Email: data.Email}
			} else {
				results[idx] = result{Filename: fileHeader.Filename, Success: true, Email: account.Email}
			}
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

	c.JSON(http.StatusOK, gin.H{
		"total":   len(files),
		"success": successCount,
		"skipped": skippedCount,
		"failed":  len(files) - successCount - skippedCount,
		"results": results,
	})
}

// ImportByScanDir scans a server-side directory for token_*.json files and imports them all
func (h *OpenAIHandler) ImportByScanDir(c *gin.Context) {
	var req struct {
		Dir string `json:"dir"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Dir == "" {
		req.Dir = "./auth"
	}

	// Expand ~ to home dir
	if strings.HasPrefix(req.Dir, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			req.Dir = filepath.Join(home, req.Dir[2:])
		}
	}

	// Security: resolve to absolute path and restrict to safe directories
	absDir, err := filepath.Abs(req.Dir)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "invalid directory path", Code: "INVALID_PATH"})
		return
	}
	cwd, _ := os.Getwd()
	safeBase := filepath.Join(cwd, "auth")
	if absDir != safeBase && !strings.HasPrefix(absDir, safeBase+string(filepath.Separator)) {
		homeDir, _ := os.UserHomeDir()
		if homeDir == "" || (absDir != homeDir && !strings.HasPrefix(absDir, homeDir+string(filepath.Separator))) {
			c.JSON(http.StatusForbidden, models.APIError{Error: "directory not allowed; only ./auth or home subdirectories are permitted", Code: "PATH_FORBIDDEN"})
			return
		}
	}

	entries, err := os.ReadDir(absDir)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "cannot read directory: " + err.Error(), Code: "DIR_ERROR"})
		return
	}

	// Filter JSON files
	var jsonFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			jsonFiles = append(jsonFiles, filepath.Join(absDir, e.Name()))
		}
	}

	if len(jsonFiles) == 0 {
		c.JSON(http.StatusOK, gin.H{"total": 0, "success": 0, "skipped": 0, "failed": 0, "results": []interface{}{}})
		return
	}

	existingMu := sync.Mutex{}
	existingAccounts, _ := h.storage.List()

	type result struct {
		Filename string `json:"filename"`
		Success  bool   `json:"success"`
		Email    string `json:"email,omitempty"`
		Skipped  bool   `json:"skipped,omitempty"`
		Error    string `json:"error,omitempty"`
	}

	results := make([]result, len(jsonFiles))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 20) // High concurrency - pure file I/O, no network

	for i, fpath := range jsonFiles {
		wg.Add(1)
		go func(idx int, filePath string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			fname := filepath.Base(filePath)
			raw, err := os.ReadFile(filePath)
			if err != nil {
				results[idx] = result{Filename: fname, Success: false, Error: err.Error()}
				return
			}

			var data tokenFileData
			if err := json.Unmarshal(raw, &data); err != nil {
				results[idx] = result{Filename: fname, Success: false, Error: "parse error: " + err.Error()}
				return
			}

			existingMu.Lock()
			account, skipped, err := h.importSingleTokenFile(&data, &existingAccounts)
			existingMu.Unlock()

			if err != nil {
				results[idx] = result{Filename: fname, Success: false, Skipped: skipped, Error: err.Error(), Email: data.Email}
			} else {
				results[idx] = result{Filename: fname, Success: true, Email: account.Email}
			}
		}(i, fpath)
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

	c.JSON(http.StatusOK, gin.H{
		"total":   len(jsonFiles),
		"success": successCount,
		"skipped": skippedCount,
		"failed":  len(jsonFiles) - successCount - skippedCount,
		"results": results,
	})
}

func findMatchingOAuthAccountIndex(existingAccounts []models.OpenAIAccount, email string, chatgptAccountID *string) int {
	targetID := strings.TrimSpace(derefStr(chatgptAccountID))
	if targetID != "" {
		for i := range existingAccounts {
			existing := existingAccounts[i]
			if existing.AccountType != models.OpenAIAccountTypeOAuth {
				continue
			}
			if strings.TrimSpace(derefStr(existing.ChatGPTAccountID)) == targetID {
				return i
			}
		}
	}

	targetEmail := strings.TrimSpace(email)
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
		existingID := strings.TrimSpace(derefStr(existing.ChatGPTAccountID))
		if targetID == "" || existingID == "" || existingID == targetID {
			return i
		}
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

	if idx := findMatchingOAuthAccountIndex(*existingAccounts, incoming.Email, incoming.ChatGPTAccountID); idx >= 0 {
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
	incoming.Status = incomingStatus
	incoming.UpdatedAt = now
	if err := h.storage.Save(incoming); err != nil {
		return nil, false, err
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
			var openaiAuthJSON string

			if tokenResp.IDToken != "" {
				if userInfo := openaiplatform.ParseIDToken(tokenResp.IDToken); userInfo != nil {
					if userInfo.Email != nil {
						email = strings.TrimSpace(*userInfo.Email)
					}
					chatgptAccountID = userInfo.ChatGPTAccountID
					chatgptUserID = userInfo.ChatGPTUserID
					orgID = userInfo.OrganizationID
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

	c.JSON(http.StatusOK, gin.H{
		"total":      len(req.RefreshTokens),
		"successful": successCount,
		"failed":     len(req.RefreshTokens) - successCount,
		"results":    results,
	})
}

// GenerateOAuthURL generates an OpenAI OAuth authorization URL
func (h *OpenAIHandler) GenerateOAuthURL(c *gin.Context) {
	var req struct {
		RedirectURI *string `json:"redirect_uri"`
	}
	c.ShouldBindJSON(&req)

	redirectURI := "http://localhost:1455/auth/callback"
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

	c.JSON(http.StatusOK, gin.H{
		"auth_url":   authURL,
		"session_id": sessionID,
	})
}

// ExchangeCode completes OAuth login with the authorization code
func (h *OpenAIHandler) ExchangeCode(c *gin.Context) {
	var req struct {
		SessionID   string  `json:"session_id"`
		Code        string  `json:"code"`
		RedirectURI *string `json:"redirect_uri"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}

	h.mu.Lock()
	session, ok := h.oauthSessions[req.SessionID]
	if ok {
		delete(h.oauthSessions, req.SessionID)
	}
	h.mu.Unlock()

	if !ok {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "Session not found or expired", Code: "SESSION_NOT_FOUND"})
		return
	}

	redirectURI := session.RedirectURI
	if req.RedirectURI != nil && *req.RedirectURI != "" {
		redirectURI = *req.RedirectURI
	}

	tokenResp, err := openaiplatform.ExchangeCode(req.Code, session.CodeVerifier, redirectURI)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "EXCHANGE_ERROR"})
		return
	}

	var email string
	var chatgptAccountID, chatgptUserID, orgID *string
	var openaiAuthJSON string

	if tokenResp.IDToken != "" {
		if userInfo := openaiplatform.ParseIDToken(tokenResp.IDToken); userInfo != nil {
			if userInfo.Email != nil {
				email = strings.TrimSpace(*userInfo.Email)
			}
			chatgptAccountID = userInfo.ChatGPTAccountID
			chatgptUserID = userInfo.ChatGPTUserID
			orgID = userInfo.OrganizationID
		}
		openaiAuthJSON = openaiplatform.ExtractOpenAIAuthJSON(tokenResp.IDToken)
	}

	if email == "" {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: "Failed to get email from token", Code: "NO_EMAIL"})
		return
	}

	existingAccounts, _ := h.storage.List()
	for _, existing := range existingAccounts {
		if strings.EqualFold(existing.Email, email) &&
			(chatgptAccountID == nil || existing.ChatGPTAccountID == nil ||
				*chatgptAccountID == *existing.ChatGPTAccountID) {
			c.JSON(http.StatusConflict, models.APIError{Error: "该账号已存在", Code: "DUPLICATE"})
			return
		}
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
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if openaiAuthJSON != "" {
		account.OpenAIAuthJSON = sPtr(openaiAuthJSON)
	}

	if err := h.storage.Save(account); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, account)
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
	c.JSON(http.StatusOK, account)
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

	existing.Email = req.ModelProvider
	if existing.Email == "" {
		existing.Email = "API Account"
	}
	existing.ModelProvider = sPtr(req.ModelProvider)
	existing.Model = sPtr(req.Model)
	existing.ModelReasoningEffort = req.ModelReasoningEffort
	existing.WireAPI = req.WireAPI
	existing.BaseURL = sPtr(req.BaseURL)
	if req.APIKey != "" {
		existing.APIKey = sPtr(req.APIKey)
	}
	existing.UpdatedAt = time.Now()

	if err := h.storage.Save(existing); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, existing)
}

// ---- Codex Pool handlers ----

func (h *OpenAIHandler) ListCodexAccounts(c *gin.Context) {
	accounts, err := h.codexStorage.LoadAllAccounts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, accounts)
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
	account.CreatedAt = time.Now()
	account.UpdatedAt = time.Now()
	account.Enabled = true
	if err := h.codexStorage.SaveAccount(&account); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, account)
}

func (h *OpenAIHandler) UpdateCodexAccount(c *gin.Context) {
	id := c.Param("id")
	var account models.CodexAccount
	if err := c.ShouldBindJSON(&account); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}
	account.ID = id
	h.codexStorage.SaveAccount(&account)
	c.JSON(http.StatusOK, account)
}

func (h *OpenAIHandler) DeleteCodexAccount(c *gin.Context) {
	if err := h.codexStorage.DeleteAccount(c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *OpenAIHandler) ToggleCodexAccount(c *gin.Context) {
	id := c.Param("id")
	accounts, _ := h.codexStorage.LoadAllAccounts()
	for _, a := range accounts {
		if a.ID == id {
			a.Enabled = !a.Enabled
			h.codexStorage.SaveAccount(a)
			c.JSON(http.StatusOK, gin.H{"enabled": a.Enabled})
			return
		}
	}
	c.JSON(http.StatusNotFound, models.APIError{Error: "Account not found", Code: "NOT_FOUND"})
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

func (h *OpenAIHandler) GetCodexLogs(c *gin.Context) {
	page := 1
	perPage := 50
	if v := c.Query("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	if v := c.Query("per_page"); v != "" {
		if pp, err := strconv.Atoi(v); err == nil && pp > 0 && pp <= 500 {
			perPage = pp
		}
	}
	offset := (page - 1) * perPage
	logs, total, err := h.codexStorage.GetLogs(perPage, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	totalPages := int(total) / perPage
	if int(total)%perPage > 0 {
		totalPages++
	}
	c.JSON(http.StatusOK, gin.H{
		"logs":        logs,
		"total":       total,
		"page":        page,
		"per_page":    perPage,
		"total_pages": totalPages,
	})
}

func (h *OpenAIHandler) ClearCodexLogs(c *gin.Context) {
	if err := h.codexStorage.ClearLogs(); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIError{Error: err.Error(), Code: "STORAGE_ERROR"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
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

// exportedOAuthAccount 是导出文件中 oauth_accounts 数组里每条记录的格式
type exportedOAuthAccount struct {
	Email            string `json:"email"`
	RefreshToken     string `json:"refresh_token"`
	AccessToken      string `json:"access_token"`
	IDToken          string `json:"id_token"`
	ChatGPTAccountID string `json:"chatgpt_account_id"`
	ExpiresAt        string `json:"expires_at"`
	Status           string `json:"status,omitempty"`
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
		ExportedAt    string                 `json:"exported_at"`
		Usage         string                 `json:"_usage"`
		OAuthAccounts []exportedOAuthAccount `json:"oauth_accounts"`
		APIAccounts   []exportedAPIAccount   `json:"api_accounts"`
	}{
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Usage:      "恢复时：在\"批量导入 → 从备份导入\"中上传此文件即可一键恢复所有账号，无需任何 API 调用。请妥善保管此文件。",
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

		expiresAt := ""
		if a.ExpiresAt != nil {
			expiresAt = a.ExpiresAt.UTC().Format(time.RFC3339)
		}

		resp.OAuthAccounts = append(resp.OAuthAccounts, exportedOAuthAccount{
			Email:            a.Email,
			RefreshToken:     derefStr(a.RefreshToken),
			AccessToken:      derefStr(a.AccessToken),
			IDToken:          derefStr(a.IDToken),
			ChatGPTAccountID: derefStr(a.ChatGPTAccountID),
			ExpiresAt:        expiresAt,
			Status:           a.Status,
		})
	}

	c.JSON(http.StatusOK, resp)
}

// ImportFromExport 直接从导出的备份 JSON 中重新导入所有账号，无需任何 OpenAI API 调用。
// 支持导出文件中的 oauth_accounts 和 api_accounts 两类账号。
func (h *OpenAIHandler) ImportFromExport(c *gin.Context) {
	var payload struct {
		OAuthAccounts []exportedOAuthAccount `json:"oauth_accounts"`
		APIAccounts   []exportedAPIAccount   `json:"api_accounts"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "无效的请求体: " + err.Error(), Code: "INVALID_REQUEST"})
		return
	}

	if len(payload.OAuthAccounts) == 0 && len(payload.APIAccounts) == 0 {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "备份文件中没有账号数据", Code: "EMPTY_INPUT"})
		return
	}

	existingAccounts, _ := h.storage.List()

	type result struct {
		Email   string `json:"email"`
		Success bool   `json:"success"`
		Skipped bool   `json:"skipped,omitempty"`
		Error   string `json:"error,omitempty"`
	}
	var results []result

	now := time.Now()

	// 导入 OAuth 账号
	for _, a := range payload.OAuthAccounts {
		if a.Email == "" && a.IDToken != "" {
			if userInfo := openaiplatform.ParseIDToken(a.IDToken); userInfo != nil && userInfo.Email != nil {
				a.Email = strings.TrimSpace(*userInfo.Email)
			}
		}
		if a.Email == "" {
			results = append(results, result{Email: "(unknown)", Success: false, Error: "缺少 email 字段"})
			continue
		}

		var expiresAt *time.Time
		if a.ExpiresAt != "" {
			if t, err := time.Parse(time.RFC3339, a.ExpiresAt); err == nil {
				expiresAt = &t
			}
		}

		account := &models.OpenAIAccount{
			Email:        a.Email,
			AccountType:  models.OpenAIAccountTypeOAuth,
			Status:       a.Status,
			AccessToken:  sPtr(a.AccessToken),
			RefreshToken: sPtr(a.RefreshToken),
			ExpiresAt:    expiresAt,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if a.IDToken != "" {
			account.IDToken = sPtr(a.IDToken)
			if userInfo := openaiplatform.ParseIDToken(a.IDToken); userInfo != nil {
				account.ChatGPTAccountID = userInfo.ChatGPTAccountID
				account.ChatGPTUserID = userInfo.ChatGPTUserID
				account.OrganizationID = userInfo.OrganizationID
			}
			if j := openaiplatform.ExtractOpenAIAuthJSON(a.IDToken); j != "" {
				account.OpenAIAuthJSON = sPtr(j)
			}
		}
		if a.ChatGPTAccountID != "" && account.ChatGPTAccountID == nil {
			account.ChatGPTAccountID = sPtr(a.ChatGPTAccountID)
		}

		if _, _, err := h.upsertImportedOAuthAccount(account, &existingAccounts); err != nil {
			results = append(results, result{Email: a.Email, Success: false, Error: err.Error()})
			continue
		}
		results = append(results, result{Email: a.Email, Success: true})
	}

	// 导入 API 账号
	for _, a := range payload.APIAccounts {
		label := a.ModelProvider
		if label == "" {
			label = a.BaseURL
		}
		if a.APIKey == "" {
			results = append(results, result{Email: label, Success: false, Error: "api_key 为空，跳过"})
			continue
		}
		wireAPI := a.WireAPI
		if wireAPI == "" {
			wireAPI = "responses"
		}
		account := &models.OpenAIAccount{
			Email:                label,
			AccountType:          models.OpenAIAccountTypeAPI,
			ModelProvider:        sPtr(a.ModelProvider),
			Model:                sPtr(a.Model),
			BaseURL:              sPtr(a.BaseURL),
			APIKey:               sPtr(a.APIKey),
			WireAPI:              sPtr(wireAPI),
			ModelReasoningEffort: sPtr(a.ModelReasoningEffort),
			ProxyEnabled:         a.ProxyEnabled,
			CreatedAt:            now,
			UpdatedAt:            now,
		}
		if _, _, err := h.upsertImportedAPIAccount(account, &existingAccounts); err != nil {
			results = append(results, result{Email: label, Success: false, Error: err.Error()})
			continue
		}
		results = append(results, result{Email: label, Success: true})
	}

	success, skipped, failed := 0, 0, 0
	for _, r := range results {
		if r.Success {
			success++
		} else if r.Skipped {
			skipped++
		} else {
			failed++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total":   len(results),
		"success": success,
		"skipped": skipped,
		"failed":  failed,
		"results": results,
	})
}

// sub2api format structs
type sub2apiAccount struct {
	Name        string `json:"name"`
	Platform    string `json:"platform"`
	AccountType string `json:"type"` // "oauth"
	Credentials struct {
		AccessToken      string            `json:"access_token"`
		ChatgptAccountID string            `json:"chatgpt_account_id"`
		ChatgptUserID    string            `json:"chatgpt_user_id"`
		ClientID         string            `json:"client_id"`
		ExpiresAt        int64             `json:"expires_at"`
		ExpiresIn        int               `json:"expires_in"`
		ModelMapping     map[string]string `json:"model_mapping"`
		OrganizationID   string            `json:"organization_id"`
		RefreshToken     string            `json:"refresh_token"`
	} `json:"credentials"`
	Extra struct {
		Email string `json:"email"`
	} `json:"extra"`
	Concurrency        int     `json:"concurrency"`
	Priority           int     `json:"priority"`
	RateMultiplier     float64 `json:"rate_multiplier"`
	AutoPauseOnExpired bool    `json:"auto_pause_on_expired"`
}

type sub2apiExport struct {
	ExportedAt string           `json:"exported_at"`
	Proxies    []interface{}    `json:"proxies"`
	Accounts   []sub2apiAccount `json:"accounts"`
}

// ImportFromSub2API handles importing accounts from the sub2api JSON format.
func (h *OpenAIHandler) ImportFromSub2API(c *gin.Context) {
	var payload sub2apiExport
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "无效的请求体: " + err.Error(), Code: "INVALID_REQUEST"})
		return
	}

	if len(payload.Accounts) == 0 {
		c.JSON(http.StatusBadRequest, models.APIError{Error: "文件中没有账号数据", Code: "EMPTY_INPUT"})
		return
	}

	existingAccounts, _ := h.storage.List()
	type result struct {
		Email   string `json:"email"`
		Success bool   `json:"success"`
		Skipped bool   `json:"skipped,omitempty"`
		Error   string `json:"error,omitempty"`
	}
	var results []result
	now := time.Now()

	for _, a := range payload.Accounts {
		email := a.Extra.Email
		if email == "" {
			email = a.Name
		}
		if email == "" {
			results = append(results, result{Email: "(unknown)", Success: false, Error: "缺少 email 或 name 字段"})
			continue
		}

		var expiresAt *time.Time
		if a.Credentials.ExpiresAt > 0 {
			t := time.Unix(a.Credentials.ExpiresAt, 0)
			expiresAt = &t
		}

		account := &models.OpenAIAccount{
			Email:            email,
			AccountType:      models.OpenAIAccountTypeOAuth,
			AccessToken:      sPtr(a.Credentials.AccessToken),
			RefreshToken:     sPtr(a.Credentials.RefreshToken),
			ExpiresAt:        expiresAt,
			ChatGPTAccountID: sPtr(a.Credentials.ChatgptAccountID),
			ChatGPTUserID:    sPtr(a.Credentials.ChatgptUserID),
			OrganizationID:   sPtr(a.Credentials.OrganizationID),
			CreatedAt:        now,
			UpdatedAt:        now,
		}

		if _, _, err := h.upsertImportedOAuthAccount(account, &existingAccounts); err != nil {
			results = append(results, result{Email: email, Success: false, Error: err.Error()})
			continue
		}
		results = append(results, result{Email: email, Success: true})
	}

	success, skipped, failed := 0, 0, 0
	for _, r := range results {
		if r.Success {
			success++
		} else if r.Skipped {
			skipped++
		} else {
			failed++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total":   len(results),
		"success": success,
		"skipped": skipped,
		"failed":  failed,
		"results": results,
	})
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
					fmt.Printf("[OpenAI] Auto refresh before quota failed for account id=%s email=%s: %v\n", account.ID, account.Email, err)
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
					fmt.Printf("[OpenAI] Auto refresh on quota 401 failed for account id=%s email=%s: %v\n", account.ID, account.Email, refreshErr)
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

// GetServiceConfig returns proxy pool status, API key (masked), strategy, and request stats.
func (h *OpenAIHandler) GetServiceConfig(c *gin.Context) {
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

	// Log stats from DB
	var logTotal int64
	if h.codexStorage != nil {
		_, logTotal, _ = h.codexStorage.GetLogs(0, 0)
	}

	proxyCount, _ := h.storage.CountProxyEnabled()

	c.JSON(http.StatusOK, gin.H{
		"proxy_pool_enabled":  enabled,
		"strategy":            strategy,
		"pool_size":           poolSize,
		"proxy_enabled_count": proxyCount,
		"total_requests":      totalReqs,
		"total_logs":          logTotal,
		"api_key_set":         apiKey != "",
		"api_key_masked":      maskedKey,
		"api_key":             apiKey,
	})
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
		valid := map[string]bool{"round_robin": true, "random": true, "least_used": true}
		if valid[*req.Strategy] {
			p.SetStrategy(*req.Strategy)
			storage.SaveSetting("proxy_strategy", *req.Strategy)
		}
	}
	if req.APIKey != nil {
		storage.SaveSetting("proxy_api_key", *req.APIKey)
	}

	h.GetServiceConfig(c)
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
		fmt.Printf("[OpenAI] Upstream refresh request failed for account id=%s email=%s: %v\n", account.ID, account.Email, err)
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
		}
		if j := openaiplatform.ExtractOpenAIAuthJSON(tokenResp.IDToken); j != "" {
			account.OpenAIAuthJSON = sPtr(j)
		}
	}
	if tokenResp.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		account.ExpiresAt = &t
	}
	account.UpdatedAt = time.Now()
	if err := h.storage.Save(account); err != nil {
		fmt.Printf("[OpenAI] Persist refreshed token failed for account id=%s email=%s: %v\n", account.ID, account.Email, err)
		return fmt.Errorf("persist refreshed token failed: %w", err)
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
	if err := h.storage.Save(account); err != nil {
		fmt.Printf("[OpenAI] Persist reauth-required status failed for account id=%s email=%s: %v\n", account.ID, account.Email, err)
	}
}

func sPtr(s string) *string { return &s }

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
