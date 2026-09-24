package cursor

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"easyllm/internal/models"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

const (
	CursorUsageSummaryURL = "https://cursor.com/api/usage-summary"
	CursorStripeURL       = "https://cursor.com/api/auth/stripe"
	CursorUsageURL        = "https://cursor.com/api/usage"
	CursorUserAgent       = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
)

// ExtractUserIdFromJWT extracts the userId from the JWT payload's "sub" field
func ExtractUserIdFromJWT(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return ""
	}
	payloadSegment := parts[1]
	// Handle URL encoding padding
	if l := len(payloadSegment) % 4; l > 0 {
		payloadSegment += strings.Repeat("=", 4-l)
	}
	data, err := base64.URLEncoding.DecodeString(payloadSegment)
	if err != nil {
		data, err = base64.StdEncoding.DecodeString(payloadSegment)
		if err != nil {
			return ""
		}
	}

	var payload struct {
		Sub string `json:"sub"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return ""
	}

	if payload.Sub == "" {
		return ""
	}

	subParts := strings.Split(payload.Sub, "|")
	return subParts[len(subParts)-1]
}

// BuildAuthCookie formats the WorkosCursorSessionToken cookie header
func BuildAuthCookie(rawToken string) string {
	token := strings.TrimSpace(rawToken)
	if strings.HasPrefix(token, "WorkosCursorSessionToken=") {
		token = strings.TrimPrefix(token, "WorkosCursorSessionToken=")
	}

	// If already in userId::token format
	if strings.Contains(token, "::") {
		return "WorkosCursorSessionToken=" + token
	}

	// If it's a raw JWT token, extract userId
	if strings.HasPrefix(token, "eyJ") {
		userId := ExtractUserIdFromJWT(token)
		if userId != "" {
			return fmt.Sprintf("WorkosCursorSessionToken=%s::%s", userId, token)
		}
	}

	return "WorkosCursorSessionToken=" + token
}

type cursorUsageSummaryResponse struct {
	BillingCycleStart string `json:"billingCycleStart"`
	BillingCycleEnd   string `json:"billingCycleEnd"`
	MembershipType    string `json:"membershipType"`
	LimitType         string `json:"limitType"`
	IsUnlimited       bool   `json:"isUnlimited"`
	IndividualUsage   struct {
		Plan struct {
			Enabled          bool `json:"enabled"`
			Used             int  `json:"used"`
			Limit            int  `json:"limit"`
			Remaining        int  `json:"remaining"`
			AutoPercentUsed  int  `json:"autoPercentUsed"`
			ApiPercentUsed   int  `json:"apiPercentUsed"`
			TotalPercentUsed int  `json:"totalPercentUsed"`
		} `json:"plan"`
		OnDemand struct {
			Enabled   bool `json:"enabled"`
			Used      int  `json:"used"`
			Limit     *int `json:"limit"`
			Remaining *int `json:"remaining"`
		} `json:"onDemand"`
	} `json:"individualUsage"`
}

type cursorStripeResponse struct {
	MembershipType     string `json:"membershipType"`
	SubscriptionStatus string `json:"subscriptionStatus"`
}

// FetchAccountQuota queries Cursor API and updates account metrics
func FetchAccountQuota(account *models.CursorAccount) error {
	client := &http.Client{Timeout: 15 * time.Second}
	cookie := BuildAuthCookie(account.SessionToken)

	// 1. Fetch usage-summary
	req, err := http.NewRequest(http.MethodGet, CursorUsageSummaryURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Cookie", cookie)
	req.Header.Set("User-Agent", CursorUserAgent)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		account.Status = "error"
		msg := fmt.Sprintf("连接 Cursor API 失败: %v", err)
		account.StatusMessage = &msg
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		account.Status = "expired"
		msg := "Cursor Token 已过期或未授权 (401)"
		account.StatusMessage = &msg
		return fmt.Errorf("cursor 凭据无效或已失效 (401)")
	}
	if resp.StatusCode == http.StatusForbidden {
		account.Status = "forbidden"
		msg := "Cursor API 拒绝访问 (403)"
		account.StatusMessage = &msg
		return fmt.Errorf("cursor 访问受限 (403)")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		account.Status = "error"
		msg := fmt.Sprintf("Cursor API 返回错误 (%d): %s", resp.StatusCode, string(body))
		account.StatusMessage = &msg
		return fmt.Errorf("%s", msg)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var summary cursorUsageSummaryResponse
	if err := json.Unmarshal(bodyBytes, &summary); err != nil {
		return fmt.Errorf("解析使用概览失败: %w", err)
	}

	account.Status = "active"
	account.StatusMessage = nil
	if summary.MembershipType != "" {
		account.MembershipType = summary.MembershipType
	}
	account.BillingCycleStart = summary.BillingCycleStart
	account.BillingCycleEnd = summary.BillingCycleEnd

	account.PlanEnabled = summary.IndividualUsage.Plan.Enabled
	account.PlanUsed = summary.IndividualUsage.Plan.Used
	account.PlanLimit = summary.IndividualUsage.Plan.Limit
	account.PlanRemaining = summary.IndividualUsage.Plan.Remaining
	if account.PlanLimit > 0 {
		account.PlanPercentage = (account.PlanRemaining * 100) / account.PlanLimit
		if account.PlanPercentage < 0 {
			account.PlanPercentage = 0
		} else if account.PlanPercentage > 100 {
			account.PlanPercentage = 100
		}
	} else if account.PlanUsed > 0 && account.PlanRemaining == 0 {
		account.PlanPercentage = 0
	} else {
		account.PlanPercentage = 100
	}

	account.OnDemandEnabled = summary.IndividualUsage.OnDemand.Enabled
	account.OnDemandUsedCents = summary.IndividualUsage.OnDemand.Used
	account.OnDemandLimitCents = summary.IndividualUsage.OnDemand.Limit

	account.AutoPercentUsed = summary.IndividualUsage.Plan.AutoPercentUsed
	account.ApiPercentUsed = summary.IndividualUsage.Plan.ApiPercentUsed
	account.TotalPercentUsed = summary.IndividualUsage.Plan.TotalPercentUsed

	rawStr := string(bodyBytes)
	account.RawUsageJSON = &rawStr
	now := time.Now()
	account.LastRefreshAt = &now

	// 2. Fetch Stripe / Subscription status
	stripeReq, err := http.NewRequest(http.MethodGet, CursorStripeURL, nil)
	if err == nil {
		stripeReq.Header.Set("Cookie", cookie)
		stripeReq.Header.Set("User-Agent", CursorUserAgent)
		if stripeResp, err := client.Do(stripeReq); err == nil {
			defer stripeResp.Body.Close()
			if stripeResp.StatusCode == http.StatusOK {
				var stripe cursorStripeResponse
				if err := json.NewDecoder(stripeResp.Body).Decode(&stripe); err == nil {
					if stripe.MembershipType != "" {
						account.MembershipType = stripe.MembershipType
					}
					if stripe.SubscriptionStatus != "" {
						account.SubscriptionStatus = stripe.SubscriptionStatus
					}
				}
			}
		}
	}

	return nil
}

// GetLocalStateDbPath returns the local path to Cursor's state.vscdb
func GetLocalStateDbPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "Cursor", "User", "globalStorage", "state.vscdb"), nil
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(homeDir, "AppData", "Roaming")
		}
		return filepath.Join(appData, "Cursor", "User", "globalStorage", "state.vscdb"), nil
	case "linux":
		return filepath.Join(homeDir, ".config", "Cursor", "User", "globalStorage", "state.vscdb"), nil
	default:
		return "", fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

// DetectLocalCursorAccount inspects the local Cursor database and returns a populated account
func DetectLocalCursorAccount() (*models.CursorAccount, error) {
	dbPath, err := GetLocalStateDbPath()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("未检测到本地 Cursor 存储文件: %s (请确认已安装并登录 Cursor)", dbPath)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开 Cursor 数据库失败: %w", err)
	}
	defer db.Close()

	var accessToken, refreshToken, cachedEmail, membershipType, profileJSON string
	_ = db.QueryRow("SELECT value FROM ItemTable WHERE key = 'cursorAuth/accessToken';").Scan(&accessToken)
	_ = db.QueryRow("SELECT value FROM ItemTable WHERE key = 'cursorAuth/refreshToken';").Scan(&refreshToken)
	_ = db.QueryRow("SELECT value FROM ItemTable WHERE key = 'cursorAuth/cachedEmail';").Scan(&cachedEmail)
	_ = db.QueryRow("SELECT value FROM ItemTable WHERE key = 'cursorAuth/stripeMembershipType';").Scan(&membershipType)
	_ = db.QueryRow("SELECT value FROM ItemTable WHERE key = 'cursorAuth/cachedScopedProfile';").Scan(&profileJSON)

	if accessToken == "" {
		return nil, fmt.Errorf("本地 Cursor 数据库中未找到登录凭据 (accessToken 为空，请先在 Cursor IDE 中登录)")
	}

	userId := ExtractUserIdFromJWT(accessToken)
	sessionToken := accessToken
	if userId != "" {
		sessionToken = fmt.Sprintf("%s::%s", userId, accessToken)
	}

	displayName := ""
	if profileJSON != "" {
		var prof struct {
			DisplayName string `json:"displayName"`
		}
		if err := json.Unmarshal([]byte(profileJSON), &prof); err == nil && prof.DisplayName != "" {
			displayName = prof.DisplayName
		}
	}

	if cachedEmail == "" {
		cachedEmail = "cursor-local-user@example.com"
	}
	if membershipType == "" {
		membershipType = "pro"
	}

	acc := &models.CursorAccount{
		ID:                 uuid.New().String(),
		Email:              cachedEmail,
		SessionToken:       sessionToken,
		AccessToken:        accessToken,
		RefreshToken:       refreshToken,
		MembershipType:     membershipType,
		SubscriptionStatus: "active",
		Status:             "active",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	if displayName != "" {
		acc.DisplayName = &displayName
	}

	// Fetch full usage quota
	_ = FetchAccountQuota(acc)
	return acc, nil
}

// SyncToLocalCursor writes the account credentials to local Cursor's state.vscdb
func SyncToLocalCursor(account *models.CursorAccount) error {
	dbPath, err := GetLocalStateDbPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("本地 Cursor 数据库不存在: %s", dbPath)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("打开本地 Cursor 数据库失败: %w", err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	upsertStmt := "INSERT INTO ItemTable(key, value) VALUES(?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value;"

	if account.AccessToken != "" {
		if _, err := tx.Exec(upsertStmt, "cursorAuth/accessToken", account.AccessToken); err != nil {
			return err
		}
	}
	if account.RefreshToken != "" {
		if _, err := tx.Exec(upsertStmt, "cursorAuth/refreshToken", account.RefreshToken); err != nil {
			return err
		}
	}
	if account.Email != "" {
		if _, err := tx.Exec(upsertStmt, "cursorAuth/cachedEmail", account.Email); err != nil {
			return err
		}
	}
	if account.MembershipType != "" {
		if _, err := tx.Exec(upsertStmt, "cursorAuth/stripeMembershipType", account.MembershipType); err != nil {
			return err
		}
	}
	if account.SubscriptionStatus != "" {
		if _, err := tx.Exec(upsertStmt, "cursorAuth/stripeSubscriptionStatus", account.SubscriptionStatus); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Wakeup tests connectivity to Cursor API with account credentials
func Wakeup(account *models.CursorAccount) (*models.CursorWakeupResponse, error) {
	start := time.Now()
	err := FetchAccountQuota(account)
	duration := time.Since(start).Milliseconds()

	if err != nil {
		return &models.CursorWakeupResponse{
			Success:            false,
			Email:              account.Email,
			MembershipType:     account.MembershipType,
			SubscriptionStatus: account.SubscriptionStatus,
			DurationMS:         duration,
			Message:            fmt.Sprintf("唤醒失败: %v", err),
		}, err
	}

	return &models.CursorWakeupResponse{
		Success:            true,
		Email:              account.Email,
		MembershipType:     account.MembershipType,
		SubscriptionStatus: account.SubscriptionStatus,
		PlanRemaining:      account.PlanRemaining,
		PlanLimit:          account.PlanLimit,
		DurationMS:         duration,
		Message:            "唤醒连通测试成功！API 响应正常。",
	}, nil
}
