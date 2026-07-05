package proxy

import (
	"easyllm/internal/storage"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	settingCodexCallHistory = "codex_proxy_call_history"
	maxCodexCallHistory     = 100
)

// CodexCallRecord is a metadata-only record for local Codex proxy traffic.
type CodexCallRecord struct {
	Timestamp     string `json:"timestamp"`
	AccountID     string `json:"account_id,omitempty"`
	AccountEmail  string `json:"account_email,omitempty"`
	AccountSource string `json:"account_source,omitempty"`
	Method        string `json:"method"`
	Path          string `json:"path"`
	Model         string `json:"model,omitempty"`
	StatusCode    int    `json:"status_code"`
	Stream        bool   `json:"stream"`
	Passthrough   bool   `json:"passthrough"`
	DurationMs    int64  `json:"duration_ms"`
	Error         string `json:"error,omitempty"`
}

// CodexCallHistoryStore keeps the recent local Codex proxy calls.
type CodexCallHistoryStore struct {
	mu    sync.RWMutex
	calls []CodexCallRecord
}

func NewCodexCallHistoryStore() *CodexCallHistoryStore {
	store := &CodexCallHistoryStore{}
	store.load()
	return store
}

func (s *CodexCallHistoryStore) Record(record CodexCallRecord) {
	if s == nil {
		return
	}
	if record.Timestamp == "" {
		record.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	if record.DurationMs < 0 {
		record.DurationMs = 0
	}

	s.mu.Lock()
	s.calls = append(s.calls, record)
	if len(s.calls) > maxCodexCallHistory {
		s.calls = s.calls[len(s.calls)-maxCodexCallHistory:]
	}
	s.mu.Unlock()

	s.persistAsync()
}

func (s *CodexCallHistoryStore) Recent(limit int) []CodexCallRecord {
	if s == nil {
		return nil
	}
	if limit <= 0 {
		limit = 20
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.calls) == 0 {
		return []CodexCallRecord{}
	}
	start := 0
	if len(s.calls) > limit {
		start = len(s.calls) - limit
	}
	slice := s.calls[start:]
	out := make([]CodexCallRecord, len(slice))
	copy(out, slice)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func (s *CodexCallHistoryStore) Clear() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.calls = nil
	s.mu.Unlock()
	saveCodexCallHistory("[]")
}

func (s *CodexCallHistoryStore) load() {
	raw, ok := storage.GetSetting(settingCodexCallHistory)
	if !ok || raw == "" || raw == "[]" {
		return
	}
	var calls []CodexCallRecord
	if err := json.Unmarshal([]byte(raw), &calls); err != nil {
		return
	}
	s.mu.Lock()
	s.calls = calls
	s.mu.Unlock()
}

func (s *CodexCallHistoryStore) persistAsync() {
	go s.persist()
}

func (s *CodexCallHistoryStore) persist() {
	if s == nil {
		return
	}
	s.mu.RLock()
	calls := make([]CodexCallRecord, len(s.calls))
	copy(calls, s.calls)
	s.mu.RUnlock()
	b, err := json.Marshal(calls)
	if err != nil {
		return
	}
	saveCodexCallHistory(string(b))
}

func saveCodexCallHistory(jsonValue string) {
	if storage.DB == nil {
		return
	}
	_ = storage.SaveSetting(settingCodexCallHistory, jsonValue)
}

func (p *CodexProxy) RecentCalls(limit int) []CodexCallRecord {
	if p == nil || p.history == nil {
		return []CodexCallRecord{}
	}
	return p.history.Recent(limit)
}

func (p *CodexProxy) ClearHistory() {
	if p == nil || p.history == nil {
		return
	}
	p.history.Clear()
}

func (p *CodexProxy) recordCall(entry *poolEntry, r *http.Request, body []byte, statusCode int, passthrough bool, start time.Time, errMsg string) {
	if p == nil || p.history == nil || r == nil {
		return
	}
	model, stream := parseCodexCallMetadata(body)
	if strings.EqualFold(r.Header.Get("Accept"), "text/event-stream") || isCodexWebSocketRequest(r) {
		stream = true
	}
	if errMsg != "" {
		errMsg = truncateRelayLog(errMsg, 200)
	}
	record := CodexCallRecord{
		Method:      r.Method,
		Path:        r.URL.Path,
		Model:       model,
		StatusCode:  statusCode,
		Stream:      stream,
		Passthrough: passthrough,
		DurationMs:  time.Since(start).Milliseconds(),
		Error:       errMsg,
	}
	if entry != nil {
		record.AccountID = entry.id
		record.AccountEmail = entry.email
		record.AccountSource = entry.source
	}
	p.history.Record(record)
}

func isCodexWebSocketRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	for _, v := range r.Header["Connection"] {
		for _, part := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(part), "upgrade") {
				return strings.EqualFold(strings.TrimSpace(r.Header.Get("Upgrade")), "websocket")
			}
		}
	}
	return false
}

func parseCodexCallMetadata(body []byte) (model string, stream bool) {
	if len(body) == 0 {
		return "", false
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", false
	}
	if m, ok := payload["model"].(string); ok {
		model = m
	}
	if v, ok := payload["stream"].(bool); ok {
		stream = v
	}
	return model, stream
}
