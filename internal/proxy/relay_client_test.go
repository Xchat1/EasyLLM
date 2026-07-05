package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveRelayAPIKeyPrefersServerConfig(t *testing.T) {
	cfg := &RelayConfig{APIKey: "server-key"}
	got := ResolveRelayAPIKey(cfg, http.Header{
		"Authorization": []string{"Bearer client-key"},
	})
	if got != "server-key" {
		t.Fatalf("expected server-key, got %q", got)
	}
}

func TestResolveRelayAPIKeyFromAuthorizationHeader(t *testing.T) {
	cfg := &RelayConfig{}
	got := ResolveRelayAPIKey(cfg, http.Header{
		"Authorization": []string{"Bearer client-key"},
	})
	if got != "client-key" {
		t.Fatalf("expected client-key, got %q", got)
	}
}

func TestResolveRelayAPIKeyCustomHeader(t *testing.T) {
	cfg := &RelayConfig{
		AuthHeader:      "api-key",
		AuthValuePrefix: "",
	}
	got := ResolveRelayAPIKey(cfg, http.Header{
		"Api-Key": []string{"secret"},
	})
	if got != "secret" {
		t.Fatalf("expected secret, got %q", got)
	}
}

func TestApplyAuthHeaderCustomNoPrefix(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://example.com", nil)
	applyAuthHeader(req, "secret", "api-key", "")
	if got := req.Header.Get("api-key"); got != "secret" {
		t.Fatalf("expected api-key secret, got %q", got)
	}
}

func TestApplyAuthHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://example.com", nil)
	applyAuthHeader(req, "abc", "Authorization", "Bearer ")
	if got := req.Header.Get("Authorization"); got != "Bearer abc" {
		t.Fatalf("expected Bearer abc, got %q", got)
	}
}

func TestSortedToolIndices(t *testing.T) {
	m := map[int]*ToolCallAccum{2: {}, 0: {}, 1: {}}
	got := sortedToolIndices(m)
	if len(got) != 3 || got[0] != 0 || got[1] != 1 || got[2] != 2 {
		t.Fatalf("unexpected order: %v", got)
	}
}

func TestMapModelNameWildcard(t *testing.T) {
	modelMap := map[string]string{"*": "deepseek-chat"}
	got := MapModelName("gpt-5.4", modelMap, "")
	if got != "deepseek-chat" {
		t.Fatalf("expected deepseek-chat, got %q", got)
	}
}

func TestNormalizeRelayUpstreams(t *testing.T) {
	got := normalizeRelayUpstreams([]RelayUpstream{
		{Enabled: true, UpstreamURL: "  https://api.deepseek.com/v1/  ", APIKey: " key ", AuthHeader: " Authorization "},
		{Enabled: true, UpstreamURL: "   ", APIKey: "drop-me"},
	})
	if len(got) != 1 {
		t.Fatalf("expected 1 upstream, got %d", len(got))
	}
	if got[0].ID == "" {
		t.Fatalf("expected generated id")
	}
	if got[0].Name != "api.deepseek.com" {
		t.Fatalf("unexpected generated name: %q", got[0].Name)
	}
	if got[0].UpstreamURL != "https://api.deepseek.com/v1" {
		t.Fatalf("unexpected upstream url: %q", got[0].UpstreamURL)
	}
	if got[0].APIKey != "key" {
		t.Fatalf("unexpected api key: %q", got[0].APIKey)
	}
	if got[0].AuthHeader != "Authorization" {
		t.Fatalf("unexpected auth header: %q", got[0].AuthHeader)
	}
}

func TestApplyRelayConfigUpdateNormalizesAndPreservesLimits(t *testing.T) {
	cfg := &RelayConfig{
		MaxSessions:     32,
		MaxSessionBytes: 4096,
		SessionTTLHours: 12,
	}

	got := applyRelayConfigUpdate(cfg, relayConfigUpdateRequest{
		Upstreams: []RelayUpstream{
			{Enabled: true, UpstreamURL: " https://api.mistral.ai/v1/ ", APIKey: " upstream-key "},
			{Enabled: true, UpstreamURL: " "},
		},
		UpstreamURL:     " https://api.openrouter.ai/v1/ ",
		APIKey:          " relay-key ",
		AuthHeader:      " Authorization ",
		AuthValuePrefix: " Bearer ",
		DefaultModel:    "gpt-5.5",
		ModelMapJSON:    `{"gpt-5.5":"mistral-large-latest"}`,
		ToolDenylistStr: "web_search, file_search",
	})

	if got.UpstreamStrategy != "round_robin" {
		t.Fatalf("expected default strategy, got %q", got.UpstreamStrategy)
	}
	if len(got.Upstreams) != 1 || got.Upstreams[0].UpstreamURL != "https://api.mistral.ai/v1" {
		t.Fatalf("unexpected normalized upstreams: %#v", got.Upstreams)
	}
	if got.UpstreamURL != "https://api.openrouter.ai/v1" {
		t.Fatalf("unexpected upstream url: %q", got.UpstreamURL)
	}
	if got.APIKey != "relay-key" || got.AuthHeader != "Authorization" || got.AuthValuePrefix != "Bearer" {
		t.Fatalf("unexpected auth fields: key=%q header=%q prefix=%q", got.APIKey, got.AuthHeader, got.AuthValuePrefix)
	}
	if got.ModelMap["gpt-5.5"] != "mistral-large-latest" {
		t.Fatalf("model map was not parsed: %#v", got.ModelMap)
	}
	if !got.ToolDenylist["web_search"] || !got.ToolDenylist["file_search"] {
		t.Fatalf("tool denylist was not parsed: %#v", got.ToolDenylist)
	}
	if got.MaxSessions != 32 || got.MaxSessionBytes != 4096 || got.SessionTTLHours != 12 {
		t.Fatalf("limits should be preserved, got sessions=%d bytes=%d ttl=%d", got.MaxSessions, got.MaxSessionBytes, got.SessionTTLHours)
	}
}
