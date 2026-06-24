package proxy

import (
	"encoding/json"
	"easyllm/internal/storage"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	settingRelayUsageStats  = "relay_usage_stats"
	settingRelayCallHistory = "relay_call_history"
	maxRelayCallHistory     = 100
)

// RelayUsageSnapshot is the aggregated Relay mode usage counters.
type RelayUsageSnapshot struct {
	RequestCount  int64  `json:"request_count"`
	StreamCount   int64  `json:"stream_count"`
	InputTokens   int64  `json:"input_tokens"`
	OutputTokens  int64  `json:"output_tokens"`
	TotalTokens   int64  `json:"total_tokens"`
	CachedTokens  int64  `json:"cached_tokens"`
	LastRequestAt string `json:"last_request_at,omitempty"`
	LastModel     string `json:"last_model,omitempty"`
}

// RelayCallRecord is a single completed Relay request for dashboard history.
type RelayCallRecord struct {
	Timestamp     string `json:"timestamp"`
	Provider      string `json:"provider,omitempty"`
	CodexModel    string `json:"codex_model"`
	UpstreamModel string `json:"upstream_model,omitempty"`
	Stream        bool   `json:"stream"`
	InputTokens   int    `json:"input_tokens"`
	OutputTokens  int    `json:"output_tokens"`
	TotalTokens   int    `json:"total_tokens"`
	CachedTokens  int    `json:"cached_tokens,omitempty"`
}

type relayUsagePersisted struct {
	RequestCount int64  `json:"request_count"`
	StreamCount  int64  `json:"stream_count"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
	TotalTokens  int64  `json:"total_tokens"`
	CachedTokens int64  `json:"cached_tokens"`
	LastUnixMs   int64  `json:"last_unix_ms"`
	LastModel    string `json:"last_model"`
}

// RelayUsageStore tracks cumulative Relay API usage (in-memory + persisted).
type RelayUsageStore struct {
	mu           sync.RWMutex
	requestCount int64
	streamCount  int64
	inputTokens  int64
	outputTokens int64
	totalTokens  int64
	cachedTokens int64
	lastUnixMs   int64
	lastModel    string
	recentCalls  []RelayCallRecord
}

// NewRelayUsageStore loads persisted usage or returns an empty store.
func NewRelayUsageStore() *RelayUsageStore {
	store := &RelayUsageStore{}
	store.loadFromSettings()
	store.loadCallHistory()
	return store
}

// Record adds token usage from a completed Relay request.
func (s *RelayUsageStore) Record(model string, stream bool, inputTokens, outputTokens, totalTokens, cachedTokens int) {
	if s == nil {
		return
	}
	atomic.AddInt64(&s.requestCount, 1)
	if stream {
		atomic.AddInt64(&s.streamCount, 1)
	}
	if inputTokens > 0 {
		atomic.AddInt64(&s.inputTokens, int64(inputTokens))
	}
	if outputTokens > 0 {
		atomic.AddInt64(&s.outputTokens, int64(outputTokens))
	}
	if totalTokens > 0 {
		atomic.AddInt64(&s.totalTokens, int64(totalTokens))
	} else if inputTokens > 0 || outputTokens > 0 {
		atomic.AddInt64(&s.totalTokens, int64(inputTokens+outputTokens))
	}
	if cachedTokens > 0 {
		atomic.AddInt64(&s.cachedTokens, int64(cachedTokens))
	}

	now := time.Now().UnixMilli()
	s.mu.Lock()
	s.lastUnixMs = now
	if model != "" {
		s.lastModel = model
	}
	s.mu.Unlock()

	s.persistAsync()
}

// RecordCall stores a per-request history entry for dashboard display.
func (s *RelayUsageStore) RecordCall(record RelayCallRecord) {
	if s == nil {
		return
	}
	if record.Timestamp == "" {
		record.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	s.mu.Lock()
	s.recentCalls = append(s.recentCalls, record)
	if len(s.recentCalls) > maxRelayCallHistory {
		s.recentCalls = s.recentCalls[len(s.recentCalls)-maxRelayCallHistory:]
	}
	s.mu.Unlock()

	s.persistCallHistoryAsync()
}

// RecentCalls returns the latest call records (newest first).
func (s *RelayUsageStore) RecentCalls(limit int) []RelayCallRecord {
	if s == nil {
		return nil
	}
	if limit <= 0 {
		limit = 20
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.recentCalls) == 0 {
		return []RelayCallRecord{}
	}
	start := 0
	if len(s.recentCalls) > limit {
		start = len(s.recentCalls) - limit
	}
	slice := s.recentCalls[start:]
	out := make([]RelayCallRecord, len(slice))
	copy(out, slice)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// Snapshot returns a copy of current usage counters.
func (s *RelayUsageStore) Snapshot() RelayUsageSnapshot {
	if s == nil {
		return RelayUsageSnapshot{}
	}
	s.mu.RLock()
	lastModel := s.lastModel
	lastMs := s.lastUnixMs
	s.mu.RUnlock()

	snap := RelayUsageSnapshot{
		RequestCount: atomic.LoadInt64(&s.requestCount),
		StreamCount:  atomic.LoadInt64(&s.streamCount),
		InputTokens:  atomic.LoadInt64(&s.inputTokens),
		OutputTokens: atomic.LoadInt64(&s.outputTokens),
		TotalTokens:  atomic.LoadInt64(&s.totalTokens),
		CachedTokens: atomic.LoadInt64(&s.cachedTokens),
		LastModel:    lastModel,
	}
	if lastMs > 0 {
		snap.LastRequestAt = time.UnixMilli(lastMs).UTC().Format(time.RFC3339)
	}
	return snap
}

// Clear resets all usage counters.
func (s *RelayUsageStore) Clear() {
	if s == nil {
		return
	}
	atomic.StoreInt64(&s.requestCount, 0)
	atomic.StoreInt64(&s.streamCount, 0)
	atomic.StoreInt64(&s.inputTokens, 0)
	atomic.StoreInt64(&s.outputTokens, 0)
	atomic.StoreInt64(&s.totalTokens, 0)
	atomic.StoreInt64(&s.cachedTokens, 0)
	s.mu.Lock()
	s.lastUnixMs = 0
	s.lastModel = ""
	s.recentCalls = nil
	s.mu.Unlock()
	saveRelayUsageStats("{}")
	saveRelayCallHistory("[]")
}

func saveRelayUsageStats(jsonValue string) {
	if storage.DB == nil {
		return
	}
	_ = storage.SaveSetting(settingRelayUsageStats, jsonValue)
}

func (s *RelayUsageStore) loadFromSettings() {
	raw, ok := storage.GetSetting(settingRelayUsageStats)
	if !ok || raw == "" || raw == "{}" {
		return
	}
	var data relayUsagePersisted
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return
	}
	atomic.StoreInt64(&s.requestCount, data.RequestCount)
	atomic.StoreInt64(&s.streamCount, data.StreamCount)
	atomic.StoreInt64(&s.inputTokens, data.InputTokens)
	atomic.StoreInt64(&s.outputTokens, data.OutputTokens)
	atomic.StoreInt64(&s.totalTokens, data.TotalTokens)
	atomic.StoreInt64(&s.cachedTokens, data.CachedTokens)
	s.mu.Lock()
	s.lastUnixMs = data.LastUnixMs
	s.lastModel = data.LastModel
	s.mu.Unlock()
}

func (s *RelayUsageStore) persistAsync() {
	go s.persist()
}

func (s *RelayUsageStore) persist() {
	if s == nil {
		return
	}
	s.mu.RLock()
	data := relayUsagePersisted{
		RequestCount: atomic.LoadInt64(&s.requestCount),
		StreamCount:  atomic.LoadInt64(&s.streamCount),
		InputTokens:  atomic.LoadInt64(&s.inputTokens),
		OutputTokens: atomic.LoadInt64(&s.outputTokens),
		TotalTokens:  atomic.LoadInt64(&s.totalTokens),
		CachedTokens: atomic.LoadInt64(&s.cachedTokens),
		LastUnixMs:   s.lastUnixMs,
		LastModel:    s.lastModel,
	}
	s.mu.RUnlock()
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	saveRelayUsageStats(string(b))
}

func saveRelayCallHistory(jsonValue string) {
	if storage.DB == nil {
		return
	}
	_ = storage.SaveSetting(settingRelayCallHistory, jsonValue)
}

func (s *RelayUsageStore) loadCallHistory() {
	raw, ok := storage.GetSetting(settingRelayCallHistory)
	if !ok || raw == "" || raw == "[]" {
		return
	}
	var calls []RelayCallRecord
	if err := json.Unmarshal([]byte(raw), &calls); err != nil {
		return
	}
	s.mu.Lock()
	s.recentCalls = calls
	s.mu.Unlock()
}

func (s *RelayUsageStore) persistCallHistoryAsync() {
	go s.persistCallHistory()
}

func (s *RelayUsageStore) persistCallHistory() {
	if s == nil {
		return
	}
	s.mu.RLock()
	calls := make([]RelayCallRecord, len(s.recentCalls))
	copy(calls, s.recentCalls)
	s.mu.RUnlock()
	b, err := json.Marshal(calls)
	if err != nil {
		return
	}
	saveRelayCallHistory(string(b))
}

func resolveRelayProvider(upstreamURL string) string {
	u := strings.ToLower(upstreamURL)
	switch {
	case strings.Contains(u, "xiaomimimo.com"):
		return "小米 MiMo"
	case strings.Contains(u, "deepseek.com"):
		return "DeepSeek"
	case strings.Contains(u, "moonshot.cn"):
		return "Kimi"
	case strings.Contains(u, "dashscope.aliyuncs.com"):
		return "通义千问"
	case strings.Contains(u, "openai.com"):
		return "OpenAI"
	case strings.Contains(u, "openrouter.ai"):
		return "OpenRouter"
	default:
		return "自定义上游"
	}
}

func recordResponsesUsage(store *RelayUsageStore, upstreamURL, codexModel, upstreamModel string, stream bool, usage ResponsesUsage) {
	if store == nil {
		return
	}
	cached := 0
	if usage.InputTokensDetails != nil {
		cached = usage.InputTokensDetails.CachedTokens
	}
	store.Record(codexModel, stream, usage.InputTokens, usage.OutputTokens, usage.TotalTokens, cached)
	recordRelayCall(store, upstreamURL, codexModel, upstreamModel, stream, usage.InputTokens, usage.OutputTokens, usage.TotalTokens, cached)
}

func recordChatUsage(store *RelayUsageStore, upstreamURL, codexModel, upstreamModel string, stream bool, usage *ChatUsage) {
	if store == nil || usage == nil {
		return
	}
	store.Record(codexModel, stream, usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens, usage.CacheHit())
	recordRelayCall(store, upstreamURL, codexModel, upstreamModel, stream, usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens, usage.CacheHit())
}

func recordRelayCall(store *RelayUsageStore, upstreamURL, codexModel, upstreamModel string, stream bool, input, output, total, cached int) {
	if store == nil {
		return
	}
	store.RecordCall(RelayCallRecord{
		Provider:      resolveRelayProvider(upstreamURL),
		CodexModel:    codexModel,
		UpstreamModel: upstreamModel,
		Stream:        stream,
		InputTokens:   input,
		OutputTokens:  output,
		TotalTokens:   total,
		CachedTokens:  cached,
	})
}
