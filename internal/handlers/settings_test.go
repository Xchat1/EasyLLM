package handlers

import "testing"

func TestIsSupportedSettingsUpdateKey(t *testing.T) {
	if !isSupportedSettingsUpdateKey("proxy_enabled") {
		t.Fatalf("expected proxy_enabled to be allowed")
	}
	if isSupportedSettingsUpdateKey("auth_password") {
		t.Fatalf("expected auth_password to be rejected")
	}
}

func TestIsSupportedSettingsUpdateValue(t *testing.T) {
	if !isSupportedSettingsUpdateValue("localhost") {
		t.Fatalf("expected string value to be allowed")
	}
	if !isSupportedSettingsUpdateValue(true) {
		t.Fatalf("expected bool value to be allowed")
	}
	if isSupportedSettingsUpdateValue(map[string]any{"nested": true}) {
		t.Fatalf("expected nested object value to be rejected")
	}
}
