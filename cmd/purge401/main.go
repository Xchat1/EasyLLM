// purge401: 按 EasyLLM「配额」按钮逻辑测试系统内 OAuth 账号并清理。
//
// 仅删除凭证明确失效的账号：HTTP 401 在任何模式下都删除；forbidden/403 仅在默认
// 模式下删除（-401-only 模式保留）。网络超时、DNS/TLS 失败、HTTP 429/503 等瞬时
// 错误以及未知错误一律保留，避免网络抖动或上游故障时误删有效账号。
// 删除前会打印待删清单并要求输入 yes 确认。
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
	yes := flag.Bool("yes", false, "跳过确认提示直接执行删除 (适用于自动化脚本)")
	flag.BoolVar(yes, "y", false, "跳过确认提示直接执行删除 (同 -yes)")
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
		fmt.Printf("测试 %d 个 OAuth 账号，删除凭证明确失效的账号（401；默认模式下含 forbidden/403）...\n", len(oauthAccounts))
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

	fmt.Println("\n待删除账号清单（将永久删除，无法恢复）:")
	for _, r := range results {
		if r.delete {
			fmt.Printf("  - %s (HTTP %d, %s: %s)\n", r.account.Email, r.httpCode, r.category, r.errMsg)
		}
	}
	if !*yes {
		fmt.Print("确认删除吗？输入 yes 继续: ")
		var confirm string
		fmt.Scanln(&confirm)
		if strings.ToLower(strings.TrimSpace(confirm)) != "yes" {
			fmt.Println("已取消，未删除任何账号")
			return
		}
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
			result.delete = shouldDeleteAccount(err, result.httpCode, only401)
			return result
		}
		accessToken = derefStr(account.AccessToken)
		chatgptID = derefStr(account.ChatGPTAccountID)
	}

	if accessToken == "" {
		result.category = "missing_token"
		result.httpCode = 0
		result.errMsg = "missing access_token"
		// 没有任何可用 token 的账号无法判定凭证明确失效，一律保留，由用户手动处理。
		result.delete = false
		return result
	}

	info, err := openaiplatform.FetchQuota(accessToken, chatgptID)
	if err != nil && is401Error(err) && derefStr(account.RefreshToken) != "" {
		if refreshErr := refreshOAuthTokens(account, store); refreshErr != nil {
			result.category = "failed"
			result.httpCode = extractHTTPCode(refreshErr.Error())
			result.errMsg = refreshErr.Error()
			result.delete = shouldDeleteAccount(refreshErr, result.httpCode, only401)
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
		result.delete = shouldDeleteAccount(err, result.httpCode, only401)
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

// isNetworkError 识别瞬时网络故障：这类错误说明不了账号本身有问题，绝不能触发删除。
// （与 cmd/filtersub2api 的同名函数保持一致的分类口径。）
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "context deadline exceeded") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "EOF") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "TLS handshake") ||
		strings.Contains(msg, "no such host")
}

// shouldDeleteAccount 决定一次失败的账号检测是否值得永久删除。
// 只有凭证明确失效（HTTP 401）才删除；网络错误、429/503 等瞬时错误和未知错误一律保留。
// forbidden/403 沿用项目惯例：默认模式删除，-401-only 模式保留。
func shouldDeleteAccount(err error, httpCode int, only401 bool) bool {
	if isNetworkError(err) {
		return false
	}
	code := httpCode
	if code == 0 && err != nil {
		code = extractHTTPCode(err.Error())
	}
	switch code {
	case 401:
		return true
	case 403:
		return !only401
	default:
		return false
	}
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
