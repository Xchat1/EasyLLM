// oauth_flow.go — Gemini OAuth authorization flow (Google OAuth 2.0).
// Ported from cockpit-tools-main/src-tauri/src/modules/gemini_oauth.rs.
package gemini

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ---- OAuth flow constants ----

const (
	geminiOAuthAuthURL  = "https://accounts.google.com/o/oauth2/v2/auth"
	geminiOAuthTokenURL = "https://oauth2.googleapis.com/token"
	googleUserInfoURL   = "https://www.googleapis.com/oauth2/v2/userinfo"
	oauthCallbackPath   = "/oauth2callback"
	oauthTimeoutSec     = 300
)

var oauthScopes = []string{
	"https://www.googleapis.com/auth/cloud-platform",
	"https://www.googleapis.com/auth/userinfo.email",
	"https://www.googleapis.com/auth/userinfo.profile",
}

var (
	pendingLoginMu    sync.Mutex
	pendingLoginState *PendingOAuthState
)

// PendingOAuthState holds the state of an in-progress OAuth flow.
type PendingOAuthState struct {
	LoginID        string `json:"login_id"`
	ExpiresAt      int64  `json:"expires_at"`
	CallbackURL    string `json:"callback_url"`
	CallbackPort   int    `json:"callback_port"`
	StateToken     string `json:"state_token"`
	AuthURL        string `json:"auth_url"`
	CallbackResult chan *OAuthCallbackResult
}

// OAuthCallbackResult is the result from the callback server.
type OAuthCallbackResult struct {
	Code  string
	Error string
}

// OAuthStartResponse is returned by StartOAuthFlow.
type OAuthStartResponse struct {
	LoginID     string `json:"login_id"`
	AuthURL     string `json:"auth_url"`
	CallbackURL string `json:"callback_url"`
	ExpiresIn   int64  `json:"expires_in"`
}

// OAuthCompletePayload holds the token exchange result.
type OAuthCompletePayload struct {
	Email         string                 `json:"email"`
	AuthID        string                 `json:"auth_id,omitempty"`
	Name          *string                `json:"name,omitempty"`
	AccessToken   string                 `json:"access_token"`
	RefreshToken  *string                `json:"refresh_token,omitempty"`
	IDToken       *string                `json:"id_token,omitempty"`
	TokenType     *string                `json:"token_type,omitempty"`
	Scope         *string                `json:"scope,omitempty"`
	ExpiryDate    *int64                 `json:"expiry_date,omitempty"`
	ProjectID     *string                `json:"project_id,omitempty"`
	TierID        *string                `json:"tier_id,omitempty"`
	TierName      *string                `json:"tier_name,omitempty"`
	GeminiAuthRaw map[string]interface{} `json:"gemini_auth_raw,omitempty"`
	Status        string                 `json:"status,omitempty"`
}

// ---- PKCE helpers ----

func generateToken() string {
	b := make([]byte, 24)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// ---- OAuth flow ----

// StartOAuthFlow initiates the Gemini OAuth flow.
func StartOAuthFlow() (*OAuthStartResponse, error) {
	pendingLoginMu.Lock()
	defer pendingLoginMu.Unlock()

	// Find available port
	port, err := findAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("no available callback port: %w", err)
	}

	loginID := generateToken()
	stateToken := generateToken()
	callbackURL := fmt.Sprintf("http://127.0.0.1:%d%s", port, oauthCallbackPath)

	authURL := buildAuthURL(callbackURL, stateToken)

	state := &PendingOAuthState{
		LoginID:        loginID,
		ExpiresAt:      time.Now().Unix() + oauthTimeoutSec,
		CallbackURL:    callbackURL,
		CallbackPort:   port,
		StateToken:     stateToken,
		AuthURL:        authURL,
		CallbackResult: make(chan *OAuthCallbackResult, 1),
	}
	pendingLoginState = state

	// Start callback server
	go startCallbackServer(state)

	return &OAuthStartResponse{
		LoginID:     loginID,
		AuthURL:     authURL,
		CallbackURL: callbackURL,
		ExpiresIn:   oauthTimeoutSec,
	}, nil
}

// CompleteOAuthFlow waits for the OAuth callback and exchanges the code for tokens.
func CompleteOAuthFlow(loginID string, timeoutSec int) (*OAuthCompletePayload, error) {
	pendingLoginMu.Lock()
	state := pendingLoginState
	pendingLoginMu.Unlock()

	if state == nil || state.LoginID != loginID {
		return nil, fmt.Errorf("login session not found or expired")
	}

	timeout := time.Duration(timeoutSec) * time.Second
	select {
	case result := <-state.CallbackResult:
		clearPendingOAuthState(loginID)
		if result == nil {
			return nil, fmt.Errorf("oauth flow cancelled")
		}
		if result.Error != "" {
			return nil, fmt.Errorf("oauth callback error: %s", result.Error)
		}
		if result.Code == "" {
			return nil, fmt.Errorf("oauth callback missing code")
		}

		// Exchange code for token
		tokenResp, err := exchangeCodeForToken(result.Code, state.CallbackURL)
		if err != nil {
			return nil, fmt.Errorf("token exchange failed: %w", err)
		}

		// Fetch user info
		accessToken := stringFromMap(tokenResp, "access_token")
		userInfo, _ := fetchUserInfo(accessToken)

		email := ""
		if userInfo != nil {
			email = stringFromMap(userInfo, "email")
		}
		authID := ""
		if userInfo != nil {
			authID = stringFromMap(userInfo, "id")
		}
		if authID == "" && tokenResp["id_token"] != nil {
			authID = parseJWTClaimString(stringFromMap(tokenResp, "id_token"), "sub")
		}
		name := ""
		if userInfo != nil {
			name = stringFromMap(userInfo, "name")
		}
		if name == "" && tokenResp["id_token"] != nil {
			name = parseJWTClaimString(stringFromMap(tokenResp, "id_token"), "name")
		}

		// Fetch code assist status
		codeAssist, _ := LoadCodeAssist(accessToken)

		payload := &OAuthCompletePayload{
			Email:        email,
			AuthID:       authID,
			AccessToken:  accessToken,
			RefreshToken: stringPtrFromMap(tokenResp, "refresh_token"),
			IDToken:      stringPtrFromMap(tokenResp, "id_token"),
			TokenType:    stringPtrFromMap(tokenResp, "token_type"),
			Scope:        stringPtrFromMap(tokenResp, "scope"),
			Status:       "active",
		}
		if strings.TrimSpace(name) != "" {
			payload.Name = &name
		}

		if expiresIn, ok := tokenResp["expires_in"].(float64); ok {
			expiryMs := time.Now().UnixMilli() + int64(expiresIn)*1000
			payload.ExpiryDate = &expiryMs
		}

		if codeAssist != nil {
			payload.ProjectID = codeAssist.ProjectID
			payload.TierID = codeAssist.TierID
			payload.TierName = codeAssist.TierName
		}

		authRaw := make(map[string]interface{})
		for _, key := range []string{"access_token", "refresh_token", "id_token", "token_type", "scope", "expires_in"} {
			if value, ok := tokenResp[key]; ok {
				authRaw[key] = value
			}
		}
		if payload.ExpiryDate != nil {
			authRaw["expiry_date"] = *payload.ExpiryDate
		}
		if email != "" {
			authRaw["email"] = email
		}
		if authID != "" {
			authRaw["sub"] = authID
		}
		payload.GeminiAuthRaw = authRaw

		return payload, nil

	case <-time.After(timeout):
		clearPendingOAuthState(loginID)
		return nil, fmt.Errorf("oauth flow timeout")
	}
}

// CancelOAuthFlow cancels the pending OAuth flow.
func CancelOAuthFlow(loginID string) error {
	pendingLoginMu.Lock()
	state := pendingLoginState
	if pendingLoginState != nil && (loginID == "" || pendingLoginState.LoginID == loginID) {
		pendingLoginState = nil
	}
	pendingLoginMu.Unlock()

	if state != nil && (loginID == "" || state.LoginID == loginID) {
		select {
		case state.CallbackResult <- &OAuthCallbackResult{Error: "cancelled"}:
		default:
		}
	}
	return nil
}

func SubmitOAuthCallbackURL(loginID, callbackURL string) error {
	pendingLoginMu.Lock()
	state := pendingLoginState
	pendingLoginMu.Unlock()

	if state == nil || state.LoginID != loginID {
		return fmt.Errorf("login session not found or expired")
	}
	parsed, err := parseOAuthCallbackURL(callbackURL, state)
	if err != nil {
		return err
	}
	if parsed.Path != oauthCallbackPath {
		return fmt.Errorf("callback path must be %s", oauthCallbackPath)
	}
	result := callbackResultFromURL(parsed, state.StateToken)
	select {
	case state.CallbackResult <- result:
	default:
	}
	return nil
}

// ---- callback server ----

func startCallbackServer(state *PendingOAuthState) {
	mux := http.NewServeMux()
	mux.HandleFunc(oauthCallbackPath, func(w http.ResponseWriter, r *http.Request) {
		handleCallback(w, r, state)
	})

	server := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", state.CallbackPort),
		Handler: mux,
	}

	go func() {
		_ = server.ListenAndServe()
	}()

	// Auto-shutdown after timeout
	time.AfterFunc(time.Duration(oauthTimeoutSec)*time.Second, func() {
		_ = server.Close()
	})
}

func handleCallback(w http.ResponseWriter, r *http.Request, state *PendingOAuthState) {
	result := callbackResultFromURL(r.URL, state.StateToken)
	if result.Error != "" {
		select {
		case state.CallbackResult <- result:
		default:
		}
		http.Redirect(w, r, "https://developers.google.com/gemini-code-assist/auth_failure_gemini", http.StatusFound)
		return
	}

	select {
	case state.CallbackResult <- result:
	default:
	}

	http.Redirect(w, r, "https://developers.google.com/gemini-code-assist/auth_success_gemini", http.StatusFound)
}

func callbackResultFromURL(u *url.URL, expectedState string) *OAuthCallbackResult {
	query := u.Query()
	if errCode := query.Get("error"); errCode != "" {
		if desc := query.Get("error_description"); desc != "" {
			errCode += ": " + desc
		}
		return &OAuthCallbackResult{Error: errCode}
	}
	if query.Get("state") != expectedState {
		return &OAuthCallbackResult{Error: "state mismatch"}
	}
	code := strings.TrimSpace(query.Get("code"))
	if code == "" {
		return &OAuthCallbackResult{Error: "missing code"}
	}
	return &OAuthCallbackResult{Code: code}
}

func parseOAuthCallbackURL(raw string, state *PendingOAuthState) (*url.URL, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return nil, fmt.Errorf("callback_url is empty")
	}
	if strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://") {
		return url.Parse(text)
	}
	base := fmt.Sprintf("http://127.0.0.1:%d", state.CallbackPort)
	if strings.HasPrefix(text, "/") {
		return url.Parse(base + text)
	}
	return url.Parse(base + oauthCallbackPath + "?" + strings.TrimPrefix(text, "?"))
}

func clearPendingOAuthState(loginID string) {
	pendingLoginMu.Lock()
	defer pendingLoginMu.Unlock()
	if pendingLoginState != nil && pendingLoginState.LoginID == loginID {
		pendingLoginState = nil
	}
}

// ---- helpers ----

func findAvailablePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

func buildAuthURL(redirectURI, state string) string {
	u, _ := url.Parse(geminiOAuthAuthURL)
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", geminiOAuthClientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("access_type", "offline")
	q.Set("prompt", "consent")
	q.Set("state", state)
	q.Set("scope", strings.Join(oauthScopes, " "))
	u.RawQuery = q.Encode()
	return u.String()
}

func exchangeCodeForToken(code, redirectURI string) (map[string]interface{}, error) {
	clientSecret, err := geminiOAuthClientSecret()
	if err != nil {
		return nil, err
	}

	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", geminiOAuthClientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequest(http.MethodPost, geminiOAuthTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange returned HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("token exchange parse failed: %w", err)
	}

	return result, nil
}

func fetchUserInfo(accessToken string) (map[string]interface{}, error) {
	req, err := http.NewRequest(http.MethodGet, googleUserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo returned HTTP %d", resp.StatusCode)
	}

	raw, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("userinfo parse failed: %w", err)
	}

	return result, nil
}

func stringFromMap(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func parseJWTClaimString(token, key string) string {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	if value, ok := claims[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func stringPtrFromMap(m map[string]interface{}, key string) *string {
	if v, ok := m[key].(string); ok && strings.TrimSpace(v) != "" {
		s := strings.TrimSpace(v)
		return &s
	}
	return nil
}
