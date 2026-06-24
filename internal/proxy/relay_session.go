package proxy

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultMaxSessions       = 256
	DefaultMaxSessionBytes   = 512 * 1024 * 1024 // 512MB
	DefaultSessionTTLHours = 168                       // 7 days
)

// DiskStore handles persistent storage of sessions and reasoning.
type DiskStore struct {
	root         string
	sessionsDir string
	reasoningDir string
	turnsDir     string
}

// NewDiskStore creates a new disk store at the specified root directory.
func NewDiskStore(root string) (*DiskStore, error) {
	ds := &DiskStore{
		root:         root,
		sessionsDir: filepath.Join(root, "sessions"),
		reasoningDir: filepath.Join(root, "reasoning"),
		turnsDir:     filepath.Join(root, "turns"),
	}

	// Create directories if they don't exist
	if err := os.MkdirAll(ds.sessionsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sessions dir: %w", err)
	}
	if err := os.MkdirAll(ds.reasoningDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create reasoning dir: %w", err)
	}
	if err := os.MkdirAll(ds.turnsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create turns dir: %w", err)
	}

	return ds, nil
}

// ── DiskStore methods ─────────────────────────────────────────────

func (ds *DiskStore) sessionPath(id string) string {
	return filepath.Join(ds.sessionsDir, encodeKey(id)+".json")
}

func (ds *DiskStore) reasoningPath(key string) string {
	return filepath.Join(ds.reasoningDir, encodeKey(key)+".json")
}

func (ds *DiskStore) turnPath(key uint64) string {
	return filepath.Join(ds.turnsDir, fmt.Sprintf("%d.json", key))
}

// writeJSONAtomic writes JSON to disk atomically.
func (ds *DiskStore) writeJSONAtomic(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to temporary file first, then rename atomically
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// readJSON reads JSON from disk.
func (ds *DiskStore) readJSON(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// ── Session store types ──────────────────────────────────────────

// RelaySessionEntry stores session data.
type RelaySessionEntry struct {
	Messages   []ChatMessage
	Bytes      int
	LastUsedAt time.Time
}

// StoredString stores a string value with metadata.
type StoredString struct {
	Value      *string
	Bytes      int
	LastUsedAt time.Time
}

// RelaySessionStore manages conversation history for Relay mode.
type RelaySessionStore struct {
	mu             sync.RWMutex
	sessions       map[string]*RelaySessionEntry
	sessionOrder   []string
	reasoning      map[string]*StoredString
	reasoningOrder []string
	turnReasoning  map[uint64]*StoredString
	turnOrder      []uint64
	storedBytes    int
	maxSessions    int
	maxStoredBytes int
	ttl            time.Duration
	disk           *DiskStore
}

// NewRelaySessionStore creates a new session store with default limits.
func NewRelaySessionStore() *RelaySessionStore {
	return &RelaySessionStore{
		sessions:       make(map[string]*RelaySessionEntry),
		sessionOrder:   make([]string, 0),
		reasoning:      make(map[string]*StoredString),
		reasoningOrder: make([]string, 0),
		turnReasoning:  make(map[uint64]*StoredString),
		turnOrder:      make([]uint64, 0),
		storedBytes:    0,
		maxSessions:    DefaultMaxSessions,
		maxStoredBytes: DefaultMaxSessionBytes,
		ttl:            time.Duration(DefaultSessionTTLHours) * time.Hour,
		disk:           nil,
	}
}

// NewRelaySessionStoreWithDisk creates a new session store with disk persistence.
func NewRelaySessionStoreWithDisk(rootDir string) (*RelaySessionStore, error) {
	disk, err := NewDiskStore(rootDir)
	if err != nil {
		return nil, err
	}

	store := &RelaySessionStore{
		sessions:       make(map[string]*RelaySessionEntry),
		sessionOrder:   make([]string, 0),
		reasoning:      make(map[string]*StoredString),
		reasoningOrder: make([]string, 0),
		turnReasoning:  make(map[uint64]*StoredString),
		turnOrder:      make([]uint64, 0),
		storedBytes:    0,
		maxSessions:    DefaultMaxSessions,
		maxStoredBytes: DefaultMaxSessionBytes,
		ttl:            time.Duration(DefaultSessionTTLHours) * time.Hour,
		disk:           disk,
	}

	// Load existing data from disk
	store.loadDiskIndex()

	return store, nil
}

// NewRelaySessionStoreWithLimits creates a new session store with custom limits.
func NewRelaySessionStoreWithLimits(maxSessions, maxStoredBytes, ttlHours int, diskDir string) (*RelaySessionStore, error) {
	if maxSessions <= 0 {
		maxSessions = DefaultMaxSessions
	}
	if maxStoredBytes <= 0 {
		maxStoredBytes = DefaultMaxSessionBytes
	}
	ttl := time.Duration(ttlHours) * time.Hour
	if ttl <= 0 {
		ttl = time.Duration(DefaultSessionTTLHours) * time.Hour
	}

	store := &RelaySessionStore{
		sessions:       make(map[string]*RelaySessionEntry),
		sessionOrder:   make([]string, 0),
		reasoning:      make(map[string]*StoredString),
		reasoningOrder: make([]string, 0),
		turnReasoning:  make(map[uint64]*StoredString),
		turnOrder:      make([]uint64, 0),
		storedBytes:    0,
		maxSessions:    maxSessions,
		maxStoredBytes: maxStoredBytes,
		ttl:            ttl,
	}

	if diskDir != "" {
		disk, err := NewDiskStore(diskDir)
		if err != nil {
			return nil, err
		}
		store.disk = disk
		store.loadDiskIndex()
	}

	return store, nil
}

// ── Public API ──────────────────────────────────────────────────

// GetHistory returns the message history for a response_id.
func (s *RelaySessionStore) GetHistory(responseID string) []ChatMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enforceLimits()
	entry, ok := s.sessions[responseID]
	if !ok {
		return nil
	}
	s.touchSession(responseID)
	// Return a copy
	msgs := make([]ChatMessage, len(entry.Messages))
	copy(msgs, entry.Messages)
	return msgs
}

// SaveWithID stores messages under a pre-allocated response_id (used by streaming path).
func (s *RelaySessionStore) SaveWithID(id string, messages []ChatMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.insertSession(id, messages)
	s.enforceLimits()
}

// Save stores messages and returns a new response_id (used by non-streaming path).
func (s *RelaySessionStore) Save(messages []ChatMessage) string {
	id := fmt.Sprintf("resp_%s", generateShortID())
	s.SaveWithID(id, messages)
	return id
}

// NewID allocates a fresh response_id without storing anything yet.
func (s *RelaySessionStore) NewID() string {
	return fmt.Sprintf("resp_%s", generateShortID())
}

// Cleanup removes expired or over-budget entries.
func (s *RelaySessionStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enforceLimits()
}

// ── Reasoning content round-trip ──────────────────────────────────────

// StoreReasoning stores reasoning_content keyed by tool call_id.
func (s *RelaySessionStore) StoreReasoning(callID string, reasoning string) {
	if reasoning == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.insertReasoning(callID, reasoning)
	s.enforceLimits()
}

// GetReasoning looks up stored reasoning_content for a call_id.
func (s *RelaySessionStore) GetReasoning(callID string) *string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enforceLimits()
	entry, ok := s.reasoning[callID]
	if !ok || entry.Value == nil {
		return nil
	}
	s.touchReasoning(callID)
	return entry.Value
}

// StoreTurnReasoning stores reasoning_content keyed by a fingerprint of the
// assistant message. This allows recovery when Codex replays the full
// conversation in input[] without using previous_response_id.
func (s *RelaySessionStore) StoreTurnReasoning(prior []ChatMessage, assistant *ChatMessage, reasoning string) {
	if reasoning == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store under content-only key
	content := assistant.TextContent()
	if content != "" {
		key := contentKey(content)
		s.insertTurnReasoning(key, reasoning)
	}
	// Also store under each tool call_id
	if assistant.ToolCalls != nil {
		for _, tc := range assistant.ToolCalls {
			if m, ok := tc.(map[string]interface{}); ok {
				if id, ok := m["id"].(string); ok && id != "" {
					s.insertReasoning(id, reasoning)
				}
			}
		}
	}
	s.enforceLimits()
}

// GetTurnReasoning looks up reasoning_content for an assistant turn by its text content.
func (s *RelaySessionStore) GetTurnReasoning(prior []ChatMessage, assistant *ChatMessage) *string {
	content := assistant.TextContent()
	if content == "" {
		return nil
	}
	key := contentKey(content)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enforceLimits()
	entry, ok := s.turnReasoning[key]
	if !ok || entry.Value == nil {
		return nil
	}
	s.touchTurnReasoning(key)
	return entry.Value
}

// ── Internal methods ────────────────────────────────────────────────

func (s *RelaySessionStore) insertSession(id string, messages []ChatMessage) {
	bytes := messagesBytes(messages)
	if bytes > s.maxStoredBytes {
		// Too large, don't cache
		return
	}
	s.removeSession(id)
	now := time.Now()
	s.sessions[id] = &RelaySessionEntry{
		Messages:   messages,
		Bytes:      bytes,
		LastUsedAt: now,
	}
	s.storedBytes += bytes
	s.sessionOrder = append(s.sessionOrder, id)

	// Persist to disk if enabled
	if s.disk != nil {
		if err := s.disk.writeJSONAtomic(s.disk.sessionPath(id), map[string]interface{}{
			"schema_version":     1,
			"response_id":        id,
			"created_at_unix_ms": now.UnixMilli(),
			"last_used_at_unix_ms": now.UnixMilli(),
			"bytes":              bytes,
			"messages":           messages,
		}); err != nil {
			// Log error but don't fail
		}
	}
}

func (s *RelaySessionStore) insertReasoning(callID string, reasoning string) {
	if old, ok := s.reasoning[callID]; ok {
		s.storedBytes -= old.Bytes
		delete(s.reasoning, callID)
	}
	var order []string
	for _, k := range s.reasoningOrder {
		if k != callID {
			order = append(order, k)
		}
	}
	s.reasoningOrder = order

	bytes := len(callID) + len(reasoning)
	now := time.Now()
	value := reasoning
	s.reasoning[callID] = &StoredString{
		Value:      &value,
		Bytes:      bytes,
		LastUsedAt: now,
	}
	s.storedBytes += bytes
	s.reasoningOrder = append(s.reasoningOrder, callID)

	// Persist to disk if enabled
	if s.disk != nil {
		if err := s.disk.writeJSONAtomic(s.disk.reasoningPath(callID), map[string]interface{}{
			"schema_version":     1,
			"key":                callID,
			"created_at_unix_ms": now.UnixMilli(),
			"last_used_at_unix_ms": now.UnixMilli(),
			"bytes":              bytes,
			"value":              reasoning,
		}); err != nil {
			// Log error but don't fail
		}
	}
}

func (s *RelaySessionStore) insertTurnReasoning(key uint64, reasoning string) {
	if old, ok := s.turnReasoning[key]; ok {
		s.storedBytes -= old.Bytes
		delete(s.turnReasoning, key)
	}
	var order []uint64
	for _, k := range s.turnOrder {
		if k != key {
			order = append(order, k)
		}
	}
	s.turnOrder = order

	bytes := 8 + len(reasoning) // uint64 key + string
	now := time.Now()
	value := reasoning
	s.turnReasoning[key] = &StoredString{
		Value:      &value,
		Bytes:      bytes,
		LastUsedAt: now,
	}
	s.storedBytes += bytes
	s.turnOrder = append(s.turnOrder, key)

	// Persist to disk if enabled
	if s.disk != nil {
		if err := s.disk.writeJSONAtomic(s.disk.turnPath(key), map[string]interface{}{
			"schema_version":     1,
			"key":                fmt.Sprintf("%d", key),
			"created_at_unix_ms": now.UnixMilli(),
			"last_used_at_unix_ms": now.UnixMilli(),
			"bytes":              bytes,
			"value":              reasoning,
		}); err != nil {
			// Log error but don't fail
		}
	}
}

func (s *RelaySessionStore) enforceLimits() {
	// Remove expired
	cutoff := time.Now().Add(-s.ttl)
	for len(s.sessionOrder) > 0 {
		id := s.sessionOrder[0]
		entry, ok := s.sessions[id]
		if !ok {
			s.removeSessionOrderHead()
			continue
		}
		if entry.LastUsedAt.Before(cutoff) {
			s.removeSession(id)
		} else {
			break
		}
	}
	for len(s.reasoningOrder) > 0 {
		key := s.reasoningOrder[0]
		entry, ok := s.reasoning[key]
		if !ok || entry.LastUsedAt.Before(cutoff) {
			s.removeReasoningEntry(key)
		} else {
			break
		}
	}
	for len(s.turnOrder) > 0 {
		key := s.turnOrder[0]
		entry, ok := s.turnReasoning[key]
		if !ok || entry.LastUsedAt.Before(cutoff) {
			s.removeTurnEntry(key)
		} else {
			break
		}
	}

	// Enforce max sessions
	for len(s.sessions) > s.maxSessions && len(s.sessionOrder) > 0 {
		s.removeSession(s.sessionOrder[0])
	}

	// Enforce max bytes
	for s.storedBytes > s.maxStoredBytes && len(s.sessionOrder) > 0 {
		s.removeSession(s.sessionOrder[0])
	}
	for s.storedBytes > s.maxStoredBytes && len(s.reasoningOrder) > 0 {
		s.removeReasoningEntry(s.reasoningOrder[0])
	}
	for s.storedBytes > s.maxStoredBytes && len(s.turnOrder) > 0 {
		s.removeTurnEntry(s.turnOrder[0])
	}
}

func (s *RelaySessionStore) removeSessionOrderHead() {
	if len(s.sessionOrder) == 0 {
		return
	}
	s.sessionOrder = s.sessionOrder[1:]
}

func (s *RelaySessionStore) removeSession(id string) {
	var order []string
	for _, k := range s.sessionOrder {
		if k != id {
			order = append(order, k)
		}
	}
	s.sessionOrder = order
	s.removeSessionEntry(id)
}

// UpdateLimits updates eviction limits without discarding stored sessions.
func (s *RelaySessionStore) UpdateLimits(maxSessions, maxStoredBytes int, ttlHours int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if maxSessions > 0 {
		s.maxSessions = maxSessions
	}
	if maxStoredBytes > 0 {
		s.maxStoredBytes = maxStoredBytes
	}
	if ttlHours > 0 {
		s.ttl = time.Duration(ttlHours) * time.Hour
	}
	s.enforceLimits()
}

func (s *RelaySessionStore) removeSessionEntry(id string) {
	if entry, ok := s.sessions[id]; ok {
		s.storedBytes -= entry.Bytes
		delete(s.sessions, id)
	}
	// Remove from disk if enabled
	if s.disk != nil {
		os.Remove(s.disk.sessionPath(id))
	}
}

func (s *RelaySessionStore) removeReasoningEntry(key string) {
	if entry, ok := s.reasoning[key]; ok {
		s.storedBytes -= entry.Bytes
		delete(s.reasoning, key)
	}
	var order []string
	for _, k := range s.reasoningOrder {
		if k != key {
			order = append(order, k)
		}
	}
	s.reasoningOrder = order
	// Remove from disk if enabled
	if s.disk != nil {
		os.Remove(s.disk.reasoningPath(key))
	}
}

func (s *RelaySessionStore) removeTurnEntry(key uint64) {
	if entry, ok := s.turnReasoning[key]; ok {
		s.storedBytes -= entry.Bytes
		delete(s.turnReasoning, key)
	}
	var order []uint64
	for _, k := range s.turnOrder {
		if k != key {
			order = append(order, k)
		}
	}
	s.turnOrder = order
	// Remove from disk if enabled
	if s.disk != nil {
		os.Remove(s.disk.turnPath(key))
	}
}

func (s *RelaySessionStore) touchSession(id string) {
	now := time.Now()
	if entry, ok := s.sessions[id]; ok {
		entry.LastUsedAt = now
	}
	var order []string
	for _, k := range s.sessionOrder {
		if k != id {
			order = append(order, k)
		}
	}
	s.sessionOrder = append(order, id)

	// Disk update: snapshot path outside lock, write after return.
	if s.disk != nil {
		path := s.disk.sessionPath(id)
		go func() {
			if data, err := os.ReadFile(path); err == nil {
				var record map[string]interface{}
				if json.Unmarshal(data, &record) == nil {
					record["last_used_at_unix_ms"] = now.UnixMilli()
					s.disk.writeJSONAtomic(path, record)
				}
			}
		}()
	}
}

func (s *RelaySessionStore) touchReasoning(callID string) {
	now := time.Now()
	if entry, ok := s.reasoning[callID]; ok {
		entry.LastUsedAt = now
	}
	var order []string
	for _, k := range s.reasoningOrder {
		if k != callID {
			order = append(order, k)
		}
	}
	s.reasoningOrder = append(order, callID)

	if s.disk != nil {
		path := s.disk.reasoningPath(callID)
		go func() {
			if data, err := os.ReadFile(path); err == nil {
				var record map[string]interface{}
				if json.Unmarshal(data, &record) == nil {
					record["last_used_at_unix_ms"] = now.UnixMilli()
					s.disk.writeJSONAtomic(path, record)
				}
			}
		}()
	}
}

func (s *RelaySessionStore) touchTurnReasoning(key uint64) {
	now := time.Now()
	if entry, ok := s.turnReasoning[key]; ok {
		entry.LastUsedAt = now
	}
	var order []uint64
	for _, k := range s.turnOrder {
		if k != key {
			order = append(order, k)
		}
	}
	s.turnOrder = append(order, key)

	if s.disk != nil {
		path := s.disk.turnPath(key)
		go func() {
			if data, err := os.ReadFile(path); err == nil {
				var record map[string]interface{}
				if json.Unmarshal(data, &record) == nil {
					record["last_used_at_unix_ms"] = now.UnixMilli()
					s.disk.writeJSONAtomic(path, record)
				}
			}
		}()
	}
}

// ── Disk loading ──────────────────────────────────────────────────

func (s *RelaySessionStore) loadDiskIndex() {
	if s.disk == nil {
		return
	}

	// Load sessions
	if entries, err := os.ReadDir(s.disk.sessionsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			path := filepath.Join(s.disk.sessionsDir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var record map[string]interface{}
			if err := json.Unmarshal(data, &record); err != nil {
				continue
			}

			responseID, _ := record["response_id"].(string)
			if responseID == "" {
				continue
			}

			// Parse messages
			var messages []ChatMessage
			if msgData, ok := record["messages"]; ok {
				if b, err := json.Marshal(msgData); err == nil {
					json.Unmarshal(b, &messages)
				}
			}

			bytes := int(getFloat(record["bytes"]))
			lastUsed := time.Now()
			if unixMs := getFloat(record["last_used_at_unix_ms"]); unixMs > 0 {
				lastUsed = time.UnixMilli(int64(unixMs))
			}

			s.storedBytes += bytes
			s.sessions[responseID] = &RelaySessionEntry{
				Messages:   messages,
				Bytes:      bytes,
				LastUsedAt: lastUsed,
			}
			s.sessionOrder = append(s.sessionOrder, responseID)
		}
	}

	// Load reasoning
	if entries, err := os.ReadDir(s.disk.reasoningDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			path := filepath.Join(s.disk.reasoningDir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var record map[string]interface{}
			if err := json.Unmarshal(data, &record); err != nil {
				continue
			}

			key, _ := record["key"].(string)
			if key == "" {
				continue
			}

			value, _ := record["value"].(string)
			bytes := int(getFloat(record["bytes"]))
			lastUsed := time.Now()
			if unixMs := getFloat(record["last_used_at_unix_ms"]); unixMs > 0 {
				lastUsed = time.UnixMilli(int64(unixMs))
			}

			s.storedBytes += bytes
			s.reasoning[key] = &StoredString{
				Value:      &value,
				Bytes:      bytes,
				LastUsedAt: lastUsed,
			}
			s.reasoningOrder = append(s.reasoningOrder, key)
		}
	}

	// Load turn reasoning
	if entries, err := os.ReadDir(s.disk.turnsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			path := filepath.Join(s.disk.turnsDir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var record map[string]interface{}
			if err := json.Unmarshal(data, &record); err != nil {
				continue
			}

			keyStr, _ := record["key"].(string)
			key := uint64(0)
			if n, err := fmt.Sscanf(keyStr, "%d", &key); err != nil || n != 1 {
				continue
			}

			value, _ := record["value"].(string)
			bytes := int(getFloat(record["bytes"]))
			lastUsed := time.Now()
			if unixMs := getFloat(record["last_used_at_unix_ms"]); unixMs > 0 {
				lastUsed = time.UnixMilli(int64(unixMs))
			}

			s.storedBytes += bytes
			s.turnReasoning[key] = &StoredString{
				Value:      &value,
				Bytes:      bytes,
				LastUsedAt: lastUsed,
			}
			s.turnOrder = append(s.turnOrder, key)
		}
	}
}

// ── Helpers ─────────────────────────────────────────────────────────

var relayIDCounter uint64

func generateShortID() string {
	counter := atomic.AddUint64(&relayIDCounter, 1)
	buf := make([]byte, 8)
	_, _ = rand.Read(buf)
	h := sha256.New()
	h.Write(buf)
	h.Write([]byte(fmt.Sprintf("%d-%d", time.Now().UnixNano(), counter)))
	hash := fmt.Sprintf("%x", h.Sum(nil))
	if len(hash) > 16 {
		hash = hash[:16]
	}
	return hash
}

func contentKey(content string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(content))
	return h.Sum64()
}

func messagesBytes(messages []ChatMessage) int {
	total := 0
	for _, m := range messages {
		total += messageBytes(m)
	}
	return total
}

func messageBytes(m ChatMessage) int {
	total := len(m.Role)
	if m.Content != nil {
		if s, ok := m.Content.(string); ok {
			total += len(s)
		} else if b, err := json.Marshal(m.Content); err == nil {
			total += len(b)
		}
	}
	if m.ReasoningContent != nil {
		total += len(*m.ReasoningContent)
	}
	if m.ToolCalls != nil {
		if b, err := json.Marshal(m.ToolCalls); err == nil {
			total += len(b)
		}
	}
	if m.ToolCallID != nil {
		total += len(*m.ToolCallID)
	}
	if m.Name != nil {
		total += len(*m.Name)
	}
	return total
}

// encodeKey encodes a key for safe filesystem usage.
func encodeKey(key string) string {
	var out strings.Builder
	for _, b := range []byte(key) {
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_' || b == '-' || b == '.' {
			out.WriteByte(b)
		} else {
			out.WriteString(fmt.Sprintf("%%%02X", b))
		}
	}
	return out.String()
}

// getFloat extracts a float64 from an interface{}.
func getFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}

// GetStats returns session store statistics.
func (s *RelaySessionStore) GetStats() (sessionCount int, reasoningCount int, turnCount int, bytes int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions), len(s.reasoning), len(s.turnReasoning), s.storedBytes
}

// ClearAll clears all stored sessions and reasoning.
func (s *RelaySessionStore) ClearAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions = make(map[string]*RelaySessionEntry)
	s.sessionOrder = nil
	s.reasoning = make(map[string]*StoredString)
	s.reasoningOrder = nil
	s.turnReasoning = make(map[uint64]*StoredString)
	s.turnOrder = nil
	s.storedBytes = 0

	// Clear disk if enabled
	if s.disk != nil {
		os.RemoveAll(s.disk.sessionsDir)
		os.RemoveAll(s.disk.reasoningDir)
		os.RemoveAll(s.disk.turnsDir)
		os.MkdirAll(s.disk.sessionsDir, 0755)
		os.MkdirAll(s.disk.reasoningDir, 0755)
		os.MkdirAll(s.disk.turnsDir, 0755)
	}
}
