package openai

import (
	"strings"
	"testing"
)

func TestLocalProxyOriginNormalizesLoopbackHost(t *testing.T) {
	got := LocalProxyOrigin("127.0.0.1:58855")
	want := "http://localhost:58855"
	if got != want {
		t.Fatalf("LocalProxyOrigin(host) = %q, want %q", got, want)
	}
}

func TestLocalProxyOriginAcceptsFullURL(t *testing.T) {
	got := LocalProxyOrigin("http://localhost:8022")
	want := "http://localhost:8022"
	if got != want {
		t.Fatalf("LocalProxyOrigin(url) = %q, want %q", got, want)
	}
}

func TestLocalProxyAPIBaseURL(t *testing.T) {
	got := LocalProxyAPIBaseURL("127.0.0.1:8022")
	want := "http://localhost:8022/v1"
	if got != want {
		t.Fatalf("LocalProxyAPIBaseURL() = %q, want %q", got, want)
	}
}

func TestLocalCodexProxyAPIBaseURL(t *testing.T) {
	got := LocalCodexProxyAPIBaseURL("127.0.0.1:8022")
	want := "http://localhost:8022/backend-api/codex"
	if got != want {
		t.Fatalf("LocalCodexProxyAPIBaseURL() = %q, want %q", got, want)
	}
}

func TestLocalProxyOriginDiscardsUntrustedHost(t *testing.T) {
	for _, hostile := range []string{
		"evil.attacker.com",
		"attacker.com:8022",
		"http://evil.com:9000",
		"192.168.1.100:8022",
	} {
		got := LocalProxyOrigin(hostile)
		if !strings.HasPrefix(got, "http://localhost:") {
			t.Fatalf("LocalProxyOrigin(%q) = %q, must bind to localhost", hostile, got)
		}
	}
}

