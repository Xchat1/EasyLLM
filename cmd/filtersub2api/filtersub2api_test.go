package main

import (
	"errors"
	"testing"
)

func TestExtractHTTPStatusCode(t *testing.T) {
	tests := []struct {
		msg  string
		want int
	}{
		{"failed (HTTP 401): unauthorized", 401},
		{"failed (HTTP 402): payment required", 402},
		{"failed (HTTP 403): forbidden", 403},
		{"failed (HTTP 429): rate limit exceeded", 429},
		{"failed (HTTP 503): upstream unavailable", 503},
		{"other error", 0},
	}

	for _, tt := range tests {
		got := extractHTTPStatusCode(tt.msg)
		if got != tt.want {
			t.Errorf("extractHTTPStatusCode(%q) = %d, want %d", tt.msg, got, tt.want)
		}
	}
}

func TestIsNetworkError(t *testing.T) {
	if !isNetworkError(errors.New("dial tcp: lookup api.openai.com: no such host")) {
		t.Error("expected DNS error to be recognized as network error")
	}
	if !isNetworkError(errors.New("context deadline exceeded")) {
		t.Error("expected timeout to be recognized as network error")
	}
	if isNetworkError(errors.New("HTTP 401 Unauthorized")) {
		t.Error("HTTP 401 should not be network error")
	}
}
