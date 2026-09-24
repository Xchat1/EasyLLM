package cursor

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestExtractUserIdFromJWT(t *testing.T) {
	// Build a dummy JWT with sub "github|user_12345"
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"github|user_12345","iss":"https://cursor.com"}`))
	jwt := header + "." + payload + ".sig"

	uid := ExtractUserIdFromJWT(jwt)
	if uid != "user_12345" {
		t.Fatalf("expected user_12345, got %s", uid)
	}
}

func TestBuildAuthCookie(t *testing.T) {
	// Raw token with ::
	c1 := BuildAuthCookie("user_abc::jwt123")
	if c1 != "WorkosCursorSessionToken=user_abc::jwt123" {
		t.Fatalf("expected WorkosCursorSessionToken=user_abc::jwt123, got %s", c1)
	}

	// Full cookie
	c2 := BuildAuthCookie("WorkosCursorSessionToken=user_abc::jwt123")
	if c2 != "WorkosCursorSessionToken=user_abc::jwt123" {
		t.Fatalf("expected WorkosCursorSessionToken=user_abc::jwt123, got %s", c2)
	}

	// JWT format
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"auth0|user_xyz"}`))
	jwt := header + "." + payload + ".dummy"
	c3 := BuildAuthCookie(jwt)
	if !strings.HasPrefix(c3, "WorkosCursorSessionToken=user_xyz::") {
		t.Fatalf("expected cookie with user_xyz, got %s", c3)
	}
}
