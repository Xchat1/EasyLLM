package openai

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	codexAPIServiceProviderID      = "easyllm"
	codexAPIServiceProviderName    = "EasyLLM API Service"
	codexAPIServiceDefaultModel    = "gpt-5-codex"
	codexAPIServiceDefaultWireAPI  = "responses"
	codexAPIServiceRequiresAuthKey = "requires_openai_auth"
)

type CodexLaunchResult struct {
	RunningBefore bool   `json:"running_before"`
	Restarted     bool   `json:"restarted"`
	Started       bool   `json:"started"`
	Command       string `json:"command"`
}

// SwitchCodexOAuthAccount writes OAuth tokens to ~/.codex/auth.json
// and cleans up API-related fields from ~/.codex/config.toml
func SwitchCodexOAuthAccount(accessToken, refreshToken, idToken string, accountID *string) error {
	codexDir, err := getCodexDir()
	if err != nil {
		return err
	}

	authFile := filepath.Join(codexDir, "auth.json")
	configFile := filepath.Join(codexDir, "config.toml")

	// Build auth.json in the format that Codex CLI v0.111+ expects.
	// NOTE: last_refresh belongs at the TOP LEVEL of auth.json, not inside tokens.
	// Putting it inside tokens causes "Token data is not available." errors.
	authData := map[string]interface{}{
		"OPENAI_API_KEY": nil,
		"tokens": map[string]interface{}{
			"id_token":      idToken,
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"account_id":    accountID,
		},
		"last_refresh": time.Now().UTC().Format(time.RFC3339),
	}

	authJSON, err := json.MarshalIndent(authData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal auth.json: %w", err)
	}

	if err := os.WriteFile(authFile, authJSON, 0600); err != nil {
		return fmt.Errorf("failed to write auth.json: %w", err)
	}

	// Remove API-related fields from config.toml (keep project trust entries intact),
	// then inject chatgpt_base_url so the CLI routes through the local proxy for logging.
	if _, err := os.Stat(configFile); err == nil {
		cleanConfigTOMLAPIFields(configFile)
	}
	injectChatGPTBaseURL(configFile, "http://localhost:8022")

	return nil
}

// SwitchCodexAPIAccount writes API key config to ~/.codex/auth.json and config.toml
func SwitchCodexAPIAccount(modelProvider, model, baseURL, apiKey string, wireAPI, reasoningEffort *string) error {
	codexDir, err := getCodexDir()
	if err != nil {
		return err
	}

	authFile := filepath.Join(codexDir, "auth.json")
	configFile := filepath.Join(codexDir, "config.toml")

	// 1. Update auth.json: set OPENAI_API_KEY and remove tokens
	authData := map[string]interface{}{}

	// Read existing auth.json if exists
	if data, err := os.ReadFile(authFile); err == nil {
		json.Unmarshal(data, &authData)
	}

	authData["OPENAI_API_KEY"] = apiKey
	delete(authData, "tokens")

	authJSON, err := json.MarshalIndent(authData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal auth.json: %w", err)
	}
	if err := os.WriteFile(authFile, authJSON, 0600); err != nil {
		return fmt.Errorf("failed to write auth.json: %w", err)
	}

	// 2. Update config.toml
	wireAPIVal := "responses"
	if wireAPI != nil && *wireAPI != "" {
		wireAPIVal = *wireAPI
	}

	configLines := []string{
		fmt.Sprintf(`model_provider = "%s"`, modelProvider),
		fmt.Sprintf(`model = "%s"`, model),
	}

	if reasoningEffort != nil && *reasoningEffort != "" {
		configLines = append(configLines, fmt.Sprintf(`model_reasoning_effort = "%s"`, *reasoningEffort))
	}

	// model_providers table
	configLines = append(configLines, "")
	configLines = append(configLines, fmt.Sprintf(`[model_providers.%s]`, modelProvider))
	configLines = append(configLines, fmt.Sprintf(`name = "%s"`, modelProvider))
	configLines = append(configLines, fmt.Sprintf(`base_url = "%s"`, baseURL))
	configLines = append(configLines, fmt.Sprintf(`wire_api = "%s"`, wireAPIVal))
	configLines = append(configLines, fmt.Sprintf(`%s = true`, codexAPIServiceRequiresAuthKey))

	configContent := strings.Join(configLines, "\n") + "\n"

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config.toml: %w", err)
	}

	return nil
}

// SwitchCodexAPIService writes the local EasyLLM API service into Codex CLI.
// Codex sees a custom OpenAI-compatible provider, while requests are handled
// by EasyLLM's local proxy pool.
func SwitchCodexAPIService(baseURL, apiKey string) error {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	apiKey = strings.TrimSpace(apiKey)
	if baseURL == "" {
		return fmt.Errorf("baseURL is required")
	}
	if apiKey == "" {
		return fmt.Errorf("apiKey is required")
	}

	codexDir, err := getCodexDir()
	if err != nil {
		return err
	}

	authFile := filepath.Join(codexDir, "auth.json")
	configFile := filepath.Join(codexDir, "config.toml")

	authData := map[string]interface{}{}
	if data, err := os.ReadFile(authFile); err == nil {
		_ = json.Unmarshal(data, &authData)
	}
	authData["OPENAI_API_KEY"] = apiKey
	delete(authData, "tokens")

	authJSON, err := json.MarshalIndent(authData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal auth.json: %w", err)
	}
	if err := os.WriteFile(authFile, authJSON, 0600); err != nil {
		return fmt.Errorf("failed to write auth.json: %w", err)
	}

	configContent := buildCodexAPIServiceConfig(configFile, baseURL)
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config.toml: %w", err)
	}
	return nil
}

// RestartCodexApp starts Codex after config injection. On macOS it restarts
// Codex.app so a running UI picks up the newly written ~/.codex config.
func RestartCodexApp() (*CodexLaunchResult, error) {
	if runtime.GOOS == "darwin" {
		return restartCodexDarwin()
	}

	path, err := exec.LookPath("codex")
	if err != nil {
		return nil, fmt.Errorf("Codex executable not found: %w", err)
	}
	cmd := exec.Command(path)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start Codex: %w", err)
	}
	return &CodexLaunchResult{Started: true, Command: path}, nil
}

func restartCodexDarwin() (*CodexLaunchResult, error) {
	result := &CodexLaunchResult{Command: "open -a Codex"}
	running := codexDarwinIsRunning()
	result.RunningBefore = running
	result.Restarted = running

	if running {
		_ = exec.Command("/usr/bin/osascript", "-e", `tell application "Codex" to quit`).Run()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) && codexDarwinIsRunning() {
			time.Sleep(250 * time.Millisecond)
		}
		if codexDarwinIsRunning() {
			_ = exec.Command("/usr/bin/pkill", "-x", "Codex").Run()
			time.Sleep(500 * time.Millisecond)
		}
	}

	if err := exec.Command("/usr/bin/open", "-a", "Codex").Run(); err != nil {
		return nil, fmt.Errorf("failed to start Codex.app: %w", err)
	}
	result.Started = true
	return result, nil
}

func codexDarwinIsRunning() bool {
	out, err := exec.Command("/usr/bin/osascript", "-e", `application "Codex" is running`).Output()
	if err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(string(out)), "true")
}

// RemoveCodexAPIService removes the EasyLLM managed provider from Codex CLI config.
func RemoveCodexAPIService(apiKey string) error {
	codexDir, err := getCodexDir()
	if err != nil {
		return err
	}

	authFile := filepath.Join(codexDir, "auth.json")
	configFile := filepath.Join(codexDir, "config.toml")

	authData := map[string]interface{}{}
	if data, err := os.ReadFile(authFile); err == nil {
		_ = json.Unmarshal(data, &authData)
		currentKey, _ := authData["OPENAI_API_KEY"].(string)
		if currentKey == "" || currentKey == strings.TrimSpace(apiKey) || strings.HasPrefix(currentKey, "easyllm_codex_") {
			delete(authData, "OPENAI_API_KEY")
			authJSON, err := json.MarshalIndent(authData, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal auth.json: %w", err)
			}
			if err := os.WriteFile(authFile, authJSON, 0600); err != nil {
				return fmt.Errorf("failed to write auth.json: %w", err)
			}
		}
	}

	if data, err := os.ReadFile(configFile); err == nil {
		configContent := stripCodexAPIServiceManagedConfig(string(data), codexAPIServiceProviderID)
		if err := os.WriteFile(configFile, []byte(strings.TrimLeft(configContent, "\n")), 0644); err != nil {
			return fmt.Errorf("failed to write config.toml: %w", err)
		}
	}
	return nil
}

// GetCodexAuthInfo reads ~/.codex/auth.json and returns it
func GetCodexAuthInfo() (map[string]interface{}, error) {
	codexDir, err := getCodexDir()
	if err != nil {
		return nil, err
	}

	authFile := filepath.Join(codexDir, "auth.json")
	data, err := os.ReadFile(authFile)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{}, nil
		}
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func getCodexDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	codexDir := filepath.Join(homeDir, ".codex")
	if err := os.MkdirAll(codexDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create .codex directory: %w", err)
	}
	return codexDir, nil
}

func buildCodexAPIServiceConfig(configFile, baseURL string) string {
	data, _ := os.ReadFile(configFile)
	existing := stripCodexAPIServiceManagedConfig(string(data), codexAPIServiceProviderID)
	serviceBlock := strings.Join([]string{
		fmt.Sprintf("model_provider = %s", tomlString(codexAPIServiceProviderID)),
		fmt.Sprintf("model = %s", tomlString(codexAPIServiceDefaultModel)),
		"",
		fmt.Sprintf("[model_providers.%s]", codexAPIServiceProviderID),
		fmt.Sprintf("name = %s", tomlString(codexAPIServiceProviderName)),
		fmt.Sprintf("base_url = %s", tomlString(baseURL)),
		fmt.Sprintf("wire_api = %s", tomlString(codexAPIServiceDefaultWireAPI)),
		fmt.Sprintf("%s = true", codexAPIServiceRequiresAuthKey),
	}, "\n") + "\n"

	existing = strings.TrimLeft(existing, "\n")
	if strings.TrimSpace(existing) == "" {
		return serviceBlock
	}
	return serviceBlock + "\n" + existing
}

func stripCodexAPIServiceManagedConfig(content, providerID string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	lines := strings.Split(content, "\n")
	filtered := make([]string, 0, len(lines))
	inSection := false
	skipProviderSection := false
	targetSection := "model_providers." + providerID
	topLevelKeys := map[string]bool{
		"model_provider":         true,
		"model":                  true,
		"model_reasoning_effort": true,
		"openai_base_url":        true,
		"chatgpt_base_url":       true,
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			sectionName := strings.Trim(trimmed, "[]")
			inSection = true
			skipProviderSection = sectionName == targetSection
			if skipProviderSection {
				continue
			}
		}
		if skipProviderSection {
			continue
		}
		if !inSection {
			key := topLevelKey(trimmed)
			if topLevelKeys[key] {
				continue
			}
		}
		filtered = append(filtered, line)
	}

	return strings.TrimRight(strings.Join(filtered, "\n"), "\n") + "\n"
}

func topLevelKey(line string) string {
	if line == "" || strings.HasPrefix(line, "#") {
		return ""
	}
	idx := strings.Index(line, "=")
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(line[:idx])
}

func tomlString(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(encoded)
}

// injectChatGPTBaseURL ensures chatgpt_base_url is set in config.toml so the
// Codex CLI routes requests through the local proxy (enabling request logging).
func injectChatGPTBaseURL(configFile, baseURL string) {
	data, _ := os.ReadFile(configFile)
	content := string(data)

	key := "chatgpt_base_url"
	line := fmt.Sprintf(`%s = "%s"`, key, baseURL)

	// Already present?
	for _, l := range strings.Split(content, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), key+" ") || strings.HasPrefix(strings.TrimSpace(l), key+"=") {
			return
		}
	}

	// Prepend before any [section]
	if content == "" {
		content = line + "\n"
	} else {
		content = line + "\n" + content
	}
	os.WriteFile(configFile, []byte(content), 0644)
}

// cleanConfigTOMLAPIFields removes API-related keys from config.toml
func cleanConfigTOMLAPIFields(configFile string) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")
	apiKeys := map[string]bool{
		"model_provider":         true,
		"model":                  true,
		"model_reasoning_effort": true,
		"model_providers":        true,
		"chatgpt_base_url":       true,
	}

	var filtered []string
	skipSection := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect [model_providers...] section start
		if strings.HasPrefix(trimmed, "[model_providers") {
			skipSection = true
			continue
		}

		// End skip when we hit another top-level section
		if skipSection && strings.HasPrefix(trimmed, "[") && !strings.HasPrefix(trimmed, "[model_providers") {
			skipSection = false
		}

		if skipSection {
			continue
		}

		// Skip API-related top-level keys
		isAPIKey := false
		for k := range apiKeys {
			if strings.HasPrefix(trimmed, k+" ") || strings.HasPrefix(trimmed, k+"=") {
				isAPIKey = true
				break
			}
		}
		if !isAPIKey {
			filtered = append(filtered, line)
		}
	}

	os.WriteFile(configFile, []byte(strings.Join(filtered, "\n")), 0644)
}
