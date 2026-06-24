package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// RelayStreamTranslator handles SSE stream translation from Chat Completions to Responses API.
type RelayStreamTranslator struct {
	Client          *http.Client
	UpstreamURL     string
	APIKey          string
	AuthHeader      string // Custom auth header name
	AuthValuePrefix string // Prefix for auth value
	ChatReq         *ChatRequest
	ResponseID      string
	Sessions        *RelaySessionStore
	RequestMessages []ChatMessage
	NamespaceTools  NamespaceToolMap
	Model           string
	UsageStore      *RelayUsageStore
	CodexModel      string
	LogStore        *RelayLogStore
}

// SSEEvent represents an SSE event for the Responses API.
type SSEEvent struct {
	Event string
	Data  map[string]interface{}
}

// TranslateStream translates an upstream Chat Completions SSE stream
// into a Responses API SSE stream, writing events to the provided writer.
func TranslateStream(ctx context.Context, args RelayStreamTranslator, w http.ResponseWriter, flusher http.Flusher) error {
	msgItemID := fmt.Sprintf("msg_%s", uuid.New().String()[:8])

	// Send response.created event
	sendEvent(w, flusher, "response.created", map[string]interface{}{
		"type": "response.created",
		"response": map[string]interface{}{
			"id":     args.ResponseID,
			"status": "in_progress",
			"model":  args.Model,
		},
	})

	// Build upstream request
	body, err := json.Marshal(args.ChatReq)
	if err != nil {
		sendErrorEvent(w, flusher, args.ResponseID, "internal_error", "Failed to marshal request")
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, args.UpstreamURL, bytes.NewReader(body))
	if err != nil {
		sendErrorEvent(w, flusher, args.ResponseID, "internal_error", "Failed to create upstream request")
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	applyAuthHeader(req, args.APIKey, args.AuthHeader, args.AuthValuePrefix)

	resp, err := args.Client.Do(req)
	if err != nil {
		relayStreamLog(args, "error", "上游连接失败: "+err.Error())
		sendErrorEvent(w, flusher, args.ResponseID, "upstream_error", fmt.Sprintf("Upstream request failed: %v", err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		relayStreamLog(args, "error", fmt.Sprintf("上游返回 %d: %s", resp.StatusCode, truncateRelayLog(string(body), 200)))
		sendErrorEvent(w, flusher, args.ResponseID, fmt.Sprintf("%d", resp.StatusCode), string(body))
		return fmt.Errorf("upstream error: %s", string(body))
	}

	relayStreamLog(args, "info", "流式连接已建立")

	// Stream state
	var accumulatedText string
	var accumulatedReasoning string
	toolCalls := make(map[int]*ToolCallAccum)
	emittedMessageItem := false
	streamDone := false
	var streamUsage *ChatUsage

	// Process SSE stream
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parse SSE event
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data: "))

			if data == "[DONE]" {
				streamDone = true
				break
			}

			var chunk ChatStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			if chunk.Usage != nil {
				streamUsage = chunk.Usage
			}

			for _, choice := range chunk.Choices {
				// Reasoning content
				if choice.Delta.ReasoningContent != nil {
					accumulatedReasoning += *choice.Delta.ReasoningContent
				}

				// Text content
				if choice.Delta.Content != nil && *choice.Delta.Content != "" {
					if !emittedMessageItem {
						sendEvent(w, flusher, "response.output_item.added", map[string]interface{}{
							"type": "response.output_item.added",
							"output_index": 0,
							"item": map[string]interface{}{
								"type":       "message",
								"id":         msgItemID,
								"role":       "assistant",
								"status":     "in_progress",
								"content":    []interface{}{},
							},
						})
						emittedMessageItem = true
					}

					accumulatedText += *choice.Delta.Content
					sendEvent(w, flusher, "response.output_text.delta", map[string]interface{}{
						"type":         "response.output_text.delta",
						"item_id":      msgItemID,
						"output_index": 0,
						"delta":        *choice.Delta.Content,
					})
				}

				// Tool call deltas (emit incrementally)
				if choice.Delta.ToolCalls != nil {
					for _, tc := range choice.Delta.ToolCalls {
						idx := tc.Index
						if _, ok := toolCalls[idx]; !ok {
							toolCalls[idx] = &ToolCallAccum{
								ItemID:      fmt.Sprintf("fc_%s", uuid.New().String()[:8]),
								OutputIndex: -1,
							}
						}
						accum := toolCalls[idx]
						prevArgsLen := len(accum.Args)
						if tc.ID != nil && *tc.ID != "" {
							accum.ID = *tc.ID
						} else if accum.ID == "" && ((tc.Function != nil && tc.Function.Name != nil && *tc.Function.Name != "") || accum.Name != "") {
							accum.ID = fmt.Sprintf("call_%s", uuid.New().String()[:12])
						}
						if tc.Function != nil {
							if tc.Function.Name != nil && *tc.Function.Name != "" && accum.Name == "" {
								accum.Name = *tc.Function.Name
							}
							if tc.Function.Arguments != nil && *tc.Function.Arguments != "" {
								accum.Args += *tc.Function.Arguments
							}
						}
						emitToolCallStreamEvents(w, flusher, accum, args.NamespaceTools, prevArgsLen)
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		sendErrorEvent(w, flusher, args.ResponseID, "stream_error", fmt.Sprintf("SSE parse error: %v", err))
		return err
	}

	if !streamDone && (emittedMessageItem || len(toolCalls) > 0) {
		streamDone = true
	}

	// Emit message item done
	if emittedMessageItem {
		sendEvent(w, flusher, "response.output_item.done", map[string]interface{}{
			"type":         "response.output_item.done",
			"output_index": 0,
			"item": map[string]interface{}{
				"type":       "message",
				"id":         msgItemID,
				"role":       "assistant",
				"status":     "completed",
				"content": []interface{}{
					map[string]interface{}{
						"type": "output_text",
						"text": accumulatedText,
					},
				},
			},
		})
	}

	// Finalize function_call items
	baseIndex := 0
	if emittedMessageItem {
		baseIndex = 1
	}

	var fcItems []interface{}
	indices := sortedToolIndices(toolCalls)
	for _, idx := range indices {
		tc := toolCalls[idx]
		if tc.OutputIndex < 0 {
			tc.OutputIndex = baseIndex + idx
		}
		if tc.ID == "" {
			tc.ID = fmt.Sprintf("call_%s", uuid.New().String()[:12])
		}

		ns, name := responseFunctionNameForResponses(tc.Name, args.NamespaceTools)
		doneItem := map[string]interface{}{
			"type":      "function_call",
			"id":        tc.ItemID,
			"call_id":   tc.ID,
			"name":      name,
			"arguments": tc.Args,
			"status":    "completed",
		}
		if ns != "" {
			doneItem["namespace"] = ns
		}

		if !tc.Added {
			emitToolCallStreamEvents(w, flusher, tc, args.NamespaceTools, 0)
		}

		sendEvent(w, flusher, "response.output_item.done", map[string]interface{}{
			"type":         "response.output_item.done",
			"output_index": tc.OutputIndex,
			"item":         doneItem,
		})

		fcItems = append(fcItems, doneItem)
	}

	// Save session history
	if streamDone {
		// Build assistant message
		var assistantToolCalls []interface{}
		if len(toolCalls) > 0 {
			for _, idx := range indices {
				tc := toolCalls[idx]
				assistantToolCalls = append(assistantToolCalls, map[string]interface{}{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]interface{}{
						"name":      tc.Name,
						"arguments": tc.Args,
					},
				})
			}
		}

		assistantMsg := ChatMessage{
			Role: "assistant",
		}
		if accumulatedText != "" {
			assistantMsg.Content = accumulatedText
		}
		if accumulatedReasoning != "" {
			assistantMsg.ReasoningContent = &accumulatedReasoning
		}
		if len(assistantToolCalls) > 0 {
			assistantMsg.ToolCalls = assistantToolCalls
		}

		// Store turn reasoning
		if accumulatedReasoning != "" {
			args.Sessions.StoreTurnReasoning(args.RequestMessages, &assistantMsg, accumulatedReasoning)
		}

		// Save history
		messages := append(args.RequestMessages, assistantMsg)
		args.Sessions.SaveWithID(args.ResponseID, messages)

		// Build output for response.completed
		var outputItems []interface{}
		if emittedMessageItem {
			outputItems = append(outputItems, map[string]interface{}{
				"type":       "message",
				"id":         msgItemID,
				"role":       "assistant",
				"status":     "completed",
				"content": []interface{}{
					map[string]interface{}{
						"type": "output_text",
						"text": accumulatedText,
					},
				},
			})
		}
		outputItems = append(outputItems, fcItems...)

		usage := streamUsage
		if usage == nil {
			usage = &ChatUsage{}
		}

		// Send response.completed
		sendEvent(w, flusher, "response.completed", map[string]interface{}{
			"type": "response.completed",
			"response": map[string]interface{}{
				"id":     args.ResponseID,
				"status": "completed",
				"model":  args.Model,
				"output": outputItems,
				"usage": map[string]interface{}{
					"input_tokens":        usage.PromptTokens,
					"output_tokens":       usage.CompletionTokens,
					"total_tokens":        usage.TotalTokens,
					"input_tokens_details": map[string]interface{}{
						"cached_tokens": usage.CacheHit(),
					},
				},
			},
		})

		model := args.CodexModel
		if model == "" {
			model = args.Model
		}
		recordChatUsage(args.UsageStore, args.UpstreamURL, model, args.Model, true, usage)
		toolCount := len(toolCalls)
		if toolCount > 0 {
			relayStreamLog(args, "info", fmt.Sprintf("流式完成 input=%d output=%d total=%d cached=%d tools=%d", usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens, usage.CacheHit(), toolCount))
		} else {
			relayStreamLog(args, "info", fmt.Sprintf("流式完成 input=%d output=%d total=%d cached=%d", usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens, usage.CacheHit()))
		}
	} else {
		// Stream did not complete cleanly
		relayStreamLog(args, "warn", "流式中断，未正常完成")
		sendErrorEvent(w, flusher, args.ResponseID, "stream_incomplete", "Stream disconnected before completion")
	}

	return nil
}

// ── Helpers ─────────────────────────────────────────────────────

func relayStreamLog(args RelayStreamTranslator, level, message string) {
	if args.LogStore == nil {
		return
	}
	model := args.CodexModel
	if model == "" {
		model = args.Model
	}
	args.LogStore.Log(level, message, model, args.ResponseID)
}

func sendEvent(w http.ResponseWriter, flusher http.Flusher, event string, data map[string]interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\n", event)
	fmt.Fprintf(w, "data: %s\n\n", string(jsonData))
	flusher.Flush()
}

func sendErrorEvent(w http.ResponseWriter, flusher http.Flusher, responseID, code, message string) {
	sendEvent(w, flusher, "response.failed", map[string]interface{}{
		"type": "response.failed",
		"response": map[string]interface{}{
			"id":     responseID,
			"status": "failed",
			"error": map[string]interface{}{
				"code":    code,
				"message": message,
			},
		},
	})
}

// ToolCallAccum accumulates tool call delta information.
type ToolCallAccum struct {
	ID          string
	Name        string
	Args        string
	ItemID      string
	OutputIndex int  // -1 = not yet assigned
	Added       bool
}

func sortedToolIndices(toolCalls map[int]*ToolCallAccum) []int {
	indices := make([]int, 0, len(toolCalls))
	for k := range toolCalls {
		indices = append(indices, k)
	}
	sort.Ints(indices)
	return indices
}

func emitToolCallStreamEvents(w http.ResponseWriter, flusher http.Flusher, accum *ToolCallAccum, nsMap NamespaceToolMap, prevArgsLen int) {
	if accum == nil || accum.ItemID == "" {
		return
	}
	ns, name := responseFunctionNameForResponses(accum.Name, nsMap)
	if !accum.Added && (accum.Name != "" || accum.ID != "") {
		addedItem := map[string]interface{}{
			"type":      "function_call",
			"id":        accum.ItemID,
			"call_id":   accum.ID,
			"name":      name,
			"arguments": "",
			"status":    "in_progress",
		}
		if ns != "" {
			addedItem["namespace"] = ns
		}
		sendEvent(w, flusher, "response.output_item.added", map[string]interface{}{
			"type":         "response.output_item.added",
			"output_index": accum.OutputIndex,
			"item":         addedItem,
		})
		accum.Added = true
	}
	if len(accum.Args) > prevArgsLen {
		sendEvent(w, flusher, "response.function_call_arguments.delta", map[string]interface{}{
			"type":         "response.function_call_arguments.delta",
			"item_id":      accum.ItemID,
			"output_index": accum.OutputIndex,
			"delta":        accum.Args[prevArgsLen:],
		})
	}
}

// TranslateStreamSimple is a simpler version that uses eventsource stream.
// This is an alternative implementation using the sse package.
func TranslateStreamSimple(ctx context.Context, args RelayStreamTranslator, w http.ResponseWriter, flusher http.Flusher) error {
	// This is a simplified wrapper - the main logic is in TranslateStream
	return TranslateStream(ctx, args, w, flusher)
}

// RelayHandlerWrapper wraps the relay handler with common setup.
type RelayHandlerWrapper struct {
	Sessions   *RelaySessionStore
	HTTPClient *http.Client
	Config     *RelayConfig
}
