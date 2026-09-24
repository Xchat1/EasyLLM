package antigravity

import (
	"net/url"
	"strings"
	"testing"
)

func TestOAuthRequiresLocalClientSecret(t *testing.T) {
	t.Setenv("ANTIGRAVITY_OAUTH_CLIENT_SECRET", "")
	if _, err := StartOAuthFlow(); err == nil {
		t.Fatal("OAuth must require a locally configured client secret")
	}
	if _, err := ExchangeCode("test-code", "http://localhost/oauth-callback"); err == nil {
		t.Fatal("code exchange must reject missing credentials")
	}
	if _, err := RefreshAccessToken("test-token"); err == nil {
		t.Fatal("token refresh must reject missing credentials")
	}
	t.Setenv("ANTIGRAVITY_OAUTH_CLIENT_SECRET", "local-test-secret")
	if _, secret := getClientCredentials(); secret != "local-test-secret" {
		t.Fatal("client secret must come from the local environment")
	}
}

func TestBuildAuthURL(t *testing.T) {
	callbackURL := "http://localhost:12345/oauth-callback"
	state := "test-state-token"
	authURL := BuildAuthURL(callbackURL, state)

	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("parse auth URL failed: %v", err)
	}

	q := u.Query()
	if q.Get("client_id") != DefaultClientID {
		t.Errorf("expected client_id %s, got %s", DefaultClientID, q.Get("client_id"))
	}
	if q.Get("redirect_uri") != callbackURL {
		t.Errorf("expected redirect_uri %s, got %s", callbackURL, q.Get("redirect_uri"))
	}
	if q.Get("state") != state {
		t.Errorf("expected state %s, got %s", state, q.Get("state"))
	}
	if !strings.Contains(q.Get("scope"), "https://www.googleapis.com/auth/cloud-platform") {
		t.Errorf("expected cloud-platform scope in %s", q.Get("scope"))
	}
}

func TestProtobufEncoding(t *testing.T) {
	// Test encode/decode varint
	val := uint64(123456789)
	encoded := encodeVarint(val)
	decoded, pos, err := readVarint(encoded, 0)
	if err != nil {
		t.Fatalf("readVarint failed: %v", err)
	}
	if decoded != val {
		t.Errorf("expected %d, got %d", val, decoded)
	}
	if pos != len(encoded) {
		t.Errorf("expected pos %d, got %d", len(encoded), pos)
	}

	// Test createOAuthInfo
	oauthInfo := createOAuthInfo("acc-token", "ref-token", 1800000000, false, "id-tok", "test@example.com")
	if len(oauthInfo) == 0 {
		t.Fatalf("expected non-empty oauthInfo")
	}

	// Test createUnifiedTopicEntry
	entry := createUnifiedTopicEntry("oauthTokenInfoSentinelKey", oauthInfo)
	if len(entry) == 0 {
		t.Fatalf("expected non-empty entry")
	}

	// Test removeUnifiedTopicEntry
	cleaned, err := removeUnifiedTopicEntry(entry, "oauthTokenInfoSentinelKey")
	if err != nil {
		t.Fatalf("removeUnifiedTopicEntry failed: %v", err)
	}
	if len(cleaned) != 0 {
		t.Errorf("expected empty after removal, got len %d", len(cleaned))
	}
}
