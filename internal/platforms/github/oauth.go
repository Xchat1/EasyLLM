// Package github implements GitHub Copilot account token refresh and quota fetching.
// Logic ported from cockpit-tools-main/src-tauri/src/modules/github_copilot_oauth.rs
// and github_copilot_account.rs.
package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ---- endpoints (mirrors github_copilot_oauth.rs constants) ----

const (
	githubDeviceCodeEndpoint  = "https://github.com/login/device/code"
	githubDeviceTokenEndpoint = "https://github.com/login/oauth/access_token"
	githubUserEndpoint        = "https://api.github.com/user"
	githubUserEmailsEndpoint  = "https://api.github.com/user/emails"
	copilotTokenEndpoint      = "https://api.github.com/copilot_internal/v2/token"
	copilotUserInfoEndpoint   = "https://api.github.com/copilot_internal/user"
	githubOAuthClientID       = "01ab8ac9400c4e429b23"
	githubOAuthScope          = "read:user user:email repo workflow"
	appUserAgent              = "antigravity-cockpit-tools"
	githubAPIVersion          = "2025-04-01"
)

// ---- public types ----

// CopilotTokenBundle holds the Copilot token and associated quota/plan info.
// Mirrors CopilotTokenBundle in github_copilot_oauth.rs.
type CopilotTokenBundle struct {
	Token                string      `json:"token"`
	Plan                 *string     `json:"plan,omitempty"`
	ChatEnabled          *bool       `json:"chat_enabled,omitempty"`
	ExpiresAt            *int64      `json:"expires_at,omitempty"`
	RefreshIn            *int64      `json:"refresh_in,omitempty"`
	QuotaSnapshots       interface{} `json:"quota_snapshots,omitempty"`
	QuotaResetDate       *string     `json:"quota_reset_date,omitempty"`
	LimitedUserQuotas    interface{} `json:"limited_user_quotas,omitempty"`
	LimitedUserResetDate *int64      `json:"limited_user_reset_date,omitempty"`
}

// GitHubUser holds basic GitHub user profile.
type GitHubUser struct {
	ID    int64   `json:"id"`
	Login string  `json:"login"`
	Name  *string `json:"name,omitempty"`
	Email *string `json:"email,omitempty"`
}

// DeviceCodeResponse is the response from the GitHub device code endpoint.
type DeviceCodeResponse struct {
	DeviceCode              string  `json:"device_code"`
	UserCode                string  `json:"user_code"`
	VerificationURI         string  `json:"verification_uri"`
	VerificationURIComplete *string `json:"verification_uri_complete,omitempty"`
	ExpiresIn               int64   `json:"expires_in"`
	Interval                *int64  `json:"interval,omitempty"`
}

// DeviceTokenResponse is the polling response from the GitHub device token endpoint.
type DeviceTokenResponse struct {
	AccessToken      *string `json:"access_token,omitempty"`
	TokenType        *string `json:"token_type,omitempty"`
	Scope            *string `json:"scope,omitempty"`
	Error            *string `json:"error,omitempty"`
	ErrorDescription *string `json:"error_description,omitempty"`
}

// ---- HTTP helpers ----

func httpClient() *http.Client {
	return &http.Client{Timeout: 20 * time.Second}
}

func githubGet(endpoint, token string) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", appUserAgent)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", githubAPIVersion)
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}
	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return body, resp.StatusCode, nil
}

// ---- Device flow (mirrors start_login / complete_login in github_copilot_oauth.rs) ----

// RequestDeviceCode initiates the GitHub device flow.
// Mirrors request_device_code in github_copilot_oauth.rs.
func RequestDeviceCode() (*DeviceCodeResponse, error) {
	form := url.Values{}
	form.Set("client_id", githubOAuthClientID)
	form.Set("scope", githubOAuthScope)

	req, err := http.NewRequest(http.MethodPost, githubDeviceCodeEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", appUserAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("github device code request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github device code returned HTTP %d", resp.StatusCode)
	}

	var result DeviceCodeResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("github device code parse failed: %w", err)
	}
	return &result, nil
}

// PollDeviceToken polls the GitHub device token endpoint once.
// Mirrors exchange_device_token in github_copilot_oauth.rs.
func PollDeviceToken(deviceCode string) (*DeviceTokenResponse, error) {
	form := url.Values{}
	form.Set("client_id", githubOAuthClientID)
	form.Set("device_code", deviceCode)
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	req, err := http.NewRequest(http.MethodPost, githubDeviceTokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", appUserAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("github device token request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github device token returned HTTP %d", resp.StatusCode)
	}

	var result DeviceTokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("github device token parse failed: %w", err)
	}
	return &result, nil
}

// ---- user info ----

// FetchUser fetches the GitHub user profile.
// Mirrors fetch_github_user in github_copilot_oauth.rs.
func FetchUser(githubAccessToken string) (*GitHubUser, error) {
	body, status, err := githubGet(githubUserEndpoint, githubAccessToken)
	if err != nil {
		return nil, fmt.Errorf("github user request failed: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("github user returned HTTP %d", status)
	}
	var user GitHubUser
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("github user parse failed: %w", err)
	}
	return &user, nil
}

// FetchPrimaryEmail fetches the primary verified email from GitHub.
// Mirrors fetch_github_email in github_copilot_oauth.rs.
func FetchPrimaryEmail(githubAccessToken string) (*string, error) {
	body, status, err := githubGet(githubUserEmailsEndpoint, githubAccessToken)
	if err != nil {
		return nil, fmt.Errorf("github emails request failed: %w", err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("github emails returned HTTP %d", status)
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  *bool  `json:"primary"`
		Verified *bool  `json:"verified"`
	}
	if err := json.Unmarshal(body, &emails); err != nil {
		return nil, fmt.Errorf("github emails parse failed: %w", err)
	}

	// prefer primary + verified
	for _, e := range emails {
		if e.Primary != nil && *e.Primary && e.Verified != nil && *e.Verified {
			email := e.Email
			return &email, nil
		}
	}
	// fallback: any verified
	for _, e := range emails {
		if e.Verified != nil && *e.Verified {
			email := e.Email
			return &email, nil
		}
	}
	return nil, nil
}

// ---- Copilot token (mirrors fetch_copilot_token) ----

// FetchCopilotToken fetches the Copilot short-lived token and quota info.
// Mirrors fetch_copilot_token in github_copilot_oauth.rs.
func FetchCopilotToken(githubAccessToken string) (*CopilotTokenBundle, error) {
	req, err := http.NewRequest(http.MethodGet, copilotTokenEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", appUserAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-GitHub-Api-Version", githubAPIVersion)
	req.Header.Set("Authorization", "token "+githubAccessToken)

	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("copilot token request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("copilot token returned HTTP %d", resp.StatusCode)
	}

	var payload struct {
		Token                string      `json:"token"`
		ExpiresAt            *int64      `json:"expires_at"`
		RefreshIn            *int64      `json:"refresh_in"`
		SKU                  *string     `json:"sku"`
		ChatEnabled          *bool       `json:"chat_enabled"`
		LimitedUserQuotas    interface{} `json:"limited_user_quotas"`
		LimitedUserResetDate *int64      `json:"limited_user_reset_date"`
		Message              *string     `json:"message"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("copilot token parse failed: %w", err)
	}
	if payload.Token == "" {
		msg := "copilot token missing"
		if payload.Message != nil {
			msg = *payload.Message
		}
		return nil, fmt.Errorf("%s", msg)
	}

	bundle := &CopilotTokenBundle{
		Token:                payload.Token,
		ChatEnabled:          payload.ChatEnabled,
		ExpiresAt:            payload.ExpiresAt,
		RefreshIn:            payload.RefreshIn,
		LimitedUserQuotas:    payload.LimitedUserQuotas,
		LimitedUserResetDate: payload.LimitedUserResetDate,
	}

	// fetch user info for plan + quota snapshots
	userInfo, _ := fetchCopilotUserInfo(githubAccessToken)
	if userInfo != nil {
		bundle.Plan = userInfo.CopilotPlan
		bundle.QuotaSnapshots = userInfo.QuotaSnapshots
		bundle.QuotaResetDate = userInfo.QuotaResetDate
	} else {
		bundle.Plan = payload.SKU
	}

	return bundle, nil
}

type copilotUserInfoResponse struct {
	CopilotPlan    *string     `json:"copilot_plan"`
	QuotaSnapshots interface{} `json:"quota_snapshots"`
	QuotaResetDate *string     `json:"quota_reset_date"`
}

func fetchCopilotUserInfo(githubAccessToken string) (*copilotUserInfoResponse, error) {
	body, status, err := githubGet(copilotUserInfoEndpoint, githubAccessToken)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("copilot user info returned HTTP %d", status)
	}
	var info copilotUserInfoResponse
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// ---- RefreshCopilotToken: refresh flow (mirrors refresh_account_token_once) ----

// RefreshCopilotToken refreshes the Copilot short-lived token using the stored
// GitHub access token. This is the main "refresh" operation for GitHub Copilot accounts.
// Mirrors refresh_copilot_token in github_copilot_oauth.rs.
func RefreshCopilotToken(githubAccessToken string) (*CopilotTokenBundle, error) {
	githubAccessToken = strings.TrimSpace(githubAccessToken)
	if githubAccessToken == "" {
		return nil, fmt.Errorf("github_access_token is empty")
	}
	return FetchCopilotToken(githubAccessToken)
}

// BuildPayloadFromGitHubToken builds a full account payload from a raw GitHub access token.
// Mirrors build_payload_from_github_access_token in github_copilot_oauth.rs.
func BuildPayloadFromGitHubToken(githubAccessToken string) (*AccountPayload, error) {
	user, err := FetchUser(githubAccessToken)
	if err != nil {
		return nil, fmt.Errorf("fetch github user failed: %w", err)
	}

	email := user.Email
	if email == nil {
		email, _ = FetchPrimaryEmail(githubAccessToken)
	}

	copilot, err := FetchCopilotToken(githubAccessToken)
	if err != nil {
		return nil, fmt.Errorf("fetch copilot token failed: %w", err)
	}

	return &AccountPayload{
		GitHubLogin:       user.Login,
		GitHubID:          user.ID,
		GitHubName:        user.Name,
		GitHubEmail:       email,
		GitHubAccessToken: githubAccessToken,
		CopilotBundle:     copilot,
	}, nil
}

// AccountPayload is the full payload for creating/updating a GitHub Copilot account.
type AccountPayload struct {
	GitHubLogin       string              `json:"github_login"`
	GitHubID          int64               `json:"github_id"`
	GitHubName        *string             `json:"github_name,omitempty"`
	GitHubEmail       *string             `json:"github_email,omitempty"`
	GitHubAccessToken string              `json:"github_access_token"`
	CopilotBundle     *CopilotTokenBundle `json:"copilot_bundle,omitempty"`
}
