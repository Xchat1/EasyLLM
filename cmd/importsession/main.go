package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"easyllm/config"
	"easyllm/internal/handlers"
	openaiplatform "easyllm/internal/openai"
	"easyllm/internal/models"
	"easyllm/internal/proxy"
	"easyllm/internal/storage"

	"github.com/google/uuid"
)

type sessionExport struct {
	AccessToken string `json:"accessToken"`
	Expires     string `json:"expires"`
	User        struct {
		Email string `json:"email"`
	} `json:"user"`
	Account struct {
		ID       string `json:"id"`
		PlanType string `json:"planType"`
	} `json:"account"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "用法: go run ./cmd/importsession <session.json>")
		os.Exit(1)
	}

	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取文件失败: %v\n", err)
		os.Exit(1)
	}

	var session sessionExport
	if err := json.Unmarshal(raw, &session); err != nil {
		fmt.Fprintf(os.Stderr, "解析 JSON 失败: %v\n", err)
		os.Exit(1)
	}

	accessToken := strings.TrimSpace(session.AccessToken)
	email := strings.TrimSpace(session.User.Email)
	accountID := strings.TrimSpace(session.Account.ID)
	planType := strings.TrimSpace(session.Account.PlanType)
	if accessToken == "" || email == "" {
		fmt.Fprintln(os.Stderr, "缺少 accessToken 或 user.email")
		os.Exit(1)
	}

	cfg := config.Load()
	if err := storage.InitDB(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "初始化数据库失败: %v\n", err)
		os.Exit(1)
	}
	defer storage.CloseDB()

	db := storage.GetDB()
	openaiStore := storage.NewOpenAIStorage(db)
	codexStore := storage.NewCodexStorage(db)
	proxy.InitProxy(codexStore, openaiStore, "auto")
	handler := handlers.NewOpenAIHandler(openaiStore, codexStore)

	now := time.Now()
	var expiresAt *time.Time
	if session.Expires != "" {
		if t, err := time.Parse(time.RFC3339, session.Expires); err == nil {
			expiresAt = &t
		}
	}

	existingAccounts, _ := openaiStore.List()
	incoming := &models.OpenAIAccount{
		ID:           uuid.New().String(),
		Email:        email,
		AccountType:  models.OpenAIAccountTypeOAuth,
		AccessToken:  &accessToken,
		ExpiresAt:    expiresAt,
		Plan:         strPtr(planType),
		CreatedAt:    now,
		UpdatedAt:    now,
		ProxyEnabled: true,
	}
	if accountID != "" {
		incoming.ChatGPTAccountID = &accountID
	}

	account, err := handler.ImportOAuthSessionAccount(incoming, &existingAccounts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "导入失败: %v\n", err)
		os.Exit(1)
	}

	chatgptID := ""
	if account.ChatGPTAccountID != nil {
		chatgptID = *account.ChatGPTAccountID
	}
	quotaInfo, quotaErr := openaiplatform.FetchQuota(accessToken, chatgptID)
	if quotaErr != nil {
		fmt.Printf("账号已导入: %s (id=%s)\n", account.Email, account.ID)
		fmt.Printf("配额查询失败: %v\n", quotaErr)
		os.Exit(0)
	}

	http200 := http.StatusOK
	account.QuotaHTTPStatus = &http200
	account.QuotaVerified = true
	account.Quota5hUsedPercent = quotaInfo.Codex5hUsedPercent
	account.Quota5hResetSeconds = quotaInfo.Codex5hResetSeconds
	account.Quota5hWindowMinutes = quotaInfo.Codex5hWindowMinutes
	account.Quota7dUsedPercent = quotaInfo.Codex7dUsedPercent
	account.Quota7dResetSeconds = quotaInfo.Codex7dResetSeconds
	account.Quota7dWindowMinutes = quotaInfo.Codex7dWindowMinutes
	quotaNow := time.Now()
	account.QuotaUpdatedAt = &quotaNow
	if err := openaiStore.Save(account); err != nil {
		fmt.Fprintf(os.Stderr, "保存配额失败: %v\n", err)
		os.Exit(1)
	}

	if p := proxy.GetProxy(); p != nil {
		p.Refresh()
	}

	fmt.Printf("导入成功: %s\n", account.Email)
	fmt.Printf("账号 ID: %s\n", account.ID)
	fmt.Printf("计划: %s\n", planType)
	if quotaInfo.Codex5hUsedPercent != nil {
		fmt.Printf("5h 配额使用: %.0f%%\n", *quotaInfo.Codex5hUsedPercent)
	}
	if quotaInfo.Codex7dUsedPercent != nil {
		fmt.Printf("7d 配额使用: %.0f%%\n", *quotaInfo.Codex7dUsedPercent)
	}
	fmt.Println("已加入代理池与 Codex 本地接入")
}

func strPtr(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	v := strings.TrimSpace(s)
	return &v
}
