package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ProxyRequest forwards a /v1/* request to the correct upstream.
// For Codex-compatible paths it routes to chatgpt.com/backend-api/codex/*
// and injects the chatgpt-account-id header required by the ChatGPT Codex API.
//
// Passthrough mode: when the incoming request carries an Authorization token
// that matches a known managed account, the proxy forwards the request as-is
// (no pool rotation) but still logs the request. This enables Codex CLI to
// route through the proxy for logging while keeping its own auth.
func (p *CodexProxy) ProxyRequest(w http.ResponseWriter, r *http.Request) {
	// Try passthrough first: match the incoming token to a managed account
	entry := p.matchIncomingToken(r)
	passthrough := entry != nil

	if !passthrough {
		if !p.enabled {
			writeError(w, http.StatusServiceUnavailable, "Proxy is disabled", "service_unavailable")
			return
		}
		entry = p.pickEntry()
		if entry == nil {
			writeError(w, http.StatusServiceUnavailable, "No available accounts in pool", "no_available_account")
			return
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to read request body", "invalid_request")
		return
	}

	if !passthrough {
		// Normalize body for chatgpt.com Codex backend requirements.
		body, _ = normalizeCodexBody(body)
	}

	upstreamURL := buildUpstreamURL(r.URL.Path, r.URL.RawQuery)

	upstreamReq, err := http.NewRequest(r.Method, upstreamURL, bytes.NewReader(body))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create upstream request", "internal_error")
		return
	}

	// Copy headers; in passthrough mode keep original Authorization
	for key, values := range r.Header {
		lower := strings.ToLower(key)
		if lower == "chatgpt-account-id" {
			continue
		}
		if lower == "authorization" && !passthrough {
			continue
		}
		for _, v := range values {
			upstreamReq.Header.Add(key, v)
		}
	}
	if !passthrough {
		upstreamReq.Header.Set("Authorization", "Bearer "+entry.accessToken)
	}

	// chatgpt-account-id is required by the ChatGPT Codex backend API
	if entry.chatgptAccountID != "" {
		upstreamReq.Header.Set("chatgpt-account-id", entry.chatgptAccountID)
	}

	setCodexCLIHeaders(upstreamReq)

	shouldLog := r.Method == http.MethodPost

	startTime := time.Now()
	resp, err := p.httpClient.Do(upstreamReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("Upstream request failed: %v", err), "upstream_error")
		return
	}
	defer resp.Body.Close()

	// Capture rate-limit headers and persist to the OpenAI account
	p.saveRateLimits(entry, resp)

	// Persist stats for codex-source accounts
	if entry.source == "codex" && p.codexDB != nil {
		p.codexDB.IncrementRequestCount(entry.id)
	}
	if entry.requests != nil {
		atomicAddInt64(entry.requests, 1)
	}

	// For /models responses in passthrough mode, disable WebSocket support
	// so Codex CLI falls back to HTTP (which respects chatgpt_base_url).
	isModelsReq := r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/models")
	if isModelsReq && passthrough {
		respBody, readErr := io.ReadAll(resp.Body)
		if readErr == nil {
			respBody = disableWebSocketInModels(respBody)
		}
		copyResponseHeaders(w, resp)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(respBody)))
		w.WriteHeader(resp.StatusCode)
		w.Write(respBody) //nolint:errcheck
		return
	}

	copyResponseHeaders(w, resp)
	w.WriteHeader(resp.StatusCode)

	// Stream the response body; capture the last SSE "data:" line for token usage.
	flusher, canFlush := w.(http.Flusher)
	buf := make([]byte, 8192)
	var lastDataLine string
	var streamBuf strings.Builder
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			w.Write(buf[:n]) //nolint:errcheck
			if canFlush {
				flusher.Flush()
			}
			if shouldLog {
				streamBuf.Write(buf[:n])
				remaining := streamBuf.String()
				for {
					idx := strings.Index(remaining, "\n")
					if idx < 0 {
						break
					}
					line := strings.TrimSpace(remaining[:idx])
					remaining = remaining[idx+1:]
					if strings.HasPrefix(line, "data: ") {
						lastDataLine = line[6:]
					}
				}
				streamBuf.Reset()
				if remaining != "" {
					streamBuf.WriteString(remaining)
				}
			}
		}
		if readErr != nil {
			break
		}
	}

	duration := time.Since(startTime).Milliseconds()

	if shouldLog {
		p.saveLog(entry, body, r.URL.Path, lastDataLine, resp.StatusCode, duration, r.UserAgent())
	}
}

// ProxyChatCompletions implements a minimal OpenAI-compatible /v1/chat/completions endpoint
// by translating the request to the ChatGPT Codex Responses backend and converting the
// final text back into a Chat Completions JSON response.
//
// This is primarily for curl/testing and simple integrations; it supports non-streaming
// usage. If the client requests stream=true, the request is rejected for now.
func (p *CodexProxy) ProxyChatCompletions(w http.ResponseWriter, r *http.Request) {
	// Pick account (passthrough token match first, otherwise pool rotation).
	entry := p.matchIncomingToken(r)
	passthrough := entry != nil
	if !passthrough {
		if !p.enabled {
			writeError(w, http.StatusServiceUnavailable, "Proxy is disabled", "service_unavailable")
			return
		}
		entry = p.pickEntry()
		if entry == nil {
			writeError(w, http.StatusServiceUnavailable, "No available accounts in pool", "no_available_account")
			return
		}
	}

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to read request body", "invalid_request")
		return
	}

	var reqBody map[string]interface{}
	if err := json.Unmarshal(raw, &reqBody); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body", "invalid_request")
		return
	}

	// Reject streaming for now (the upstream requires streaming; we convert to non-stream).
	if v, ok := reqBody["stream"].(bool); ok && v {
		writeError(w, http.StatusBadRequest, "stream=true is not supported on this endpoint", "not_supported")
		return
	}

	model := ""
	if m, ok := reqBody["model"].(string); ok {
		model = m
	}

	// Convert messages[] → a simple "input" string.
	inputText := buildPromptFromMessages(reqBody["messages"])
	if inputText == "" {
		writeError(w, http.StatusBadRequest, "messages is required", "invalid_request")
		return
	}

	up := map[string]interface{}{
		"model":        model,
		"input":        inputText,
		"instructions": "You are a helpful assistant.",
		"stream":       true,
		"store":        false,
	}
	upBytes, _ := json.Marshal(up)
	upBytes, _ = normalizeCodexBody(upBytes)

	// Force upstream to the real responses endpoint.
	const upstreamURL = "https://chatgpt.com/backend-api/codex/responses?client_version=0.98.0"
	maxAttempts := 1
	if !passthrough {
		// Try multiple accounts if upstream says token invalidated.
		// Pool sizes can be large; cap retries to a reasonable number.
		maxAttempts = 20
	}

	var lastText string
	tried := map[string]bool{}
	if entry != nil {
		tried[entry.accessToken] = true
	}
	for attempt := 0; attempt < maxAttempts; attempt++ {
		upReq, err := http.NewRequest(http.MethodPost, upstreamURL, bytes.NewReader(upBytes))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to create upstream request", "internal_error")
			return
		}

		// Copy headers (except Authorization), then set required ones.
		for key, values := range r.Header {
			lower := strings.ToLower(key)
			if lower == "authorization" || lower == "chatgpt-account-id" {
				continue
			}
			for _, v := range values {
				upReq.Header.Add(key, v)
			}
		}
		upReq.Header.Set("Content-Type", "application/json")
		if !passthrough {
			upReq.Header.Set("Authorization", "Bearer "+entry.accessToken)
		} else if auth := r.Header.Get("Authorization"); auth != "" {
			upReq.Header.Set("Authorization", auth)
		}
		if entry.chatgptAccountID != "" {
			upReq.Header.Set("chatgpt-account-id", entry.chatgptAccountID)
		}
		setCodexCLIHeaders(upReq)
		upReq.Header.Set("Accept", "text/event-stream")

		resp, err := p.httpClient.Do(upReq)
		if err != nil {
			writeError(w, http.StatusBadGateway, fmt.Sprintf("Upstream request failed: %v", err), "upstream_error")
			return
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()

			// Retry only for rotated pool mode when the chosen account can't auth.
			if !passthrough && isRetryableAuthFailure(body) && attempt < maxAttempts-1 {
				// Pick a different entry for next attempt.
				if next := p.pickEntryExcluding(tried); next != nil {
					entry = next
					tried[entry.accessToken] = true
				} else {
					// No more candidates; fall through and return the error.
				}
				continue
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(resp.StatusCode)
			if len(body) > 0 {
				w.Write(body) //nolint:errcheck
			} else {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]interface{}{"message": "Upstream error", "type": "upstream_error", "code": fmt.Sprintf("%d", resp.StatusCode)},
				})
			}
			return
		}

		// Parse SSE until we can extract output text.
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
			if data == "[DONE]" {
				break
			}
			t := extractOutputTextFromResponsesEvent(data)
			if t != "" {
				lastText = t
			}
		}
		_ = resp.Body.Close()
		break
	}

	out := map[string]interface{}{
		"id":      "chatcmpl-" + uuid.New().String(),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []interface{}{
			map[string]interface{}{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": lastText,
				},
				"finish_reason": "stop",
			},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(out)
}
