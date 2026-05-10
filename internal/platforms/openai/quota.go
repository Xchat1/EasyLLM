package openai

import (
	"bytes"
	"easyllm/config"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Use wham/usage endpoint with the same response pattern as Quotio.
const usageURL = "https://chatgpt.com/backend-api/wham/usage"
const codexResponsesQuotaURL = "https://chatgpt.com/backend-api/codex/responses?client_version=0.98.0"

// QuotaInfo holds percentage-based quota data extracted from wham/usage JSON response.
type QuotaInfo struct {
	// Legacy fields (kept for backward compat; populated from percentage data)
	Total     int64  // synthetic: 100
	Remaining int64  // synthetic: 100 - 7d used percent
	Used      int64  // synthetic: 7d used percent
	ResetAt   string // from 7d reset seconds, formatted

	// New percentage-based fields (from wham/usage JSON)
	Codex5hUsedPercent   *float64 // 5h window used %
	Codex5hResetSeconds  *int64   // 5h reset countdown (seconds)
	Codex5hWindowMinutes *int64   // 5h window duration (minutes)
	Codex7dUsedPercent   *float64 // 7d window used %
	Codex7dResetSeconds  *int64   // 7d reset countdown (seconds)
	Codex7dWindowMinutes *int64   // 7d window duration (minutes)

	PlanType    *string
	IsForbidden bool // 402/403 response
}

// JSON response structs matching the usage payload
type windowInfo struct {
	UsedPercent        *int   `json:"used_percent,omitempty"`
	LimitWindowSeconds *int64 `json:"limit_window_seconds,omitempty"`
	ResetAfterSeconds  *int64 `json:"reset_after_seconds,omitempty"`
	ResetAt            *int64 `json:"reset_at,omitempty"`
}

type rateLimitInfo struct {
	Allowed         *bool       `json:"allowed,omitempty"`
	LimitReached    *bool       `json:"limit_reached,omitempty"`
	PrimaryWindow   *windowInfo `json:"primary_window,omitempty"`
	SecondaryWindow *windowInfo `json:"secondary_window,omitempty"`
}

type usageResponse struct {
	PlanType            *string        `json:"plan_type,omitempty"`
	RateLimit           *rateLimitInfo `json:"rate_limit,omitempty"`
	CodeReviewRateLimit *rateLimitInfo `json:"code_review_rate_limit,omitempty"`
}

// FetchQuota combines ChatGPT usage data with Codex response headers so the UI
// can show both 5h and 7d windows even when wham/usage only exposes one side.
func FetchQuota(accessToken, chatgptAccountID string) (*QuotaInfo, error) {
	client := newQuotaHTTPClient()
	info, err := fetchUsageQuota(client, accessToken, chatgptAccountID)
	if err != nil {
		return nil, err
	}
	if info != nil && info.IsForbidden {
		return info, nil
	}

	headerInfo, headerErr := fetchCodexHeadersQuota(client, accessToken, chatgptAccountID)
	if headerErr == nil {
		info = mergeQuotaInfo(info, headerInfo)
	}

	return info, nil
}

func newQuotaHTTPClient() *http.Client {
	cfg := config.Get()
	transport := &http.Transport{}
	if cfg.Proxy.Enabled && cfg.Proxy.Host != "" {
		proxyURLStr := fmt.Sprintf("http://%s:%d", cfg.Proxy.Host, cfg.Proxy.Port)
		if cfg.Proxy.Username != "" {
			proxyURLStr = fmt.Sprintf("http://%s:%s@%s:%d",
				url.QueryEscape(cfg.Proxy.Username),
				url.QueryEscape(cfg.Proxy.Password),
				cfg.Proxy.Host, cfg.Proxy.Port)
		}
		if u, err := url.Parse(proxyURLStr); err == nil {
			transport.Proxy = http.ProxyURL(u)
		}
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
	}
	return client
}

func fetchUsageQuota(client *http.Client, accessToken, chatgptAccountID string) (*QuotaInfo, error) {
	req, err := http.NewRequest("GET", usageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	if chatgptAccountID != "" {
		req.Header.Set("ChatGPT-Account-Id", chatgptAccountID)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("quota request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("HTTP 401: Token expired or invalid")
	}
	if resp.StatusCode == 402 || resp.StatusCode == 403 {
		return &QuotaInfo{IsForbidden: true}, nil
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		var usage usageResponse
		if err := json.Unmarshal(body, &usage); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}

		info := parseQuotaFromUsage(&usage)
		return info, nil
	}

	return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
}

func fetchCodexHeadersQuota(client *http.Client, accessToken, chatgptAccountID string) (*QuotaInfo, error) {
	reqBody := map[string]interface{}{
		"model": "gpt-5.1-codex",
		"input": []interface{}{
			map[string]interface{}{
				"role": "user",
				"content": []interface{}{
					map[string]interface{}{"type": "input_text", "text": "quota"},
				},
			},
		},
		"instructions": "You are Codex, based on GPT-5. You are running as a coding agent in the Codex CLI on a user's computer.",
		"stream":       true,
		"store":        false,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, codexResponsesQuotaURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "codex_cli_rs/0.98.0")
	req.Header.Set("OpenAI-Beta", "responses=experimental")
	req.Header.Set("originator", "codex_cli_rs")
	if chatgptAccountID != "" {
		req.Header.Set("ChatGPT-Account-Id", chatgptAccountID)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("codex quota request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("HTTP 401: Token expired or invalid")
	}

	info := ParseCodexHeaders(resp.Header)
	if info.Codex5hUsedPercent == nil &&
		info.Codex7dUsedPercent == nil &&
		info.Codex5hResetSeconds == nil &&
		info.Codex7dResetSeconds == nil &&
		info.Codex5hWindowMinutes == nil &&
		info.Codex7dWindowMinutes == nil {
		return nil, fmt.Errorf("codex quota headers unavailable")
	}
	return info, nil
}

// parseQuotaFromUsage extracts quota info from the wham/usage JSON response.
// OpenAI's usage payload is not stable: 5h/7d windows may appear in either
// primary_window or secondary_window, so we classify by window duration first.
func parseQuotaFromUsage(usage *usageResponse) *QuotaInfo {
	info := &QuotaInfo{}
	if usage != nil && usage.PlanType != nil {
		if plan := NormalizePlanType(*usage.PlanType); plan != "" {
			info.PlanType = &plan
		}
	}

	applyWindow := func(w *windowInfo) {
		if w == nil {
			return
		}
		assignQuotaWindow(info, normalizeUsedPercent(w.UsedPercent), normalizeResetSeconds(w), normalizeWindowMinutes(w))
	}

	if usage.RateLimit != nil {
		applyWindow(usage.RateLimit.PrimaryWindow)
		applyWindow(usage.RateLimit.SecondaryWindow)
	}
	if usage.CodeReviewRateLimit != nil {
		applyWindow(usage.CodeReviewRateLimit.PrimaryWindow)
		applyWindow(usage.CodeReviewRateLimit.SecondaryWindow)
	}

	// Populate legacy fields from 7d data for backward compat
	if info.Codex7dUsedPercent != nil {
		info.Total = 100
		info.Used = int64(math.Round(*info.Codex7dUsedPercent))
		info.Remaining = 100 - info.Used
		if info.Codex7dResetSeconds != nil {
			info.ResetAt = formatResetSeconds(*info.Codex7dResetSeconds)
		}
	}

	return info
}

func mergeQuotaInfo(base, extra *QuotaInfo) *QuotaInfo {
	if base == nil {
		base = &QuotaInfo{}
	}
	if extra == nil {
		return base
	}

	if base.Codex5hUsedPercent == nil {
		base.Codex5hUsedPercent = extra.Codex5hUsedPercent
	}
	if base.Codex5hResetSeconds == nil {
		base.Codex5hResetSeconds = extra.Codex5hResetSeconds
	}
	if base.Codex5hWindowMinutes == nil {
		base.Codex5hWindowMinutes = extra.Codex5hWindowMinutes
	}
	if base.Codex7dUsedPercent == nil {
		base.Codex7dUsedPercent = extra.Codex7dUsedPercent
	}
	if base.Codex7dResetSeconds == nil {
		base.Codex7dResetSeconds = extra.Codex7dResetSeconds
	}
	if base.Codex7dWindowMinutes == nil {
		base.Codex7dWindowMinutes = extra.Codex7dWindowMinutes
	}
	if base.Total == 0 {
		base.Total = extra.Total
	}
	if base.Used == 0 {
		base.Used = extra.Used
	}
	if base.Remaining == 0 {
		base.Remaining = extra.Remaining
	}
	if base.ResetAt == "" {
		base.ResetAt = extra.ResetAt
	}
	if base.PlanType == nil {
		base.PlanType = extra.PlanType
	}
	base.IsForbidden = base.IsForbidden || extra.IsForbidden

	return base
}

func NormalizePlanType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return ""
	}
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.Join(strings.Fields(normalized), "-")
	switch {
	case strings.Contains(normalized, "team"):
		return "team"
	case strings.Contains(normalized, "enterprise"):
		return "enterprise"
	case strings.Contains(normalized, "business"):
		return "business"
	case strings.Contains(normalized, "pro-max"), strings.Contains(normalized, "promax"):
		return "promax"
	case strings.Contains(normalized, "pro-lite"), strings.Contains(normalized, "prolite"):
		return "prolite"
	case normalized == "pro" || strings.Contains(normalized, "chatgpt-pro"):
		return "pro"
	case strings.Contains(normalized, "plus"):
		return "plus"
	case strings.Contains(normalized, "free"):
		return "free"
	default:
		return normalized
	}
}

func assignQuotaWindow(info *QuotaInfo, used *float64, reset *int64, minutes *int64) {
	if info == nil {
		return
	}

	if minutes != nil {
		if *minutes <= 360 {
			if info.Codex5hUsedPercent == nil && info.Codex5hResetSeconds == nil && info.Codex5hWindowMinutes == nil {
				info.Codex5hUsedPercent = used
				info.Codex5hResetSeconds = reset
				info.Codex5hWindowMinutes = minutes
			}
			return
		}

		if info.Codex7dUsedPercent == nil && info.Codex7dResetSeconds == nil && info.Codex7dWindowMinutes == nil {
			info.Codex7dUsedPercent = used
			info.Codex7dResetSeconds = reset
			info.Codex7dWindowMinutes = minutes
		}
		return
	}

	// Fallback for payloads without explicit window size.
	if info.Codex5hUsedPercent == nil && info.Codex5hResetSeconds == nil && info.Codex5hWindowMinutes == nil {
		info.Codex5hUsedPercent = used
		info.Codex5hResetSeconds = reset
		info.Codex5hWindowMinutes = minutes
		return
	}
	if info.Codex7dUsedPercent == nil && info.Codex7dResetSeconds == nil && info.Codex7dWindowMinutes == nil {
		info.Codex7dUsedPercent = used
		info.Codex7dResetSeconds = reset
		info.Codex7dWindowMinutes = minutes
	}
}

// normalizeUsedPercent returns used_percent (clamped 0-100), or nil if not available.
func normalizeUsedPercent(val *int) *float64 {
	if val == nil {
		return nil
	}
	used := float64(max(0, min(100, *val)))
	return &used
}

// normalizeWindowMinutes returns limit_window_seconds / 60 (ceiling), or nil.
func normalizeWindowMinutes(w *windowInfo) *int64 {
	if w == nil || w.LimitWindowSeconds == nil || *w.LimitWindowSeconds <= 0 {
		return nil
	}
	minutes := (*w.LimitWindowSeconds + 59) / 60
	return &minutes
}

// normalizeResetSeconds returns reset countdown (now + reset_after_seconds or reset_at), or nil.
func normalizeResetSeconds(w *windowInfo) *int64 {
	if w == nil {
		return nil
	}
	if w.ResetAt != nil {
		// reset_at is Unix timestamp, convert to countdown
		now := time.Now().Unix()
		countdown := *w.ResetAt - now
		if countdown < 0 {
			countdown = 0
		}
		return &countdown
	}
	if w.ResetAfterSeconds != nil && *w.ResetAfterSeconds >= 0 {
		return w.ResetAfterSeconds
	}
	return nil
}

func formatResetSeconds(seconds int64) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60
	var parts []string
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	return strings.Join(parts, "")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ParseCodexHeaders extracts quota info from x-codex-* response headers.
// Used by proxy to parse quota from upstream responses.
func ParseCodexHeaders(h http.Header) *QuotaInfo {
	info := &QuotaInfo{}

	primaryUsedPct := parseFloatHeader(h, "x-codex-primary-used-percent")
	primaryResetSec := parseInt64Header(h, "x-codex-primary-reset-after-seconds")
	primaryWindowMin := parseInt64Header(h, "x-codex-primary-window-minutes")

	secondaryUsedPct := parseFloatHeader(h, "x-codex-secondary-used-percent")
	secondaryResetSec := parseInt64Header(h, "x-codex-secondary-reset-after-seconds")
	secondaryWindowMin := parseInt64Header(h, "x-codex-secondary-window-minutes")

	assignQuotaWindow(info, primaryUsedPct, primaryResetSec, primaryWindowMin)
	assignQuotaWindow(info, secondaryUsedPct, secondaryResetSec, secondaryWindowMin)

	// Fallback to legacy x-ratelimit-* headers
	if info.Codex5hUsedPercent == nil && info.Codex7dUsedPercent == nil {
		parseLegacyHeaders(h, info)
	}

	// Populate legacy fields from 7d data for backward compat
	if info.Codex7dUsedPercent != nil {
		info.Total = 100
		info.Used = int64(math.Round(*info.Codex7dUsedPercent))
		info.Remaining = 100 - info.Used
		if info.Codex7dResetSeconds != nil {
			info.ResetAt = formatResetSeconds(*info.Codex7dResetSeconds)
		}
	}

	return info
}

func parseFloatHeader(h http.Header, key string) *float64 {
	v := findHeader(h, key)
	if v == "" {
		return nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return nil
	}
	return &f
}

func parseInt64Header(h http.Header, key string) *int64 {
	v := findHeader(h, key)
	if v == "" {
		return nil
	}
	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil
	}
	return &i
}

func findHeader(h http.Header, key string) string {
	if v := h.Get(key); v != "" {
		return v
	}
	lower := strings.ToLower(key)
	for k, vals := range h {
		if strings.ToLower(k) == lower && len(vals) > 0 {
			return vals[0]
		}
	}
	return ""
}

func parseLegacyHeaders(h http.Header, info *QuotaInfo) {
	if v := findHeader(h, "x-ratelimit-limit-requests"); v != "" {
		info.Total, _ = strconv.ParseInt(v, 10, 64)
	}
	if v := findHeader(h, "x-ratelimit-remaining-requests"); v != "" {
		info.Remaining, _ = strconv.ParseInt(v, 10, 64)
	}
	if v := findHeader(h, "x-ratelimit-reset-requests"); v != "" {
		info.ResetAt = v
	}

	if info.Total > 0 {
		info.Used = info.Total - info.Remaining
		pct := float64(info.Used) / float64(info.Total) * 100
		info.Codex7dUsedPercent = &pct
	}
}
