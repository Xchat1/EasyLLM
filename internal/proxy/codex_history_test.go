package proxy

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCodexCallHistoryRecentReturnsNewestFirst(t *testing.T) {
	store := &CodexCallHistoryStore{}
	store.Record(CodexCallRecord{Timestamp: "2026-01-01T00:00:00Z", Model: "first"})
	store.Record(CodexCallRecord{Timestamp: "2026-01-01T00:00:01Z", Model: "second"})

	calls := store.Recent(10)
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}
	if calls[0].Model != "second" || calls[1].Model != "first" {
		t.Fatalf("expected newest first, got %#v", calls)
	}
}

func TestCodexCallHistoryCapsAtMax(t *testing.T) {
	store := &CodexCallHistoryStore{}
	for i := 0; i < maxCodexCallHistory+5; i++ {
		store.Record(CodexCallRecord{Model: "m"})
	}

	calls := store.Recent(maxCodexCallHistory + 10)
	if len(calls) != maxCodexCallHistory {
		t.Fatalf("expected %d calls, got %d", maxCodexCallHistory, len(calls))
	}
}

func TestParseCodexCallMetadata(t *testing.T) {
	model, stream := parseCodexCallMetadata([]byte(`{"model":"gpt-5.4","stream":true,"input":"secret prompt"}`))
	if model != "gpt-5.4" {
		t.Fatalf("unexpected model %q", model)
	}
	if !stream {
		t.Fatal("expected stream=true")
	}
}

func TestRecordCallStoresOnlyMetadata(t *testing.T) {
	proxy := &CodexProxy{history: &CodexCallHistoryStore{}}
	req := httptest.NewRequest(http.MethodPost, "/backend-api/codex/responses", strings.NewReader(`{"model":"gpt-5.4","stream":true,"input":"do not store"}`))
	entry := &poolEntry{id: "acct_1", email: "user@example.com", source: "openai"}

	proxy.recordCall(entry, req, []byte(`{"model":"gpt-5.4","stream":true,"input":"do not store"}`), http.StatusOK, false, time.Now(), "")

	calls := proxy.RecentCalls(1)
	if len(calls) != 1 {
		t.Fatalf("expected one call, got %d", len(calls))
	}
	call := calls[0]
	if call.AccountEmail != "user@example.com" || call.Model != "gpt-5.4" || !call.Stream {
		t.Fatalf("unexpected record: %#v", call)
	}
	if strings.Contains(call.Error+call.Path+call.Model+call.AccountEmail, "do not store") {
		t.Fatalf("record leaked request body: %#v", call)
	}
}
