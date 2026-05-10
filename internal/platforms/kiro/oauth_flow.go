// oauth_flow.go — Kiro OAuth authorization flow (PKCE + local callback server).
// Ported from cockpit-tools-main/src-tauri/src/modules/kiro_oauth.rs.
package kiro

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ---- OAuth flow state management ----

const (
	kiroAuthPortalURL = "https://app.kiro.dev/signin"
	oauthTimeoutSec   = 600
)

var (
	pendingLoginMu    sync.Mutex
	pendingLoginState *PendingOAuthState
)

// PendingOAuthState holds the state of an in-progress OAuth flow.
type PendingOAuthState struct {
	LoginID                 string `json:"login_id"`
	ExpiresAt               int64  `json:"expires_at"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	CallbackURL             string `json:"callback_url"`
	CallbackPort            int    `json:"callback_port"`
	StateToken              string `json:"state_token"`
	CodeVerifier            string `json:"code_verifier"`
	CallbackResult          chan *OAuthCallbackResult
}

// OAuthCallbackResult is the result from the callback server.
type OAuthCallbackResult struct {
	Code        string
	LoginOption string
	IssuerURL   string
	IDCRegion   string
	ClientID    string
	Scopes      string
	LoginHint   string
	Error       string
}

// OAuthStartResponse is returned by StartOAuthFlow.
type OAuthStartResponse struct {
	LoginID                 string `json:"login_id"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	CallbackURL             string `json:"callback_url"`
	ExpiresIn               int64  `json:"expires_in"`
}

// ---- PKCE helpers ----

func generateToken() string {
	b := make([]byte, 24)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func generateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// ---- OAuth flow ----

// StartOAuthFlow initiates the Kiro OAuth flow.
// Returns the verification URL for the user to visit.
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
	codeVerifier := generateToken()
	codeChallenge := generateCodeChallenge(codeVerifier)
	callbackURL := fmt.Sprintf("http://localhost:%d/oauth/callback", port)

	verificationURIComplete := buildPortalAuthURL(stateToken, codeChallenge, callbackURL)

	state := &PendingOAuthState{
		LoginID:                 loginID,
		ExpiresAt:               time.Now().Unix() + oauthTimeoutSec,
		VerificationURI:         kiroAuthPortalURL,
		VerificationURIComplete: verificationURIComplete,
		CallbackURL:             callbackURL,
		CallbackPort:            port,
		StateToken:              stateToken,
		CodeVerifier:            codeVerifier,
		CallbackResult:          make(chan *OAuthCallbackResult, 1),
	}
	pendingLoginState = state

	// Start callback server
	go startCallbackServer(state)

	return &OAuthStartResponse{
		LoginID:                 loginID,
		VerificationURI:         kiroAuthPortalURL,
		VerificationURIComplete: verificationURIComplete,
		CallbackURL:             callbackURL,
		ExpiresIn:               oauthTimeoutSec,
	}, nil
}

// CompleteOAuthFlow waits for the OAuth callback and exchanges the code for tokens.
func CompleteOAuthFlow(loginID string, timeoutSec int) (*RefreshResult, error) {
	pendingLoginMu.Lock()
	state := pendingLoginState
	pendingLoginMu.Unlock()

	if state == nil || state.LoginID != loginID {
		return nil, fmt.Errorf("login session not found or expired")
	}

	timeout := time.Duration(timeoutSec) * time.Second
	select {
	case result := <-state.CallbackResult:
		if result.Error != "" {
			return nil, fmt.Errorf("oauth callback error: %s", result.Error)
		}
		if result.Code == "" {
			return nil, fmt.Errorf("oauth callback missing code")
		}

		// Exchange code for token
		redirectURI := buildTokenExchangeRedirectURI(state.CallbackURL, result.LoginOption)
		tokenResp, err := exchangeCodeForToken(result.Code, state.CodeVerifier, redirectURI)
		if err != nil {
			return nil, fmt.Errorf("token exchange failed: %w", err)
		}

		// Extract profile ARN and region from callback or token
		profileArn := extractProfileArn(tokenResp, result)
		region := result.IDCRegion
		if region == "" {
			region = "us-east-1"
		}

		// Build refresh result
		refreshResult := &RefreshResult{
			TokenPayload: TokenPayload{
				AccessToken:  tokenResp["accessToken"].(string),
				RefreshToken: stringPtr(tokenResp, "refreshToken"),
				TokenType:    stringPtr(tokenResp, "tokenType"),
				ExpiresAt:    int64Ptr(tokenResp, "expiresAt"),
				RawJSON:      tokenResp,
			},
		}

		// Fetch usage if we have profile ARN
		if profileArn != "" {
			usage, _ := FetchUsageLimits(refreshResult.AccessToken, profileArn, region)
			if usage != nil {
				refreshResult.UsageInfo = *usage
			}
		}

		// Extract email from token or callback
		if email := extractEmail(tokenResp, result); email != "" {
			refreshResult.UsageInfo.Email = email
		}

		return refreshResult, nil

	case <-time.After(timeout):
		return nil, fmt.Errorf("oauth flow timeout")
	}
}

// CancelOAuthFlow cancels the pending OAuth flow.
func CancelOAuthFlow(loginID string) error {
	pendingLoginMu.Lock()
	defer pendingLoginMu.Unlock()

	if pendingLoginState != nil && (loginID == "" || pendingLoginState.LoginID == loginID) {
		close(pendingLoginState.CallbackResult)
		pendingLoginState = nil
	}
	return nil
}

// ---- callback server ----

func startCallbackServer(state *PendingOAuthState) {
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/callback", func(w http.ResponseWriter, r *http.Request) {
		handleCallback(w, r, state)
	})
	mux.HandleFunc("/signin/callback", func(w http.ResponseWriter, r *http.Request) {
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
	query := r.URL.Query()

	// Check for errors
	if errCode := query.Get("error"); errCode != "" {
		errDesc := query.Get("error_description")
		result := &OAuthCallbackResult{
			Error: fmt.Sprintf("%s: %s", errCode, errDesc),
		}
		select {
		case state.CallbackResult <- result:
		default:
		}
		http.Redirect(w, r, authErrorRedirectURL(result.Error), http.StatusFound)
		return
	}

	// Verify state
	if query.Get("state") != state.StateToken {
		result := &OAuthCallbackResult{Error: "state mismatch"}
		select {
		case state.CallbackResult <- result:
		default:
		}
		http.Redirect(w, r, authErrorRedirectURL("state mismatch"), http.StatusFound)
		return
	}

	// Extract callback data
	result := &OAuthCallbackResult{
		Code:        query.Get("code"),
		LoginOption: query.Get("login_option"),
		IssuerURL:   query.Get("issuer_url"),
		IDCRegion:   query.Get("idc_region"),
		ClientID:    query.Get("client_id"),
		Scopes:      query.Get("scopes"),
		LoginHint:   query.Get("login_hint"),
	}

	select {
	case state.CallbackResult <- result:
	default:
	}

	http.Redirect(w, r, authSuccessRedirectURL(), http.StatusFound)
}

// ---- helpers ----

func findAvailablePort() (int, error) {
	candidates := []int{3128, 4649, 6588, 8008, 9091, 49153, 50153, 51153, 52153, 53153}
	for _, port := range candidates {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			ln.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port")
}

func buildPortalAuthURL(state, codeChallenge, redirectURI string) string {
	return fmt.Sprintf("%s?state=%s&code_challenge=%s&code_challenge_method=S256&redirect_uri=%s&redirect_from=KiroIDE",
		kiroAuthPortalURL,
		url.QueryEscape(state),
		url.QueryEscape(codeChallenge),
		url.QueryEscape(redirectURI),
	)
}

func buildTokenExchangeRedirectURI(baseURL, loginOption string) string {
	return fmt.Sprintf("%s?login_option=%s", baseURL, url.QueryEscape(loginOption))
}

func exchangeCodeForToken(code, codeVerifier, redirectURI string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"code":          code,
		"code_verifier": codeVerifier,
		"redirect_uri":  redirectURI,
	}
	result, status, err := postJSON(kiroTokenEndpoint, body)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("token exchange returned HTTP %d", status)
	}
	// Unwrap "data" wrapper if present
	if data, ok := result["data"].(map[string]interface{}); ok {
		return data, nil
	}
	return result, nil
}

func authSuccessRedirectURL() string {
	return fmt.Sprintf("%s?auth_status=success&redirect_from=KiroIDE", kiroAuthPortalURL)
}

func authErrorRedirectURL(message string) string {
	return fmt.Sprintf("%s?auth_status=error&redirect_from=KiroIDE&error_message=%s",
		kiroAuthPortalURL, url.QueryEscape(message))
}

func extractProfileArn(token map[string]interface{}, callback *OAuthCallbackResult) string {
	if arn := stringFromMap(token, "profileArn", "profile_arn", "arn"); arn != "" {
		return arn
	}
	return ""
}

func extractEmail(token map[string]interface{}, callback *OAuthCallbackResult) string {
	if email := stringFromMap(token, "email"); email != "" {
		return email
	}
	if callback != nil && callback.LoginHint != "" {
		return callback.LoginHint
	}
	return ""
}

func stringFromMap(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func stringPtr(m map[string]interface{}, key string) *string {
	if v, ok := m[key].(string); ok && strings.TrimSpace(v) != "" {
		s := strings.TrimSpace(v)
		return &s
	}
	return nil
}

func int64Ptr(m map[string]interface{}, key string) *int64 {
	if v, ok := m[key].(float64); ok {
		i := int64(v)
		return &i
	}
	return nil
}
