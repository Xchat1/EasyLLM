package proxy

import (
	"testing"
)

func TestTextInputBecomesUserMessage(t *testing.T) {
	sessions := NewRelaySessionStore()
	req := &ResponsesRequest{
		Model: "test",
		Input: []byte(`"hello"`),
	}

	chatReq, _, _, err := ToChatRequest(req, nil, sessions, nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chatReq.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(chatReq.Messages))
	}
	if chatReq.Messages[0].Role != "user" {
		t.Errorf("expected role 'user', got '%s'", chatReq.Messages[0].Role)
	}
	if chatReq.Messages[0].TextContent() != "hello" {
		t.Errorf("expected content 'hello', got '%s'", chatReq.Messages[0].TextContent())
	}
}

func TestSystemPromptFromInstructions(t *testing.T) {
	sessions := NewRelaySessionStore()
	instructions := "be helpful"
	req := &ResponsesRequest{
		Model:        "test",
		Input:        []byte(`"hi"`),
		Instructions: &instructions,
	}

	chatReq, _, _, err := ToChatRequest(req, nil, sessions, nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chatReq.Messages) < 1 {
		t.Fatal("expected at least 1 message")
	}
	if chatReq.Messages[0].Role != "system" {
		t.Errorf("expected role 'system', got '%s'", chatReq.Messages[0].Role)
	}
	if chatReq.Messages[0].TextContent() != "be helpful" {
		t.Errorf("expected content 'be helpful', got '%s'", chatReq.Messages[0].TextContent())
	}
}

func TestDeveloperRoleMappedToSystem(t *testing.T) {
	sessions := NewRelaySessionStore()
	req := &ResponsesRequest{
		Model: "test",
		Input: []byte(`[{"type":"message","role":"developer","content":"secret instructions"}]`),
	}

	chatReq, _, _, err := ToChatRequest(req, nil, sessions, nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chatReq.Messages) < 1 {
		t.Fatal("expected at least 1 message")
	}
	if chatReq.Messages[0].Role != "system" {
		t.Errorf("expected role 'system', got '%s'", chatReq.Messages[0].Role)
	}
}

func TestFunctionCallGrouping(t *testing.T) {
	sessions := NewRelaySessionStore()
	req := &ResponsesRequest{
		Model: "test",
		Input: []byte(`[{"type":"function_call","call_id":"c1","name":"fn_a","arguments":"{}"},{"type":"function_call","call_id":"c2","name":"fn_b","arguments":"{}"}]`),
	}

	chatReq, _, _, err := ToChatRequest(req, nil, sessions, nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have 1 assistant message with 2 tool calls
	if len(chatReq.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(chatReq.Messages))
	}
	if chatReq.Messages[0].Role != "assistant" {
		t.Errorf("expected role 'assistant', got '%s'", chatReq.Messages[0].Role)
	}
	if chatReq.Messages[0].ToolCalls == nil {
		t.Fatal("expected tool calls")
	}
	if len(chatReq.Messages[0].ToolCalls) != 2 {
		t.Errorf("expected 2 tool calls, got %d", len(chatReq.Messages[0].ToolCalls))
	}
}

func TestFunctionCallOutputBecomesToolMessage(t *testing.T) {
	sessions := NewRelaySessionStore()
	req := &ResponsesRequest{
		Model: "test",
		Input: []byte(`[{"type":"function_call_output","call_id":"c1","output":"result"}]`),
	}

	chatReq, _, _, err := ToChatRequest(req, nil, sessions, nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, msg := range chatReq.Messages {
		if msg.Role == "tool" {
			found = true
			if msg.ToolCallID == nil || *msg.ToolCallID != "c1" {
				t.Errorf("expected tool_call_id 'c1', got '%s'", *msg.ToolCallID)
			}
		}
	}
	if !found {
		t.Error("expected a tool message")
	}
}

func TestConvertToolFlatToNested(t *testing.T) {
	flat := map[string]interface{}{
		"type":     "function",
		"name":     "my_fn",
		"description": "does stuff",
		"parameters":  map[string]interface{}{"type": "object"},
	}

	nested := convertTool(flat)
	if getType(nested) != "function" {
		t.Errorf("expected type 'function'")
	}
	funcMap, ok := nested["function"].(map[string]interface{})
	if !ok {
		t.Fatal("expected function to be a map")
	}
	if funcMap["name"] != "my_fn" {
		t.Errorf("expected name 'my_fn'")
	}
}

func TestConvertToolAlreadyNested(t *testing.T) {
	already := map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "my_fn",
			"description": "does stuff",
		},
	}

	result := convertTool(already)
	if result["type"] != "function" {
		t.Error("expected type to be preserved")
	}
}

func TestPreferredCodexModel(t *testing.T) {
	tests := []struct {
		name         string
		modelMap     map[string]string
		defaultModel string
		expected     string
	}{
		{
			name: "prefer gpt-5.5 from map",
			modelMap: map[string]string{
				"gpt-5.4": "mimo-v2.5",
				"gpt-5.5": "mimo-v2.5-pro",
			},
			defaultModel: "mimo-v2.5-pro",
			expected:     "gpt-5.5",
		},
		{
			name:         "fallback to codex placeholder default",
			modelMap:     nil,
			defaultModel: "gpt-5.4",
			expected:     "gpt-5.4",
		},
		{
			name:         "upstream default without map",
			modelMap:     nil,
			defaultModel: "mimo-v2.5-pro",
			expected:     "gpt-5-codex",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PreferredCodexModel(tt.modelMap, tt.defaultModel); got != tt.expected {
				t.Errorf("PreferredCodexModel() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestMapModelName(t *testing.T) {
	modelMap := map[string]string{
		"gpt-5.4":        "deepseek-v4-pro",
		"gpt-5.5":        "deepseek-v4-pro",
	}

	tests := []struct {
		input         string
		modelMap      map[string]string
		defaultModel  string
		expected      string
	}{
		{"gpt-5.4", modelMap, "", "deepseek-v4-pro"},
		{"gpt-5.5", modelMap, "", "deepseek-v4-pro"},
		{"unknown-model", modelMap, "", "unknown-model"},
		{"unknown-model", modelMap, "gpt-5.5", "unknown-model"},
		{"gpt-5.5", nil, "mimo-v2.5-pro", "mimo-v2.5-pro"},
		{"gpt-5.5", map[string]string{"gpt-5.5": "custom"}, "mimo-v2.5-pro", "custom"},
		{"", nil, "gpt-5.5", "gpt-5.5"},
	}

	for _, tt := range tests {
		result := MapModelName(tt.input, tt.modelMap, tt.defaultModel)
		if result != tt.expected {
			t.Errorf("MapModelName(%s, %v, %s) = %s, expected %s", tt.input, tt.modelMap, tt.defaultModel, result, tt.expected)
		}
	}
}

func TestParseModelMap(t *testing.T) {
	// Test JSON format
	jsonStr := `{"gpt-5.4":"deepseek-v4-pro","gpt-5.5":"deepseek-v4-flash"}`
	result := ParseModelMap(jsonStr)
	if result["gpt-5.4"] != "deepseek-v4-pro" {
		t.Errorf("expected 'deepseek-v4-pro', got '%s'", result["gpt-5.4"])
	}

	// Test comma format
	commaStr := "gpt-5.4:deepseek-v4-pro, gpt-5.5:deepseek-v4-flash"
	result2 := ParseModelMap(commaStr)
	if result2["gpt-5.4"] != "deepseek-v4-pro" {
		t.Errorf("expected 'deepseek-v4-pro', got '%s'", result2["gpt-5.4"])
	}

	// Test empty
	empty := ParseModelMap("")
	if empty != nil && len(empty) > 0 {
		t.Error("expected nil or empty map for empty input")
	}
}

func TestParseToolDenylist(t *testing.T) {
	input := "web_search, image_generation, exec_command"
	result := ParseToolDenylist(input)
	if !result["web_search"] {
		t.Error("expected 'web_search' to be in denylist")
	}
	if !result["image_generation"] {
		t.Error("expected 'image_generation' to be in denylist")
	}
	if !result["exec_command"] {
		t.Error("expected 'exec_command' to be in denylist")
	}
	if len(result) != 3 {
		t.Errorf("expected 3 items, got %d", len(result))
	}
}

func TestValueToChatContentTextString(t *testing.T) {
	// Plain string should stay as string
	result := valueToChatContent("plain text")
	if s, ok := result.(string); !ok || s != "plain text" {
		t.Errorf("expected string 'plain text', got %v", result)
	}
}

func TestValueToChatContentPartsArray(t *testing.T) {
	// Text-only parts should collapse to string
	parts := []interface{}{
		map[string]interface{}{"type": "input_text", "text": "hello "},
		map[string]interface{}{"type": "input_text", "text": "world"},
	}
	result := valueToChatContent(parts)
	if s, ok := result.(string); !ok || s != "hello world" {
		t.Errorf("expected string 'hello world', got %v", result)
	}
}

func TestValueToChatContentMultimodal(t *testing.T) {
	// Parts with non-text should stay as array
	parts := []interface{}{
		map[string]interface{}{"type": "input_text", "text": "what is this?"},
		map[string]interface{}{"type": "input_image", "image_url": "data:image/png;base64,AAA"},
	}
	result := valueToChatContent(parts)
	if arr, ok := result.([]interface{}); !ok {
		t.Errorf("expected array for multimodal content, got %T", result)
	} else if len(arr) != 2 {
		t.Errorf("expected 2 parts, got %d", len(arr))
	}
}

func TestSkipDuplicateFunctionCallFromHistory(t *testing.T) {
	sessions := NewRelaySessionStore()

	// Simulate history: user -> assistant with tool_call
	history := []ChatMessage{
		{Role: "user", Content: "run command"},
		{
			Role:      "assistant",
			ToolCalls: []interface{}{
				map[string]interface{}{
					"id":   "call_1",
					"type": "function",
					"function": map[string]interface{}{"name": "exec", "arguments": `{"cmd":"ls"}`},
				},
			},
		},
	}

	// Input replays function_call + output + new user message
	req := &ResponsesRequest{
		Model: "test",
		Input: []byte(`[{"type":"function_call","call_id":"call_1","name":"exec","arguments":"{\"cmd\":\"ls\"}"},{"type":"function_call_output","call_id":"call_1","output":"file.txt"},{"type":"message","role":"user","content":"next"}]`),
	}

	chatReq, _, _, err := ToChatRequest(req, history, sessions, nil, "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have: user, assistant{tool_calls}, tool, user(next)
	// NOT: user, assistant, assistant(dup), tool, user
	if len(chatReq.Messages) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(chatReq.Messages))
	}

	// Check that there's only one assistant message with tool_calls
	assistantCount := 0
	for _, msg := range chatReq.Messages {
		if msg.Role == "assistant" && msg.ToolCalls != nil {
			assistantCount++
		}
	}
	if assistantCount != 1 {
		t.Errorf("expected 1 assistant with tool_calls, got %d", assistantCount)
	}
}
