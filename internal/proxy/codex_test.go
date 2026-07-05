package proxy

import "testing"

func TestGetPoolStatusRedactsAccessTokens(t *testing.T) {
	requests := int64(7)
	p := &CodexProxy{
		pool: []poolEntry{
			{
				id:          "account-1",
				email:       "user@example.com",
				accessToken: "secret-access-token",
				source:      "openai",
				requests:    &requests,
			},
		},
	}

	status := p.GetPoolStatus()
	if status.TotalAccounts != 1 || status.EnabledAccounts != 1 || status.TotalRequests != requests {
		t.Fatalf("unexpected status summary: %#v", status)
	}
	if len(status.Accounts) != 1 {
		t.Fatalf("expected one account, got %d", len(status.Accounts))
	}
	if status.Accounts[0].AccessToken != "" {
		t.Fatalf("pool status must not expose access tokens")
	}
	if status.Accounts[0].Email != "user@example.com" {
		t.Fatalf("expected non-secret account metadata to remain available")
	}
}

func TestPickEntryReturnsPoolEntryPointerWhenFilteringQuota(t *testing.T) {
	zero := 0
	one := 1
	requests := int64(0)
	inFlight := int64(0)
	p := &CodexProxy{
		strategy: "round_robin",
		pool: []poolEntry{
			{id: "limited", accessToken: "old-limited", remainingQuota: &zero},
			{id: "available", accessToken: "old-available", requests: &requests, inFlight: &inFlight, remainingQuota: &one},
		},
	}

	entry := p.pickEntry()
	if entry == nil {
		t.Fatal("expected entry")
	}
	entry.accessToken = "new-available"

	if p.pool[1].accessToken != "new-available" {
		t.Fatalf("pickEntry returned a copy; pool token is still %q", p.pool[1].accessToken)
	}
}

func TestAutoStrategySpreadsConcurrentReservations(t *testing.T) {
	p := &CodexProxy{
		strategy: "auto",
		pool: []poolEntry{
			testPoolEntry("account-a"),
			testPoolEntry("account-b"),
			testPoolEntry("account-c"),
		},
	}

	first := p.pickEntryReserved()
	second := p.pickEntryReserved()
	third := p.pickEntryReserved()
	defer releasePoolEntry(first)
	defer releasePoolEntry(second)
	defer releasePoolEntry(third)

	if first == nil || second == nil || third == nil {
		t.Fatal("expected three reserved entries")
	}
	seen := map[string]bool{
		first.id:  true,
		second.id: true,
		third.id:  true,
	}
	if len(seen) != 3 {
		t.Fatalf("expected concurrent auto picks to spread across accounts, got %s, %s, %s", first.id, second.id, third.id)
	}
}

func testPoolEntry(id string) poolEntry {
	requests := int64(0)
	inFlight := int64(0)
	quota := 100
	return poolEntry{
		id:             id,
		accessToken:    "token-" + id,
		source:         "openai",
		requests:       &requests,
		inFlight:       &inFlight,
		planRank:       2,
		remainingQuota: &quota,
	}
}
