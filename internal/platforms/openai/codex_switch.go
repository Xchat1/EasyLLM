package openai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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

	configContent := strings.Join(configLines, "\n") + "\n"

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config.toml: %w", err)
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
