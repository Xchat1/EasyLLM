// importcodex: 将 codex_accounts.json 导入 EasyLLM 数据库的 codex_accounts 表。
//
// 用法:
//   go run ./cmd/importcodex <codex_accounts.json>
//
// 会自动使用与 EasyLLM 主程序相同的数据库路径。
// 对于有 access_token 的账号（从 open_ai_accounts 表按邮箱关联），会自动填充 token。
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"easyllm/config"
	"easyllm/internal/models"
	"easyllm/internal/storage"
)

type codexExport struct {
	Version  string          `json:"version"`
	Accounts []codexEntry    `json:"accounts"`
}

type codexEntry struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	PlanType  string `json:"plan_type"`
	CreatedAt int64  `json:"created_at"`
	LastUsed  int64  `json:"last_used"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "用法: go run ./cmd/importcodex <codex_accounts.json>")
		os.Exit(1)
	}
	filePath := os.Args[1]

	// 1. 读取 JSON 文件
	raw, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("读取文件失败: %v", err)
	}

	var export codexExport
	if err := json.Unmarshal(raw, &export); err != nil {
		log.Fatalf("解析 JSON 失败: %v", err)
	}
	if len(export.Accounts) == 0 {
		log.Fatalf("文件中没有账户数据 (accounts 数组为空)")
	}

	// 2. 初始化数据库
	cfg := config.Load()
	if err := storage.InitDB(cfg); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer storage.CloseDB()

	db := storage.GetDB()
	codexStore := storage.NewCodexStorage(db)

	// 3. 从 open_ai_accounts 表按邮箱建立 access_token 索引（用于关联填充）
	type oauthRow struct {
		Email       string
		AccessToken *string
	}
	var oauthRows []oauthRow
	db.Table("open_ai_accounts").
		Select("email, access_token").
		Where("account_type = ? AND access_token IS NOT NULL AND access_token != ''", "oauth").
		Scan(&oauthRows)

	tokenByEmail := make(map[string]string, len(oauthRows))
	for _, r := range oauthRows {
		if r.AccessToken != nil && *r.AccessToken != "" {
			tokenByEmail[toLower(r.Email)] = *r.AccessToken
		}
	}

	// 4. 获取 codex_accounts 表中已有的 ID 用于去重
	existing, _ := codexStore.LoadAllAccounts()
	existingIDs := make(map[string]bool, len(existing))
	for _, a := range existing {
		existingIDs[a.ID] = true
	}

	// 5. 逐个导入
	now := time.Now()
	success, skipped, withToken := 0, 0, 0

	for _, entry := range export.Accounts {
		if entry.ID == "" || entry.Email == "" {
			log.Printf("[跳过] 缺少 id 或 email")
			skipped++
			continue
		}

		if existingIDs[entry.ID] {
			log.Printf("[跳过] %s: ID %s 已存在", entry.Email, entry.ID)
			skipped++
			continue
		}

		// 解析时间戳
		createdAt := time.Unix(entry.CreatedAt, 0)
		if entry.CreatedAt == 0 {
			createdAt = now
		}
		var lastUsedAt *time.Time
		if entry.LastUsed > 0 {
			t := time.Unix(entry.LastUsed, 0)
			lastUsedAt = &t
		}

		// 关联 access_token
		accessToken := tokenByEmail[toLower(entry.Email)]

		account := &models.CodexAccount{
			ID:           entry.ID,
			AccountID:    "",
			Email:        entry.Email,
			AccessToken:  accessToken,
			Enabled:      true,
			RequestCount: 0,
			LastUsedAt:   lastUsedAt,
			CreatedAt:    createdAt,
			UpdatedAt:    now,
		}

		if err := codexStore.SaveAccount(account); err != nil {
			log.Printf("[失败] %s: 保存失败: %v", entry.Email, err)
			skipped++
			continue
		}

		existingIDs[entry.ID] = true
		if accessToken != "" {
			withToken++
			log.Printf("[成功] %s (有 token, plan=%s)", entry.Email, entry.PlanType)
		} else {
			log.Printf("[成功] %s (无 token, plan=%s)", entry.Email, entry.PlanType)
		}
		success++
	}

	fmt.Println()
	fmt.Printf("导入完成: 成功 %d (其中有 token %d), 跳过/失败 %d, 共 %d 个账户\n",
		success, withToken, skipped, len(export.Accounts))
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		b[i] = c
	}
	return string(b)
}
