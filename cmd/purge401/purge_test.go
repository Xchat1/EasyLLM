package main

import (
	"errors"
	"testing"
)

func TestShouldDeleteAccount(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		httpCode int
		only401  bool
		want     bool
	}{
		{
			name:     "HTTP 401 always deletes",
			err:      errors.New("HTTP 401 Unauthorized"),
			httpCode: 401,
			only401:  false,
			want:     true,
		},
		{
			name:     "HTTP 401 in 401-only mode deletes",
			err:      errors.New("HTTP 401 Unauthorized"),
			httpCode: 401,
			only401:  true,
			want:     true,
		},
		{
			name:     "HTTP 403 in default mode deletes",
			err:      errors.New("HTTP 403 Forbidden"),
			httpCode: 403,
			only401:  false,
			want:     true,
		},
		{
			name:     "HTTP 403 in 401-only mode is kept",
			err:      errors.New("HTTP 403 Forbidden"),
			httpCode: 403,
			only401:  true,
			want:     false,
		},
		{
			name:     "Network timeout is never deleted",
			err:      errors.New("context deadline exceeded (Client.Timeout exceeded)"),
			httpCode: 0,
			only401:  false,
			want:     false,
		},
		{
			name:     "Network reset is never deleted",
			err:      errors.New("read: connection reset by peer"),
			httpCode: 0,
			only401:  false,
			want:     false,
		},
		{
			name:     "DNS failure is never deleted",
			err:      errors.New("dial tcp: lookup api.openai.com: no such host"),
			httpCode: 0,
			only401:  false,
			want:     false,
		},
		{
			name:     "HTTP 429 rate limit is never deleted",
			err:      errors.New("HTTP 429 Too Many Requests"),
			httpCode: 429,
			only401:  false,
			want:     false,
		},
		{
			name:     "HTTP 503 service unavailable is never deleted",
			err:      errors.New("HTTP 503 Service Unavailable"),
			httpCode: 503,
			only401:  false,
			want:     false,
		},
		{
			name:     "Unknown error is kept",
			err:      errors.New("something went wrong"),
			httpCode: 500,
			only401:  false,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldDeleteAccount(tt.err, tt.httpCode, tt.only401)
			if got != tt.want {
				t.Errorf("shouldDeleteAccount(%v, %d, %v) = %v, want %v", tt.err, tt.httpCode, tt.only401, got, tt.want)
			}
		})
	}
}
