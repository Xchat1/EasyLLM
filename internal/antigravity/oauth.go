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
	DefaultClientID     = "1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com"
	AuthURL             = "https://accounts.google.com/o/oauth2/v2/auth"
	TokenURL            = "https://oauth2.googleapis.com/token"
	UserInfoURL         = "https://www.googleapis.com/oauth2/v2/userinfo"
	OAuthCallbackPath   = "/oauth-callback"
	OAuthDefaultTimeout = 600
)

var OAuthScopes = []string{
	"openid",
	"https://www.googleapis.com/auth/cloud-platform",
	"https://www.googleapis.com/auth/userinfo.email",
	"https://www.googleapis.com/auth/userinfo.profile",
	"https://www.googleapis.com/auth/cclog",
	"https://www.googleapis.com/auth/experimentsandconfigs",
	"https://www.googleapis.com/auth/aicode",
}

var (
	oauthMu    sync.Mutex
	oauthState *PendingOAuthState
)

type TokenResponse struct {
	AccessToken  string  `json:"access_token"`
	ExpiresIn    int64   `json:"expires_in"`
	TokenType    string  `json:"token_type"`
	RefreshToken *string `json:"refresh_token,omitempty"`
	IDToken      *string `json:"id_token,omitempty"`
	Error        *string `json:"error,omitempty"`
	ErrorDesc    *string `json:"error_description,omitempty"`
}

type UserInfo struct {
	ID         *string `json:"id,omitempty"`
	Email      string  `json:"email"`
	Name       *string `json:"name,omitempty"`
	GivenName  *string `json:"given_name,omitempty"`
	FamilyName *string `json:"family_name,omitempty"`
	Picture    *string `json:"picture,omitempty"`
}

func (u *UserInfo) DisplayName() string {
	if u.Name != nil && strings.TrimSpace(*u.Name) != "" {
		return strings.TrimSpace(*u.Name)
	}
	given := ""
	if u.GivenName != nil {
		given = strings.TrimSpace(*u.GivenName)
	}
	family := ""
	if u.FamilyName != nil {
		family = strings.TrimSpace(*u.FamilyName)
	}
	combined := strings.TrimSpace(given + " " + family)
	if combined != "" {
		return combined
	}
	return u.Email
}

type OAuthCallbackResult struct {
	Code  string
	Error string
}

type PendingOAuthState struct {
	LoginID        string                    `json:"login_id"`
	AuthURL        string                    `json:"auth_url"`
	CallbackURL    string                    `json:"callback_url"`
	CallbackPort   int                       `json:"callback_port"`
	StateToken     string                    `json:"state_token"`
	CallbackResult chan *OAuthCallbackResult `json:"-"`
}

func getClientCredentials() (string, string) {
	clientID := strings.TrimSpace(os.Getenv("ANTIGRAVITY_OAUTH_CLIENT_ID"))
	if clientID == "" {
		clientID = DefaultClientID
	}
	clientSecret := strings.TrimSpace(os.Getenv("ANTIGRAVITY_OAUTH_CLIENT_SECRET"))
	return clientID, clientSecret
}

// StartOAuthFlow starts a local OAuth flow and returns the auth URL and login state.
func StartOAuthFlow() (*PendingOAuthState, error) {
	if _, secret := getClientCredentials(); secret == "" {
		return nil, fmt.Errorf("ANTIGRAVITY_OAUTH_CLIENT_SECRET is required")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("无法监听本地回调端口: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	b := make([]byte, 16)
	_, _ = rand.Read(b)
	loginID := base64.RawURLEncoding.EncodeToString(b)

	s := make([]byte, 24)
	_, _ = rand.Read(s)
	stateToken := base64.RawURLEncoding.EncodeToString(s)

	callbackURL := fmt.Sprintf("http://localhost:%d%s", port, OAuthCallbackPath)
	authURL := BuildAuthURL(callbackURL, stateToken)

	state := &PendingOAuthState{
		LoginID:        loginID,
		AuthURL:        authURL,
		CallbackURL:    callbackURL,
		CallbackPort:   port,
		StateToken:     stateToken,
		CallbackResult: make(chan *OAuthCallbackResult, 1),
	}

	oauthMu.Lock()
	oauthState = state
	oauthMu.Unlock()

	startCallbackServer(listener, state)
	return state, nil
}

func BuildAuthURL(callbackURL, stateToken string) string {
	clientID, _ := getClientCredentials()
	u, _ := url.Parse(AuthURL)
	q := u.Query()
	q.Set("client_id", clientID)
	q.Set("redirect_uri", callbackURL)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(OAuthScopes, " "))
	q.Set("access_type", "offline")
	q.Set("prompt", "consent")
	q.Set("state", stateToken)
	u.RawQuery = q.Encode()
	return u.String()
}

func startCallbackServer(listener net.Listener, state *PendingOAuthState) {
	mux := http.NewServeMux()
	var server *http.Server
	mux.HandleFunc(OAuthCallbackPath, func(w http.ResponseWriter, r *http.Request) {
		result := parseCallbackURL(r.URL, state.StateToken)
		select {
		case state.CallbackResult <- result:
		default:
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if result.Error != "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprintf(w, "<html><body style='font-family:sans-serif;padding:40px;background:#18181b;color:#f43f5e'><h2>授权失败: %s</h2><p>可关闭此页面返回 EasyLLM。</p></body></html>", result.Error)
		} else {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<html><body style='font-family:sans-serif;padding:40px;background:#18181b;color:#10b981'><h2>Antigravity OAuth 授权成功！</h2><p>可关闭此页面，返回 EasyLLM 继续操作。</p></body></html>"))
		}
		if server != nil {
			go func() {
				time.Sleep(500 * time.Millisecond)
				_ = server.Close()
			}()
		}
	})

	server = &http.Server{Handler: mux}
	go func() {
		_ = server.Serve(listener)
	}()
	time.AfterFunc(time.Duration(OAuthDefaultTimeout)*time.Second, func() {
		_ = server.Close()
	})
}

func parseCallbackURL(u *url.URL, expectedState string) *OAuthCallbackResult {
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

// CompleteOAuthFlow waits for callback code, exchanges it for tokens, and fetches userinfo.
func CompleteOAuthFlow(loginID string, timeoutSec int) (*TokenResponse, *UserInfo, error) {
	oauthMu.Lock()
	state := oauthState
	oauthMu.Unlock()

	if state == nil || state.LoginID != loginID {
		return nil, nil, fmt.Errorf("OAuth 流程不存在或已失效")
	}

	if timeoutSec <= 0 {
		timeoutSec = OAuthDefaultTimeout
	}

	select {
	case res := <-state.CallbackResult:
		clearOAuthState(loginID)
		if res.Error != "" {
			return nil, nil, fmt.Errorf("授权失败: %s", res.Error)
		}
		token, err := ExchangeCode(res.Code, state.CallbackURL)
		if err != nil {
			return nil, nil, err
		}
		userInfo, err := FetchUserInfo(token.AccessToken)
		if err != nil {
			return nil, nil, err
		}
		return token, userInfo, nil

	case <-time.After(time.Duration(timeoutSec) * time.Second):
		clearOAuthState(loginID)
		return nil, nil, fmt.Errorf("Antigravity OAuth 等待超时")
	}
}

// SubmitManualCallbackURL allows user to paste callback redirect URL manually.
func SubmitManualCallbackURL(loginID, callbackRaw string) error {
	oauthMu.Lock()
	state := oauthState
	oauthMu.Unlock()

	if state == nil || state.LoginID != loginID {
		return fmt.Errorf("OAuth 流程不存在或已失效")
	}

	trimmed := strings.TrimSpace(callbackRaw)
	if trimmed == "" {
		return fmt.Errorf("回调链接不能为空")
	}

	var parsed *url.URL
	var err error
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		parsed, err = url.Parse(trimmed)
	} else {
		base := fmt.Sprintf("http://localhost:%d%s", state.CallbackPort, OAuthCallbackPath)
		if strings.HasPrefix(trimmed, "/") {
			parsed, err = url.Parse(fmt.Sprintf("http://localhost:%d%s", state.CallbackPort, trimmed))
		} else {
			parsed, err = url.Parse(base + "?" + strings.TrimPrefix(trimmed, "?"))
		}
	}
	if err != nil {
		return fmt.Errorf("解析回调链接失败: %w", err)
	}

	result := parseCallbackURL(parsed, state.StateToken)
	select {
	case state.CallbackResult <- result:
	default:
	}
	return nil
}

func CancelOAuthFlow(loginID string) error {
	oauthMu.Lock()
	state := oauthState
	if state != nil && (loginID == "" || state.LoginID == loginID) {
		oauthState = nil
	}
	oauthMu.Unlock()
	if state != nil && (loginID == "" || state.LoginID == loginID) {
		select {
		case state.CallbackResult <- &OAuthCallbackResult{Error: "cancelled"}:
		default:
		}
	}
	return nil
}

func clearOAuthState(loginID string) {
	oauthMu.Lock()
	defer oauthMu.Unlock()
	if oauthState != nil && oauthState.LoginID == loginID {
		oauthState = nil
	}
}

// ExchangeCode exchanges an authorization code for access and refresh tokens.
func ExchangeCode(code, redirectURI string) (*TokenResponse, error) {
	clientID, clientSecret := getClientCredentials()
	if clientSecret == "" {
		return nil, fmt.Errorf("ANTIGRAVITY_OAUTH_CLIENT_SECRET is required")
	}
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("grant_type", "authorization_code")

	req, err := http.NewRequest(http.MethodPost, TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Token 交换网络失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 Token 响应失败: %w", err)
	}

	var tokenRes TokenResponse
	if err := json.Unmarshal(body, &tokenRes); err != nil {
		return nil, fmt.Errorf("解析 Token 响应失败: %w (body: %s)", err, string(body))
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 || tokenRes.AccessToken == "" {
		errDesc := ""
		if tokenRes.Error != nil {
			errDesc = *tokenRes.Error
		}
		if tokenRes.ErrorDesc != nil {
			errDesc += ": " + *tokenRes.ErrorDesc
		}
		if errDesc == "" {
			errDesc = string(body)
		}
		return nil, fmt.Errorf("Token 交换未成功 (HTTP %d): %s", resp.StatusCode, errDesc)
	}

	return &tokenRes, nil
}

// RefreshAccessToken refreshes an access token using a refresh token.
func RefreshAccessToken(refreshToken string) (*TokenResponse, error) {
	clientID, clientSecret := getClientCredentials()
	if clientSecret == "" {
		return nil, fmt.Errorf("ANTIGRAVITY_OAUTH_CLIENT_SECRET is required")
	}
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("refresh_token", refreshToken)
	form.Set("grant_type", "refresh_token")

	req, err := http.NewRequest(http.MethodPost, TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("刷新 Token 网络失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取刷新响应失败: %w", err)
	}

	var tokenRes TokenResponse
	if err := json.Unmarshal(body, &tokenRes); err != nil {
		return nil, fmt.Errorf("解析刷新响应失败: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 || tokenRes.AccessToken == "" {
		errDesc := ""
		if tokenRes.Error != nil {
			errDesc = *tokenRes.Error
		}
		if tokenRes.ErrorDesc != nil {
			errDesc += ": " + *tokenRes.ErrorDesc
		}
		if errDesc == "" {
			errDesc = string(body)
		}
		return nil, fmt.Errorf("刷新 Token 失败 (HTTP %d): %s", resp.StatusCode, errDesc)
	}

	return &tokenRes, nil
}

// FetchUserInfo fetches user profile information from Google.
func FetchUserInfo(accessToken string) (*UserInfo, error) {
	req, err := http.NewRequest(http.MethodGet, UserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息网络失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取用户信息失败: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("获取用户信息失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var userInfo UserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("解析用户信息失败: %w", err)
	}

	if strings.TrimSpace(userInfo.Email) == "" {
		return nil, fmt.Errorf("用户信息缺少 email")
	}

	return &userInfo, nil
}
