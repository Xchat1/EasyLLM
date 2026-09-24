package antigravity

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"easyllm/internal/models"
)

const (
	AntigravityStreamPath   = "/v1internal:streamGenerateContent?alt=sse"
	AntigravityWakeupUA     = "antigravity"
	AntigravitySystemPrompt = "You are Antigravity, a powerful agentic AI coding assistant designed by the Google Deepmind team working on Advanced Agentic Coding.You are pair programming with a USER to solve their coding task. The task may require creating a new codebase, modifying or debugging an existing codebase, or simply answering a question.**Absolute paths only****Proactiveness**"
)

func randomSuffix(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

func generateSessionID() string {
	return fmt.Sprintf("sess_%d_%s", time.Now().UnixMilli(), randomSuffix(6))
}

func generateRequestID() string {
	return fmt.Sprintf("req_%d_%s", time.Now().UnixMilli(), randomSuffix(6))
}

// ExecuteWakeup executes a wakeup request against the Antigravity PA endpoint.
func ExecuteWakeup(ctx context.Context, account *models.AntigravityAccount, req models.AntigravityWakeupRequest) (*models.AntigravityWakeupResponse, error) {
	if account.AccessToken == "" {
		return nil, fmt.Errorf("账号 Access Token 为空")
	}

	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = "gemini-2.5-flash"
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		prompt = "ping"
	}
	maxOutputTokens := req.MaxOutputTokens
	if maxOutputTokens == 0 {
		maxOutputTokens = 64
	}

	projectID := ""
	if account.ProjectID != nil {
		projectID = *account.ProjectID
	}

	requestID := generateRequestID()
	sessionID := generateSessionID()

	reqBody := map[string]interface{}{
		"project":     projectID,
		"requestId":   requestID,
		"model":       model,
		"userAgent":   AntigravityWakeupUA,
		"requestType": "agent",
		"request": map[string]interface{}{
			"contents": []map[string]interface{}{
				{
					"role": "user",
					"parts": []map[string]interface{}{
						{"text": prompt},
					},
				},
			},
			"session_id": sessionID,
			"systemInstruction": map[string]interface{}{
				"parts": []map[string]interface{}{
					{"text": AntigravitySystemPrompt},
				},
			},
			"generationConfig": map[string]interface{}{
				"temperature":     0,
				"maxOutputTokens": maxOutputTokens,
			},
		},
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("构建唤醒请求失败: %w", err)
	}

	projID := ""
	if account.ProjectID != nil {
		projID = *account.ProjectID
	}
	baseURL := resolveCloudCodeBaseURL(account.IsGcpTos, projID)
	url := baseURL + AntigravityStreamPath

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+account.AccessToken)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", AntigravityWakeupUA)
	httpReq.Header.Set("Accept", "text/event-stream")

	startTime := time.Now()
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("唤醒请求网络错误: %w", err)
	}
	defer resp.Body.Close()

	durationMS := time.Since(startTime).Milliseconds()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("唤醒请求失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	scanner := bufio.NewScanner(resp.Body)
	// Buffer up to 1MB lines
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var replyBuilder strings.Builder
	var promptTokens *uint32
	var completionTokens *uint32
	var totalTokens *uint32
	var responseID *string

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") && !strings.HasPrefix(line, "data:") {
			continue
		}
		dataStr := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if dataStr == "" || dataStr == "[DONE]" {
			continue
		}

		var chunk map[string]interface{}
		if err := json.Unmarshal([]byte(dataStr), &chunk); err != nil {
			continue
		}

		// Parse candidate parts
		var candidate map[string]interface{}
		if respObj, ok := chunk["response"].(map[string]interface{}); ok {
			if candidates, ok := respObj["candidates"].([]interface{}); ok && len(candidates) > 0 {
				candidate, _ = candidates[0].(map[string]interface{})
			}
			if respID, ok := respObj["responseId"].(string); ok && respID != "" {
				responseID = &respID
			}
		} else if candidates, ok := chunk["candidates"].([]interface{}); ok && len(candidates) > 0 {
			candidate, _ = candidates[0].(map[string]interface{})
		}

		if candidate != nil {
			if content, ok := candidate["content"].(map[string]interface{}); ok {
				if parts, ok := content["parts"].([]interface{}); ok {
					for _, p := range parts {
						if partMap, ok := p.(map[string]interface{}); ok {
							if text, ok := partMap["text"].(string); ok {
								replyBuilder.WriteString(text)
							}
						}
					}
				}
			}
		}

		// Parse token usage metadata
		var usage map[string]interface{}
		if u, ok := chunk["usageMetadata"].(map[string]interface{}); ok {
			usage = u
		} else if respObj, ok := chunk["response"].(map[string]interface{}); ok {
			if u, ok := respObj["usageMetadata"].(map[string]interface{}); ok {
				usage = u
			}
		}

		if usage != nil {
			if pt, ok := usage["promptTokenCount"].(float64); ok {
				v := uint32(pt)
				promptTokens = &v
			}
			if ct, ok := usage["candidatesTokenCount"].(float64); ok {
				v := uint32(ct)
				completionTokens = &v
			}
			if tt, ok := usage["totalTokenCount"].(float64); ok {
				v := uint32(tt)
				totalTokens = &v
			}
		}
	}

	reply := replyBuilder.String()
	if reply == "" {
		reply = "(无文本回复，但请求成功)"
	}

	return &models.AntigravityWakeupResponse{
		Reply:            reply,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		DurationMS:       durationMS,
		ResponseID:       responseID,
	}, nil
}
