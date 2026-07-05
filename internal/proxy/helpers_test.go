package proxy

import (
	"net/http"
	"testing"
)

func TestIsUsageLimitReached(t *testing.T) {
	body := []byte(`{"error":{"type":"usage_limit_reached","message":"The usage limit has been reached"}}`)
	if !isUsageLimitReached(http.StatusTooManyRequests, body) {
		t.Fatal("expected usage_limit_reached body to be retryable")
	}
	if isUsageLimitReached(http.StatusUnauthorized, body) {
		t.Fatal("expected non-429 status to be ignored")
	}
}

func TestIsRetryablePoolFailureIncludesAuthAndUsage(t *testing.T) {
	usageBody := []byte(`{"error":{"type":"usage_limit_reached"}}`)
	if !isRetryablePoolFailure(http.StatusTooManyRequests, usageBody) {
		t.Fatal("expected usage limit to be retryable")
	}
	authBody := []byte(`{"detail":"Unauthorized"}`)
	if !isRetryablePoolFailure(http.StatusUnauthorized, authBody) {
		t.Fatal("expected auth failure to remain retryable")
	}
}
