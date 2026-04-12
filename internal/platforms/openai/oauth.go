package openai

import (
	"crypto/rand"
	"crypto/sha256"
	"easyllm/config"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	clientID           = "app_EMoamEEZ73f0CkXaXp7hrann"
	authorizeURL       = "https://auth.openai.com/oauth/authorize"
	tokenURL           = "https://auth.openai.com/oauth/token"
	defaultRedirectURI = "http://localhost:1455/auth/callback"
	defaultScopes      = "openid profile email offline_access"
	defaultOriginator  = "codex_vscode"
)

// TokenResponse is the OpenAI OAuth token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// IDTokenClaims holds parsed JWT claims from id_token
type IDTokenClaims struct {
	Email      *string           `json:"email"`
	OpenAIAuth *OpenAIAuthClaims `json:"https://api.openai.com/auth"`
}

type OpenAIAuthClaims struct {
	ChatGPTAccountID *string   `json:"chatgpt_account_id"`
	ChatGPTUserID    *string   `json:"chatgpt_user_id"`
	Organizations    []OrgInfo `json:"organizations"`
}

type OrgInfo struct {
	ID        *string `json:"id"`
	IsDefault *bool   `json:"is_default"`
}

// UserInfo holds parsed user info from id_token
type UserInfo struct {
	Email            *string
	ChatGPTAccountID *string
	ChatGPTUserID    *string
	OrganizationID   *string
}

// GenerateState generates a random hex state string
func GenerateState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// GenerateCodeVerifier generates a PKCE code verifier
func GenerateCodeVerifier() string {
	b := make([]byte, 64)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// GenerateCodeChallenge generates the S256 code challenge
func GenerateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// BuildAuthorizationURL constructs the OpenAI OAuth authorization URL
func BuildAuthorizationURL(state, codeChallenge, redirectURI string) string {
	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256")
	params.Set("codex_cli_simplified_flow", "true")
	params.Set("id_token_add_organizations", "true")
	params.Set("redirect_uri", redirectURI)
	params.Set("response_type", "code")
	params.Set("scope", defaultScopes)
	params.Set("state", state)
	params.Set("originator", defaultOriginator)
	return authorizeURL + "?" + params.Encode()
}

// RefreshToken exchanges a refresh_token for a new access_token + id_token
func RefreshToken(refreshToken string) (*TokenResponse, error) {
	params := url.Values{}
	params.Set("grant_type", "refresh_token")
	params.Set("refresh_token", refreshToken)
	params.Set("client_id", clientID)

	client := createHTTPClient()
	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("token refresh failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}
	return &tokenResp, nil
}

// ExchangeCode exchanges an OAuth authorization code for tokens
func ExchangeCode(code, codeVerifier, redirectURI string) (*TokenResponse, error) {
	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("client_id", clientID)
	params.Set("code", code)
	params.Set("redirect_uri", redirectURI)
	params.Set("code_verifier", codeVerifier)

	client := createHTTPClient()
	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("token exchange failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}
	return &tokenResp, nil
}

// ParseIDToken parses the JWT id_token and extracts user info
func ParseIDToken(idToken string) *UserInfo {
	parts := strings.Split(idToken, ".")
	if len(parts) < 2 {
		return nil
	}

	// Decode base64url payload
	payload := parts[1]
	// Add padding if needed
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		// Try without padding
		decoded, err = base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return nil
		}
	}

	var claims IDTokenClaims
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil
	}

	info := &UserInfo{
		Email: claims.Email,
	}

	if claims.OpenAIAuth != nil {
		info.ChatGPTAccountID = claims.OpenAIAuth.ChatGPTAccountID
		info.ChatGPTUserID = claims.OpenAIAuth.ChatGPTUserID

		// Find default org
		for _, org := range claims.OpenAIAuth.Organizations {
			if org.IsDefault != nil && *org.IsDefault {
				info.OrganizationID = org.ID
				break
			}
		}
		if info.OrganizationID == nil && len(claims.OpenAIAuth.Organizations) > 0 {
			info.OrganizationID = claims.OpenAIAuth.Organizations[0].ID
		}
	}

	return info
}

// ExtractOpenAIAuthJSON extracts the openai auth claims as JSON string
func ExtractOpenAIAuthJSON(idToken string) string {
	parts := strings.Split(idToken, ".")
	if len(parts) < 2 {
		return ""
	}

	decoded, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return ""
	}

	authObj, ok := claims["https://api.openai.com/auth"]
	if !ok {
		return ""
	}

	b, err := json.Marshal(authObj)
	if err != nil {
		return ""
	}
	return string(b)
}

// createHTTPClient returns an HTTP client with optional proxy support
func createHTTPClient() *http.Client {
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

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
}
