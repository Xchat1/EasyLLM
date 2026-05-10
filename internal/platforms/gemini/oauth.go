// Package gemini implements Gemini CLI account token refresh and quota fetching.
// Logic ported from cockpit-tools-main/src-tauri/src/modules/gemini_account.rs
// and gemini_oauth.rs.
package gemini

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ---- endpoints (mirrors gemini_account.rs / gemini_oauth.rs constants) ----

const (
	googleTokenEndpoint             = "https://oauth2.googleapis.com/token"
	googleUserInfoEndpoint          = "https://www.googleapis.com/oauth2/v2/userinfo"
	codeAssistLoadEndpoint          = "https://cloudcode-pa.googleapis.com/v1internal:loadCodeAssist"
	codeAssistRetrieveQuotaEndpoint = "https://cloudcode-pa.googleapis.com/v1internal:retrieveUserQuota"

	geminiOAuthClientID = "681255809395-oo8ft2oprdrnp9e3aqf6av3hmdib135j.apps.googleusercontent.com"
)

func geminiOAuthClientSecret() (string, error) {
	secret := strings.TrimSpace(os.Getenv("GEMINI_OAUTH_CLIENT_SECRET"))
	if secret == "" {
		return "", fmt.Errorf("GEMINI_OAUTH_CLIENT_SECRET is not configured")
	}
	return secret, nil
}

// ---- public types ----

// TokenRefreshResult holds the refreshed Google OAuth token fields.
type TokenRefreshResult struct {
	AccessToken  string  `json:"access_token"`
	RefreshToken *string `json:"refresh_token,omitempty"`
	IDToken      *string `json:"id_token,omitempty"`
	TokenType    *string `json:"token_type,omitempty"`
	Scope        *string `json:"scope,omitempty"`
	// ExpiryDate is Unix timestamp in milliseconds (matches Gemini local storage format)
	ExpiryDate *int64 `json:"expiry_date,omitempty"`
}

// UserInfo holds Google user profile fields.
type UserInfo struct {
	ID    *string `json:"id,omitempty"`
	Email *string `json:"email,omitempty"`
	Name  *string `json:"name,omitempty"`
}

// LoadCodeAssistStatus holds the tier/project info from loadCodeAssist.
type LoadCodeAssistStatus struct {
	TierID    *string `json:"tier_id,omitempty"`
	TierName  *string `json:"tier_name,omitempty"`
	ProjectID *string `json:"project_id,omitempty"`
}

// QuotaInfo holds the raw quota JSON from retrieveUserQuota.
type QuotaInfo struct {
	Raw map[string]interface{} `json:"raw,omitempty"`
}

// RefreshResult is the combined result of a Gemini account refresh.
type RefreshResult struct {
	Token      TokenRefreshResult
	UserInfo   *UserInfo
	CodeAssist *LoadCodeAssistStatus
	Quota      *QuotaInfo
	// Error from quota fetch (non-fatal)
	QuotaError *string
}

// ---- HTTP helpers ----

func httpClient() *http.Client {
	return &http.Client{Timeout: 20 * time.Second}
}

// ---- token refresh (mirrors force_refresh_access_token / refresh_access_token) ----

// RefreshAccessToken exchanges a refresh_token for a new access_token via Google OAuth.
// Mirrors refresh_access_token in gemini_account.rs.
func RefreshAccessToken(refreshToken string) (*TokenRefreshResult, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil, fmt.Errorf("gemini refresh_token is empty")
	}
	clientSecret, err := geminiOAuthClientSecret()
	if err != nil {
		return nil, err
	}

	form := strings.NewReader(fmt.Sprintf(
		"client_id=%s&client_secret=%s&refresh_token=%s&grant_type=refresh_token",
		geminiOAuthClientID, clientSecret, refreshToken,
	))
	req, err := http.NewRequest(http.MethodPost, googleTokenEndpoint, form)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini token refresh request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini token refresh returned HTTP %d", resp.StatusCode)
	}

	var payload struct {
		AccessToken  string  `json:"access_token"`
		RefreshToken *string `json:"refresh_token"`
		IDToken      *string `json:"id_token"`
		TokenType    *string `json:"token_type"`
		Scope        *string `json:"scope"`
		ExpiresIn    *int64  `json:"expires_in"`
		Error        *string `json:"error"`
		ErrorDesc    *string `json:"error_description"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("gemini token refresh parse failed: %w", err)
	}
	if payload.AccessToken == "" {
		errMsg := "unknown error"
		if payload.Error != nil {
			errMsg = *payload.Error
		}
		if payload.ErrorDesc != nil {
			errMsg += ": " + *payload.ErrorDesc
		}
		return nil, fmt.Errorf("gemini token refresh missing access_token: %s", errMsg)
	}

	result := &TokenRefreshResult{
		AccessToken:  payload.AccessToken,
		RefreshToken: payload.RefreshToken,
		IDToken:      payload.IDToken,
		TokenType:    payload.TokenType,
		Scope:        payload.Scope,
	}
	if payload.ExpiresIn != nil {
		expiryMs := time.Now().UnixMilli() + (*payload.ExpiresIn)*1000
		result.ExpiryDate = &expiryMs
	}
	return result, nil
}

// ---- Google userinfo (mirrors fetch_google_userinfo) ----

// FetchUserInfo fetches the Google user profile for the given access token.
func FetchUserInfo(accessToken string) (*UserInfo, error) {
	req, err := http.NewRequest(http.MethodGet, googleUserInfoEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini userinfo request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini userinfo returned HTTP %d", resp.StatusCode)
	}

	var info UserInfo
	raw, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil, fmt.Errorf("gemini userinfo parse failed: %w", err)
	}
	return &info, nil
}

// ---- loadCodeAssist (mirrors load_code_assist_status) ----

// LoadCodeAssist calls the Cloud Code loadCodeAssist endpoint to get tier/project info.
func LoadCodeAssist(accessToken string) (*LoadCodeAssistStatus, error) {
	body := map[string]interface{}{
		"metadata": map[string]interface{}{
			"ideType":    "IDE_UNSPECIFIED",
			"platform":   "PLATFORM_UNSPECIFIED",
			"pluginType": "GEMINI",
		},
	}
	data, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, codeAssistLoadEndpoint, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini loadCodeAssist request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("UNAUTHORIZED")
	}
	if !isSuccessStatus(resp.StatusCode) {
		return nil, fmt.Errorf("gemini loadCodeAssist returned HTTP %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("gemini loadCodeAssist parse failed: %w", err)
	}

	status := &LoadCodeAssistStatus{}
	if ct, ok := result["currentTier"].(map[string]interface{}); ok {
		if id, ok := ct["id"].(string); ok && strings.TrimSpace(id) != "" {
			status.TierID = &id
		}
		if name, ok := ct["name"].(string); ok && strings.TrimSpace(name) != "" {
			status.TierName = &name
		}
	}
	// project_id may come from currentTier or top-level
	if pid, ok := result["projectId"].(string); ok && strings.TrimSpace(pid) != "" {
		status.ProjectID = &pid
	} else if ct, ok := result["currentTier"].(map[string]interface{}); ok {
		if pid, ok := ct["projectId"].(string); ok && strings.TrimSpace(pid) != "" {
			status.ProjectID = &pid
		}
	}
	return status, nil
}

// ---- retrieveUserQuota (mirrors retrieve_user_quota) ----

// RetrieveUserQuota fetches the Gemini quota for the given project.
func RetrieveUserQuota(accessToken, projectID string) (*QuotaInfo, error) {
	body := map[string]interface{}{"project": projectID}
	data, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, codeAssistRetrieveQuotaEndpoint, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini retrieveUserQuota request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("FORBIDDEN")
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("UNAUTHORIZED")
	}
	if !isSuccessStatus(resp.StatusCode) {
		return nil, fmt.Errorf("gemini retrieveUserQuota returned HTTP %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("gemini retrieveUserQuota parse failed: %w", err)
	}
	return &QuotaInfo{Raw: result}, nil
}

// ---- RefreshAccount: full refresh flow (mirrors refresh_account_token_once) ----

// RefreshAccount performs a full Gemini account refresh:
//  1. Refresh access token if expiry is near or forced
//  2. Call loadCodeAssist to get tier/project
//  3. Fetch userinfo
//  4. Retrieve quota
//
// accessToken and refreshToken come from the stored PlatformAccount fields.
// expiryDateMs is the stored expiry_date in milliseconds (0 = unknown).
func RefreshAccount(accessToken, refreshToken string, expiryDateMs int64) (*RefreshResult, error) {
	result := &RefreshResult{}

	// Step 1: ensure access token is valid (refresh if expiring within 60s)
	shouldRefresh := expiryDateMs > 0 && time.Now().UnixMilli() >= expiryDateMs-60_000
	if shouldRefresh || accessToken == "" {
		if refreshToken == "" {
			return nil, fmt.Errorf("gemini refresh_token missing, cannot refresh access_token")
		}
		newToken, err := RefreshAccessToken(refreshToken)
		if err != nil {
			return nil, err
		}
		result.Token = *newToken
		accessToken = newToken.AccessToken
	} else {
		result.Token.AccessToken = accessToken
	}

	// Step 2: loadCodeAssist (retry once after token refresh on 401)
	codeAssist, err := LoadCodeAssist(accessToken)
	if err != nil {
		if strings.Contains(err.Error(), "UNAUTHORIZED") && refreshToken != "" {
			// force refresh and retry
			newToken, rerr := RefreshAccessToken(refreshToken)
			if rerr != nil {
				return nil, fmt.Errorf("gemini loadCodeAssist 401, token re-refresh failed: %w", rerr)
			}
			result.Token = *newToken
			accessToken = newToken.AccessToken
			codeAssist, err = LoadCodeAssist(accessToken)
			if err != nil {
				return nil, fmt.Errorf("gemini loadCodeAssist failed after token refresh: %w", err)
			}
		} else {
			return nil, fmt.Errorf("gemini loadCodeAssist failed: %w", err)
		}
	}
	result.CodeAssist = codeAssist

	// Step 3: userinfo
	userInfo, _ := FetchUserInfo(accessToken)
	result.UserInfo = userInfo

	// Step 4: quota (non-fatal)
	if codeAssist != nil && codeAssist.ProjectID != nil {
		quota, qerr := RetrieveUserQuota(accessToken, *codeAssist.ProjectID)
		if qerr != nil {
			errStr := qerr.Error()
			result.QuotaError = &errStr
		} else {
			result.Quota = quota
		}
	}

	return result, nil
}

// ---- helpers ----

func isSuccessStatus(code int) bool {
	return code >= 200 && code < 300
}

// ParseTierPlanName maps a tier_id to a human-readable plan name.
// Mirrors parse_tier_plan_name in gemini_account.rs.
func ParseTierPlanName(tierID string) string {
	switch strings.ToUpper(strings.TrimSpace(tierID)) {
	case "GEMINI_CODE_ASSIST_ENTERPRISE":
		return "Enterprise"
	case "GEMINI_CODE_ASSIST_STANDARD":
		return "Standard"
	case "GEMINI_CODE_ASSIST_FREE":
		return "Free"
	default:
		return tierID
	}
}
