package proxy

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// NamespaceToolName represents a namespaced tool name for round-trip.
type NamespaceToolName struct {
	Namespace string
	Name      string
}

// NamespaceToolMap maps Chat Completions tool names to their namespace origin.
type NamespaceToolMap map[string]NamespaceToolName

// ToChatRequest converts a Responses API request to a Chat Completions request.
// history contains previous messages loaded from SessionStore.
func ToChatRequest(req *ResponsesRequest, history []ChatMessage, sessions *RelaySessionStore, modelMap map[string]string, defaultModel string, toolDenylist map[string]bool) (*ChatRequest, NamespaceToolMap, []ChatMessage, error) {
	var messages []ChatMessage
	if len(history) > 0 {
		messages = append(messages, history...)
	}

	// Prefer instructions (Codex CLI) over system (other clients)
	var systemText string
	if req.Instructions != nil && *req.Instructions != "" {
		systemText = *req.Instructions
	} else if req.System != nil && *req.System != "" {
		systemText = *req.System
	}
	if systemText != "" {
		if len(messages) == 0 || messages[0].Role != "system" {
			messages = append([]ChatMessage{{
				Role:    "system",
				Content: systemText,
			}}, messages...)
		}
	}

	// Build namespace tool map for response conversion
	nsMap := buildNamespaceToolMap(req.Tools)

	// Collect existing call_ids and tool_response call_ids from history for dedup
	existingCallIDs := make(map[string]bool)
	existingToolResponses := make(map[string]bool)
	for _, msg := range messages {
		if msg.ToolCalls != nil {
			for _, tc := range msg.ToolCalls {
				if m, ok := tc.(map[string]interface{}); ok {
					if id, ok := m["id"].(string); ok && id != "" {
						existingCallIDs[id] = true
					}
				}
			}
		}
		if msg.ToolCallID != nil {
			existingToolResponses[*msg.ToolCallID] = true
		}
	}

	// Parse input
	if str, ok := req.inputAsString(); ok {
		messages = append(messages, ChatMessage{
			Role:    "user",
			Content: str,
		})
	} else if arr, ok := req.inputAsArray(); ok {
		i := 0
		for i < len(arr) {
			item, ok := arr[i].(map[string]interface{})
			if !ok {
				i++
				continue
			}
			itemType := ""
			if t, ok := item["type"].(string); ok {
				itemType = t
			}

			if itemType == "function_call" {
				// Check for duplicate call_id
				callID := ""
				if c, ok := item["call_id"].(string); ok {
					callID = c
				}
				if existingCallIDs[callID] {
					i++
					continue
				}

				// Group consecutive function_call items into one assistant message
				var grouped []interface{}
				var reasoningContent *string
				for i < len(arr) {
					cur, ok := arr[i].(map[string]interface{})
					if !ok || getType(cur) != "function_call" {
						break
					}
					cID := getStr(cur, "call_id")
					name := responseFunctionNameForChat(cur)
					args := getStr(cur, "arguments")
					if args == "" {
						args = "{}"
					}
					if reasoningContent == nil {
						reasoningContent = sessions.GetReasoning(cID)
					}
					grouped = append(grouped, map[string]interface{}{
						"id":   cID,
						"type": "function",
						"function": map[string]interface{}{
							"name":      name,
							"arguments": args,
						},
					})
					i++
				}

				msg := ChatMessage{
					Role:      "assistant",
					Content:    nil,
					ToolCalls: grouped,
				}
				if reasoningContent != nil {
					msg.ReasoningContent = reasoningContent
				}
				// Fallback: try turn-level fingerprint
				if msg.ReasoningContent == nil {
					msg.ReasoningContent = sessions.GetTurnReasoning(messages, &msg)
				}
				messages = append(messages, msg)
			} else {
				switch itemType {
				case "function_call_output":
					callID := getStr(item, "call_id")
					if existingToolResponses[callID] {
						i++
						continue
					}
					output := getStr(item, "output")
					messages = append(messages, ChatMessage{
						Role:       "tool",
						Content:    output,
						ToolCallID: &callID,
					})

				case "reasoning":
					// Skip reasoning items — handled by session store

				default:
					role := getStr(item, "role")
					if role == "developer" {
						role = "system"
					}
					if role == "" {
						role = "user"
					}
					msg := ChatMessage{
						Role:    role,
						Content: valueToChatContent(item["content"]),
					}
					// Try to recover reasoning_content for assistant messages
					if role == "assistant" {
						msg.ReasoningContent = sessions.GetTurnReasoning(messages, &msg)
					}
					// System messages must go to front
					if role == "system" {
						if len(messages) > 0 && messages[0].Role == "system" {
							messages[0] = msg
						} else {
							messages = append([]ChatMessage{msg}, messages...)
						}
					} else {
						messages = append(messages, msg)
					}
				}
				i++
			}
		}
	}

	// Map model name (with default model support)
	model := MapModelName(req.Model, modelMap, defaultModel)

	// Convert tools
	tools := convertTools(req.Tools, toolDenylist)

	chatReq := &ChatRequest{
		Model:             model,
		Messages:            messages,
		Tools:               tools,
		ToolChoice:          req.ToolChoice,
		ParallelToolCalls:   req.ParallelToolCalls,
		Stream:              req.Stream,
	}
	if req.Temperature != nil {
		chatReq.Temperature = req.Temperature
	}
	if req.MaxOutputTokens != nil {
		v := *req.MaxOutputTokens
		chatReq.MaxTokens = &v
	}
	if req.Stream {
		chatReq.StreamOptions = &ChatStreamOptions{IncludeUsage: true}
	}

	for i := range chatReq.Messages {
		normalizeAssistantMessageContent(&chatReq.Messages[i])
	}

	return chatReq, nsMap, messages, nil
}

// FromChatResponse converts a Chat Completions response to a Responses API response.
func FromChatResponse(id string, model string, chat ChatResponse, nsMap NamespaceToolMap) (ResponsesResponse, []ChatMessage) {
	var output []interface{}
	var outputMessages []ChatMessage

	if len(chat.Choices) > 0 {
		choice := chat.Choices[0]
		text := choice.Message.TextContent()
		if text != "" || choice.Message.ToolCalls == nil {
			output = append(output, map[string]interface{}{
				"type":       "message",
				"role":       "assistant",
				"status":     "completed",
				"content": []interface{}{
					map[string]interface{}{
						"type": "output_text",
						"text": text,
					},
				},
			})
		}

		if choice.Message.ToolCalls != nil {
			for _, tc := range choice.Message.ToolCalls {
				tcMap, ok := tc.(map[string]interface{})
				if !ok {
					continue
				}
				funcMap, _ := tcMap["function"].(map[string]interface{})
				rawName := ""
				if funcMap != nil {
					rawName, _ = funcMap["name"].(string)
				}
				arguments := ""
				if funcMap != nil {
					arguments, _ = funcMap["arguments"].(string)
				}
				callID := ""
				if id, ok := tcMap["id"].(string); ok {
					callID = id
				}

				ns, name := responseFunctionNameForResponses(rawName, nsMap)
				item := map[string]interface{}{
					"type":       "function_call",
					"id":         fmt.Sprintf("fc_%s", uuid.New().String()[:8]),
					"call_id":    callID,
					"name":       name,
					"arguments":  arguments,
					"status":     "completed",
				}
				if ns != "" {
					item["namespace"] = ns
				}
				output = append(output, item)
			}
		}

		outputMessages = []ChatMessage{choice.Message}
	}

	usage := chat.Usage
	if usage == nil {
		usage = &ChatUsage{}
	}

	resp := ResponsesResponse{
		ID:     id,
		Object: "response",
		Model:  model,
		Output: output,
		Usage: ResponsesUsage{
			InputTokens:  usage.PromptTokens,
			OutputTokens: usage.CompletionTokens,
			TotalTokens:  usage.TotalTokens,
			InputTokensDetails: &InputTokensDetails{
				CachedTokens: usage.CacheHit(),
			},
		},
	}

	return resp, outputMessages
}

// ── Tool conversion ─────────────────────────────────────────────────────────────

func buildNamespaceToolMap(tools []interface{}) NamespaceToolMap {
	nsMap := make(NamespaceToolMap)
	for _, tool := range tools {
		t, ok := tool.(map[string]interface{})
		if !ok {
			continue
		}
		if getType(t) != "namespace" {
			continue
		}
		namespace := getStr(t, "name")
		subs, _ := t["tools"].([]interface{})
		for _, sub := range subs {
			subMap, ok := sub.(map[string]interface{})
			if !ok || getType(subMap) != "function" {
				continue
			}
			name := getStr(subMap, "name")
			chatName := chatFunctionNameForNamespaceTool(namespace, name)
			nsMap[chatName] = NamespaceToolName{
				Namespace: namespace,
				Name:      name,
			}
		}
	}
	return nsMap
}

func convertTools(tools []interface{}, denylist map[string]bool) []interface{} {
	var out []interface{}
	for _, tool := range tools {
		t, ok := tool.(map[string]interface{})
		if !ok {
			continue
		}
		switch getType(t) {
		case "function":
			if !toolIsDenied(t, nil, denylist) {
				out = append(out, convertTool(t))
			}
		case "namespace":
			namespace := getStr(t, "name")
			subs, _ := t["tools"].([]interface{})
			for _, sub := range subs {
				subMap, ok := sub.(map[string]interface{})
				if !ok || getType(subMap) != "function" {
					continue
				}
				name := chatFunctionNameForNamespaceTool(namespace, getStr(subMap, "name"))
				if !toolIsDenied(subMap, &name, denylist) {
					out = append(out, convertToolWithName(subMap, &name))
				}
			}
		default:
			// Drop unsupported tool types (web_search, image_generation, etc.)
		}
	}
	return out
}

func toolIsDenied(tool interface{}, overrideName *string, denylist map[string]bool) bool {
	if len(denylist) == 0 {
		return false
	}
	var name string
	if overrideName != nil {
		name = *overrideName
	} else {
		t, ok := tool.(map[string]interface{})
		if !ok {
			return false
		}
		if funcMap, ok := t["function"].(map[string]interface{}); ok {
			name, _ = funcMap["name"].(string)
		} else {
			name = getStr(t, "name")
		}
	}
	return denylist[name]
}

func convertTool(tool map[string]interface{}) map[string]interface{} {
	return convertToolWithName(tool, nil)
}

func convertToolWithName(tool map[string]interface{}, overrideName *string) map[string]interface{} {
	// Already in Chat Completions format
	if _, ok := tool["function"]; ok {
		result := make(map[string]interface{})
		for k, v := range tool {
			result[k] = v
		}
		if overrideName != nil {
			if funcMap, ok := result["function"].(map[string]interface{}); ok {
				funcMap["name"] = *overrideName
			}
		}
		return result
	}

	// Convert from Responses API flat format
	if getType(tool) == "function" {
		funcMap := make(map[string]interface{})
		if overrideName != nil {
			funcMap["name"] = *overrideName
		} else if v, ok := tool["name"]; ok {
			funcMap["name"] = v
		}
		if v, ok := tool["description"]; ok {
			funcMap["description"] = v
		}
		if v, ok := tool["parameters"]; ok {
			funcMap["parameters"] = v
		}
		if v, ok := tool["strict"]; ok {
			funcMap["strict"] = v
		}
		return map[string]interface{}{
			"type":     "function",
			"function": funcMap,
		}
	}
	return tool
}

func responseFunctionNameForChat(item map[string]interface{}) string {
	name := getStr(item, "name")
	namespace := getStr(item, "namespace")
	if namespace != "" {
		return chatFunctionNameForNamespaceTool(namespace, name)
	}
	return name
}

func chatFunctionNameForNamespaceTool(namespace, name string) string {
	return fmt.Sprintf("%s-%s", namespace, name)
}

func responseFunctionNameForResponses(name string, nsMap NamespaceToolMap) (namespace, finalName string) {
	if t, ok := nsMap[name]; ok {
		return t.Namespace, t.Name
	}
	// Try splitting by "." (legacy format)
	if idx := strings.Index(name, "."); idx >= 0 {
		return name[:idx], name[idx+1:]
	}
	// Try splitting by "__" (mcp__ format)
	if idx := strings.Index(name, "__"); idx >= 0 {
		return name[:idx], name[idx+2:]
	}
	return "", name
}

// ── Model name mapping ──────────────────────────────────────────────────────────

// MapModelName maps model names using the provided map.
// defaultModel is used when the incoming model name is empty, or when the name
// looks like a Codex/OpenAI placeholder (gpt-*, codex-*, o1/o3/o4*) and no explicit mapping exists.
// A "*" entry in modelMap acts as a catch-all fallback.
func MapModelName(name string, modelMap map[string]string, defaultModel string) string {
	if name == "" {
		if defaultModel != "" {
			return defaultModel
		}
		return name
	}
	if mapped, ok := modelMap[name]; ok {
		return mapped
	}
	if wildcard, ok := modelMap["*"]; ok && wildcard != "" {
		return wildcard
	}
	if defaultModel != "" && isCodexPlaceholderModel(name) {
		return defaultModel
	}
	return name
}

func isCodexPlaceholderModel(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	return strings.HasPrefix(lower, "gpt-") ||
		strings.HasPrefix(lower, "codex-") ||
		strings.HasPrefix(lower, "o1") ||
		strings.HasPrefix(lower, "o3") ||
		strings.HasPrefix(lower, "o4")
}

// PreferredCodexModel picks the Codex-facing model name for ~/.codex/config.toml injection.
// It prefers mapped gpt-* keys (e.g. gpt-5.5) over upstream default_model values (e.g. mimo-v2.5-pro).
func PreferredCodexModel(modelMap map[string]string, defaultModel string) string {
	for _, preferred := range []string{"gpt-5.5", "gpt-5.4", "gpt-5-codex", "gpt-5", "gpt-5.1-codex-max"} {
		if _, ok := modelMap[preferred]; ok {
			return preferred
		}
	}
	var best string
	for key := range modelMap {
		if key == "*" {
			continue
		}
		if isCodexPlaceholderModel(key) && (best == "" || key > best) {
			best = key
		}
	}
	if best != "" {
		return best
	}
	if isCodexPlaceholderModel(defaultModel) {
		return defaultModel
	}
	return "gpt-5-codex"
}

// ── Content conversion ─────────────────────────────────────────────────────────

func valueToChatContent(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	// String content
	if s, ok := v.(string); ok {
		return s
	}
	// Array of content parts
	if arr, ok := v.([]interface{}); ok {
		hasNonText := false
		for _, p := range arr {
			if m, ok := p.(map[string]interface{}); ok {
				t := getType(m)
				if t != "input_text" && t != "text" && t != "output_text" {
					hasNonText = true
					break
				}
			}
		}
		if !hasNonText {
			// Collapse to string
			var b strings.Builder
			for _, p := range arr {
				if m, ok := p.(map[string]interface{}); ok {
					if text, ok := m["text"].(string); ok {
						b.WriteString(text)
					}
				}
			}
			return b.String()
		}
		// Map to Chat Completions content parts
		var mapped []interface{}
		for _, p := range arr {
			mapped = append(mapped, mapContentPart(p))
		}
		return mapped
	}
	return v
}

func mapContentPart(part interface{}) interface{} {
	m, ok := part.(map[string]interface{})
	if !ok {
		return part
	}
	kind := getType(m)
	switch kind {
	case "input_text", "text", "output_text":
		text := getStr(m, "text")
		return map[string]interface{}{
			"type": "text",
			"text": text,
		}
	case "input_image":
		url := getStr(m, "image_url")
		return map[string]interface{}{
			"type": "image_url",
			"image_url": map[string]interface{}{
				"url": url,
			},
		}
	case "image_url":
		inner := m["image_url"]
		// If already an object, keep it; if string, wrap
		if _, ok := inner.(map[string]interface{}); ok {
			return map[string]interface{}{
				"type":     "image_url",
				"image_url": inner,
			}
		}
		url := ""
		if s, ok := inner.(string); ok {
			url = s
		}
		return map[string]interface{}{
			"type": "image_url",
			"image_url": map[string]interface{}{
				"url": url,
			},
		}
	default:
		return part
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func getType(m map[string]interface{}) string {
	if t, ok := m["type"].(string); ok {
		return t
	}
	return ""
}

func getStr(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

