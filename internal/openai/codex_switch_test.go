package openai

import (
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
		`model = "gpt-5-codex"`,
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
