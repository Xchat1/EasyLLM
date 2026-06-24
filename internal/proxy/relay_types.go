package proxy

import (
	"encoding/json"
	"strings"
)

// ── Responses API (inbound from Codex CLI) ──────────────────────────────────

// ResponsesRequest is the request body from Codex CLI using OpenAI Responses API format.
type ResponsesRequest struct {
	Model              string          `json:"model"`
	Input              json.RawMessage `json:"input"`  // can be string or []interface{}
	PreviousResponseID *string         `json:"previous_response_id,omitempty"`
	Tools              []interface{}   `json:"tools,omitempty"`
	ToolChoice         interface{}     `json:"tool_choice,omitempty"`
	ParallelToolCalls  *bool           `json:"parallel_tool_calls,omitempty"`
	Stream             bool            `json:"stream,omitempty"`
	Temperature        *float64        `json:"temperature,omitempty"`
	MaxOutputTokens    *int            `json:"max_output_tokens,omitempty"`
	System             *string         `json:"system,omitempty"`
	Instructions       *string         `json:"instructions,omitempty"`
}

// inputAsString returns the input as a plain string if it is one.
func (r *ResponsesRequest) inputAsString() (string, bool) {
	var s string
	if err := json.Unmarshal(r.Input, &s); err == nil {
		return s, true
	}
	return "", false
}

// inputAsArray returns the input as a slice of items.
func (r *ResponsesRequest) inputAsArray() ([]interface{}, bool) {
	var items []interface{}
	if err := json.Unmarshal(r.Input, &items); err == nil {
		return items, true
	}
	return nil, false
}

// ResponsesResponse is the response body returned to Codex CLI.
type ResponsesResponse struct {
	ID     string        `json:"id"`
	Object string        `json:"object"`
	Model  string        `json:"model"`
	Output []interface{} `json:"output"`
	Usage  ResponsesUsage `json:"usage"`
}

// ResponsesUsage maps token usage for the Responses API format.
type ResponsesUsage struct {
	InputTokens        int                  `json:"input_tokens"`
	OutputTokens       int                  `json:"output_tokens"`
	TotalTokens        int                  `json:"total_tokens"`
	InputTokensDetails *InputTokensDetails   `json:"input_tokens_details,omitempty"`
}

// InputTokensDetails holds cache hit info.
type InputTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

// ── Chat Completions API (outbound to upstream provider) ─────────────────────

// ChatRequest is the request sent to the upstream Chat Completions API.
type ChatRequest struct {
	Model         string        `json:"model"`
	Messages      []ChatMessage `json:"messages"`
	Tools               []interface{}      `json:"tools,omitempty"`
	ToolChoice          interface{}        `json:"tool_choice,omitempty"`
	ParallelToolCalls   *bool              `json:"parallel_tool_calls,omitempty"`
	Temperature         *float64           `json:"temperature,omitempty"`
	MaxTokens           *int               `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int               `json:"max_completion_tokens,omitempty"`
	StreamOptions       *ChatStreamOptions `json:"stream_options,omitempty"`
	Stream        bool          `json:"stream"`
	Thinking      *ChatThinking `json:"thinking,omitempty"`
}

// ChatThinking controls reasoning/thinking output for providers that support it (e.g. MiMo).
type ChatThinking struct {
	Type string `json:"type"`
}

// ChatStreamOptions enables usage reporting in SSE streams.
type ChatStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// ChatMessage represents a single message in Chat Completions format.
type ChatMessage struct {
	Role             string        `json:"role"`
	Content          interface{}   `json:"content,omitempty"` // string or []ChatContentPart
	ReasoningContent *string       `json:"reasoning_content,omitempty"`
	ToolCalls        []interface{} `json:"tool_calls,omitempty"`
	ToolCallID       *string       `json:"tool_call_id,omitempty"`
	Name             *string       `json:"name,omitempty"`
}

// TextContent returns the message content as a plain string.
func (m *ChatMessage) TextContent() string {
	if s, ok := m.Content.(string); ok {
		return s
	}
	return ""
}

// ChatResponse is the non-streaming response from a Chat Completions API.
type ChatResponse struct {
	Choices []ChatChoice `json:"choices"`
	Usage   *ChatUsage   `json:"usage,omitempty"`
}

// ChatChoice is a single choice in a Chat Completions response.
type ChatChoice struct {
	Message ChatMessage `json:"message"`
}

// ChatUsage is the token usage from Chat Completions API.
type ChatUsage struct {
	PromptTokens            int                  `json:"prompt_tokens"`
	CompletionTokens        int                  `json:"completion_tokens"`
	TotalTokens             int                  `json:"total_tokens"`
	PromptCacheHitTokens    *int                 `json:"prompt_cache_hit_tokens,omitempty"`
	PromptCacheMissTokens   *int                 `json:"prompt_cache_miss_tokens,omitempty"`
	PromptTokensDetails     *PromptTokensDetails  `json:"prompt_tokens_details,omitempty"`
}

// PromptTokensDetails holds cached tokens info.
type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

// CacheHit returns the number of cached (hit) prompt tokens.
func (u *ChatUsage) CacheHit() int {
	if u == nil {
		return 0
	}
	if u.PromptCacheHitTokens != nil {
		return *u.PromptCacheHitTokens
	}
	if u.PromptTokensDetails != nil {
		return u.PromptTokensDetails.CachedTokens
	}
	return 0
}

// ── SSE Streaming types ───────────────────────────────────────────────────────

// ChatStreamChunk is a single chunk from a Chat Completions SSE stream.
type ChatStreamChunk struct {
	Choices []ChatStreamChoice `json:"choices"`
	Usage   *ChatUsage         `json:"usage,omitempty"`
}

// ChatStreamChoice is a single delta choice.
type ChatStreamChoice struct {
	Delta        ChatDelta `json:"delta"`
	FinishReason *string   `json:"finish_reason,omitempty"`
}

// ChatDelta holds the delta content for streaming.
type ChatDelta struct {
	Role             *string           `json:"role,omitempty"`
	Content          *string           `json:"content,omitempty"`
	ReasoningContent *string           `json:"reasoning_content,omitempty"`
	ToolCalls        []DeltaToolCall   `json:"tool_calls,omitempty"`
}

// DeltaToolCall is a tool call delta in a streaming chunk.
type DeltaToolCall struct {
	Index    int            `json:"index"`
	ID       *string        `json:"id,omitempty"`
	Function *DeltaFunction `json:"function,omitempty"`
}

// DeltaFunction holds the function name and arguments deltas.
type DeltaFunction struct {
	Name      *string `json:"name,omitempty"`
	Arguments *string `json:"arguments,omitempty"`
}

// ── Helper utilities ─────────────────────────────────────────────────────────

// contentTypeKey returns whether the content type string represents an input_text type.
var textContentTypes = map[string]bool{
	"input_text":  true,
	"text":        true,
	"output_text": true,
}

// isTextContentType checks if a content part type is text-like.
func isTextContentType(typ string) bool {
	return textContentTypes[typ]
}

// textFromContentParts extracts concatenated text from content parts.
func textFromContentParts(parts []interface{}) string {
	var b strings.Builder
	for _, p := range parts {
		m, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		if t, ok := m["text"].(string); ok && t != "" {
			b.WriteString(t)
		}
	}
	return b.String()
}

// ── Relay Configuration ───────────────────────────────────────────

// RelayUpstream represents a single upstream provider in a multi-upstream pool.
type RelayUpstream struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Enabled         bool   `json:"enabled"`
	UpstreamURL     string `json:"upstream_url"`
	APIKey          string `json:"api_key"`
	AuthHeader      string `json:"auth_header"`
	AuthValuePrefix string `json:"auth_value_prefix"`
}

// RelayConfig holds relay configuration.
type RelayConfig struct {
	// Multi-upstream pool (preferred when non-empty).
	Upstreams        []RelayUpstream
	UpstreamStrategy string // "round_robin" (default)

	// Legacy single-upstream fields — kept for backward compatibility and as
	// fallback when Upstreams is empty.
	UpstreamURL     string
	APIKey          string
	AuthHeader      string // Custom auth header name (e.g., "api-key", default: "Authorization")
	AuthValuePrefix string // Prefix for auth value (e.g., "Bearer ", default: "Bearer ")

	// Global options (shared across all upstreams).
	DefaultModel    string // Default model when no mapping matches (empty = use original)
	ModelMap         map[string]string
	ToolDenylist     map[string]bool
	MaxSessions      int
	MaxSessionBytes  int
	SessionTTLHours int
	DiskCacheDir     string // Directory for disk persistence (empty = disabled)
	// JSON string configurations (loaded from settings)
	ModelMapJSON    string `json:"model_map_json"`
	ToolDenylistStr string `json:"tool_denylist_str"`
}

// DefaultRelayConfig returns a default relay configuration.
func DefaultRelayConfig() *RelayConfig {
	return &RelayConfig{
		UpstreamURL:      "https://api.openai.com/v1",
		APIKey:           "",
		UpstreamStrategy: "round_robin",
		ModelMap:         make(map[string]string),
		ToolDenylist:     make(map[string]bool),
		MaxSessions:      256,
		MaxSessionBytes:  512 * 1024 * 1024, // 512MB
		SessionTTLHours:  168,               // 7 days
		DiskCacheDir:     "",
	}
}

// ParseModelMap parses a JSON string into a model map.
func ParseModelMap(jsonStr string) map[string]string {
	if jsonStr == "" {
		return make(map[string]string)
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(jsonStr), &m); err == nil {
		return m
	}
	// Try parsing as comma-separated key:value pairs
	result := make(map[string]string)
	parts := strings.Split(jsonStr, ",")
	for _, part := range parts {
		kv := strings.SplitN(part, ":", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(kv[1])
			if key != "" && val != "" {
				result[key] = val
			}
		}
	}
	return result
}

// ParseToolDenylist parses a comma-separated string into a denylist map.
func ParseToolDenylist(str string) map[string]bool {
	result := make(map[string]bool)
	if str == "" {
		return result
	}
	parts := strings.Split(str, ",")
	for _, part := range parts {
		tool := strings.TrimSpace(part)
		if tool != "" {
			result[tool] = true
		}
	}
	return result
}
