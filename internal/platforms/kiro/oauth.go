// Package kiro implements Kiro IDE account token refresh and quota fetching.
// Logic ported from cockpit-tools-main/src-tauri/src/modules/kiro_oauth.rs
// and kiro_account.rs.
package kiro

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ---- endpoints (mirrors kiro_oauth.rs constants) ----

const (
	kiroRefreshEndpoint        = "https://prod.us-east-1.auth.desktop.kiro.dev/refreshToken"
	kiroTokenEndpoint          = "https://prod.us-east-1.auth.desktop.kiro.dev/oauth/token"
	kiroRuntimeDefaultEndpoint = "https://q.us-east-1.amazonaws.com"
)

// ---- public types ----

// TokenPayload holds the refreshed token fields returned by Kiro auth service.
type TokenPayload struct {
	AccessToken  string  `json:"access_token,omitempty"`
	RefreshToken *string `json:"refresh_token,omitempty"`
	TokenType    *string `json:"token_type,omitempty"`
	ExpiresAt    *int64  `json:"expires_at,omitempty"`
	// Raw JSON blob stored back into MetadataJSON
	RawJSON map[string]interface{} `json:"-"`
}

// UsageInfo holds Kiro credits/bonus quota extracted from the runtime usage API.
type UsageInfo struct {
	Email           string   `json:"email,omitempty"`
	UserID          *string  `json:"user_id,omitempty"`
	LoginProvider   *string  `json:"login_provider,omitempty"`
	PlanName        *string  `json:"plan_name,omitempty"`
	PlanTier        *string  `json:"plan_tier,omitempty"`
	CreditsTotal    *float64 `json:"credits_total,omitempty"`
	CreditsUsed     *float64 `json:"credits_used,omitempty"`
	BonusTotal      *float64 `json:"bonus_total,omitempty"`
	BonusUsed       *float64 `json:"bonus_used,omitempty"`
	UsageResetAt    *int64   `json:"usage_reset_at,omitempty"`
	BonusExpireDays *int64   `json:"bonus_expire_days,omitempty"`
	Status          *string  `json:"status,omitempty"`
	StatusReason    *string  `json:"status_reason,omitempty"`
	// Raw usage JSON
	RawUsage map[string]interface{} `json:"raw_usage,omitempty"`
}

// RefreshResult is the combined result of a token refresh + usage fetch.
type RefreshResult struct {
	TokenPayload
	UsageInfo
}

// ---- HTTP helpers ----

func httpClient() *http.Client {
	return &http.Client{Timeout: 20 * time.Second}
}

func postJSON(url string, body interface{}) (map[string]interface{}, int, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal request body: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("POST %s: %w", url, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	_ = json.Unmarshal(raw, &result)
	return result, resp.StatusCode, nil
}

func getJSON(url, bearerToken string) (map[string]interface{}, int, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}
	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	_ = json.Unmarshal(raw, &result)
	return result, resp.StatusCode, nil
}

// ---- token refresh (mirrors kiro_oauth.rs refresh_token_via_remote) ----

// RefreshAccessToken exchanges a refresh_token for a new access_token via the
// Kiro auth service. Mirrors refresh_token_via_remote in kiro_oauth.rs.
func RefreshAccessToken(refreshToken string) (*TokenPayload, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil, fmt.Errorf("refresh_token is empty")
	}

	body := map[string]interface{}{
		"refreshToken": refreshToken,
	}
	result, status, err := postJSON(kiroRefreshEndpoint, body)
	if err != nil {
		return nil, fmt.Errorf("kiro refreshToken request failed: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("kiro refreshToken returned HTTP %d", status)
	}

	// Unwrap "data" wrapper if present (mirrors unwrap_token_response)
	if data, ok := result["data"].(map[string]interface{}); ok {
		result = data
	}

	payload := &TokenPayload{RawJSON: result}
	if v := pickString(result, "accessToken", "access_token", "token", "idToken", "id_token"); v != "" {
		payload.AccessToken = v
	}
	if v := pickString(result, "refreshToken", "refresh_token", "refreshTokenJwt"); v != "" {
		payload.RefreshToken = &v
	}
	if v := pickString(result, "tokenType", "token_type", "authType"); v != "" {
		payload.TokenType = &v
	}
	if ts := parseTimestampFromMap(result, "expiresAt", "expires_at", "expiry", "expiration"); ts != nil {
		payload.ExpiresAt = ts
	} else if expiresIn := pickInt64(result, "expiresIn", "expires_in"); expiresIn > 0 {
		t := time.Now().Unix() + expiresIn
		payload.ExpiresAt = &t
	}

	if payload.AccessToken == "" {
		return nil, fmt.Errorf("kiro refreshToken response missing access_token")
	}
	return payload, nil
}

// ---- runtime usage (mirrors fetch_usage_limits_via_runtime in kiro_oauth.rs) ----

// runtimeEndpointForRegion maps an AWS region to the Q runtime endpoint.
// Mirrors runtime_endpoint_for_region in kiro_oauth.rs.
func runtimeEndpointForRegion(region string) string {
	switch strings.ToLower(strings.TrimSpace(region)) {
	case "us-east-1":
		return "https://q.us-east-1.amazonaws.com"
	case "eu-central-1":
		return "https://q.eu-central-1.amazonaws.com"
	case "us-gov-east-1":
		return "https://q-fips.us-gov-east-1.amazonaws.com"
	case "us-gov-west-1":
		return "https://q-fips.us-gov-west-1.amazonaws.com"
	default:
		return kiroRuntimeDefaultEndpoint
	}
}

// FetchUsageLimits calls the Kiro runtime getUsageLimits endpoint.
// Mirrors fetch_usage_limits_via_runtime in kiro_oauth.rs.
func FetchUsageLimits(accessToken, profileArn, region string) (*UsageInfo, error) {
	endpoint := runtimeEndpointForRegion(region)
	url := fmt.Sprintf("%s/getUsageLimits?origin=AI_EDITOR&profileArn=%s&resourceType=AGENTIC_REQUEST&isEmailRequired=true",
		strings.TrimRight(endpoint, "/"),
		profileArn,
	)

	result, status, err := getJSON(url, accessToken)
	if err != nil {
		return nil, fmt.Errorf("kiro runtime usage request failed: %w", err)
	}
	if status == http.StatusForbidden || status == http.StatusUnauthorized {
		statusStr := "banned"
		reason := extractErrorReason(result)
		return &UsageInfo{Status: &statusStr, StatusReason: &reason}, nil
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("kiro runtime usage returned HTTP %d", status)
	}

	return parseUsageInfo(result), nil
}

// parseUsageInfo extracts UsageInfo from the runtime usage JSON response.
// Mirrors apply_runtime_usage_to_payload + extract_usage_payload in kiro_oauth.rs.
func parseUsageInfo(raw map[string]interface{}) *UsageInfo {
	info := &UsageInfo{RawUsage: raw}

	// email
	if email := pickStringNested(raw, [][]string{{"userInfo", "email"}, {"email"}}); email != "" {
		info.Email = email
	}
	// user_id
	if uid := pickStringNested(raw, [][]string{{"userInfo", "userId"}, {"userId"}, {"user_id"}, {"sub"}}); uid != "" {
		info.UserID = &uid
	}
	// login_provider
	if prov := pickStringNested(raw, [][]string{
		{"userInfo", "provider", "label"},
		{"userInfo", "provider", "name"},
		{"userInfo", "provider", "id"},
		{"provider", "label"},
		{"provider", "name"},
	}); prov != "" {
		info.LoginProvider = &prov
	}

	// subscription info
	if sub, ok := raw["subscriptionInfo"].(map[string]interface{}); ok {
		if title := pickString(sub, "subscriptionTitle"); title != "" {
			info.PlanName = &title
		}
		if t := pickString(sub, "type"); t != "" {
			info.PlanTier = &t
		}
	}

	// usage breakdown list
	if list, ok := raw["usageBreakdownList"].([]interface{}); ok && len(list) > 0 {
		// find credit type or use first
		var breakdown map[string]interface{}
		for _, item := range list {
			if m, ok := item.(map[string]interface{}); ok {
				if strings.EqualFold(fmt.Sprintf("%v", m["type"]), "credit") {
					breakdown = m
					break
				}
			}
		}
		if breakdown == nil {
			if m, ok := list[0].(map[string]interface{}); ok {
				breakdown = m
			}
		}
		if breakdown != nil {
			if v := pickFloat64(breakdown, "usageLimitWithPrecision"); v != nil {
				info.CreditsTotal = v
			}
			if v := pickFloat64(breakdown, "currentUsageWithPrecision"); v != nil {
				info.CreditsUsed = v
			}
			// free trial / bonus
			if ft, ok := breakdown["freeTrialInfo"].(map[string]interface{}); ok {
				if v := pickFloat64(ft, "usageLimitWithPrecision"); v != nil {
					info.BonusTotal = v
				}
				if v := pickFloat64(ft, "currentUsageWithPrecision"); v != nil {
					info.BonusUsed = v
				}
				if expiry := pickInt64(ft, "freeTrialExpiry"); expiry > 0 {
					now := time.Now().Unix()
					if expiry > now {
						days := int64(float64(expiry-now)/86400.0 + 0.9999)
						info.BonusExpireDays = &days
					} else {
						zero := int64(0)
						info.BonusExpireDays = &zero
					}
				}
			}
		}
	}

	// usage reset
	if ts := pickInt64(raw, "nextDateReset"); ts > 0 {
		info.UsageResetAt = &ts
	}

	normal := "normal"
	info.Status = &normal
	return info
}

// ---- RefreshAccount: full refresh flow (mirrors refresh_account_token_once) ----

// RefreshAccount performs a full token refresh + usage fetch for a Kiro account.
// accountMeta is the MetadataJSON blob stored in PlatformAccount.
// Returns updated fields to be merged back into the account.
func RefreshAccount(accessToken, refreshToken, profileArn, region string) (*RefreshResult, error) {
	result := &RefreshResult{}

	// Step 1: refresh the access token
	if refreshToken != "" {
		newToken, err := RefreshAccessToken(refreshToken)
		if err != nil {
			return nil, fmt.Errorf("kiro token refresh failed: %w", err)
		}
		result.TokenPayload = *newToken
		accessToken = newToken.AccessToken
	}

	if accessToken == "" {
		return nil, fmt.Errorf("no access_token available for kiro usage fetch")
	}

	// Step 2: fetch usage limits
	if profileArn != "" {
		usage, err := FetchUsageLimits(accessToken, profileArn, region)
		if err != nil {
			// non-fatal: store error but return what we have
			errStr := err.Error()
			result.UsageInfo.Status = &errStr
		} else {
			result.UsageInfo = *usage
		}
	}

	return result, nil
}

// ---- JSON helpers ----

func pickString(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					return s
				}
			}
		}
	}
	return ""
}

func pickStringNested(m map[string]interface{}, paths [][]string) string {
	for _, path := range paths {
		cur := interface{}(m)
		for _, key := range path {
			if mm, ok := cur.(map[string]interface{}); ok {
				cur = mm[key]
			} else {
				cur = nil
				break
			}
		}
		if s, ok := cur.(string); ok {
			s = strings.TrimSpace(s)
			if s != "" {
				return s
			}
		}
	}
	return ""
}

func pickInt64(m map[string]interface{}, keys ...string) int64 {
	for _, k := range keys {
		if v, ok := m[k]; ok {
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

func pickFloat64(m map[string]interface{}, keys ...string) *float64 {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch n := v.(type) {
			case float64:
				return &n
			case json.Number:
				if f, err := n.Float64(); err == nil {
					return &f
				}
			}
		}
	}
	return nil
}

func parseTimestampFromMap(m map[string]interface{}, keys ...string) *int64 {
	for _, k := range keys {
		v, ok := m[k]
		if !ok {
			continue
		}
		switch n := v.(type) {
		case float64:
			ts := int64(n)
			if ts > 10_000_000_000 {
				ts /= 1000
			}
			if ts > 0 {
				return &ts
			}
		case string:
			n = strings.TrimSpace(n)
			if n == "" {
				continue
			}
			// try RFC3339
			if t, err := time.Parse(time.RFC3339, n); err == nil {
				ts := t.Unix()
				return &ts
			}
			// try "2006/01/02 15:04:05"
			if t, err := time.Parse("2006/01/02 15:04:05", n); err == nil {
				ts := t.Unix()
				return &ts
			}
			// try "2006-01-02 15:04:05"
			if t, err := time.Parse("2006-01-02 15:04:05", n); err == nil {
				ts := t.Unix()
				return &ts
			}
		}
	}
	return nil
}

func extractErrorReason(m map[string]interface{}) string {
	if m == nil {
		return ""
	}
	for _, k := range []string{"reason", "message", "errorMessage", "detail", "details"} {
		if v, ok := m[k].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	if errObj, ok := m["error"].(map[string]interface{}); ok {
		for _, k := range []string{"message", "reason"} {
			if v, ok := errObj[k].(string); ok && strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		}
	}
	return ""
}
