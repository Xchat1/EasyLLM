// importsub2api: 将 sub2api 导出的账户 JSON 导入 EasyLLM 数据库。
//
// 用法:
//   go run ./cmd/importsub2api <sub2api-export.json>
//
// 会自动使用与 EasyLLM 主程序相同的数据库路径（由环境变量 / 默认配置决定）。
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"easyllm/config"
	"easyllm/internal/models"
	"easyllm/internal/storage"

	"github.com/google/uuid"
)

// sub2api 导出格式
type sub2apiExport struct {
	ExportedAt string          `json:"exported_at"`
	Proxies    json.RawMessage `json:"proxies"`
	Accounts   []sub2apiAccount `json:"accounts"`
}

type sub2apiAccount struct {
	Name               string                 `json:"name"`
	Notes              string                 `json:"notes"`
	Platform           string                 `json:"platform"`
	Type               string                 `json:"type"`
	Credentials        sub2apiCredentials     `json:"credentials"`
	Extra              map[string]interface{} `json:"extra"`
	Concurrency        int                    `json:"concurrency"`
	Priority           int                    `json:"priority"`
	RateMultiplier     float64                `json:"rate_multiplier"`
	ExpiresAt          int64                  `json:"expires_at"`
	AutoPauseOnExpired bool                   `json:"auto_pause_on_expired"`
}

type sub2apiCredentials struct {
	AccessToken      string `json:"access_token"`
	ChatGPTAccountID string `json:"chatgpt_account_id"`
	ChatGPTUserID    string `json:"chatgpt_user_id"`
	ClientID         string `json:"client_id"`
	Email            string `json:"email"`
	ExpiresAt        string `json:"expires_at"`
	ExpiresIn        int    `json:"expires_in"`
	PlanType         string `json:"plan_type"`
	SessionToken     string `json:"session_token"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "用法: go run ./cmd/importsub2api <sub2api-export.json>")
		os.Exit(1)
	}
	filePath := os.Args[1]

	// 1. 读取 JSON 文件
	raw, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("读取文件失败: %v", err)
	}

	var export sub2apiExport
	if err := json.Unmarshal(raw, &export); err != nil {
		log.Fatalf("解析 JSON 失败: %v", err)
	}
	if len(export.Accounts) == 0 {
		log.Fatalf("文件中没有账户数据 (accounts 数组为空)")
	}

	// 2. 初始化数据库（与主程序使用相同的路径）
	cfg := config.Load()
	if err := storage.InitDB(cfg); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer storage.CloseDB()

	store := storage.NewOpenAIStorage(storage.GetDB())

	// 3. 获取现有账户用于去重
	existing, _ := store.List()
	existingByEmail := make(map[string]int, len(existing))
	existingByToken := make(map[string]int, len(existing))
	for i, a := range existing {
		if a.Email != "" {
			existingByEmail[strings.ToLower(a.Email)] = i
		}
		if a.AccessToken != nil && *a.AccessToken != "" {
			existingByToken[*a.AccessToken] = i
		}
	}

	// 4. 逐个导入
	now := time.Now()
	success, skipped, failed := 0, 0, 0

	for _, sa := range export.Accounts {
		email := sa.Credentials.Email
		if email == "" {
			email = sa.Name
		}
		if email == "" {
			email = fmt.Sprintf("unknown-%d", failed)
		}

		if sa.Platform != "" && !strings.EqualFold(sa.Platform, "openai") {
			log.Printf("[跳过] %s: platform=%s 非 openai", email, sa.Platform)
			skipped++
			continue
		}

		accessToken := sa.Credentials.AccessToken
		if accessToken == "" {
			log.Printf("[失败] %s: 缺少 access_token", email)
			failed++
			continue
		}

		// 去重：按 email 或 access_token
		if _, ok := existingByEmail[strings.ToLower(email)]; ok {
			log.Printf("[跳过] %s: 邮箱已存在", email)
			skipped++
			continue
		}
		if _, ok := existingByToken[accessToken]; ok {
			log.Printf("[跳过] %s: access_token 已存在", email)
			skipped++
			continue
		}

		// 解析过期时间
		var expiresAt *time.Time
		if sa.Credentials.ExpiresAt != "" {
			if t, err := time.Parse(time.RFC3339, sa.Credentials.ExpiresAt); err == nil {
				expiresAt = &t
			}
		}

		account := &models.OpenAIAccount{
			ID:           uuid.New().String(),
			Email:        email,
			AccountType:  models.OpenAIAccountTypeOAuth,
			Status:       "active",
			AccessToken:  sPtr(accessToken),
			ExpiresAt:    expiresAt,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if sa.Credentials.ChatGPTAccountID != "" {
			account.ChatGPTAccountID = sPtr(sa.Credentials.ChatGPTAccountID)
		}
		if sa.Credentials.ChatGPTUserID != "" {
			account.ChatGPTUserID = sPtr(sa.Credentials.ChatGPTUserID)
		}
		if sa.Credentials.PlanType != "" {
			account.Plan = sPtr(sa.Credentials.PlanType)
		}

		if err := store.Save(account); err != nil {
			log.Printf("[失败] %s: 保存失败: %v", email, err)
			failed++
			continue
		}

		// 更新去重索引
		existingByEmail[strings.ToLower(email)] = len(existing)
		existingByToken[accessToken] = len(existing)

		log.Printf("[成功] %s (plan=%s)", email, sa.Credentials.PlanType)
		success++
	}

	fmt.Println()
	fmt.Printf("导入完成: 成功 %d, 跳过 %d, 失败 %d, 共 %d 个账户\n",
		success, skipped, failed, len(export.Accounts))
}

func sPtr(s string) *string { return &s }
