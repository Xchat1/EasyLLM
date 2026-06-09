package handlers

import (
	"testing"
)

func TestParseCPAFileEntriesSingleObject(t *testing.T) {
	raw := []byte(`{
		"type": "codex",
		"email": "cpa-user@example.com",
		"expired": "2026-05-29T02:26:12+08:00",
		"account_id": "68d6c473-c217-471e-a128-a1816290dbc8",
		"access_token": "at_example",
		"refresh_token": "rt_example",
		"id_token": "id_example",
		"plan_type": "plus"
	}`)

	entries, err := parseCPAFileEntries(raw)
	if err != nil {
		t.Fatalf("parseCPAFileEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Email != "cpa-user@example.com" {
		t.Fatalf("email = %q", entries[0].Email)
	}
	if entries[0].PlanType != "plus" {
		t.Fatalf("plan_type = %q", entries[0].PlanType)
	}
}

func TestParseCPAFileEntriesArray(t *testing.T) {
	raw := []byte(`[
		{"email":"a@example.com","access_token":"at1"},
		{"email":"b@example.com","id_token":"id2"}
	]`)
	entries, err := parseCPAFileEntries(raw)
	if err != nil {
		t.Fatalf("parseCPAFileEntries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestParseCPAFileEntriesRejectsEmptyCredentials(t *testing.T) {
	raw := []byte(`{"email":"a@example.com"}`)
	if _, err := parseCPAFileEntries(raw); err == nil {
		t.Fatal("expected error for missing credentials")
	}
}
