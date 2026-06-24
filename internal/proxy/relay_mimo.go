package proxy

import "strings"

// applyMiMoChatRequestOptions tunes Chat Completions requests for Xiaomi MiMo.
// Codex code-fix flows require tool calling with deep thinking enabled and
// reasoning_content preserved across turns.
func applyMiMoChatRequestOptions(chatReq *ChatRequest, upstreamURL string) {
	if chatReq == nil || !isMiMoUpstream(upstreamURL) {
		return
	}

	hasTools := len(chatReq.Tools) > 0 || chatMessagesHaveToolCalls(chatReq.Messages)

	if hasTools {
		chatReq.Thinking = &ChatThinking{Type: "enabled"}
		// MiMo ignores custom temperature/top_p while thinking is enabled.
		chatReq.Temperature = nil
		if chatReq.ToolChoice == nil {
			chatReq.ToolChoice = "auto"
		}
		if chatReq.ParallelToolCalls == nil {
			v := true
			chatReq.ParallelToolCalls = &v
		}
	} else {
		chatReq.Thinking = &ChatThinking{Type: "disabled"}
	}

	if chatReq.MaxTokens != nil {
		v := *chatReq.MaxTokens
		chatReq.MaxCompletionTokens = &v
	}
}

func isMiMoUpstream(upstreamURL string) bool {
	return strings.Contains(strings.ToLower(upstreamURL), "xiaomimimo.com")
}

func chatMessagesHaveToolCalls(messages []ChatMessage) bool {
	for _, msg := range messages {
		if len(msg.ToolCalls) > 0 || msg.Role == "tool" {
			return true
		}
	}
	return false
}

func normalizeAssistantMessageContent(msg *ChatMessage) {
	if msg == nil {
		return
	}
	if s, ok := msg.Content.(string); ok && s == "" && len(msg.ToolCalls) > 0 {
		msg.Content = nil
	}
}
