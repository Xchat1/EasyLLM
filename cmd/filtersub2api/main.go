// filtersub2api: 按 EasyLLM「配额」按钮相同逻辑测试 sub2api 导出 JSON 中的账户。
//
// 与 internal/handlers/openai.go FetchQuotas 一致，调用 openai.FetchQuota：
//   - 成功（含 forbidden/verified/quota）→ 保留
//   - 失败（如 HTTP 401）→ 删除
//
// 用法:
//
//	go run ./cmd/filtersub2api [-in-place] [-workers=20] <sub2api-export.json> [more.json...]
//
// 默认输出到 <原名>-valid.json；-in-place 直接覆盖原文件。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	openaiplatform "easyllm/internal/openai"
)

type quotaCheckOutcome struct {
	account   map[string]interface{}
	keep      bool
	category  string // quota, verified, forbidden, failed
	httpCode  int
	errMsg    string
}

func main() {
	inPlace := flag.Bool("in-place", false, "覆盖原文件")
	workers := flag.Int("workers", 20, "并发数")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: go run ./cmd/filtersub2api [-in-place] <sub2api-export.json> [more.json...]")
		os.Exit(1)
	}

	totalBefore, totalAfter, totalRemoved := 0, 0, 0
	stats := map[string]int{}

	for _, inputPath := range args {
		before, after, removed, fileStats, err := filterFile(inputPath, *inPlace, *workers)
		if err != nil {
			log.Fatalf("%s: %v", inputPath, err)
		}
		totalBefore += before
		totalAfter += after
		totalRemoved += removed
		for k, v := range fileStats {
			stats[k] += v
		}
	}

	fmt.Printf("\n全部完成: 原有 %d, 保留 %d, 删除 %d\n", totalBefore, totalAfter, totalRemoved)
	if len(stats) > 0 {
		fmt.Println("分类统计:")
		for _, key := range []string{"quota", "verified", "forbidden", "network_error", "failed"} {
			if n := stats[key]; n > 0 {
				fmt.Printf("  %s: %d\n", key, n)
			}
		}
	}
}

func filterFile(inputPath string, inPlace bool, workers int) (before, after, removed int, stats map[string]int, err error) {
	raw, err := os.ReadFile(inputPath)
	if err != nil {
		return 0, 0, 0, nil, err
	}

	var export map[string]interface{}
	if err := json.Unmarshal(raw, &export); err != nil {
		return 0, 0, 0, nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}

	accountsRaw, ok := export["accounts"].([]interface{})
	if !ok || len(accountsRaw) == 0 {
		fmt.Printf("%s: 无账户，跳过\n", inputPath)
		return 0, 0, 0, nil, nil
	}

	accounts := make([]map[string]interface{}, 0, len(accountsRaw))
	for _, item := range accountsRaw {
		acc, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		accounts = append(accounts, acc)
	}

	before = len(accounts)
	stats = make(map[string]int)
	fmt.Printf("\n测试 %s (%d 个账户，逻辑同「配额」按钮)...\n", inputPath, before)

	results := make([]quotaCheckOutcome, len(accounts))
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var done int64

	for i, acc := range accounts {
		wg.Add(1)
		go func(idx int, account map[string]interface{}) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			outcome := checkAccountQuota(account)
			results[idx] = outcome

			n := atomic.AddInt64(&done, 1)
			email := accountLabel(account)
			if outcome.keep {
				log.Printf("[%d/%d] %s: %s (保留)", n, before, email, outcome.category)
			} else {
				log.Printf("[%d/%d] %s: failed HTTP %d %s (删除)", n, before, email, outcome.httpCode, outcome.errMsg)
			}
		}(i, acc)
	}
	wg.Wait()

	valid := make([]interface{}, 0, len(accounts))
	for _, r := range results {
		stats[r.category]++
		if r.keep {
			valid = append(valid, r.account)
		}
	}

	after = len(valid)
	removed = before - after

	export["accounts"] = valid
	out, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return before, after, removed, stats, err
	}
	out = append(out, '\n')

	outputPath := inputPath
	if !inPlace {
		ext := filepath.Ext(inputPath)
		base := strings.TrimSuffix(inputPath, ext)
		outputPath = base + "-valid" + ext
	}

	if err := os.WriteFile(outputPath, out, 0o644); err != nil {
		return before, after, removed, stats, err
	}

	fmt.Printf("%s: 保留 %d, 删除 %d -> %s\n", inputPath, after, removed, outputPath)
	return before, after, removed, stats, nil
}

func checkAccountQuota(account map[string]interface{}) quotaCheckOutcome {
	creds, _ := account["credentials"].(map[string]interface{})
	accessToken := strings.TrimSpace(asString(creds["access_token"]))
	chatgptID := strings.TrimSpace(asString(creds["chatgpt_account_id"]))

	outcome := quotaCheckOutcome{account: account}

	if accessToken == "" {
		outcome.keep = false
		outcome.category = "failed"
		outcome.httpCode = 0
		outcome.errMsg = "missing access_token"
		return outcome
	}

	var info *openaiplatform.QuotaInfo
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		info, lastErr = openaiplatform.FetchQuota(accessToken, chatgptID)
		if lastErr == nil {
			break
		}
		if !isQuotaUnauthorized(lastErr) {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if lastErr != nil {
		if isNetworkError(lastErr) {
			outcome.keep = true
			outcome.category = "network_error"
			outcome.httpCode = 0
			outcome.errMsg = lastErr.Error()
			return outcome
		}
		outcome.keep = false
		outcome.category = "failed"
		outcome.httpCode = extractHTTPStatusCode(lastErr.Error())
		outcome.errMsg = lastErr.Error()
		return outcome
	}

	outcome.keep = true
	if info == nil {
		outcome.category = "verified"
		outcome.httpCode = 200
		return outcome
	}
	if info.IsForbidden {
		outcome.category = "forbidden"
		outcome.httpCode = 403
		return outcome
	}
	if hasQuotaData(info) {
		outcome.category = "quota"
		outcome.httpCode = 200
		return outcome
	}

	outcome.category = "verified"
	outcome.httpCode = 200
	return outcome
}

func hasQuotaData(info *openaiplatform.QuotaInfo) bool {
	if info == nil {
		return false
	}
	return info.Codex5hUsedPercent != nil ||
		info.Codex7dUsedPercent != nil ||
		info.Total > 0
}

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

func isQuotaUnauthorized(err error) bool {
	return err != nil && strings.Contains(err.Error(), "HTTP 401")
}

func extractHTTPStatusCode(message string) int {
	for _, prefix := range []string{"HTTP 401", "HTTP 402", "HTTP 403", "HTTP 429", "HTTP 503"} {
		if strings.Contains(message, prefix) {
			switch prefix {
			case "HTTP 401":
				return 401
			case "HTTP 402":
				return 402
			case "HTTP 403":
				return 403
			case "HTTP 429":
				return 429
			case "HTTP 503":
				return 503
			}
		}
	}
	return 0
}

func accountLabel(account map[string]interface{}) string {
	if creds, ok := account["credentials"].(map[string]interface{}); ok {
		if email := strings.TrimSpace(asString(creds["email"])); email != "" {
			return email
		}
	}
	if name := strings.TrimSpace(asString(account["name"])); name != "" {
		return name
	}
	return "unknown"
}

func asString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	case float64:
		return fmt.Sprintf("%v", t)
	default:
		return fmt.Sprintf("%v", v)
	}
}
