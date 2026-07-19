package openai

import (
	"easyllm/internal/codexconfig"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSwitchCodexAPIServiceWritesLocalProviderAndPreservesOtherConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	codexDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexDir, 0755); err != nil {
		t.Fatalf("mkdir codex dir: %v", err)
	}
	configPath := filepath.Join(codexDir, "config.toml")
	initialConfig := strings.Join([]string{
		`model_provider = "old-provider"`,
		`model = "old-model"`,
		`chatgpt_base_url = "http://old.local"`,
		``,
		`[model_providers.other]`,
		`name = "other"`,
		`base_url = "https://example.com/v1"`,
		`wire_api = "responses"`,
		``,
		`[model_providers.easyllm]`,
		`name = "stale"`,
		`base_url = "http://stale.local/v1"`,
		`wire_api = "chat"`,
		``,
		`[projects."/tmp/project"]`,
		`trust_level = "trusted"`,
		``,
	}, "\n")
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := SwitchCodexAPIService("http://localhost:18080/v1", "easyllm_codex_test"); err != nil {
		t.Fatalf("switch api service: %v", err)
	}

	authData, err := os.ReadFile(filepath.Join(codexDir, "auth.json"))
	if err != nil {
		t.Fatalf("read auth: %v", err)
	}
	var auth map[string]any
	if err := json.Unmarshal(authData, &auth); err != nil {
		t.Fatalf("decode auth: %v", err)
	}
	if got := auth["OPENAI_API_KEY"]; got != "easyllm_codex_test" {
		t.Fatalf("expected OPENAI_API_KEY to be written, got %#v", got)
	}
	if _, ok := auth["tokens"]; ok {
		t.Fatalf("expected OAuth tokens to be removed in API service mode")
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	config := string(configData)
	for _, want := range []string{
		`model_provider = "easyllm"`,
		`model = "gpt-5.6-sol"`,
		`[model_providers.easyllm]`,
		`name = "EasyLLM API Service"`,
		`base_url = "http://localhost:18080/v1"`,
		`wire_api = "responses"`,
		`requires_openai_auth = true`,
		`[model_providers.other]`,
		`[projects."/tmp/project"]`,
		`trust_level = "trusted"`,
	} {
		if !strings.Contains(config, want) {
			t.Fatalf("expected config to contain %q, got:\n%s", want, config)
		}
	}
	if strings.Contains(config, "stale.local") || strings.Contains(config, "old-provider") || strings.Contains(config, "chatgpt_base_url") {
		t.Fatalf("expected stale managed service config to be removed, got:\n%s", config)
	}
}

func TestSwitchCodexAPIServiceAppliesAndClearsContextConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	codexDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexDir, 0755); err != nil {
		t.Fatalf("mkdir codex dir: %v", err)
	}
	configPath := filepath.Join(codexDir, "config.toml")

	if err := SwitchCodexAPIService("http://localhost:18080/v1", "easyllm_codex_test", codexconfig.ContextConfig{Mode: codexconfig.ModePreset516K}); err != nil {
		t.Fatalf("switch api service with context config: %v", err)
	}
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	config := string(configData)
	for _, want := range []string{
		`model_context_window = 516000`,
		`model_auto_compact_token_limit = 460000`,
	} {
		if !strings.Contains(config, want) {
			t.Fatalf("expected config to contain %q, got:\n%s", want, config)
		}
	}

	if err := SwitchCodexAPIService("http://localhost:18080/v1", "easyllm_codex_test", codexconfig.ContextConfig{Mode: codexconfig.ModeDefault}); err != nil {
		t.Fatalf("switch api service with default context config: %v", err)
	}
	configData, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	config = string(configData)
	if strings.Contains(config, "model_context_window") || strings.Contains(config, "model_auto_compact_token_limit") {
		t.Fatalf("expected context config keys to be removed in default mode, got:\n%s", config)
	}
}

func TestSwitchCodexOAuthAccountWritesTokensAndValidProxyURL(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	codexDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexDir, 0755); err != nil {
		t.Fatalf("mkdir codex dir: %v", err)
	}
	configPath := filepath.Join(codexDir, "config.toml")
	initialConfig := strings.Join([]string{
		`model_provider = "easyllm"`,
		`model = "gpt-5.6-sol"`,
		``,
		`[model_providers.easyllm]`,
		`name = "EasyLLM API Service"`,
		`base_url = "http://localhost:8022/backend-api/codex"`,
		`wire_api = "responses"`,
		``,
		`[projects."/tmp/project"]`,
		`trust_level = "trusted"`,
		``,
	}, "\n")
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	accountID := "acct-123"
	if err := SwitchCodexOAuthAccount("access-token", "refresh-token", "id-token", &accountID, "http://localhost:8022"); err != nil {
		t.Fatalf("switch oauth account: %v", err)
	}

	authData, err := os.ReadFile(filepath.Join(codexDir, "auth.json"))
	if err != nil {
		t.Fatalf("read auth: %v", err)
	}
	var auth map[string]any
	if err := json.Unmarshal(authData, &auth); err != nil {
		t.Fatalf("decode auth: %v", err)
	}
	if got, ok := auth["OPENAI_API_KEY"]; !ok || got != nil {
		t.Fatalf("expected OPENAI_API_KEY to be explicitly cleared, got %#v", got)
	}
	tokens, ok := auth["tokens"].(map[string]any)
	if !ok {
		t.Fatalf("expected tokens object, got %#v", auth["tokens"])
	}
	for key, want := range map[string]string{
		"access_token":  "access-token",
		"refresh_token": "refresh-token",
		"id_token":      "id-token",
		"account_id":    accountID,
	} {
		if got := tokens[key]; got != want {
			t.Fatalf("expected tokens.%s = %q, got %#v", key, want, got)
		}
	}
	if _, ok := tokens["last_refresh"]; ok {
		t.Fatal("last_refresh must be top-level, not inside tokens")
	}
	if _, ok := auth["last_refresh"].(string); !ok {
		t.Fatalf("expected top-level last_refresh, got %#v", auth["last_refresh"])
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	config := string(configData)
	if !strings.Contains(config, `chatgpt_base_url = "http://localhost:8022"`) {
		t.Fatalf("expected valid chatgpt_base_url, got:\n%s", config)
	}
	if strings.Contains(config, "http://[http://") || strings.Contains(config, "[model_providers.easyllm]") || strings.Contains(config, `model_provider = "easyllm"`) {
		t.Fatalf("expected API service config to be removed for OAuth mode, got:\n%s", config)
	}
	if !strings.Contains(config, `[projects."/tmp/project"]`) {
		t.Fatalf("expected project trust config to be preserved, got:\n%s", config)
	}
}

func TestSwitchCodexRelayProviderAndState(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Relay now uses /v1 (same root as the OpenAI-compatible API) instead of the
	// old /relay/v1 sub-path.
	if err := SwitchCodexRelayProvider("http://localhost:18080/v1", "deepseek-chat", "localhost:18080"); err != nil {
		t.Fatalf("SwitchCodexRelayProvider: %v", err)
	}

	state := GetCodexRelayState()
	if !state.Injected {
		t.Fatal("expected codex relay to be injected")
	}
	if state.ModelProvider != "relay" {
		t.Fatalf("expected model_provider relay, got %q", state.ModelProvider)
	}
	if state.Model != "deepseek-chat" {
		t.Fatalf("expected model deepseek-chat, got %q", state.Model)
	}
	if state.BaseURL != "http://localhost:18080/v1" {
		t.Fatalf("unexpected base_url %q", state.BaseURL)
	}
}
