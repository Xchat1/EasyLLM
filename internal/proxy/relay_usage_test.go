package proxy

import (
	"testing"
)

func TestRelayUsageStoreRecordAndSnapshot(t *testing.T) {
	store := &RelayUsageStore{}
	store.Record("gpt-5.5", false, 100, 50, 150, 10)
	store.Record("gpt-5.5", true, 200, 80, 280, 0)

	snap := store.Snapshot()
	if snap.RequestCount != 2 {
		t.Fatalf("expected 2 requests, got %d", snap.RequestCount)
	}
	if snap.StreamCount != 1 {
		t.Fatalf("expected 1 stream, got %d", snap.StreamCount)
	}
	if snap.InputTokens != 300 {
		t.Fatalf("expected 300 input tokens, got %d", snap.InputTokens)
	}
	if snap.OutputTokens != 130 {
		t.Fatalf("expected 130 output tokens, got %d", snap.OutputTokens)
	}
	if snap.TotalTokens != 430 {
		t.Fatalf("expected 430 total tokens, got %d", snap.TotalTokens)
	}
	if snap.CachedTokens != 10 {
		t.Fatalf("expected 10 cached tokens, got %d", snap.CachedTokens)
	}
	if snap.LastModel != "gpt-5.5" {
		t.Fatalf("expected last model gpt-5.5, got %q", snap.LastModel)
	}
}

func TestRelayUsageStoreRecordCallHistory(t *testing.T) {
	store := &RelayUsageStore{}
	store.RecordCall(RelayCallRecord{
		Provider:      "小米 MiMo",
		CodexModel:    "gpt-5.5",
		UpstreamModel: "mimo-v2.5-pro",
		Stream:        true,
		InputTokens:   100,
		OutputTokens:  20,
		TotalTokens:   120,
	})

	calls := store.RecentCalls(10)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call record, got %d", len(calls))
	}
	if calls[0].Provider != "小米 MiMo" {
		t.Fatalf("unexpected provider: %q", calls[0].Provider)
	}
}

func TestResolveRelayProvider(t *testing.T) {
	if got := resolveRelayProvider("https://token-plan-cn.xiaomimimo.com/v1"); got != "小米 MiMo" {
		t.Fatalf("unexpected provider: %q", got)
	}
}

func TestRelayUsageStoreClear(t *testing.T) {
	store := &RelayUsageStore{}
	store.Record("m", false, 1, 1, 2, 0)
	store.Clear()
	snap := store.Snapshot()
	if snap.RequestCount != 0 || snap.TotalTokens != 0 {
		t.Fatalf("expected cleared store, got %+v", snap)
	}
}
