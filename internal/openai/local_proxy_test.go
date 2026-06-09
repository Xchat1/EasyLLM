package openai

import "testing"

func TestLocalProxyOriginNormalizesLoopbackHost(t *testing.T) {
	got := LocalProxyOrigin("127.0.0.1:58855")
	want := "http://localhost:58855"
	if got != want {
		t.Fatalf("LocalProxyOrigin(host) = %q, want %q", got, want)
	}
}

func TestLocalProxyAPIBaseURL(t *testing.T) {
	got := LocalProxyAPIBaseURL("127.0.0.1:8022")
	want := "http://localhost:8022/v1"
	if got != want {
		t.Fatalf("LocalProxyAPIBaseURL() = %q, want %q", got, want)
	}
}
