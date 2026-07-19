package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"easyllm/config"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := config.Get().App.SecretKey
	validToken, err := generateJWT(secret)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	expiredToken := signTestJWT(t, secret, jwtHeader{Alg: "HS256", Typ: "JWT"}, jwtPayload{
		Iss: "easyllm",
		Iat: time.Now().Add(-2 * time.Hour).Unix(),
		Exp: time.Now().Add(-time.Hour).Unix(),
	})
	wrongAlgorithmToken := signTestJWT(t, secret, jwtHeader{Alg: "none", Typ: "JWT"}, jwtPayload{
		Iss: "easyllm",
		Iat: time.Now().Unix(),
		Exp: time.Now().Add(time.Hour).Unix(),
	})

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{name: "missing token", wantStatus: http.StatusUnauthorized},
		{name: "empty bearer token", authHeader: "Bearer ", wantStatus: http.StatusUnauthorized},
		{name: "invalid token", authHeader: "Bearer invalid", wantStatus: http.StatusUnauthorized},
		{name: "expired token", authHeader: "Bearer " + expiredToken, wantStatus: http.StatusUnauthorized},
		{name: "wrong algorithm", authHeader: "Bearer " + wrongAlgorithmToken, wantStatus: http.StatusUnauthorized},
		{name: "valid token", authHeader: "Bearer " + validToken, wantStatus: http.StatusNoContent},
		{name: "case insensitive scheme", authHeader: "bearer " + validToken, wantStatus: http.StatusNoContent},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(AuthMiddleware())
			router.GET("/protected", func(c *gin.Context) {
				c.Status(http.StatusNoContent)
			})

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			if recorder.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tc.wantStatus)
			}
		})
	}
}

func TestVerifyJWTRejectsWrongIssuer(t *testing.T) {
	secret := config.Get().App.SecretKey
	token := signTestJWT(t, secret, jwtHeader{Alg: "HS256", Typ: "JWT"}, jwtPayload{
		Iss: "someone-else",
		Iat: time.Now().Unix(),
		Exp: time.Now().Add(time.Hour).Unix(),
	})
	if err := verifyJWT(token, secret); err == nil {
		t.Fatal("expected token with wrong issuer to be rejected")
	}
}

func signTestJWT(t *testing.T, secret string, header jwtHeader, payload jwtPayload) string {
	t.Helper()
	headerJSON, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := headerB64 + "." + payloadB64
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signingInput))
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
