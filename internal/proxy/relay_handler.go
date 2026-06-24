package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	openaiplatform "easyllm/internal/openai"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

func (h *RelayHandler) relayLog(level, message, model, responseID string) {
	if h != nil && h.Logs != nil {
		h.Logs.Log(level, message, model, responseID)
	}
}

// RelayHandler handles Relay mode requests.
type RelayHandler struct {
	Sessions *RelaySessionStore
	Config   *RelayConfig
	Client   *http.Client
	Usage    *RelayUsageStore
	Logs     *RelayLogStore
	rrIndex  int64 // atomic round-robin counter for multi-upstream selection
}

// selectUpstream picks an upstream for the next request using round-robin.
func (h *RelayHandler) selectUpstream() RelayUpstream {
	return h.selectUpstreamFromConfig(h.Config)
}

// selectUpstreamFromConfig picks the next upstream from an explicit config.
// Enabled upstreams in config.Upstreams are round-robin'd; falls back to legacy fields.
func (h *RelayHandler) selectUpstreamFromConfig(config *RelayConfig) RelayUpstream {
	if config == nil {
		return RelayUpstream{}
	}
	enabled := make([]RelayUpstream, 0, len(config.Upstreams))
	for _, u := range config.Upstreams {
		if u.Enabled {
			enabled = append(enabled, u)
		}
	}
	if len(enabled) == 0 {
		// Fall back to legacy single-upstream fields.
		return RelayUpstream{
			UpstreamURL:     config.UpstreamURL,
			APIKey:          config.APIKey,
			AuthHeader:      config.AuthHeader,
			AuthValuePrefix: config.AuthValuePrefix,
		}
	}
	idx := int(atomic.AddInt64(&h.rrIndex, 1)-1) % len(enabled)
	return enabled[idx]
}

// NewRelayHandler creates a new Relay handler.
func NewRelayHandler(config *RelayConfig) *RelayHandler {
	if config == nil {
		config = LoadRelayConfigFromSettings()
	}
	sessions, err := NewRelaySessionStoreWithLimits(
		config.MaxSessions,
		config.MaxSessionBytes,
		config.SessionTTLHours,
		config.DiskCacheDir,
	)
	if err != nil {
		sessions = NewRelaySessionStore()
	}
	return &RelayHandler{
		Sessions: sessions,
		Config:   config,
		Client:   NewRelayHTTPClient(),
		Usage:    NewRelayUsageStore(),
		Logs:     NewRelayLogStore(),
	}
}

// UpdateConfig updates the relay configuration.
func (h *RelayHandler) UpdateConfig(config *RelayConfig) {
	h.Config = config
}

// HandleRelayResponses handles POST /v1/responses.
// This is the main endpoint that converts Responses API requests to Chat Completions.
func (h *RelayHandler) HandleRelayResponses(c *gin.Context) {
	// Read request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Failed to read request body",
				"type":    "invalid_request",
			},
		})
		return
	}

	// Parse ResponsesRequest
	var req ResponsesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid JSON: " + err.Error(),
				"type":    "invalid_request",
			},
		})
		return
	}

	if req.Model == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "model is required",
				"type":    "invalid_request",
			},
		})
		return
	}

	// Load history if previous_response_id is provided
	var history []ChatMessage
	if req.PreviousResponseID != nil && *req.PreviousResponseID != "" {
		history = h.Sessions.GetHistory(*req.PreviousResponseID)
	}

	// Build model map and tool denylist from config
	modelMap := h.Config.ModelMap
	if modelMap == nil && h.Config.ModelMapJSON != "" {
		modelMap = ParseModelMap(h.Config.ModelMapJSON)
	}
	toolDenylist := h.Config.ToolDenylist
	if toolDenylist == nil && h.Config.ToolDenylistStr != "" {
		toolDenylist = ParseToolDenylist(h.Config.ToolDenylistStr)
	}

	// Convert to Chat Completions request
	chatReq, nsMap, requestMessages, err := ToChatRequest(&req, history, h.Sessions, modelMap, h.Config.DefaultModel, toolDenylist)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to convert request: " + err.Error(),
				"type":    "internal_error",
			},
		})
		return
	}

	// Generate response ID
	responseID := h.Sessions.NewID()

	// Select upstream (round-robin across enabled upstreams; falls back to legacy fields).
	upstream := h.selectUpstream()
	upstreamURL := upstream.UpstreamURL
	if upstreamURL == "" {
		upstreamURL = "https://api.openai.com/v1"
	}
	// Resolve API key: upstream config > request header passthrough.
	apiKey := upstream.APIKey
	if apiKey == "" {
		apiKey = ResolveRelayAPIKey(h.Config, c.Request.Header)
	}
	authHeader := upstream.AuthHeader
	authValuePrefix := upstream.AuthValuePrefix

	streamLabel := "非流式"
	if req.Stream {
		streamLabel = "流式"
	}
	upstreamLabel := upstream.Name
	if upstreamLabel == "" {
		upstreamLabel = upstreamURL
	}
	h.relayLog("info", fmt.Sprintf("收到请求 %s · %s → %s [%s]", streamLabel, req.Model, chatReq.Model, upstreamLabel), req.Model, responseID)
	if len(chatReq.Tools) > 0 {
		h.relayLog("info", fmt.Sprintf("工具数量 %d", len(chatReq.Tools)), req.Model, responseID)
	}

	applyMiMoChatRequestOptions(chatReq, upstreamURL)

	// Handle streaming
	if req.Stream {
		h.handleStreaming(c, chatReq, responseID, requestMessages, nsMap, upstreamURL, apiKey, authHeader, authValuePrefix, req.Model)
		return
	}

	// Non-streaming: send request to upstream
	resp, err := h.sendChatRequest(chatReq, upstreamURL, apiKey, authHeader, authValuePrefix)
	if err != nil {
		h.relayLog("error", "上游请求失败: "+err.Error(), req.Model, responseID)
		c.JSON(http.StatusBadGateway, gin.H{
			"error": gin.H{
				"message": "Upstream request failed: " + err.Error(),
				"type":    "upstream_error",
			},
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		h.relayLog("error", fmt.Sprintf("上游返回 %d: %s", resp.StatusCode, truncateRelayLog(string(respBody), 200)), req.Model, responseID)
		c.JSON(resp.StatusCode, gin.H{
			"error": gin.H{
				"message": string(respBody),
				"type":    "upstream_error",
				"code":    fmt.Sprintf("%d", resp.StatusCode),
			},
		})
		return
	}

	// Parse Chat Completions response
	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to parse upstream response: " + err.Error(),
				"type":    "upstream_error",
			},
		})
		return
	}

	// Convert to Responses API response
	responsesResp, outputMessages := FromChatResponse(responseID, req.Model, chatResp, nsMap)

	// Save session history and reasoning for multi-turn
	if len(chatResp.Choices) > 0 {
		assistant := chatResp.Choices[0].Message
		if assistant.ReasoningContent != nil && *assistant.ReasoningContent != "" {
			h.Sessions.StoreTurnReasoning(requestMessages, &assistant, *assistant.ReasoningContent)
		}
	}
	if len(outputMessages) > 0 {
		messages := append(requestMessages, outputMessages...)
		h.Sessions.SaveWithID(responseID, messages)
	}

	recordResponsesUsage(h.Usage, upstreamURL, req.Model, chatReq.Model, false, responsesResp.Usage)

	usage := responsesResp.Usage
	h.relayLog("info", fmt.Sprintf("响应完成 input=%d output=%d total=%d", usage.InputTokens, usage.OutputTokens, usage.TotalTokens), req.Model, responseID)

	// Return response
	c.JSON(http.StatusOK, responsesResp)
}

// HandleRelayModels handles GET /v1/models.
// It proxies to the upstream /v1/models endpoint.
func (h *RelayHandler) HandleRelayModels(c *gin.Context) {
	upstreamURL := h.Config.UpstreamURL
	if upstreamURL == "" {
		upstreamURL = "https://api.openai.com/v1"
	}
	apiKey := ResolveRelayAPIKey(h.Config, c.Request.Header)

	// Build upstream models URL — TrimRight first to normalise trailing slashes
	// before the HasSuffix check; without this "…/v1/" fails the check and we'd
	// append an extra "/v1", yielding "…/v1/v1/models".
	modelsURL := strings.TrimRight(upstreamURL, "/")
	if !strings.HasSuffix(modelsURL, "/v1") {
		modelsURL = modelsURL + "/v1"
	}
	modelsURL = modelsURL + "/models"

	req, err := http.NewRequest(http.MethodGet, modelsURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to create upstream request",
				"type":    "internal_error",
			},
		})
		return
	}

	if apiKey != "" {
		applyAuthHeader(req, apiKey, h.Config.AuthHeader, h.Config.AuthValuePrefix)
	}

	resp, err := h.Client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": gin.H{
				"message": "Upstream request failed: " + err.Error(),
				"type":    "upstream_error",
			},
		})
		return
	}
	defer resp.Body.Close()

	// Copy response headers, skipping hop-by-hop headers that must not be
	// forwarded through a proxy (RFC 7230 §6.1).
	hopByHop := map[string]bool{
		"transfer-encoding": true,
		"connection":        true,
		"keep-alive":        true,
		"content-length":    true,
	}
	for k, vals := range resp.Header {
		if hopByHop[strings.ToLower(k)] {
			continue
		}
		for _, v := range vals {
			c.Writer.Header().Add(k, v)
		}
	}

	// Copy response body
	c.Writer.WriteHeader(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
}

// handleStreaming handles streaming responses.
func (h *RelayHandler) handleStreaming(c *gin.Context, chatReq *ChatRequest, responseID string, requestMessages []ChatMessage, nsMap NamespaceToolMap, upstreamURL, apiKey, authHeader, authValuePrefix, codexModel string) {
	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Streaming not supported",
				"type":    "internal_error",
			},
		})
		return
	}

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Create stream translator args
	args := RelayStreamTranslator{
		Client:          h.Client,
		UpstreamURL:     buildChatCompletionsURL(upstreamURL),
		APIKey:          apiKey,
		AuthHeader:      authHeader,
		AuthValuePrefix: authValuePrefix,
		ChatReq:         chatReq,
		ResponseID:      responseID,
		Sessions:        h.Sessions,
		RequestMessages: requestMessages,
		NamespaceTools:  nsMap,
		Model:           chatReq.Model,
		UsageStore:      h.Usage,
		CodexModel:      codexModel,
		LogStore:        h.Logs,
	}

	// Translate stream
	err := TranslateStream(ctx, args, c.Writer, flusher)
	if err != nil {
		// Error already sent via SSE
		return
	}
}

// sendChatRequest sends a non-streaming chat completions request.
func (h *RelayHandler) sendChatRequest(chatReq *ChatRequest, upstreamURL, apiKey, authHeader, authValuePrefix string) (*http.Response, error) {
	body, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := buildChatCompletionsURL(upstreamURL)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	applyAuthHeader(req, apiKey, authHeader, authValuePrefix)

	return h.Client.Do(req)
}

// buildChatCompletionsURL builds the Chat Completions URL from the upstream URL.
func buildChatCompletionsURL(upstreamURL string) string {
	base := strings.TrimRight(upstreamURL, "/")
	// If already ends with /v1, append /chat/completions
	if strings.HasSuffix(base, "/v1") {
		return base + "/chat/completions"
	}
	// If doesn't have /v1, check if we should add it
	if !strings.Contains(base, "/v1") {
		base = base + "/v1"
	}
	return base + "/chat/completions"
}

// ── Relay Config API endpoints ─────────────────────────────

// HandleGetRelayConfig handles GET /api/v1/relay/config
func (h *RelayHandler) HandleGetRelayConfig(c *gin.Context) {
	config := h.Config
	if config == nil {
		config = DefaultRelayConfig()
	}

	// Build model map JSON string (prefer persisted JSON over in-memory map)
	modelMapJSON := config.ModelMapJSON
	if modelMapJSON == "" && config.ModelMap != nil {
		if b, err := json.Marshal(config.ModelMap); err == nil {
			modelMapJSON = string(b)
		}
	}

	// Build tool denylist string
	toolDenylistStr := ""
	if config.ToolDenylist != nil {
		var names []string
		for name := range config.ToolDenylist {
			names = append(names, name)
		}
		toolDenylistStr = strings.Join(names, ",")
	}

	relayState := openaiplatform.GetCodexRelayState()
	upstreams := config.Upstreams
	if upstreams == nil {
		upstreams = []RelayUpstream{}
	}
	c.JSON(http.StatusOK, gin.H{
		// Multi-upstream fields
		"upstreams":          upstreams,
		"upstream_strategy":  config.UpstreamStrategy,
		// Legacy single-upstream fields (kept for backward compat)
		"upstream_url":        config.UpstreamURL,
		"api_key":             config.APIKey,
		"auth_header":         config.AuthHeader,
		"auth_value_prefix":   config.AuthValuePrefix,
		// Global options
		"default_model":       config.DefaultModel,
		"model_map_json":      modelMapJSON,
		"tool_denylist_str":   toolDenylistStr,
		"max_sessions":        config.MaxSessions,
		"max_session_bytes":   config.MaxSessionBytes,
		"session_ttl_hours":   config.SessionTTLHours,
		"relay_url":            openaiplatform.LocalRelayServiceURL(c.Request.Host),
		"codex_injected":       relayState.Injected,
		"codex_model_provider": relayState.ModelProvider,
		"codex_model":          relayState.Model,
	})
}

// HandleUpdateRelayConfig handles PUT /api/v1/relay/config
func (h *RelayHandler) HandleUpdateRelayConfig(c *gin.Context) {
	var req struct {
		// Multi-upstream
		Upstreams        []RelayUpstream `json:"upstreams"`
		UpstreamStrategy string          `json:"upstream_strategy"`
		// Legacy single-upstream (honored when Upstreams is empty)
		UpstreamURL     string `json:"upstream_url"`
		APIKey          string `json:"api_key"`
		AuthHeader      string `json:"auth_header"`
		AuthValuePrefix string `json:"auth_value_prefix"`
		// Global options
		DefaultModel    string `json:"default_model"`
		ModelMapJSON    string `json:"model_map_json"`
		ToolDenylistStr string `json:"tool_denylist_str"`
		MaxSessions     int    `json:"max_sessions"`
		MaxSessionBytes int    `json:"max_session_bytes"`
		SessionTTLHours int    `json:"session_ttl_hours"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid request: " + err.Error(),
				"type":    "invalid_request",
			},
		})
		return
	}

	// Parse model map
	modelMap := ParseModelMap(req.ModelMapJSON)

	// Parse tool denylist
	toolDenylist := ParseToolDenylist(req.ToolDenylistStr)

	// Update config
	config := h.Config
	if config == nil {
		config = DefaultRelayConfig()
	}

	// Multi-upstream
	if req.Upstreams != nil {
		config.Upstreams = req.Upstreams
	}
	if req.UpstreamStrategy != "" {
		config.UpstreamStrategy = req.UpstreamStrategy
	}
	// Legacy single-upstream fields
	config.UpstreamURL = req.UpstreamURL
	config.APIKey = req.APIKey
	config.AuthHeader = req.AuthHeader
	config.AuthValuePrefix = req.AuthValuePrefix
	// Global options
	config.DefaultModel = req.DefaultModel
	config.ModelMap = modelMap
	config.ToolDenylist = toolDenylist
	config.ModelMapJSON = req.ModelMapJSON
	config.ToolDenylistStr = req.ToolDenylistStr

	if req.MaxSessions > 0 {
		config.MaxSessions = req.MaxSessions
	}
	if req.MaxSessionBytes > 0 {
		config.MaxSessionBytes = req.MaxSessionBytes
	}
	if req.SessionTTLHours > 0 {
		config.SessionTTLHours = req.SessionTTLHours
	}

	h.Config = config

	if h.Sessions != nil {
		h.Sessions.UpdateLimits(config.MaxSessions, config.MaxSessionBytes, config.SessionTTLHours)
	}

	saveRelayConfigToSettings(config)

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// HandleClearRelaySessions handles POST /api/v1/relay/sessions/clear
func (h *RelayHandler) HandleClearRelaySessions(c *gin.Context) {
	if h.Sessions != nil {
		h.Sessions.ClearAll()
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// HandleGetRelaySessionStats handles GET /api/v1/relay/sessions/stats
func (h *RelayHandler) HandleGetRelaySessionStats(c *gin.Context) {
	if h.Sessions == nil {
		c.JSON(http.StatusOK, gin.H{
			"session_count":    0,
			"reasoning_count":  0,
			"turn_count":       0,
			"stored_bytes":     0,
		})
		return
	}

	sessionCount, reasoningCount, turnCount, bytes := h.Sessions.GetStats()
	c.JSON(http.StatusOK, gin.H{
		"session_count":    sessionCount,
		"reasoning_count":  reasoningCount,
		"turn_count":       turnCount,
		"stored_bytes":     bytes,
	})
}

// HandleGetRelayUsage handles GET /api/v1/relay/usage
func (h *RelayHandler) HandleGetRelayUsage(c *gin.Context) {
	snap := RelayUsageSnapshot{}
	if h.Usage != nil {
		snap = h.Usage.Snapshot()
	}

	upstreamURL := ""
	defaultModel := ""
	if h.Config != nil {
		upstreamURL = h.Config.UpstreamURL
		defaultModel = h.Config.DefaultModel
	}

	recentCalls := []RelayCallRecord{}
	if h.Usage != nil {
		recentCalls = h.Usage.RecentCalls(20)
		if len(recentCalls) == 0 && snap.RequestCount > 0 {
			recentCalls = []RelayCallRecord{{
				Timestamp:     snap.LastRequestAt,
				Provider:      resolveRelayProvider(upstreamURL),
				CodexModel:    snap.LastModel,
				UpstreamModel: defaultModel,
				Stream:        snap.StreamCount > 0,
				InputTokens:   int(snap.InputTokens),
				OutputTokens:  int(snap.OutputTokens),
				TotalTokens:   int(snap.TotalTokens),
				CachedTokens:  int(snap.CachedTokens),
			}}
		}
	}

	relayState := openaiplatform.GetCodexRelayState()
	c.JSON(http.StatusOK, gin.H{
		"usage":               snap,
		"upstream_url":        upstreamURL,
		"upstream_label":      resolveRelayProvider(upstreamURL),
		"default_model":       defaultModel,
		"upstream_configured": upstreamURL != "",
		"codex_injected":      relayState.Injected,
		"relay_url":           openaiplatform.LocalRelayServiceURL(c.Request.Host),
		"recent_calls":        recentCalls,
	})
}

// HandleGetRelayLogs handles GET /api/v1/relay/logs
func (h *RelayHandler) HandleGetRelayLogs(c *gin.Context) {
	limit := 100
	if raw := c.Query("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= defaultRelayLogMax {
			limit = n
		}
	}
	entries := []RelayLogEntry{}
	if h.Logs != nil {
		entries = h.Logs.Recent(limit)
	}
	c.JSON(http.StatusOK, gin.H{"entries": entries})
}

// HandleStreamRelayLogs handles GET /api/v1/relay/logs/stream (SSE).
func (h *RelayHandler) HandleStreamRelayLogs(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Streaming not supported",
				"type":    "internal_error",
			},
		})
		return
	}

	if h.Logs == nil {
		writeRelayLogSSE(c.Writer, flusher, "ready", gin.H{"entries": []RelayLogEntry{}})
		return
	}

	writeRelayLogSSE(c.Writer, flusher, "ready", gin.H{"entries": h.Logs.Recent(100)})

	ch := h.Logs.Subscribe()
	defer h.Logs.Unsubscribe(ch)

	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	ctx := c.Request.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case entry, ok := <-ch:
			if !ok {
				return
			}
			writeRelayLogSSE(c.Writer, flusher, "log", entry)
		case <-ticker.C:
			fmt.Fprintf(c.Writer, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

// HandleClearRelayLogs handles DELETE /api/v1/relay/logs
func (h *RelayHandler) HandleClearRelayLogs(c *gin.Context) {
	if h.Logs != nil {
		h.Logs.Clear()
		h.Logs.Log("info", "日志已清空", "", "")
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func truncateRelayLog(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// HandleInjectCodexConfig handles POST /api/v1/relay/inject-codex.
// It persists relay upstream settings and injects Codex CLI config for the local relay endpoint.
// When multiple upstreams are configured, the first enabled one is used as representative in
// ~/.codex/config.toml; all requests are still load-balanced across all enabled upstreams by EasyLLM.
func (h *RelayHandler) HandleInjectCodexConfig(c *gin.Context) {
	type InjectRequest struct {
		UpstreamURL     string `json:"upstream_url"`
		APIKey          string `json:"api_key"`
		AuthHeader      string `json:"auth_header"`
		AuthValuePrefix string `json:"auth_value_prefix"`
		DefaultModel    string `json:"default_model"`
	}

	var req InjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid request: " + err.Error()})
		return
	}

	config := h.Config
	if config == nil {
		config = DefaultRelayConfig()
	}

	// If the request provides upstream credentials, treat it as adding/updating an upstream.
	if req.UpstreamURL != "" && req.APIKey != "" {
		config.UpstreamURL = req.UpstreamURL
		config.APIKey = req.APIKey
		config.AuthHeader = req.AuthHeader
		config.AuthValuePrefix = req.AuthValuePrefix
	}
	if req.DefaultModel != "" {
		config.DefaultModel = req.DefaultModel
	}

	// Require at least one usable upstream.
	upstream := h.selectUpstreamFromConfig(config)
	if upstream.UpstreamURL == "" || upstream.APIKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "No enabled upstream configured. Add at least one upstream channel with a URL and API key."})
		return
	}

	h.Config = config
	saveRelayConfigToSettings(config)

	modelMap := ParseModelMap(config.ModelMapJSON)
	if len(modelMap) == 0 {
		modelMap = config.ModelMap
	}
	model := PreferredCodexModel(modelMap, config.DefaultModel)
	relayURL := openaiplatform.LocalRelayServiceURL(c.Request.Host)
	if err := openaiplatform.SwitchCodexRelayProvider(relayURL, model, c.Request.Host); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to inject Codex config: " + err.Error()})
		return
	}

	launchResult, err := openaiplatform.RestartCodexApp()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to restart Codex: " + err.Error()})
		return
	}

	relayState := openaiplatform.GetCodexRelayState()
	c.JSON(http.StatusOK, gin.H{
		"success":              true,
		"message":              "Codex configuration injected successfully",
		"relay_url":            relayURL,
		"codex_injected":       relayState.Injected,
		"codex_model_provider": relayState.ModelProvider,
		"codex_model":          relayState.Model,
		"codex_app_started":    launchResult.Started,
		"codex_app_restarted":  launchResult.Restarted,
		"codex_app_was_running": launchResult.RunningBefore,
	})
}

