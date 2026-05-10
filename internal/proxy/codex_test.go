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
