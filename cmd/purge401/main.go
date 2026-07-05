// purge401: 按 EasyLLM「配额」按钮逻辑测试系统内 OAuth 账号并清理。
//
// 默认删除所有非 200 账号（含 forbidden/401/网络失败等），仅保留 quota/verified。
//
// 用法:
//
//	go run ./cmd/purge401 [-dry-run] [-workers=10]
//	go run ./cmd/purge401 -401-only   # 仅删除 401
package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"easyllm/config"
	openaiplatform "easyllm/internal/openai"
	"easyllm/internal/models"
	"easyllm/internal/storage"
)

type accountResult struct {
	account  models.OpenAIAccount
	category string
	httpCode int
	errMsg   string
	delete   bool
}

func main() {
	dryRun := flag.Bool("dry-run", false, "只测试不删除")
	only401 := flag.Bool("401-only", false, "仅删除 401，保留 forbidden 等非 200")
	workers := flag.Int("workers", 10, "并发数")
	flag.Parse()

	cfg := config.Load()
	if err := storage.InitDB(cfg); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer storage.CloseDB()

	store := storage.NewOpenAIStorage(storage.GetDB())
	accounts, err := store.List()
	if err != nil {
		log.Fatalf("读取账号失败: %v", err)
	}

	var oauthAccounts []models.OpenAIAccount
	for _, a := range accounts {
		if a.AccountType == models.OpenAIAccountTypeOAuth &&
			(derefStr(a.AccessToken) != "" || derefStr(a.RefreshToken) != "") {
			oauthAccounts = append(oauthAccounts, a)
		}
	}

	if len(oauthAccounts) == 0 {
		fmt.Println("没有可测试的 OAuth 账号")
		return
	}

	fmt.Printf("数据库: %s\n", cfg.Database.SQLitePath)
	if *only401 {
		fmt.Printf("测试 %d 个 OAuth 账号，仅删除 401...\n", len(oauthAccounts))
	} else {
		fmt.Printf("测试 %d 个 OAuth 账号，删除所有非 200...\n", len(oauthAccounts))
	}
	if *dryRun {
		fmt.Println("dry-run 模式：不会删除账号")
	}

	results := make([]accountResult, len(oauthAccounts))
	sem := make(chan struct{}, *workers)
	var wg sync.WaitGroup
	var done int64

	for i, acc := range oauthAccounts {
		wg.Add(1)
		go func(idx int, account models.OpenAIAccount) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result := testAccount(&account, store, *only401)
			results[idx] = result

			n := atomic.AddInt64(&done, 1)
			label := account.Email
			if result.delete {
				log.Printf("[%d/%d] %s: HTTP %d %s -> 删除", n, len(oauthAccounts), label, result.httpCode, result.category)
			} else {
				log.Printf("[%d/%d] %s: HTTP 200 %s (保留)", n, len(oauthAccounts), label, result.category)
			}
		}(i, acc)
	}
	wg.Wait()

	stats := map[string]int{}
	var deleteIDs []string
	for _, r := range results {
		stats[r.category]++
		if r.delete {
			deleteIDs = append(deleteIDs, r.account.ID)
		}
	}

	fmt.Println("\n分类统计:")
	for _, key := range []string{"quota", "verified", "forbidden", "failed", "missing_token"} {
		if n := stats[key]; n > 0 {
			fmt.Printf("  %s: %d\n", key, n)
		}
	}

	kept := len(oauthAccounts) - len(deleteIDs)
	fmt.Printf("\n保留 %d, 待删除 %d\n", kept, len(deleteIDs))
	if len(deleteIDs) == 0 {
		fmt.Println("没有需要删除的账号")
		return
	}
	if *dryRun {
		fmt.Println("dry-run 模式，未执行删除")
		return
	}

	if err := store.DeleteMany(deleteIDs); err != nil {
		log.Fatalf("删除失败: %v", err)
	}
	fmt.Printf("已彻底删除 %d 个账号\n", len(deleteIDs))
}

func testAccount(account *models.OpenAIAccount, store *storage.OpenAIStorage, only401 bool) accountResult {
	result := accountResult{account: *account}

	accessToken := derefStr(account.AccessToken)
	chatgptID := derefStr(account.ChatGPTAccountID)

	if accessToken == "" && derefStr(account.RefreshToken) != "" {
		if err := refreshOAuthTokens(account, store); err != nil {
			result.category = "failed"
			result.httpCode = extractHTTPCode(err.Error())
			result.errMsg = err.Error()
			result.delete = !only401 || is401Error(err)
			return result
		}
		accessToken = derefStr(account.AccessToken)
		chatgptID = derefStr(account.ChatGPTAccountID)
	}

	if accessToken == "" {
		result.category = "missing_token"
		result.httpCode = 0
		result.errMsg = "missing access_token"
		result.delete = !only401
		return result
	}

	info, err := openaiplatform.FetchQuota(accessToken, chatgptID)
	if err != nil && is401Error(err) && derefStr(account.RefreshToken) != "" {
		if refreshErr := refreshOAuthTokens(account, store); refreshErr != nil {
			result.category = "failed"
			result.httpCode = extractHTTPCode(refreshErr.Error())
			result.errMsg = refreshErr.Error()
			result.delete = !only401 || is401Error(refreshErr)
			return result
		}
		accessToken = derefStr(account.AccessToken)
		chatgptID = derefStr(account.ChatGPTAccountID)
		info, err = openaiplatform.FetchQuota(accessToken, chatgptID)
	}

	if err != nil {
		result.category = "failed"
		result.httpCode = extractHTTPCode(err.Error())
		result.errMsg = err.Error()
		result.delete = !only401 || is401Error(err)
		return result
	}

	if info != nil && info.IsForbidden {
		result.category = "forbidden"
		result.httpCode = 403
		result.delete = !only401
		return result
	}

	if info != nil && (info.Codex5hUsedPercent != nil || info.Codex7dUsedPercent != nil || info.Total > 0) {
		result.category = "quota"
		result.httpCode = 200
		return result
	}

	result.category = "verified"
	result.httpCode = 200
	return result
}

func refreshOAuthTokens(account *models.OpenAIAccount, store *storage.OpenAIStorage) error {
	if account == nil || derefStr(account.RefreshToken) == "" {
		return fmt.Errorf("no refresh token available")
	}

	tokenResp, err := openaiplatform.RefreshToken(*account.RefreshToken)
	if err != nil {
		if strings.Contains(err.Error(), "refresh_token_reused") {
			return fmt.Errorf("HTTP 401: refresh_token 已轮换失效")
		}
		return err
	}

	account.Status = "active"
	account.AccessToken = strPtr(tokenResp.AccessToken)
	if tokenResp.RefreshToken != "" {
		account.RefreshToken = strPtr(tokenResp.RefreshToken)
	}
	if tokenResp.IDToken != "" {
		account.IDToken = strPtr(tokenResp.IDToken)
		if userInfo := openaiplatform.ParseIDToken(tokenResp.IDToken); userInfo != nil {
			if userInfo.Email != nil && *userInfo.Email != "" {
				account.Email = *userInfo.Email
			}
			account.ChatGPTAccountID = userInfo.ChatGPTAccountID
			account.ChatGPTUserID = userInfo.ChatGPTUserID
			account.OrganizationID = userInfo.OrganizationID
			if userInfo.PlanType != nil && *userInfo.PlanType != "" {
				account.Plan = userInfo.PlanType
			}
		}
		if j := openaiplatform.ExtractOpenAIAuthJSON(tokenResp.IDToken); j != "" {
			account.OpenAIAuthJSON = strPtr(j)
		}
	}
	if tokenResp.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		account.ExpiresAt = &t
	}
	account.UpdatedAt = time.Now()
	return store.Save(account)
}

func is401Error(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "HTTP 401") ||
		strings.Contains(msg, "refresh_token 已轮换失效")
}

func extractHTTPCode(message string) int {
	for _, item := range []struct {
		prefix string
		code   int
	}{
		{"HTTP 401", 401},
		{"HTTP 402", 402},
		{"HTTP 403", 403},
		{"HTTP 429", 429},
		{"HTTP 503", 503},
	} {
		if strings.Contains(message, item.prefix) {
			return item.code
		}
	}
	return 0
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func strPtr(s string) *string { return &s }
