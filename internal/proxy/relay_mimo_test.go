package proxy

import "testing"

func TestApplyMiMoChatRequestOptionsWithTools(t *testing.T) {
	max := 4096
	req := &ChatRequest{
		Model: "mimo-v2.5-pro",
		Tools: []interface{}{
			map[string]interface{}{"type": "function", "function": map[string]interface{}{"name": "apply_patch"}},
		},
		MaxTokens:   &max,
		Temperature: ptrFloat(0.2),
	}
	applyMiMoChatRequestOptions(req, "https://token-plan-cn.xiaomimimo.com/v1")

	if req.Thinking == nil || req.Thinking.Type != "enabled" {
		t.Fatalf("expected thinking enabled, got %+v", req.Thinking)
	}
	if req.ToolChoice == nil {
		t.Fatal("expected tool_choice auto")
	}
	if req.ParallelToolCalls == nil || !*req.ParallelToolCalls {
		t.Fatal("expected parallel_tool_calls true")
	}
	if req.Temperature != nil {
		t.Fatal("expected temperature cleared for thinking mode")
	}
	if req.MaxCompletionTokens == nil || *req.MaxCompletionTokens != max {
		t.Fatalf("expected max_completion_tokens %d, got %+v", max, req.MaxCompletionTokens)
	}
}

func TestApplyMiMoChatRequestOptionsWithoutTools(t *testing.T) {
	req := &ChatRequest{Model: "mimo-v2.5-pro"}
	applyMiMoChatRequestOptions(req, "https://token-plan-cn.xiaomimimo.com/v1")
	if req.Thinking == nil || req.Thinking.Type != "disabled" {
		t.Fatalf("expected thinking disabled, got %+v", req.Thinking)
	}
}

func TestNormalizeAssistantMessageContent(t *testing.T) {
	msg := ChatMessage{
		Role:    "assistant",
		Content: "",
		ToolCalls: []interface{}{
			map[string]interface{}{"id": "call_1", "type": "function"},
		},
	}
	normalizeAssistantMessageContent(&msg)
	if msg.Content != nil {
		t.Fatalf("expected nil content, got %#v", msg.Content)
	}
}

func ptrFloat(v float64) *float64 { return &v }
