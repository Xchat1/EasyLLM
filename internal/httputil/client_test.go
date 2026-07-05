package httputil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestIsTransientNetworkError(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{fmt.Errorf(`read tcp: read: connection reset by peer`), true},
		{fmt.Errorf("HTTP 401: Token expired or invalid"), false},
		{fmt.Errorf("HTTP 500"), false},
	}
	for _, tc := range cases {
		if got := IsTransientNetworkError(tc.err); got != tc.want {
			t.Fatalf("IsTransientNetworkError(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}

func TestDoWithRetriesStopsWhenContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.test", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext: %v", err)
	}

	transport := &retryBodyTransport{}
	client := &http.Client{Transport: transport}
	_, err = DoWithRetries(client, req, 3)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
	if transport.attempts != 0 {
		t.Fatalf("expected no attempts after context cancellation, got %d", transport.attempts)
	}
}

type retryBodyTransport struct {
	attempts int
	bodies   []string
}

func (t *retryBodyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.attempts++
	if req.Body != nil {
		body, _ := io.ReadAll(req.Body)
		_ = req.Body.Close()
		t.bodies = append(t.bodies, string(body))
	}
	if t.attempts == 1 {
		return nil, fmt.Errorf("read tcp: read: connection reset by peer")
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func TestDoWithRetriesRebuildsRequestBody(t *testing.T) {
	transport := &retryBodyTransport{}
	client := &http.Client{Transport: transport}
	req, err := http.NewRequest(http.MethodPost, "https://example.test/v1/responses", bytes.NewReader([]byte(`{"input":"hello"}`)))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	resp, err := DoWithRetries(client, req, 2)
	if err != nil {
		t.Fatalf("DoWithRetries: %v", err)
	}
	_ = resp.Body.Close()

	if transport.attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", transport.attempts)
	}
	for i, body := range transport.bodies {
		if body != `{"input":"hello"}` {
			t.Fatalf("attempt %d body = %q", i+1, body)
		}
	}
}
