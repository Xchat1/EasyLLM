package proxy

import (
	"testing"
	"time"
)

func TestRelayLogStoreLogAndRecent(t *testing.T) {
	store := NewRelayLogStore()
	store.Log("info", "hello", "gpt-5.5", "resp_1")
	store.Log("error", "failed", "", "")

	recent := store.Recent(10)
	if len(recent) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(recent))
	}
	if recent[0].Message != "hello" || recent[1].Level != "error" {
		t.Fatalf("unexpected entries: %+v", recent)
	}
}

func TestRelayLogStoreSubscribe(t *testing.T) {
	store := NewRelayLogStore()
	ch := store.Subscribe()

	store.Log("info", "stream start", "mimo", "resp_2")

	select {
	case entry := <-ch:
		if entry.Message != "stream start" {
			t.Fatalf("unexpected message: %q", entry.Message)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for log entry")
	}

	store.Unsubscribe(ch)
}

func TestRelayLogStoreClear(t *testing.T) {
	store := NewRelayLogStore()
	store.Log("info", "x", "", "")
	store.Clear()
	if len(store.Recent(10)) != 0 {
		t.Fatal("expected cleared logs")
	}
}
