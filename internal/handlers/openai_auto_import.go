package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"
	"time"

	"easyllm/internal/models"
	openaiplatform "easyllm/internal/openai"
	"easyllm/internal/proxy"
	"easyllm/internal/storage"
	"strconv"

	"github.com/gin-gonic/gin"
)

// 自适应导入格式标识（与前端展示一致）
const (
	scanFormatEasyLLMExport = "easyllm-export"
	scanFormatCPA           = "cpa"
	scanFormatToken         = "token"
)

func scanFormatLabel(format string) string {
	switch format {
	case scanFormatEasyLLMExport:
		return "EasyLLM 备份"
	case scanFormatCPA:
		return "CPA"
	case scanFormatToken:
		return "Token"
	default:
		return format
	}
}

func detectAutoImportFormat(raw []byte, filename string) string {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return ""
	}
	lowerName := strings.ToLower(filename)

	if looksLikeEasyLLMExportJSON(trimmed) {
		return scanFormatEasyLLMExport
	}

	// 文件名或 type 字段优先识别 CPA。
	if strings.Contains(lowerName, "-cpa.") || strings.HasSuffix(lowerName, ".codex.cpa.json") {
		if _, err := parseCPAFileEntries(trimmed); err == nil {
			return scanFormatCPA
		}
	}
	if trimmed[0] == '{' {
		var root map[string]json.RawMessage
		if err := json.Unmarshal(trimmed, &root); err == nil {
			if typeRaw, ok := root["type"]; ok {
				var typ string
				_ = json.Unmarshal(typeRaw, &typ)
				if strings.EqualFold(strings.TrimSpace(typ), "codex") {
					if _, err := parseCPAFileEntries(trimmed); err == nil {
						return scanFormatCPA
					}
				}
			}
			if _, ok := root["expired"]; ok {
				if _, err := parseCPAFileEntries(trimmed); err == nil {
					return scanFormatCPA
				}
			}
		}
	}
	if _, err := parseCPAFileEntries(trimmed); err == nil {
		if looksLikeCPAEntries(trimmed) {
			return scanFormatCPA
		}
	}

	if _, err := parseTokenFileEntries(trimmed); err == nil {
		return scanFormatToken
	}
	return ""
}

func looksLikeEasyLLMExportJSON(raw []byte) bool {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return false
	}
	for _, key := range []string{"oauth_accounts", "api_accounts", "local_access"} {
		if payload, ok := root[key]; ok && len(bytes.TrimSpace(payload)) > 2 && string(bytes.TrimSpace(payload)) != "null" {
			return true
		}
	}
	return false
}

func looksLikeCPAEntries(raw []byte) bool {
	entries, err := parseCPAFileEntries(raw)
	if err != nil || len(entries) == 0 {
		return false
	}
	for _, e := range entries {
		if strings.EqualFold(strings.TrimSpace(e.Type), "codex") {
			return true
		}
		if strings.TrimSpace(e.Expired) != "" && strings.TrimSpace(e.PlanType) != "" {
			return true
		}
	}
	return false
}

func ginResultsToTokenImportResults(filename, format string, payload gin.H) []tokenImportResult {
	b, err := json.Marshal(payload["results"])
	if err != nil {
		return []tokenImportResult{{Filename: filename, Format: format, Success: false, Error: "无法解析导入结果"}}
	}
	var generic []struct {
		Email   string `json:"email"`
		Success bool   `json:"success"`
		Skipped bool   `json:"skipped"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(b, &generic); err != nil {
		return []tokenImportResult{{Filename: filename, Format: format, Success: false, Error: err.Error()}}
	}
	out := make([]tokenImportResult, 0, len(generic))
	for _, r := range generic {
		out = append(out, tokenImportResult{
			Filename: filename,
			Format:   format,
			Success:  r.Success,
			Email:    r.Email,
			Skipped:  r.Skipped,
			Error:    r.Error,
		})
	}
	return out
}

// importAutoJSONFile 自动识别格式并导入单个 JSON 文件。
func (h *OpenAIHandler) importAutoJSONFile(filename string, raw []byte, existingAccounts *[]models.OpenAIAccount) []tokenImportResult {
	format := detectAutoImportFormat(raw, filename)
	if format == "" {
		return []tokenImportResult{{
			Filename: filename,
			Success:  false,
			Error:    "无法识别 JSON 格式，支持 Token、CPA、EasyLLM 备份",
		}}
	}

	switch format {
	case scanFormatEasyLLMExport:
		out, err := h.importEasyLLMExportBytes(raw, existingAccounts)
		if err != nil {
			return []tokenImportResult{{Filename: filename, Format: format, Success: false, Error: err.Error()}}
		}
		return ginResultsToTokenImportResults(filename, format, out)

	case scanFormatCPA:
		entries, err := parseCPAFileEntries(raw)
		if err != nil {
			return []tokenImportResult{{Filename: filename, Format: format, Success: false, Error: "parse error: " + err.Error()}}
		}
		results := make([]tokenImportResult, 0, len(entries))
		for _, data := range entries {
			entry := data
			account, skipped, err := h.importSingleTokenFile(&entry, existingAccounts)
			if err != nil {
				email := entry.Email
				if email == "" {
					email = filename
				}
				results = append(results, tokenImportResult{
					Filename: filename, Format: format, Success: false, Skipped: skipped, Error: err.Error(), Email: email,
				})
				continue
			}
			results = append(results, tokenImportResult{
				Filename: filename, Format: format, Success: true, Email: account.Email,
			})
		}
		return results

	case scanFormatToken:
		entries, err := parseTokenFileEntries(raw)
		if err != nil {
			return []tokenImportResult{{Filename: filename, Format: format, Success: false, Error: "parse error: " + err.Error()}}
		}
		results := make([]tokenImportResult, 0, len(entries))
		for _, data := range entries {
			entry := data
			account, skipped, err := h.importSingleTokenFile(&entry, existingAccounts)
			if err != nil {
				results = append(results, tokenImportResult{
					Filename: filename,
					Format:   format,
					Success:  false,
					Skipped:  skipped,
					Error:    err.Error(),
					Email:    entry.Email,
				})
				continue
			}
			results = append(results, tokenImportResult{
				Filename: filename,
				Format:   format,
				Success:  true,
				Email:    account.Email,
			})
		}
		return results
	default:
		return []tokenImportResult{{Filename: filename, Success: false, Error: "未知格式: " + format}}
	}
}

type easyLLMExportPayload struct {
	OAuthAccounts []exportedOAuthAccount             `json:"oauth_accounts"`
	APIAccounts   []exportedAPIAccount               `json:"api_accounts"`
	LocalAccess   *models.CodexLocalAccessCollection `json:"local_access"`
}

// applyEasyLLMExportPayload 导入 EasyLLM 备份中的账号与本地服务配置。
func (h *OpenAIHandler) applyEasyLLMExportPayload(payload *easyLLMExportPayload, existingAccounts *[]models.OpenAIAccount) (gin.H, error) {
	type result struct {
		Email   string `json:"email"`
		Success bool   `json:"success"`
		Skipped bool   `json:"skipped,omitempty"`
		Error   string `json:"error,omitempty"`
	}
	var results []result
	oauthIDMap := make(map[string]string)
	now := time.Now()

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
		if t, ok := parseExportedOAuthTime(exportedOAuthExpiresSource(a)); ok {
			expiresAt = t
		}
		status := strings.TrimSpace(a.Status)
		if status == "" {
			if a.Disabled {
				status = "inactive"
			} else {
				status = "active"
			}
		}
		account := &models.OpenAIAccount{
			ID:           strings.TrimSpace(a.ID),
			Email:        a.Email,
			AccountType:  models.OpenAIAccountTypeOAuth,
			Status:       status,
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
				if plan := normalizedOpenAIPlanPtr(userInfo.PlanType); plan != nil {
					account.Plan = plan
				}
			}
			if j := openaiplatform.ExtractOpenAIAuthJSON(a.IDToken); j != "" {
				account.OpenAIAuthJSON = sPtr(j)
			}
		}
		if cid := exportedOAuthChatGPTAccountID(a); cid != "" && account.ChatGPTAccountID == nil {
			account.ChatGPTAccountID = sPtr(cid)
		}
		if plan := exportedOAuthPlan(a); plan != nil {
			account.Plan = plan
		}
		saved, _, err := h.upsertImportedOAuthAccount(account, existingAccounts)
		if err != nil {
			results = append(results, result{Email: a.Email, Success: false, Error: err.Error()})
			continue
		}
		if saved != nil {
			if oldID := strings.TrimSpace(a.ID); oldID != "" {
				oauthIDMap[oldID] = saved.ID
			}
			if cid := exportedOAuthChatGPTAccountID(a); cid != "" {
				oauthIDMap[cid] = saved.ID
			}
		}
		results = append(results, result{Email: a.Email, Success: true})
	}

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
		if _, _, err := h.upsertImportedAPIAccount(account, existingAccounts); err != nil {
			results = append(results, result{Email: label, Success: false, Error: err.Error()})
			continue
		}
		results = append(results, result{Email: label, Success: true})
	}

	if payload.LocalAccess != nil {
		accountIDs := make([]string, 0, len(payload.LocalAccess.AccountIDs))
		for _, id := range payload.LocalAccess.AccountIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			if mapped := oauthIDMap[id]; mapped != "" {
				accountIDs = append(accountIDs, mapped)
			} else {
				accountIDs = append(accountIDs, id)
			}
		}
		ids, err := h.filterCodexLocalAccessAccountIDs(accountIDs, payload.LocalAccess.RestrictFreeAccounts)
		if err == nil {
			_ = saveCodexLocalAccessAccountIDs(ids)
			_ = storage.SaveSetting(codexLocalAccessRestrictFreeAccountsKey, fmt.Sprintf("%v", payload.LocalAccess.RestrictFreeAccounts))
			if payload.LocalAccess.Port > 0 && payload.LocalAccess.Port <= 65535 {
				_ = storage.SaveSetting(codexLocalAccessPortKey, strconv.Itoa(payload.LocalAccess.Port))
			}
			if isValidCodexProxyStrategy(payload.LocalAccess.RoutingStrategy) {
				_ = storage.SaveSetting(codexLocalAccessRoutingStrategyKey, payload.LocalAccess.RoutingStrategy)
				_ = storage.SaveSetting("proxy_strategy", payload.LocalAccess.RoutingStrategy)
			}
			_ = storage.SaveSetting(codexLocalAccessEnabledKey, fmt.Sprintf("%v", payload.LocalAccess.Enabled))
			_ = saveCodexLocalAccessTimestamp(true)
			if h.storage != nil && len(ids) > 0 {
				_, _ = h.storage.SetProxyForIDs(ids, true)
			}
			if p := proxy.GetProxy(); p != nil {
				p.Refresh()
			}
		}
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
	if success > 0 {
		refreshCodexProxyPool()
	}
	return gin.H{
		"total":   len(results),
		"success": success,
		"skipped": skipped,
		"failed":  failed,
		"results": results,
	}, nil
}

// importEasyLLMExportBytes 从 EasyLLM 备份 JSON 导入（供扫描目录与 HTTP 共用）。
func (h *OpenAIHandler) importEasyLLMExportBytes(raw []byte, existingAccounts *[]models.OpenAIAccount) (gin.H, error) {
	var payload easyLLMExportPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("无效的 EasyLLM 备份: %w", err)
	}
	if len(payload.OAuthAccounts) == 0 && len(payload.APIAccounts) == 0 && payload.LocalAccess == nil {
		return nil, fmt.Errorf("备份文件中没有账号数据")
	}
	if existingAccounts == nil {
		list, _ := h.storage.List()
		existingAccounts = &list
	}
	return h.applyEasyLLMExportPayload(&payload, existingAccounts)
}

// ImportByAutoFiles 上传单个或多个 JSON 文件，自动识别格式并导入（与扫描目录逻辑一致）。
func (h *OpenAIHandler) ImportByAutoFiles(c *gin.Context) {
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

			fname := fileHeader.Filename
			if !strings.HasSuffix(strings.ToLower(fname), ".json") {
				resultsMu.Lock()
				results = append(results, tokenImportResult{
					Filename: fname,
					Success:  false,
					Error:    "仅支持 .json 文件",
				})
				resultsMu.Unlock()
				return
			}

			f, err := fileHeader.Open()
			if err != nil {
				resultsMu.Lock()
				results = append(results, tokenImportResult{Filename: fname, Success: false, Error: "open error: " + err.Error()})
				resultsMu.Unlock()
				return
			}
			defer f.Close()

			raw, err := io.ReadAll(f)
			if err != nil {
				resultsMu.Lock()
				results = append(results, tokenImportResult{Filename: fname, Success: false, Error: "read error: " + err.Error()})
				resultsMu.Unlock()
				return
			}

			existingMu.Lock()
			fileResults := h.importAutoJSONFile(fname, raw, &existingAccounts)
			existingMu.Unlock()

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
