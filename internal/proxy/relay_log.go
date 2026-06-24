package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const defaultRelayLogMax = 500

// RelayLogEntry is a single relay activity log line.
type RelayLogEntry struct {
	ID         string `json:"id"`
	Timestamp  string `json:"timestamp"`
	Level      string `json:"level"`
	Message    string `json:"message"`
	Model      string `json:"model,omitempty"`
	ResponseID string `json:"response_id,omitempty"`
}

// RelayLogStore keeps recent relay logs and broadcasts new entries to subscribers.
type RelayLogStore struct {
	mu          sync.RWMutex
	entries     []RelayLogEntry
	maxEntries  int
	subscribers map[chan RelayLogEntry]struct{}
	seq         uint64
}

// NewRelayLogStore creates an in-memory relay log buffer.
func NewRelayLogStore() *RelayLogStore {
	return &RelayLogStore{
		maxEntries:  defaultRelayLogMax,
		subscribers: make(map[chan RelayLogEntry]struct{}),
	}
}

// Log appends a log entry and notifies subscribers.
func (s *RelayLogStore) Log(level, message, model, responseID string) {
	if s == nil {
		return
	}
	if level == "" {
		level = "info"
	}

	entry := RelayLogEntry{
		ID:         fmt.Sprintf("rl_%d", atomic.AddUint64(&s.seq, 1)),
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Level:      level,
		Message:    message,
		Model:      model,
		ResponseID: responseID,
	}

	s.mu.Lock()
	s.entries = append(s.entries, entry)
	if len(s.entries) > s.maxEntries {
		s.entries = s.entries[len(s.entries)-s.maxEntries:]
	}
	subs := make([]chan RelayLogEntry, 0, len(s.subscribers))
	for ch := range s.subscribers {
		subs = append(subs, ch)
	}
	s.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- entry:
		default:
		}
	}
}

// Recent returns the latest log entries (oldest first).
func (s *RelayLogStore) Recent(limit int) []RelayLogEntry {
	if s == nil {
		return nil
	}
	if limit <= 0 {
		limit = 100
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.entries) == 0 {
		return []RelayLogEntry{}
	}
	start := 0
	if len(s.entries) > limit {
		start = len(s.entries) - limit
	}
	out := make([]RelayLogEntry, len(s.entries)-start)
	copy(out, s.entries[start:])
	return out
}

// Clear removes all buffered logs.
func (s *RelayLogStore) Clear() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.entries = nil
	s.mu.Unlock()
}

// Subscribe returns a channel that receives new log entries.
func (s *RelayLogStore) Subscribe() chan RelayLogEntry {
	ch := make(chan RelayLogEntry, 64)
	if s == nil {
		return ch
	}
	s.mu.Lock()
	s.subscribers[ch] = struct{}{}
	s.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber channel.
func (s *RelayLogStore) Unsubscribe(ch chan RelayLogEntry) {
	if s == nil || ch == nil {
		return
	}
	s.mu.Lock()
	if _, ok := s.subscribers[ch]; ok {
		delete(s.subscribers, ch)
		close(ch)
	}
	s.mu.Unlock()
}

func writeRelayLogSSE(w io.Writer, flusher http.Flusher, event string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
	if flusher != nil {
		flusher.Flush()
	}
}
