package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	openai "easyllm/internal/openai"
)

func main() {
	apiKey := strings.TrimSpace(os.Getenv("PROXY_API_KEY"))
	if apiKey == "" {
		apiKey = loadProxyAPIKey()
	}
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "proxy_api_key not found in EasyLLM database")
		os.Exit(1)
	}

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8022"
	}
	baseURL := fmt.Sprintf("http://localhost:%s/backend-api/codex", port)

	if err := openai.SwitchCodexAPIService(baseURL, apiKey); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("Codex config injected: base_url=%s\n", baseURL)
	if result, err := openai.RestartCodexApp(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to restart Codex.app: %v\n", err)
	} else if result != nil {
		fmt.Printf("Codex.app restarted (was_running=%v)\n", result.RunningBefore)
	}
}

func loadProxyAPIKey() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	paths := []string{
		filepath.Join(home, "Library", "Application Support", "EasyLLM", "data", "easyllm.db"),
		filepath.Join(home, "Library", "Application Support", "EasyLLM", "easyllm.db"),
	}
	for _, path := range paths {
		if key := readKeyFromDB(path); key != "" {
			return key
		}
	}
	return ""
}

func readKeyFromDB(path string) string {
	out, err := exec.Command("sqlite3", path, `SELECT value FROM app_settings WHERE key = 'proxy_api_key' LIMIT 1;`).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
