package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/joho/godotenv"

	"easyllm/config"
	"easyllm/internal/handlers"
	openaiplatform "easyllm/internal/platforms/openai"
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
		fmt.Fprintln(os.Stderr, "用法:")
		fmt.Fprintln(os.Stderr, "  go run ./cmd/importsub2api <sub2api导出.json>")
		fmt.Fprintln(os.Stderr, "  go run ./cmd/importsub2api -           # 从 stdin 读完整 Sub2API JSON")
		fmt.Fprintln(os.Stderr, "  go run ./cmd/importsub2api --credentials <仅含credentials对象的.json>")
		os.Exit(1)
	}

	var body []byte
	var err error

	if args[0] == "--credentials" {
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "请指定 credentials JSON 文件路径")
			os.Exit(1)
		}
		raw, err := os.ReadFile(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		var credObj map[string]interface{}
		if err := json.Unmarshal(raw, &credObj); err != nil {
			fmt.Fprintln(os.Stderr, "解析 credentials JSON:", err)
			os.Exit(1)
		}
		idTok, _ := credObj["id_token"].(string)
		email := ""
		if idTok != "" {
			if u := openaiplatform.ParseIDToken(idTok); u != nil && u.Email != nil {
				email = strings.TrimSpace(*u.Email)
			}
		}
		if email == "" {
			fmt.Fprintln(os.Stderr, "无法从 id_token 解析邮箱；请使用带 extra.email / name 的完整 Sub2API JSON")
			os.Exit(1)
		}
		wrap := map[string]interface{}{
			"accounts": []interface{}{
				map[string]interface{}{
					"name":        email,
					"type":        "oauth",
					"platform":    "openai",
					"extra":       map[string]string{"email": email},
					"credentials": credObj,
				},
			},
		}
		body, err = json.Marshal(wrap)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else if args[0] == "-" {
		body, err = io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		body, err = os.ReadFile(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	out, err := h.ImportSub2APIBytes(body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "导入失败:", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}
