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
