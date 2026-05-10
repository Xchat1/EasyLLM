// Package antigravity implements the Antigravity platform wakeup execution logic.
// Logic ported from cockpit-tools-main/src-tauri/src/modules/wakeup.rs.
package antigravity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// ---- constants (mirrors wakeup.rs) ----

var cloudCodeBaseURLs = []string{
	"https://cloudcode-pa.googleapis.com",
	"https://daily-cloudcode-pa.googleapis.com",
	"https://daily-cloudcode-pa.sandbox.googleapis.com",
}

const (
	streamPath              = "/v1internal:streamGenerateContent?alt=sse"
	wakeupUserAgent         = "antigravity"
	defaultAttempts         = 2
	backoffBaseMS           = 500
	backoffMaxMS            = 4000
	antigravitySystemPrompt = "You are Antigravity, a powerful agentic AI coding assistant designed by the Google Deepmind team working on Advanced Agentic Coding.You are pair programming with a USER to solve their coding task. The task may require creating a new codebase, modifying or debugging an existing codebase, or simply answering a question.**Absolute paths only****Proactiveness**"
)

// ---- public types ----

// WakeupRequest is the input for a wakeup call.
type WakeupRequest struct {
	ProjectID       string `json:"project_id"`
	Model           string `json:"model"`
	Prompt          string `json:"prompt"`
	MaxOutputTokens uint32 `json:"max_output_tokens,omitempty"`
}

// WakeupResponse mirrors WakeupResponse in wakeup.rs.
type WakeupResponse struct {
	Reply            string  `json:"reply"`
	PromptTokens     *uint32 `json:"prompt_tokens,omitempty"`
	CompletionTokens *uint32 `json:"completion_tokens,omitempty"`
	TotalTokens      *uint32 `json:"total_tokens,omitempty"`
	TraceID          *string `json:"trace_id,omitempty"`
	ResponseID       *string `json:"response_id,omitempty"`
	DurationMS       int64   `json:"duration_ms"`
}

// ---- request building (mirrors build_request_body) ----

func generateID(prefix string) string {
	ts := time.Now().UnixMilli()
	suffix := make([]byte, 6)
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	for i := range suffix {
		suffix[i] = charset[rand.Intn(len(charset))]
	}
	return fmt.Sprintf("%s_%d_%s", prefix, ts, string(suffix))
}

func buildRequestBody(projectID, model, prompt string, maxOutputTokens uint32) map[string]interface{} {
	generationConfig := map[string]interface{}{"temperature": 0}
	if maxOutputTokens > 0 {
		generationConfig["maxOutputTokens"] = maxOutputTokens
	}
	return map[string]interface{}{
		"project":     projectID,
		"requestId":   generateID("req"),
		"model":       model,
		"userAgent":   wakeupUserAgent,
		"requestType": "agent",
		"request": map[string]interface{}{
			"contents": []interface{}{
				map[string]interface{}{
					"role":  "user",
					"parts": []interface{}{map[string]interface{}{"text": prompt}},
				},
			},
			"session_id": generateID("sess"),
			"systemInstruction": map[string]interface{}{
				"parts": []interface{}{map[string]interface{}{"text": antigravitySystemPrompt}},
			},
			"generationConfig": generationConfig,
		},
	}
}

// ---- backoff (mirrors get_backoff_delay_ms) ----

func getBackoffDelayMS(attempt int) int64 {
	if attempt < 2 {
		return 0
	}
	raw := int64(backoffBaseMS) * int64(math.Pow(2, float64(attempt-2)))
	jitter := rand.Int63n(100)
	if raw+jitter > backoffMaxMS {
		return backoffMaxMS
	}
	return raw + jitter
}

// ---- stream parsing (mirrors parse_stream_result) ----

type streamParseResult struct {
	Reply            string
	PromptTokens     *uint32
	CompletionTokens *uint32
	TotalTokens      *uint32
	TraceID          *string
	ResponseID       *string
}

func processStreamObject(obj map[string]interface{}, result *streamParseResult, parts *[]string) {
	// extract text from candidates
	var candidate map[string]interface{}
	if resp, ok := obj["response"].(map[string]interface{}); ok {
		if cands, ok := resp["candidates"].([]interface{}); ok && len(cands) > 0 {
			candidate, _ = cands[0].(map[string]interface{})
		}
	}
	if candidate == nil {
		if cands, ok := obj["candidates"].([]interface{}); ok && len(cands) > 0 {
			candidate, _ = cands[0].(map[string]interface{})
		}
	}
	if candidate != nil {
		if content, ok := candidate["content"].(map[string]interface{}); ok {
			if ps, ok := content["parts"].([]interface{}); ok {
				for _, p := range ps {
					if pm, ok := p.(map[string]interface{}); ok {
						if thought, _ := pm["thought"].(bool); thought {
							continue
						}
						if text, ok := pm["text"].(string); ok && text != "" {
							*parts = append(*parts, text)
						}
					}
				}
			}
		}
	}

	// usage metadata
	var usage map[string]interface{}
	if resp, ok := obj["response"].(map[string]interface{}); ok {
		usage, _ = resp["usageMetadata"].(map[string]interface{})
	}
	if usage == nil {
		usage, _ = obj["usageMetadata"].(map[string]interface{})
	}
	if usage != nil {
		if result.PromptTokens == nil {
			if v := uint32FromInterface(usage["promptTokenCount"]); v != nil {
				result.PromptTokens = v
			}
		}
		if result.CompletionTokens == nil {
			if v := uint32FromInterface(usage["candidatesTokenCount"]); v != nil {
				result.CompletionTokens = v
			}
		}
		if result.TotalTokens == nil {
			if v := uint32FromInterface(usage["totalTokenCount"]); v != nil {
				result.TotalTokens = v
			}
		}
	}

	// trace/response IDs
	if result.TraceID == nil {
		if v, ok := obj["traceId"].(string); ok && v != "" {
			result.TraceID = &v
		}
	}
	if result.ResponseID == nil {
		var rid string
		if resp, ok := obj["response"].(map[string]interface{}); ok {
			rid, _ = resp["responseId"].(string)
		}
		if rid == "" {
			rid, _ = obj["responseId"].(string)
		}
		if rid != "" {
			result.ResponseID = &rid
		}
	}
}

func parseStreamResult(text string) (*streamParseResult, error) {
	result := &streamParseResult{}
	var parts []string
	gotEvent := false
	var lastData map[string]interface{}

	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		var payload string
		if strings.HasPrefix(trimmed, "data:") {
			payload = strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
			if payload == "" || payload == "[DONE]" {
				continue
			}
		} else if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
			payload = trimmed
		} else {
			continue
		}

		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(payload), &obj); err == nil {
			gotEvent = true
			processStreamObject(obj, result, &parts)
			lastData = obj
		}
	}

	if !gotEvent {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(text), &obj); err == nil {
			gotEvent = true
			processStreamObject(obj, result, &parts)
		}
	}

	if !gotEvent {
		return nil, fmt.Errorf("cloud code stream received no data")
	}

	if len(parts) == 0 && lastData != nil {
		processStreamObject(lastData, result, &parts)
	}

	if len(parts) == 0 {
		result.Reply = "(无回复)"
	} else {
		result.Reply = strings.Join(parts, "")
	}
	if result.CompletionTokens == nil {
		zero := uint32(0)
		result.CompletionTokens = &zero
	}
	return result, nil
}

// ---- HTTP stream request (mirrors send_stream_request) ----

func sendStreamRequest(ctx context.Context, accessToken string, body map[string]interface{}) (*streamParseResult, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal wakeup body: %w", err)
	}

	var lastErr error
	for _, base := range cloudCodeBaseURLs {
		for attempt := 1; attempt <= defaultAttempts; attempt++ {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("唤醒测试已取消")
			default:
			}

			url := base + streamPath
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
			if err != nil {
				lastErr = err
				continue
			}
			req.Header.Set("Authorization", "Bearer "+accessToken)
			req.Header.Set("User-Agent", wakeupUserAgent)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept-Encoding", "gzip")

			client := &http.Client{Timeout: 60 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				lastErr = fmt.Errorf("wakeup request failed: %w", err)
				if attempt < defaultAttempts {
					delay := getBackoffDelayMS(attempt + 1)
					if delay > 0 {
						time.Sleep(time.Duration(delay) * time.Millisecond)
					}
				}
				continue
			}

			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			if resp.StatusCode == http.StatusUnauthorized {
				return nil, fmt.Errorf("authorization expired")
			}
			if resp.StatusCode == http.StatusForbidden {
				return nil, fmt.Errorf("cloud code access forbidden")
			}
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				parsed, err := parseStreamResult(string(respBody))
				if err != nil {
					lastErr = err
					if attempt < defaultAttempts {
						delay := getBackoffDelayMS(attempt + 1)
						if delay > 0 {
							time.Sleep(time.Duration(delay) * time.Millisecond)
						}
					}
					continue
				}
				return parsed, nil
			}

			retryable := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
			lastErr = fmt.Errorf("wakeup request failed: %d", resp.StatusCode)
			if retryable && attempt < defaultAttempts {
				delay := getBackoffDelayMS(attempt + 1)
				if delay > 0 {
					time.Sleep(time.Duration(delay) * time.Millisecond)
				}
				continue
			}
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("wakeup request failed: all endpoints exhausted")
}

// ---- public API ----

// ExecuteWakeup sends a wakeup prompt to the Antigravity Cloud Code API.
// Mirrors the core logic of wakeup_with_account in wakeup.rs.
func ExecuteWakeup(ctx context.Context, accessToken string, req WakeupRequest) (*WakeupResponse, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("access_token is required for wakeup")
	}
	if req.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required for wakeup")
	}
	if req.Model == "" {
		req.Model = "gemini-2.0-flash-001"
	}
	if req.Prompt == "" {
		req.Prompt = "Hello"
	}

	start := time.Now()
	body := buildRequestBody(req.ProjectID, req.Model, req.Prompt, req.MaxOutputTokens)
	parsed, err := sendStreamRequest(ctx, accessToken, body)
	if err != nil {
		return nil, err
	}

	return &WakeupResponse{
		Reply:            parsed.Reply,
		PromptTokens:     parsed.PromptTokens,
		CompletionTokens: parsed.CompletionTokens,
		TotalTokens:      parsed.TotalTokens,
		TraceID:          parsed.TraceID,
		ResponseID:       parsed.ResponseID,
		DurationMS:       time.Since(start).Milliseconds(),
	}, nil
}

// ---- helpers ----

func uint32FromInterface(v interface{}) *uint32 {
	switch n := v.(type) {
	case float64:
		u := uint32(n)
		return &u
	case json.Number:
		if i, err := n.Int64(); err == nil {
			u := uint32(i)
			return &u
		}
	}
	return nil
}
