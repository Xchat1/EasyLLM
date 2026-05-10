// Package antigravity implements the Google OAuth flow used by Antigravity.
// The endpoints and scopes mirror cockpit-tools-main's generic Antigravity OAuth logic.
package antigravity

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	antigravityOAuthClientID     = "1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com"
	antigravityOAuthAuthURL      = "https://accounts.google.com/o/oauth2/v2/auth"
	antigravityOAuthTokenURL     = "https://oauth2.googleapis.com/token"
	antigravityUserInfoURL       = "https://www.googleapis.com/oauth2/v2/userinfo"
	antigravityOAuthCallbackPath = "/oauth-callback"
	antigravityOAuthTimeoutSec   = 600
)

var antigravityOAuthScopes = []string{
	"https://www.googleapis.com/auth/cloud-platform",
	"https://www.googleapis.com/auth/userinfo.email",
	"https://www.googleapis.com/auth/userinfo.profile",
	"https://www.googleapis.com/auth/cclog",
	"https://www.googleapis.com/auth/experimentsandconfigs",
}

var (
	antigravityOAuthMu    sync.Mutex
	antigravityOAuthState *PendingOAuthState
)

type PendingOAuthState struct {
	LoginID        string
	ExpiresAt      int64
	AuthURL        string
	CallbackURL    string
	CallbackPort   int
	StateToken     string
	CallbackResult chan *OAuthCallbackResult
}

type OAuthCallbackResult struct {
	Code  string
	Error string
}

type OAuthStartResponse struct {
	LoginID     string `json:"login_id"`
	AuthURL     string `json:"auth_url"`
	CallbackURL string `json:"callback_url"`
	ExpiresIn   int64  `json:"expires_in"`
}

type OAuthCompletePayload struct {
	Email           string  `json:"email"`
	AuthID          string  `json:"auth_id,omitempty"`
	Name            *string `json:"name,omitempty"`
	AccessToken     string  `json:"access_token"`
	RefreshToken    string  `json:"refresh_token"`
	ExpiresIn       int64   `json:"expires_in"`
	ExpiryTimestamp int64   `json:"expiry_timestamp"`
	TokenType       string  `json:"token_type"`
	ProjectID       *string `json:"project_id,omitempty"`
	SessionID       *string `json:"session_id,omitempty"`
}

type tokenResponse struct {
	AccessToken      string  `json:"access_token"`
	ExpiresIn        int64   `json:"expires_in"`
	TokenType        string  `json:"token_type"`
	RefreshToken     *string `json:"refresh_token"`
	Error            *string `json:"error"`
	ErrorDescription *string `json:"error_description"`
}

type userInfoResponse struct {
	ID         *string `json:"id"`
	Email      string  `json:"email"`
	Name       *string `json:"name"`
	GivenName  *string `json:"given_name"`
	FamilyName *string `json:"family_name"`
}

func randomOAuthToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func findOAuthPort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

func StartOAuthFlow() (*OAuthStartResponse, error) {
	antigravityOAuthMu.Lock()
	defer antigravityOAuthMu.Unlock()

	port, err := findOAuthPort()
	if err != nil {
		return nil, fmt.Errorf("分配 Antigravity OAuth 回调端口失败: %w", err)
	}

	loginID := randomOAuthToken()
	stateToken := randomOAuthToken()
	callbackURL := fmt.Sprintf("http://localhost:%d%s", port, antigravityOAuthCallbackPath)
	authURL := buildAntigravityAuthURL(callbackURL, stateToken)

	state := &PendingOAuthState{
		LoginID:        loginID,
		ExpiresAt:      time.Now().Unix() + antigravityOAuthTimeoutSec,
		AuthURL:        authURL,
		CallbackURL:    callbackURL,
		CallbackPort:   port,
		StateToken:     stateToken,
		CallbackResult: make(chan *OAuthCallbackResult, 1),
	}
	antigravityOAuthState = state
	go startAntigravityCallbackServer(state)

	return &OAuthStartResponse{
		LoginID:     loginID,
		AuthURL:     authURL,
		CallbackURL: callbackURL,
		ExpiresIn:   antigravityOAuthTimeoutSec,
	}, nil
}

func CompleteOAuthFlow(loginID string, timeoutSec int) (*OAuthCompletePayload, error) {
	antigravityOAuthMu.Lock()
	state := antigravityOAuthState
	antigravityOAuthMu.Unlock()

	if state == nil || state.LoginID != loginID {
		return nil, fmt.Errorf("Antigravity OAuth 登录流程不存在，请重新发起")
	}
	if timeoutSec <= 0 {
		timeoutSec = antigravityOAuthTimeoutSec
	}

	select {
	case result := <-state.CallbackResult:
		clearAntigravityOAuthState(loginID)
		if result == nil {
			return nil, fmt.Errorf("Antigravity OAuth 登录已取消")
		}
		if result.Error != "" {
			return nil, fmt.Errorf("Antigravity OAuth 回调失败: %s", result.Error)
		}
		if strings.TrimSpace(result.Code) == "" {
			return nil, fmt.Errorf("Antigravity OAuth 回调缺少 code")
		}

		token, err := exchangeAntigravityCode(result.Code, state.CallbackURL)
		if err != nil {
			return nil, err
		}
		if token.RefreshToken == nil || strings.TrimSpace(*token.RefreshToken) == "" {
			return nil, fmt.Errorf("未获取到 refresh_token，请到 https://myaccount.google.com/permissions 撤销旧授权后重新授权")
		}
		userInfo, err := fetchAntigravityUserInfo(token.AccessToken)
		if err != nil {
			return nil, err
		}
		name := userInfo.displayName()
		expiry := time.Now().Unix() + token.ExpiresIn
		payload := &OAuthCompletePayload{
			Email:           userInfo.Email,
			AccessToken:     token.AccessToken,
			RefreshToken:    strings.TrimSpace(*token.RefreshToken),
			ExpiresIn:       token.ExpiresIn,
			ExpiryTimestamp: expiry,
			TokenType:       firstOAuthText(token.TokenType, "Bearer"),
			Name:            name,
		}
		if userInfo.ID != nil && strings.TrimSpace(*userInfo.ID) != "" {
			id := strings.TrimSpace(*userInfo.ID)
			payload.AuthID = id
			payload.SessionID = &id
		}
		return payload, nil

	case <-time.After(time.Duration(timeoutSec) * time.Second):
		clearAntigravityOAuthState(loginID)
		return nil, fmt.Errorf("Antigravity OAuth 等待回调超时")
	}
}

func SubmitOAuthCallbackURL(loginID, callbackURL string) error {
	antigravityOAuthMu.Lock()
	state := antigravityOAuthState
	antigravityOAuthMu.Unlock()
	if state == nil || state.LoginID != loginID {
		return fmt.Errorf("Antigravity OAuth 登录流程不存在，请重新发起")
	}
	parsed, err := parseAntigravityCallbackURL(callbackURL, state)
	if err != nil {
		return err
	}
	if parsed.Path != antigravityOAuthCallbackPath {
		return fmt.Errorf("回调链接路径无效，必须为 %s", antigravityOAuthCallbackPath)
	}
	result := callbackResultFromURL(parsed, state.StateToken)
	sendAntigravityOAuthResult(state, result)
	return nil
}

func CancelOAuthFlow(loginID string) error {
	antigravityOAuthMu.Lock()
	state := antigravityOAuthState
	if state != nil && (loginID == "" || state.LoginID == loginID) {
		antigravityOAuthState = nil
	}
	antigravityOAuthMu.Unlock()
	if state != nil && (loginID == "" || state.LoginID == loginID) {
		sendAntigravityOAuthResult(state, &OAuthCallbackResult{Error: "cancelled"})
	}
	return nil
}

func buildAntigravityAuthURL(callbackURL, stateToken string) string {
	u, _ := url.Parse(antigravityOAuthAuthURL)
	q := u.Query()
	q.Set("client_id", antigravityOAuthClientID)
	q.Set("redirect_uri", callbackURL)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(antigravityOAuthScopes, " "))
	q.Set("access_type", "offline")
	q.Set("prompt", "consent")
	q.Set("state", stateToken)
	u.RawQuery = q.Encode()
	return u.String()
}

func startAntigravityCallbackServer(state *PendingOAuthState) {
	mux := http.NewServeMux()
	var server *http.Server
	mux.HandleFunc(antigravityOAuthCallbackPath, func(w http.ResponseWriter, r *http.Request) {
		result := callbackResultFromURL(r.URL, state.StateToken)
		sendAntigravityOAuthResult(state, result)
		if result.Error != "" {
			http.Error(w, result.Error, http.StatusBadRequest)
		} else {
			_, _ = w.Write([]byte("Antigravity OAuth 授权成功，可以关闭此窗口返回 EasyLLM。"))
		}
		if server != nil {
			go server.Close()
		}
	})
	server = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", state.CallbackPort),
		Handler: mux,
	}
	go func() {
		_ = server.ListenAndServe()
	}()
	time.AfterFunc(time.Duration(antigravityOAuthTimeoutSec)*time.Second, func() {
		_ = server.Close()
	})
}

func callbackResultFromURL(u *url.URL, expectedState string) *OAuthCallbackResult {
	query := u.Query()
	if errCode := query.Get("error"); errCode != "" {
		errDesc := query.Get("error_description")
		if errDesc != "" {
			errCode += ": " + errDesc
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

func parseAntigravityCallbackURL(raw string, state *PendingOAuthState) (*url.URL, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return nil, fmt.Errorf("回调链接不能为空")
	}
	if strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://") {
		return url.Parse(text)
	}
	base := fmt.Sprintf("http://localhost:%d", state.CallbackPort)
	if strings.HasPrefix(text, "/") {
		return url.Parse(base + text)
	}
	return url.Parse(base + antigravityOAuthCallbackPath + "?" + strings.TrimPrefix(text, "?"))
}

func sendAntigravityOAuthResult(state *PendingOAuthState, result *OAuthCallbackResult) {
	select {
	case state.CallbackResult <- result:
	default:
	}
}

func clearAntigravityOAuthState(loginID string) {
	antigravityOAuthMu.Lock()
	defer antigravityOAuthMu.Unlock()
	if antigravityOAuthState != nil && antigravityOAuthState.LoginID == loginID {
		antigravityOAuthState = nil
	}
}

func exchangeAntigravityCode(code, redirectURI string) (*tokenResponse, error) {
	clientSecret := strings.TrimSpace(os.Getenv("ANTIGRAVITY_OAUTH_CLIENT_SECRET"))
	if clientSecret == "" {
		return nil, fmt.Errorf("ANTIGRAVITY_OAUTH_CLIENT_SECRET 未配置，无法完成 Antigravity OAuth")
	}

	form := url.Values{}
	form.Set("client_id", antigravityOAuthClientID)
	form.Set("client_secret", clientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("grant_type", "authorization_code")

	req, err := http.NewRequest(http.MethodPost, antigravityOAuthTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("Google OAuth token 交换请求失败: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Google OAuth token 交换失败: HTTP %d", resp.StatusCode)
	}
	var token tokenResponse
	if err := json.Unmarshal(raw, &token); err != nil {
		return nil, fmt.Errorf("Google OAuth token 响应解析失败: %w", err)
	}
	if token.AccessToken == "" {
		errText := "missing access_token"
		if token.Error != nil {
			errText = *token.Error
		}
		if token.ErrorDescription != nil {
			errText += ": " + *token.ErrorDescription
		}
		return nil, fmt.Errorf("Google OAuth token 交换失败: %s", errText)
	}
	return &token, nil
}

func fetchAntigravityUserInfo(accessToken string) (*userInfoResponse, error) {
	req, err := http.NewRequest(http.MethodGet, antigravityUserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("Google 用户信息请求失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Google 用户信息请求失败: HTTP %d", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	var info userInfoResponse
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil, fmt.Errorf("Google 用户信息解析失败: %w", err)
	}
	if strings.TrimSpace(info.Email) == "" {
		return nil, fmt.Errorf("Google 用户信息缺少 email")
	}
	return &info, nil
}

func (u userInfoResponse) displayName() *string {
	for _, text := range []string{
		ptrString(u.Name),
		strings.TrimSpace(strings.Join([]string{ptrString(u.GivenName), ptrString(u.FamilyName)}, " ")),
		ptrString(u.GivenName),
		ptrString(u.FamilyName),
	} {
		if strings.TrimSpace(text) != "" {
			value := strings.TrimSpace(text)
			return &value
		}
	}
	return nil
}

func ptrString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func firstOAuthText(values ...string) string {
	for _, value := range values {
		if text := strings.TrimSpace(value); text != "" {
			return text
		}
	}
	return ""
}
