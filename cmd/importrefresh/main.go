package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"

	"easyllm/config"
	"easyllm/internal/handlers"
	"easyllm/internal/storage"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	if err := storage.InitDB(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "数据库初始化失败:", err)
		os.Exit(1)
	}
	db := storage.GetDB()
	h := handlers.NewOpenAIHandler(storage.NewOpenAIStorage(db), storage.NewCodexStorage(db))

	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法: go run ./cmd/importrefresh <refresh_token>")
		fmt.Fprintln(os.Stderr, "说明: 向 OpenAI 换票后写入本地库，与网页「Refresh Token 导入」一致。")
		os.Exit(1)
	}
	rt := strings.TrimSpace(strings.Join(args, " "))
	acct, err := h.ImportOAuthAccountByRefreshToken(rt)
	if err != nil {
		fmt.Fprintln(os.Stderr, "导入失败:", err)
		os.Exit(1)
	}
	fmt.Printf("已写入/更新账号: %s (id=%s)\n", acct.Email, acct.ID)
}
