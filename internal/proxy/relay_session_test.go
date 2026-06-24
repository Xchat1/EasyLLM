package proxy

import (
	"testing"
	"time"
)

func TestStoreAndGetReasoning(t *testing.T) {
	store := NewRelaySessionStore()
	store.StoreReasoning("call_1", "think")
	result := store.GetReasoning("call_1")
	if result == nil || *result != "think" {
		t.Errorf("expected 'think', got '%v'", result)
	}
}

func TestGetReasoningMissing(t *testing.T) {
	store := NewRelaySessionStore()
	result := store.GetReasoning("nonexistent")
	if result != nil {
		t.Errorf("expected nil, got '%v'", result)
	}
}

func TestEmptyReasoningNotStored(t *testing.T) {
	store := NewRelaySessionStore()
	store.StoreReasoning("call_e", "")
	result := store.GetReasoning("call_e")
	if result != nil {
		t.Error("expected nil for empty reasoning")
	}
}

func TestTurnReasoningByContent(t *testing.T) {
	store := NewRelaySessionStore()
	assistant := ChatMessage{
		Role:    "assistant",
		Content: "hello world",
	}
	store.StoreTurnReasoning(nil, &assistant, "deep thought")

	result := store.GetTurnReasoning(nil, &assistant)
	if result == nil || *result != "deep thought" {
		t.Errorf("expected 'deep thought', got '%v'", result)
	}
}

func TestTurnReasoningEmptyContent(t *testing.T) {
	store := NewRelaySessionStore()
	assistant := ChatMessage{
		Role:    "assistant",
		Content: "",
	}
	store.StoreTurnReasoning(nil, &assistant, "reason")
	result := store.GetTurnReasoning(nil, &assistant)
	if result != nil {
		t.Error("expected nil for empty content")
	}
}

func TestTurnReasoningAlsoStoresCallIDs(t *testing.T) {
	store := NewRelaySessionStore()
	assistant := ChatMessage{
		Role:    "assistant",
		Content: "hi",
		ToolCalls: []interface{}{
			map[string]interface{}{
				"id":   "call_123",
				"type": "function",
				"function": map[string]interface{}{"name": "exec", "arguments": "{}"},
			},
		},
	}
	store.StoreTurnReasoning(nil, &assistant, "reason_tc")
	result := store.GetReasoning("call_123")
	if result == nil || *result != "reason_tc" {
		t.Errorf("expected 'reason_tc', got '%v'", result)
	}
}

func TestHistorySaveAndGet(t *testing.T) {
	store := NewRelaySessionStore()
	msgs := []ChatMessage{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "hey"},
	}
	id := store.Save(msgs)
	got := store.GetHistory(id)
	if len(got) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(got))
	}
	if got[0].TextContent() != "hi" {
		t.Errorf("expected 'hi', got '%s'", got[0].TextContent())
	}
}

func TestContentKeyDeterministic(t *testing.T) {
	key1 := contentKey("same text")
	key2 := contentKey("same text")
	if key1 != key2 {
		t.Error("content key should be deterministic")
	}
	key3 := contentKey("different")
	if key1 == key3 {
		t.Error("different content should have different keys")
	}
}

func TestEvictsOldestSessionByCount(t *testing.T) {
	store, _ := NewRelaySessionStoreWithLimits(2, 1024*1024, 168, "")
	id1 := store.Save([]ChatMessage{{Role: "user", Content: "one"}})
	id2 := store.Save([]ChatMessage{{Role: "user", Content: "two"}})
	store.Save([]ChatMessage{{Role: "user", Content: "three"}})

	if len(store.GetHistory(id1)) != 0 {
		t.Error("expected oldest session to be evicted")
	}
	if len(store.GetHistory(id2)) != 1 {
		t.Error("expected second session to still exist")
	}
}

func TestEvictsOldestSessionByBytes(t *testing.T) {
	store, _ := NewRelaySessionStoreWithLimits(10, 64, 168, "")
	// Each message is roughly 30+ bytes
	store.Save([]ChatMessage{{Role: "user", Content: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}})
	id2 := store.Save([]ChatMessage{{Role: "user", Content: "bbbbbbbbbbbbbbbbbbbbbbbbbbbb"}})
	store.Save([]ChatMessage{{Role: "user", Content: "c"}})

	if len(store.GetHistory(id2)) != 1 {
		t.Error("expected session 2 to still exist")
	}
}

func TestOversizedSessionNotCached(t *testing.T) {
	store, _ := NewRelaySessionStoreWithLimits(10, 10, 168, "")
	id := store.Save([]ChatMessage{{Role: "user", Content: "this message is too large"}})
	if len(store.GetHistory(id)) != 0 {
		t.Error("expected oversized session to not be cached")
	}
}

func TestReasoningEntriesAreBoundedByBytes(t *testing.T) {
	store, _ := NewRelaySessionStoreWithLimits(10, 36, 168, "")
	store.StoreReasoning("call_1", "aaaaaaaaaaaaaaaaaaaaaaaa")
	store.StoreReasoning("call_2", "bbbbbbbbbbbbbbbbbbbbbbbb")

	result1 := store.GetReasoning("call_1")
	if result1 != nil {
		t.Error("expected call_1 to be evicted")
	}
	result2 := store.GetReasoning("call_2")
	if result2 == nil || *result2 != "bbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Error("expected call_2 to still exist")
	}
}

func TestCleanupRemovesExpiredSession(t *testing.T) {
	store, _ := NewRelaySessionStoreWithLimits(10, 1024, 1, "") // 1 hour TTL
	id := store.Save([]ChatMessage{{Role: "user", Content: "old"}})

	// Manually set last_used_at to past
	store.mu.Lock()
	if entry, ok := store.sessions[id]; ok {
		entry.LastUsedAt = time.Now().Add(-2 * time.Hour)
	}
	store.mu.Unlock()

	store.Cleanup()
	if len(store.GetHistory(id)) != 0 {
		t.Error("expected expired session to be cleaned up")
	}
}

func TestCleanupRemovesExpiredReasoning(t *testing.T) {
	store, _ := NewRelaySessionStoreWithLimits(10, 1024, 1, "")
	store.StoreReasoning("call_old", "old thought")

	store.mu.Lock()
	if entry, ok := store.reasoning["call_old"]; ok {
		entry.LastUsedAt = time.Now().Add(-2 * time.Hour)
	}
	store.mu.Unlock()

	store.Cleanup()
	result := store.GetReasoning("call_old")
	if result != nil {
		t.Error("expected expired reasoning to be cleaned up")
	}
}

func TestGetStats(t *testing.T) {
	store := NewRelaySessionStore()
	store.Save([]ChatMessage{{Role: "user", Content: "hi"}})
	store.StoreReasoning("call_1", "think")

	sessionCount, reasoningCount, turnCount, bytes := store.GetStats()
	if sessionCount != 1 {
		t.Errorf("expected 1 session, got %d", sessionCount)
	}
	if reasoningCount != 1 {
		t.Errorf("expected 1 reasoning, got %d", reasoningCount)
	}
	_ = turnCount
	_ = bytes
}

func TestClearAll(t *testing.T) {
	store := NewRelaySessionStore()
	store.Save([]ChatMessage{{Role: "user", Content: "hi"}})
	store.StoreReasoning("call_1", "think")

	store.ClearAll()

	sessionCount, _, _, _ := store.GetStats()
	if sessionCount != 0 {
		t.Errorf("expected 0 sessions after clear, got %d", sessionCount)
	}
}

func TestNewID(t *testing.T) {
	store := NewRelaySessionStore()
	id1 := store.NewID()
	id2 := store.NewID()
	if id1 == "" || id2 == "" {
		t.Error("expected non-empty IDs")
	}
	if id1 == id2 {
		t.Error("expected unique IDs")
	}
}

func TestSaveWithID(t *testing.T) {
	store := NewRelaySessionStore()
	id := store.NewID()
	store.SaveWithID(id, []ChatMessage{{Role: "user", Content: "q"}})
	got := store.GetHistory(id)
	if len(got) != 1 {
		t.Errorf("expected 1 message, got %d", len(got))
	}
}
