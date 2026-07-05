package proxy

import (
	"bufio"
	"bytes"
	"easyllm/internal/httputil"
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
// (no pool rotation). This enables Codex CLI to route through the proxy while
// keeping its own auth; EasyLLM only retains metadata for the recent-call view.
func (p *CodexProxy) ProxyRequest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Try passthrough first: match the incoming token to a managed account
	entry := p.matchIncomingToken(r)
	passthrough := entry != nil
	var reservedEntry *poolEntry

	if !passthrough {
		if !p.IsEnabled() {
			p.recordCall(entry, r, nil, http.StatusServiceUnavailable, passthrough, start, "Proxy is disabled")
			writeError(w, http.StatusServiceUnavailable, "Proxy is disabled", "service_unavailable")
			return
		}
		entry = p.pickEntryReserved()
		if entry == nil {
			p.recordCall(entry, r, nil, http.StatusServiceUnavailable, passthrough, start, "No available accounts in pool")
			writeError(w, http.StatusServiceUnavailable, "No available accounts in pool", "no_available_account")
			return
		}
		reservedEntry = entry
		defer func() {
			releasePoolEntry(reservedEntry)
		}()
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		p.recordCall(entry, r, nil, http.StatusBadRequest, passthrough, start, "Failed to read request body")
		writeError(w, http.StatusBadRequest, "Failed to read request body", "invalid_request")
		return
	}

	if !passthrough {
		// Normalize body for chatgpt.com Codex backend requirements.
		body, _ = normalizeCodexBody(body)
	}

	upstreamURL := buildUpstreamURL(r.URL.Path, r.URL.RawQuery)
	maxAttempts := 1
	if !passthrough {
		maxAttempts = p.poolMaxRetryAttempts()
	}
	tried := map[string]bool{}
	if entry != nil && entry.accessToken != "" {
		tried[entry.accessToken] = true
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		upstreamReq, err := http.NewRequest(r.Method, upstreamURL, bytes.NewReader(body))
		if err != nil {
			p.recordCall(entry, r, body, http.StatusInternalServerError, passthrough, start, "Failed to create upstream request")
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

		if entry.chatgptAccountID != "" {
			upstreamReq.Header.Set("chatgpt-account-id", entry.chatgptAccountID)
		}

		setCodexCLIHeaders(upstreamReq)

		resp, err := httputil.DoWithRetries(p.httpClient, upstreamReq, 3)
		if err != nil {
			if !passthrough && httputil.IsTransientNetworkError(err) && attempt < maxAttempts-1 {
				if next := p.pickEntryExcludingReserved(tried); next != nil {
					releasePoolEntry(reservedEntry)
					entry = next
					reservedEntry = next
					tried[entry.accessToken] = true
					continue
				}
			}
			p.recordCall(entry, r, body, http.StatusBadGateway, passthrough, start, err.Error())
			writeError(w, http.StatusBadGateway, fmt.Sprintf("Upstream request failed: %v", err), "upstream_error")
			return
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			errBody, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if !passthrough && isRetryablePoolFailure(resp.StatusCode, errBody) && attempt < maxAttempts-1 {
				if isUsageLimitReached(resp.StatusCode, errBody) {
					p.markPoolEntryUsageLimited(entry, errBody)
				} else if isRetryableAuthFailure(resp.StatusCode, errBody) && p.refreshPoolEntryToken(entry) {
					tried[entry.accessToken] = true
					continue
				}
				if next := p.pickEntryExcludingReserved(tried); next != nil {
					releasePoolEntry(reservedEntry)
					entry = next
					reservedEntry = next
					tried[entry.accessToken] = true
					continue
				}
			}
			copyResponseHeaders(w, resp)
			w.WriteHeader(resp.StatusCode)
			if len(errBody) > 0 {
				w.Write(errBody) //nolint:errcheck
			}
			p.recordCall(entry, r, body, resp.StatusCode, passthrough, start, string(errBody))
			return
		}

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
			_ = resp.Body.Close()
			if readErr == nil {
				respBody = disableWebSocketInModels(respBody)
			}
			copyResponseHeaders(w, resp)
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(respBody)))
			w.WriteHeader(resp.StatusCode)
			w.Write(respBody) //nolint:errcheck
			p.recordCall(entry, r, body, resp.StatusCode, passthrough, start, "")
			return
		}

		copyResponseHeaders(w, resp)
		w.WriteHeader(resp.StatusCode)

		flusher, canFlush := w.(http.Flusher)
		buf := make([]byte, 8192)
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				w.Write(buf[:n]) //nolint:errcheck
				if canFlush {
					flusher.Flush()
				}
			}
			if readErr != nil {
				break
			}
		}
		_ = resp.Body.Close()
		p.recordCall(entry, r, body, resp.StatusCode, passthrough, start, "")
		return
	}
}

// ProxyChatCompletions implements a minimal OpenAI-compatible /v1/chat/completions endpoint
// by translating the request to the ChatGPT Codex Responses backend and converting the
// final text back into a Chat Completions JSON response.
//
// This is primarily for curl/testing and simple integrations; it supports non-streaming
// usage. If the client requests stream=true, the request is rejected for now.
func (p *CodexProxy) ProxyChatCompletions(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	// Pick account (passthrough token match first, otherwise pool rotation).
	entry := p.matchIncomingToken(r)
	passthrough := entry != nil
	var reservedEntry *poolEntry
	if !passthrough {
		if !p.IsEnabled() {
			p.recordCall(entry, r, nil, http.StatusServiceUnavailable, passthrough, start, "Proxy is disabled")
			writeError(w, http.StatusServiceUnavailable, "Proxy is disabled", "service_unavailable")
			return
		}
		entry = p.pickEntryReserved()
		if entry == nil {
			p.recordCall(entry, r, nil, http.StatusServiceUnavailable, passthrough, start, "No available accounts in pool")
			writeError(w, http.StatusServiceUnavailable, "No available accounts in pool", "no_available_account")
			return
		}
		reservedEntry = entry
		defer func() {
			releasePoolEntry(reservedEntry)
		}()
	}

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		p.recordCall(entry, r, nil, http.StatusBadRequest, passthrough, start, "Failed to read request body")
		writeError(w, http.StatusBadRequest, "Failed to read request body", "invalid_request")
		return
	}

	var reqBody map[string]interface{}
	if err := json.Unmarshal(raw, &reqBody); err != nil {
		p.recordCall(entry, r, raw, http.StatusBadRequest, passthrough, start, "Invalid JSON body")
		writeError(w, http.StatusBadRequest, "Invalid JSON body", "invalid_request")
		return
	}

	// Reject streaming for now (the upstream requires streaming; we convert to non-stream).
	if v, ok := reqBody["stream"].(bool); ok && v {
		p.recordCall(entry, r, raw, http.StatusBadRequest, passthrough, start, "stream=true is not supported on this endpoint")
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
		p.recordCall(entry, r, raw, http.StatusBadRequest, passthrough, start, "messages is required")
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
		maxAttempts = p.poolMaxRetryAttempts()
	}

	var lastText string
	tried := map[string]bool{}
	if entry != nil {
		tried[entry.accessToken] = true
	}
	for attempt := 0; attempt < maxAttempts; attempt++ {
		upReq, err := http.NewRequest(http.MethodPost, upstreamURL, bytes.NewReader(upBytes))
		if err != nil {
			p.recordCall(entry, r, raw, http.StatusInternalServerError, passthrough, start, "Failed to create upstream request")
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

		resp, err := httputil.DoWithRetries(p.httpClient, upReq, 3)
		if err != nil {
			if !passthrough && httputil.IsTransientNetworkError(err) && attempt < maxAttempts-1 {
				if next := p.pickEntryExcludingReserved(tried); next != nil {
					releasePoolEntry(reservedEntry)
					entry = next
					reservedEntry = next
					tried[entry.accessToken] = true
					continue
				}
			}
			p.recordCall(entry, r, raw, http.StatusBadGateway, passthrough, start, err.Error())
			writeError(w, http.StatusBadGateway, fmt.Sprintf("Upstream request failed: %v", err), "upstream_error")
			return
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()

			if !passthrough && isRetryablePoolFailure(resp.StatusCode, body) && attempt < maxAttempts-1 {
				if isUsageLimitReached(resp.StatusCode, body) {
					p.markPoolEntryUsageLimited(entry, body)
				} else if isRetryableAuthFailure(resp.StatusCode, body) && p.refreshPoolEntryToken(entry) {
					tried[entry.accessToken] = true
					continue
				}
				if next := p.pickEntryExcludingReserved(tried); next != nil {
					releasePoolEntry(reservedEntry)
					entry = next
					reservedEntry = next
					tried[entry.accessToken] = true
					continue
				}
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
			p.recordCall(entry, r, raw, resp.StatusCode, passthrough, start, string(body))
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
	p.recordCall(entry, r, raw, http.StatusOK, passthrough, start, "")
}
