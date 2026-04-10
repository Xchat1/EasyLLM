package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, message, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"type":    code,
			"code":    fmt.Sprintf("%d", status),
		},
	})
}

// copyResponseHeaders copies upstream response headers to the downstream writer,
// skipping hop-by-hop headers.
func copyResponseHeaders(w http.ResponseWriter, resp *http.Response) {
	for key, values := range resp.Header {
		lower := strings.ToLower(key)
		if lower == "transfer-encoding" || lower == "connection" || lower == "keep-alive" || lower == "content-length" {
			continue
		}
		for _, v := range values {
			w.Header().Add(key, v)
		}
	}
}

// setCodexCLIHeaders sets the headers required by chatgpt.com/backend-api/codex
// to identify the client as a legitimate Codex CLI agent.
func setCodexCLIHeaders(req *http.Request) {
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "codex_cli_rs/0.98.0")
	}
	if req.Header.Get("OpenAI-Beta") == "" {
		req.Header.Set("OpenAI-Beta", "responses=experimental")
	}
	if req.Header.Get("originator") == "" {
		req.Header.Set("originator", "codex_cli_rs")
	}
}

// buildUpstreamURL maps /v1/* to https://chatgpt.com/backend-api/codex/*
// which is the real Codex backend. This matches the upstream target used by
// the original augment-token-mng Tauri app.
// client_version is always appended because the chatgpt.com Codex API requires it.
func buildUpstreamURL(path, query string) string {
	const base = "https://chatgpt.com"
	const clientVersion = "0.98.0"
	mapped := mapCodexPath(path)
	if strings.Contains(query, "client_version=") {
		return base + mapped + "?" + query
	}
	var qs string
	if query != "" {
		qs = query + "&client_version=" + clientVersion
	} else {
		qs = "client_version=" + clientVersion
	}
	return base + mapped + "?" + qs
}

// mapCodexPath converts /v1/<tail> → /backend-api/codex/<tail>
// and passes through any path already starting with /backend-api/codex.
func mapCodexPath(path string) string {
	if path == "/v1" {
		return "/backend-api/codex"
	}
	if tail := strings.TrimPrefix(path, "/v1/"); tail != path {
		return "/backend-api/codex/" + tail
	}
	if strings.HasPrefix(path, "/backend-api/codex") {
		return path
	}
	return path
}

// isTokenInvalidated checks if the upstream response indicates the token was invalidated.
func isTokenInvalidated(body []byte) bool {
	if len(body) == 0 {
		return false
	}
	return bytes.Contains(body, []byte(`"code":"token_invalidated"`)) || bytes.Contains(body, []byte(`"code": "token_invalidated"`))
}

// isRetryableAuthFailure checks if the upstream error is a retryable auth failure.
func isRetryableAuthFailure(body []byte) bool {
	if len(body) == 0 {
		return false
	}
	// Try JSON parsing first.
	var obj map[string]interface{}
	if err := json.Unmarshal(body, &obj); err == nil {
		if e, ok := obj["error"].(map[string]interface{}); ok {
			if code, _ := e["code"].(string); code != "" {
				switch code {
				case "token_invalidated", "account_deactivated":
					return true
				}
			}
		}
	}
	// Fallback substring matching.
	return bytes.Contains(body, []byte(`"code":"token_invalidated"`)) ||
		bytes.Contains(body, []byte(`"code": "token_invalidated"`)) ||
		bytes.Contains(body, []byte(`"code":"account_deactivated"`)) ||
		bytes.Contains(body, []byte(`"code": "account_deactivated"`))
}

// buildPromptFromMessages converts a chat messages array into a plain text prompt.
func buildPromptFromMessages(v interface{}) string {
	msgs, ok := v.([]interface{})
	if !ok || len(msgs) == 0 {
		return ""
	}
	var b strings.Builder
	for _, m := range msgs {
		obj, ok := m.(map[string]interface{})
		if !ok {
			continue
		}
		role, _ := obj["role"].(string)
		content, _ := obj["content"].(string)
		role = strings.TrimSpace(role)
		content = strings.TrimSpace(content)
		if content == "" {
			continue
		}
		if role == "" {
			role = "user"
		}
		b.WriteString(role)
		b.WriteString(": ")
		b.WriteString(content)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

// extractOutputTextFromResponsesEvent extracts output text from a Codex Responses SSE event.
func extractOutputTextFromResponsesEvent(data string) string {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(data), &obj); err != nil {
		return ""
	}
	resp, _ := obj["response"].(map[string]interface{})
	if resp == nil {
		return ""
	}
	output, _ := resp["output"].([]interface{})
	if len(output) == 0 {
		return ""
	}
	var b strings.Builder
	for _, item := range output {
		m, _ := item.(map[string]interface{})
		if m == nil {
			continue
		}
		content, _ := m["content"].([]interface{})
		for _, c := range content {
			cm, _ := c.(map[string]interface{})
			if cm == nil {
				continue
			}
			typ, _ := cm["type"].(string)
			if typ != "output_text" {
				continue
			}
			text, _ := cm["text"].(string)
			if text != "" {
				b.WriteString(text)
			}
		}
	}
	return b.String()
}

// disableWebSocketInModels rewrites the models JSON to prevent clients
// from using WebSocket connections, forcing them to use HTTP POST instead.
func disableWebSocketInModels(body []byte) []byte {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return body
	}
	models, ok := data["models"].([]interface{})
	if !ok {
		return body
	}
	for _, m := range models {
		model, ok := m.(map[string]interface{})
		if !ok {
			continue
		}
		model["prefer_websockets"] = false
		model["supports_websockets"] = false
	}
	out, err := json.Marshal(data)
	if err != nil {
		return body
	}
	return out
}

// parsePlatform detects the client platform from the User-Agent string.
func parsePlatform(ua string) string {
	ua = strings.ToLower(ua)
	switch {
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad"):
		return "iOS"
	case strings.Contains(ua, "android"):
		return "Android"
	case strings.Contains(ua, "macintosh") || strings.Contains(ua, "mac os") || strings.Contains(ua, "darwin"):
		return "macOS"
	case strings.Contains(ua, "windows"):
		return "Windows"
	case strings.Contains(ua, "linux"):
		return "Linux"
	case strings.Contains(ua, "codex_cli"):
		return "Codex CLI"
	case ua == "":
		return ""
	default:
		return "Other"
	}
}

// extractUsageFromSSE parses token usage from an SSE data payload.
func extractUsageFromSSE(data string) (inputTokens, outputTokens int64) {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(data), &obj); err != nil {
		return 0, 0
	}
	// Look for usage in response.completed events or top-level response object
	resp, _ := obj["response"].(map[string]interface{})
	if resp == nil {
		resp = obj
	}
	usage, _ := resp["usage"].(map[string]interface{})
	if usage == nil {
		return 0, 0
	}
	if v, ok := usage["input_tokens"].(float64); ok {
		inputTokens = int64(v)
	}
	if v, ok := usage["output_tokens"].(float64); ok {
		outputTokens = int64(v)
	}
	return
}

// knownUnsupportedParams are stripped before forwarding to avoid upstream errors.
var knownUnsupportedParams = map[string]bool{
	"max_output_tokens":      true,
	"prompt_cache_retention": true,
	"safety_identifier":      true,
}

// normalizeCodexBody normalises a Responses-API request body for the
// chatgpt.com/backend-api/codex backend which has a few quirks:
//   - input must be a JSON array of message objects, not a plain string
//   - instructions field must be present
//   - stream must be true (the backend only supports streaming)
//
// Returns the normalised body and whether streaming was force-enabled.
func normalizeCodexBody(body []byte) ([]byte, bool) {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return body, false
	}

	modified := false
	streamForced := false

	// Convert string input to message-list format
	if inputRaw, ok := data["input"]; ok {
		if text, ok := inputRaw.(string); ok {
			data["input"] = []interface{}{
				map[string]interface{}{
					"role": "user",
					"content": []interface{}{
						map[string]interface{}{"type": "input_text", "text": text},
					},
				},
			}
			modified = true
		}
	}

	// Add default instructions if missing
	if _, ok := data["instructions"]; !ok {
		data["instructions"] = "You are a helpful assistant."
		modified = true
	}

	// chatgpt.com Codex backend only supports streaming responses
	if v, ok := data["stream"]; !ok || v == false || v == nil {
		data["stream"] = true
		streamForced = true
		modified = true
	}

	// chatgpt.com Codex backend requires store = false
	if v, ok := data["store"]; !ok || v == true {
		data["store"] = false
		modified = true
	}

	// Strip unsupported params
	for param := range knownUnsupportedParams {
		if _, ok := data[param]; ok {
			delete(data, param)
			modified = true
		}
	}

	if !modified {
		return body, false
	}
	cleaned, err := json.Marshal(data)
	if err != nil {
		return body, false
	}
	return cleaned, streamForced
}

// cleanRequestBody is kept for compatibility; actual normalisation happens in normalizeCodexBody.
func cleanRequestBody(body []byte) []byte {
	cleaned, _ := normalizeCodexBody(body)
	return cleaned
}
