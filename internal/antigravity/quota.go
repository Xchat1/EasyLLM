package antigravity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"easyllm/internal/models"
)

const (
	CloudCodeProdBaseURL  = "https://cloudcode-pa.googleapis.com"
	CloudCodeDailyBaseURL = "https://daily-cloudcode-pa.googleapis.com"

	LoadCodeAssistPath           = "v1internal:loadCodeAssist"
	OnboardUserPath              = "v1internal:onboardUser"
	FetchAvailableModelsPath     = "v1internal:fetchAvailableModels"
	RetrieveUserQuotaSummaryPath = "v1internal:retrieveUserQuotaSummary"

	DefaultIdeVersion           = "1.20.5"
	DefaultGoogleNodeClientVer = "10.3.0"
	DefaultNodeAPIVersion       = "22.21.1"
)

func FilterProjectID(proj string) string {
	p := strings.TrimSpace(proj)
	if p == "" || p == "aicode-consumers" {
		return ""
	}
	return p
}

func resolveCloudCodeBaseURL(isGcpTos bool, projectID string) string {
	if override := strings.TrimSpace(os.Getenv("ANTIGRAVITY_CLOUD_CODE_URL_OVERRIDE")); override != "" {
		return override
	}
	if isGcpTos && FilterProjectID(projectID) != "" {
		return CloudCodeProdBaseURL
	}
	return CloudCodeDailyBaseURL
}

func getPlatformString() string {
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "DARWIN_ARM64"
		}
		return "DARWIN_AMD64"
	case "windows":
		return "WINDOWS_AMD64"
	case "linux":
		if runtime.GOARCH == "arm64" {
			return "LINUX_ARM64"
		}
		return "LINUX_AMD64"
	default:
		return "PLATFORM_UNSPECIFIED"
	}
}

func getUserAgentOS() string {
	switch runtime.GOOS {
	case "darwin":
		return "darwin"
	case "windows":
		return "windows"
	case "linux":
		return "linux"
	default:
		return "darwin"
	}
}

func getUserAgentArch() string {
	switch runtime.GOARCH {
	case "arm64":
		return "arm64"
	default:
		return "amd64"
	}
}

func buildLoadCodeAssistUserAgent() string {
	return fmt.Sprintf("antigravity/%s %s/%s google-api-nodejs-client/%s",
		DefaultIdeVersion, getUserAgentOS(), getUserAgentArch(), DefaultGoogleNodeClientVer)
}

func buildCloudCodeUserAgent() string {
	return fmt.Sprintf("antigravity/%s %s/%s", DefaultIdeVersion, getUserAgentOS(), getUserAgentArch())
}

func buildCloudCodeMetadata(duetProject string) map[string]interface{} {
	meta := map[string]interface{}{
		"ideName":       "antigravity",
		"ideType":       "ANTIGRAVITY",
		"ideVersion":    DefaultIdeVersion,
		"pluginVersion": "unknown",
		"platform":      getPlatformString(),
		"updateChannel": "stable",
		"pluginType":    "GEMINI",
	}
	if p := FilterProjectID(duetProject); p != "" {
		meta["duetProject"] = p
	}
	return meta
}

type loadProjectResponse struct {
	Project      interface{} `json:"cloudaicompanionProject"`
	CurrentTier  *tierRaw    `json:"currentTier"`
	PaidTier     *tierRaw    `json:"paidTier"`
	AllowedTiers []struct {
		ID        *string `json:"id"`
		IsDefault *bool   `json:"isDefault"`
	} `json:"allowedTiers"`
}

type tierRaw struct {
	ID               *string `json:"id"`
	AvailableCredits []struct {
		CreditType                  *string `json:"creditType"`
		CreditAmount                *string `json:"creditAmount"`
		MinimumCreditAmountForUsage *string `json:"minimumCreditAmountForUsage"`
	} `json:"availableCredits"`
}

type onboardResponse struct {
	Name     *string          `json:"name"`
	Done     *bool            `json:"done"`
	Response *loadProjectResponse `json:"response"`
}

type availableModelsResponse struct {
	Models map[string]struct {
		DisplayName *string `json:"displayName"`
		QuotaInfo   *struct {
			RemainingFraction *float64 `json:"remainingFraction"`
			ResetTime         *string  `json:"resetTime"`
		} `json:"quotaInfo"`
	} `json:"models"`
}

type quotaSummaryResponse struct {
	Groups []struct {
		Buckets []struct {
			BucketID          *string  `json:"bucketId"`
			DisplayName       *string  `json:"displayName"`
			RemainingFraction *float64 `json:"remainingFraction"`
			ResetTime         *string  `json:"resetTime"`
		} `json:"buckets"`
	} `json:"groups"`
}

func extractProjectID(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return FilterProjectID(s)
	}
	if m, ok := v.(map[string]interface{}); ok {
		if id, ok := m["id"].(string); ok {
			return FilterProjectID(id)
		}
	}
	return ""
}

// FetchProjectAndSubscription calls v1internal:loadCodeAssist to discover project ID, tier, and credits.
func FetchProjectAndSubscription(accessToken string, preferredProjectID string, isGcpTos bool) (projectID string, tier string, credits []models.AntigravityCreditInfo, err error) {
	effectivePreferred := FilterProjectID(preferredProjectID)
	baseURL := resolveCloudCodeBaseURL(isGcpTos, effectivePreferred)
	ua := buildLoadCodeAssistUserAgent()
	xGoogClient := fmt.Sprintf("gl-node/%s", DefaultNodeAPIVersion)

	payload := map[string]interface{}{
		"metadata": buildCloudCodeMetadata(effectivePreferred),
		"mode":     "FULL_ELIGIBILITY_CHECK",
	}
	if effectivePreferred != "" {
		payload["cloudaicompanionProject"] = effectivePreferred
	}

	jsonBytes, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/%s", baseURL, LoadCodeAssistPath)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBytes))
	if err != nil {
		return "", "", nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", ua)
	req.Header.Set("x-goog-api-client", xGoogClient)
	req.Header.Set("Accept", "*/*")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", nil, fmt.Errorf("loadCodeAssist 网络失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", nil, fmt.Errorf("读取 loadCodeAssist 响应失败: %w", err)
	}

	if resp.StatusCode == http.StatusForbidden {
		return "", "", nil, fmt.Errorf("HTTP 403 Forbidden: 账号无访问权限或已被限制")
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return "", "", nil, fmt.Errorf("HTTP 401 Unauthorized: Access Token 已失效")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", nil, fmt.Errorf("loadCodeAssist 失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var data loadProjectResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return "", "", nil, fmt.Errorf("解析 loadCodeAssist 响应失败: %w", err)
	}

	if data.PaidTier != nil && data.PaidTier.ID != nil {
		tier = *data.PaidTier.ID
	} else if data.CurrentTier != nil && data.CurrentTier.ID != nil {
		tier = *data.CurrentTier.ID
	}

	if data.PaidTier != nil {
		for _, raw := range data.PaidTier.AvailableCredits {
			if raw.CreditType != nil && raw.CreditAmount != nil {
				credits = append(credits, models.AntigravityCreditInfo{
					CreditType:                  *raw.CreditType,
					CreditAmount:                raw.CreditAmount,
					MinimumCreditAmountForUsage: raw.MinimumCreditAmountForUsage,
				})
			}
		}
	}

	projectID = extractProjectID(data.Project)
	if projectID != "" {
		return projectID, tier, credits, nil
	}

	// Try onboarding if project is missing
	var onboardTier string
	for _, t := range data.AllowedTiers {
		if t.IsDefault != nil && *t.IsDefault && t.ID != nil {
			onboardTier = *t.ID
			break
		}
	}
	if onboardTier == "" && len(data.AllowedTiers) > 0 && data.AllowedTiers[0].ID != nil {
		onboardTier = *data.AllowedTiers[0].ID
	}
	if onboardTier == "" && tier != "" {
		onboardTier = tier
	}
	if onboardTier == "" {
		onboardTier = "LEGACY"
	}

	projectID, _ = tryOnboardUser(client, baseURL, accessToken, onboardTier, preferredProjectID, ua)
	return projectID, tier, credits, nil
}

func tryOnboardUser(client *http.Client, baseURL, accessToken, tierID, preferredProjectID, ua string) (string, error) {
	payload := map[string]interface{}{
		"tierId":   tierID,
		"metadata": buildCloudCodeMetadata(preferredProjectID),
	}
	if preferredProjectID != "" {
		payload["cloudaicompanionProject"] = preferredProjectID
	}
	jsonBytes, _ := json.Marshal(payload)
	url := fmt.Sprintf("%s/%s", baseURL, OnboardUserPath)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", ua)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("onboardUser 失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var data onboardResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}

	for i := 0; i < 15; i++ {
		if data.Done != nil && *data.Done {
			if data.Response != nil {
				return extractProjectID(data.Response.Project), nil
			}
			return "", nil
		}
		if data.Name == nil || strings.TrimSpace(*data.Name) == "" {
			break
		}

		time.Sleep(600 * time.Millisecond)
		pollURL := fmt.Sprintf("%s/v1internal/%s", baseURL, *data.Name)
		pollReq, err := http.NewRequest(http.MethodGet, pollURL, nil)
		if err != nil {
			break
		}
		pollReq.Header.Set("Authorization", "Bearer "+accessToken)
		pollReq.Header.Set("Content-Type", "application/json")
		pollReq.Header.Set("User-Agent", ua)

		pollResp, err := client.Do(pollReq)
		if err != nil {
			break
		}
		pollBody, _ := io.ReadAll(pollResp.Body)
		pollResp.Body.Close()

		if pollResp.StatusCode == http.StatusOK {
			_ = json.Unmarshal(pollBody, &data)
		}
	}

	return "", nil
}

// FetchAvailableModels queries model list and remaining fractions.
func FetchAvailableModels(accessToken string, projectID string, isGcpTos bool) ([]models.AntigravityQuotaModel, error) {
	effectiveProj := FilterProjectID(projectID)
	baseURL := resolveCloudCodeBaseURL(isGcpTos, effectiveProj)
	ua := buildCloudCodeUserAgent()

	payload := map[string]interface{}{}
	if effectiveProj != "" {
		payload["project"] = effectiveProj
	}
	jsonBytes, _ := json.Marshal(payload)

	url := fmt.Sprintf("%s/%s", baseURL, FetchAvailableModelsPath)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", ua)

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetchAvailableModels 网络失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 fetchAvailableModels 响应失败: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetchAvailableModels 失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var data availableModelsResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("解析 fetchAvailableModels 响应失败: %w", err)
	}

	var result []models.AntigravityQuotaModel
	for name, info := range data.Models {
		if !strings.Contains(name, "gemini") && !strings.Contains(name, "claude") {
			continue
		}
		pct := 0
		resetTime := ""
		if info.QuotaInfo != nil {
			if info.QuotaInfo.RemainingFraction != nil {
				pct = int(*info.QuotaInfo.RemainingFraction * 100.0)
			}
			if info.QuotaInfo.ResetTime != nil {
				resetTime = *info.QuotaInfo.ResetTime
			}
		}
		disp := ""
		if info.DisplayName != nil {
			disp = *info.DisplayName
		}
		result = append(result, models.AntigravityQuotaModel{
			Name:        name,
			DisplayName: disp,
			Percentage:  pct,
			ResetTime:   resetTime,
		})
	}

	return result, nil
}

// RetrieveUserQuotaSummary queries user quota summary (weekly and 5h buckets).
func RetrieveUserQuotaSummary(accessToken string, projectID string, isGcpTos bool) ([]models.AntigravityQuotaModel, error) {
	effectiveProj := FilterProjectID(projectID)
	baseURL := resolveCloudCodeBaseURL(isGcpTos, effectiveProj)
	ua := buildCloudCodeUserAgent()

	payload := map[string]interface{}{}
	if effectiveProj != "" {
		payload["project"] = effectiveProj
	}
	jsonBytes, _ := json.Marshal(payload)

	url := fmt.Sprintf("%s/%s", baseURL, RetrieveUserQuotaSummaryPath)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", ua)

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("retrieveUserQuotaSummary 网络失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 retrieveUserQuotaSummary 响应失败: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("retrieveUserQuotaSummary 失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var data quotaSummaryResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("解析 retrieveUserQuotaSummary 响应失败: %w", err)
	}

	var result []models.AntigravityQuotaModel
	for _, group := range data.Groups {
		for _, b := range group.Buckets {
			if b.BucketID == nil || b.RemainingFraction == nil {
				continue
			}
			pct := int(*b.RemainingFraction * 100.0)
			reset := ""
			if b.ResetTime != nil {
				reset = *b.ResetTime
			}
			disp := ""
			if b.DisplayName != nil {
				disp = *b.DisplayName
			}
			result = append(result, models.AntigravityQuotaModel{
				Name:        *b.BucketID,
				DisplayName: disp,
				Percentage:  pct,
				ResetTime:   reset,
			})
		}
	}

	return result, nil
}

// RefreshAccountQuota updates token if needed, fetches project, tier, credits, and model quotas.
func RefreshAccountQuota(account *models.AntigravityAccount) error {
	now := time.Now().Unix()

	doRefresh := func() error {
		if account.RefreshToken == "" {
			return fmt.Errorf("缺少 refresh_token")
		}
		tokenRes, err := RefreshAccessToken(account.RefreshToken)
		if err != nil {
			account.Status = "error"
			msg := fmt.Sprintf("Token 刷新失败: %s", err.Error())
			account.StatusMessage = &msg
			return err
		}
		account.AccessToken = tokenRes.AccessToken
		account.ExpiresIn = tokenRes.ExpiresIn
		account.ExpiryTimestamp = now + tokenRes.ExpiresIn
		if tokenRes.IDToken != nil && *tokenRes.IDToken != "" {
			account.IDToken = tokenRes.IDToken
		}
		return nil
	}

	// 1. Check token expiry, refresh if within 5 minutes (300s)
	if account.RefreshToken != "" && (account.ExpiryTimestamp == 0 || account.ExpiryTimestamp <= now+300) {
		if err := doRefresh(); err != nil {
			return err
		}
	}

	// 2. Fetch Project ID, Tier, and Credits
	preferredProject := ""
	if account.ProjectID != nil {
		preferredProject = FilterProjectID(*account.ProjectID)
	}
	projectID, tier, credits, err := FetchProjectAndSubscription(account.AccessToken, preferredProject, account.IsGcpTos)
	if err != nil {
		// If 401 Unauthorized and refresh token exists, refresh token and retry once
		if strings.Contains(err.Error(), "401") && account.RefreshToken != "" {
			if rErr := doRefresh(); rErr == nil {
				projectID, tier, credits, err = FetchProjectAndSubscription(account.AccessToken, preferredProject, account.IsGcpTos)
			}
		}
	}
	if err != nil {
		if strings.Contains(err.Error(), "403") {
			account.Status = "forbidden"
			msg := err.Error()
			account.StatusMessage = &msg
		} else if strings.Contains(err.Error(), "401") {
			account.Status = "expired"
			msg := err.Error()
			account.StatusMessage = &msg
		} else {
			account.Status = "error"
			msg := err.Error()
			account.StatusMessage = &msg
		}
		return err
	}

	cleanedProj := FilterProjectID(projectID)
	if cleanedProj != "" {
		account.ProjectID = &cleanedProj
	} else {
		// Clean up placeholder aicode-consumers
		account.ProjectID = nil
	}
	if tier != "" {
		account.SubscriptionTier = &tier
	}
	if len(credits) > 0 {
		creditsBytes, _ := json.Marshal(credits)
		creditsStr := string(creditsBytes)
		account.CreditsJSON = &creditsStr
	}

	effectiveProject := ""
	if account.ProjectID != nil {
		effectiveProject = FilterProjectID(*account.ProjectID)
	}

	// 3. Fetch Available Models
	modelsList, _ := FetchAvailableModels(account.AccessToken, effectiveProject, account.IsGcpTos)

	// 4. Retrieve Quota Summary (5h and weekly buckets)
	summaryList, _ := RetrieveUserQuotaSummary(account.AccessToken, effectiveProject, account.IsGcpTos)

	// Merge models: summary buckets first (e.g. claude:5h, claude:weekly, gemini:5h, gemini:weekly), then individual models
	seen := make(map[string]bool)
	var combined []models.AntigravityQuotaModel
	for _, m := range summaryList {
		if !seen[m.Name] {
			seen[m.Name] = true
			combined = append(combined, m)
		}
	}
	for _, m := range modelsList {
		if !seen[m.Name] {
			seen[m.Name] = true
			combined = append(combined, m)
		}
	}

	if len(combined) > 0 {
		modelsBytes, _ := json.Marshal(combined)
		modelsStr := string(modelsBytes)
		account.QuotaModelsJSON = &modelsStr
	}

	account.Status = "active"
	account.StatusMessage = nil
	t := time.Now()
	account.LastRefreshAt = &t
	return nil
}
