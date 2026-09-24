package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"easyllm/config"
	"easyllm/internal/models"
	"easyllm/internal/storage"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestAuthDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}
	if err := db.AutoMigrate(&models.AppSettings{}); err != nil {
		t.Fatalf("failed to automigrate: %v", err)
	}
	storage.DB = db
}

func TestAuthMiddlewareDefaultDisabled(t *testing.T) {
	setupTestAuthDB(t)
	gin.SetMode(gin.TestMode)

	// Auth is disabled by default, so unauthenticated request should succeed
	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204 when auth is disabled, got %d", recorder.Code)
	}
}

func TestAuthMiddlewareWhenEnabled(t *testing.T) {
	setupTestAuthDB(t)
	_ = storage.SaveSetting("auth_enabled", "true")
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

func TestAuthEnableDisableAndCheck(t *testing.T) {
	setupTestAuthDB(t)
	gin.SetMode(gin.TestMode)

	h := NewAuthHandler()
	r := gin.New()
	rg := r.Group("/api/v1")
	h.RegisterRoutes(rg)

	// 1. Initial check: should be auth_enabled = false
	reqCheck, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/check", nil)
	wCheck := httptest.NewRecorder()
	r.ServeHTTP(wCheck, reqCheck)
	if wCheck.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", wCheck.Code)
	}
	var checkResp map[string]interface{}
	_ = json.Unmarshal(wCheck.Body.Bytes(), &checkResp)
	if checkResp["auth_enabled"] != false {
		t.Fatalf("expected auth_enabled = false by default, got %v", checkResp["auth_enabled"])
	}

	// 2. Enable auth
	enableBody, _ := json.Marshal(map[string]string{"password": "test-password"})
	reqEnable, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/enable", bytes.NewReader(enableBody))
	reqEnable.Header.Set("Content-Type", "application/json")
	wEnable := httptest.NewRecorder()
	r.ServeHTTP(wEnable, reqEnable)
	if wEnable.Code != http.StatusOK {
		t.Fatalf("expected 200 on enable, got %d: %s", wEnable.Code, wEnable.Body.String())
	}
	var enableResp map[string]interface{}
	_ = json.Unmarshal(wEnable.Body.Bytes(), &enableResp)
	if enableResp["auth_enabled"] != true || enableResp["token"] == "" {
		t.Fatalf("expected auth_enabled = true and token, got %v", enableResp)
	}

	// 3. Verify check now returns auth_enabled = true
	wCheck2 := httptest.NewRecorder()
	r.ServeHTTP(wCheck2, reqCheck)
	_ = json.Unmarshal(wCheck2.Body.Bytes(), &checkResp)
	if checkResp["auth_enabled"] != true {
		t.Fatalf("expected auth_enabled = true after enable, got %v", checkResp["auth_enabled"])
	}

	// 4. Disable auth with wrong password should fail
	badDisable, _ := json.Marshal(map[string]string{"password": "wrong"})
	reqBadDisable, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/disable", bytes.NewReader(badDisable))
	reqBadDisable.Header.Set("Content-Type", "application/json")
	wBadDisable := httptest.NewRecorder()
	r.ServeHTTP(wBadDisable, reqBadDisable)
	if wBadDisable.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on wrong disable password, got %d", wBadDisable.Code)
	}

	// 5. Disable auth with correct password should succeed
	goodDisable, _ := json.Marshal(map[string]string{"password": "test-password"})
	reqGoodDisable, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/disable", bytes.NewReader(goodDisable))
	reqGoodDisable.Header.Set("Content-Type", "application/json")
	wGoodDisable := httptest.NewRecorder()
	r.ServeHTTP(wGoodDisable, reqGoodDisable)
	if wGoodDisable.Code != http.StatusOK {
		t.Fatalf("expected 200 on disable, got %d: %s", wGoodDisable.Code, wGoodDisable.Body.String())
	}

	// 6. Verify check now returns auth_enabled = false
	wCheck3 := httptest.NewRecorder()
	r.ServeHTTP(wCheck3, reqCheck)
	_ = json.Unmarshal(wCheck3.Body.Bytes(), &checkResp)
	if checkResp["auth_enabled"] != false {
		t.Fatalf("expected auth_enabled = false after disable, got %v", checkResp["auth_enabled"])
	}
}

func TestExistingPasswordCannotBeBypassed(t *testing.T) {
	for _, flag := range []string{"", "true"} {
		for _, body := range []string{
			`{"password":"replacement"}`,
			`{}`,
			`{"password":"replacement","old_password":"wrong"}`,
		} {
			t.Run(flag+body, func(t *testing.T) {
				setupTestAuthDB(t)
				hash, err := bcrypt.GenerateFromPassword([]byte("original"), bcrypt.MinCost)
				if err != nil {
					t.Fatal(err)
				}
				if err := storage.SaveSetting("auth_password", string(hash)); err != nil {
					t.Fatal(err)
				}
				if flag != "" {
					if err := storage.SaveSetting("auth_enabled", flag); err != nil {
						t.Fatal(err)
					}
				}
				if !IsAuthEnabled() {
					t.Fatal("existing password must enable authentication")
				}
				r := gin.New()
				NewAuthHandler().RegisterRoutes(r.Group("/auth-test"))
				for _, endpoint := range []string{"enable", "setup"} {
					req := httptest.NewRequest(http.MethodPost, "/auth-test/auth/"+endpoint, bytes.NewBufferString(body))
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()
					r.ServeHTTP(w, req)
					if w.Code < 400 {
						t.Fatalf("%s accepted unauthenticated request: %d", endpoint, w.Code)
					}
					stored, _ := storage.GetSetting("auth_password")
					if stored != string(hash) {
						t.Fatal("password changed without authentication")
					}
				}
			})
		}
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

func TestEnableAuthPreventsAccountTakeoverWhenDisabled(t *testing.T) {
	setupTestAuthDB(t)
	gin.SetMode(gin.TestMode)

	// Save existing password and explicitly disable auth
	hash, _ := bcrypt.GenerateFromPassword([]byte("original-secret"), bcrypt.MinCost)
	_ = storage.SaveSetting("auth_password", string(hash))
	_ = storage.SaveSetting("auth_enabled", "false")

	h := NewAuthHandler()
	r := gin.New()
	rg := r.Group("/api/v1")
	h.RegisterRoutes(rg)

	// Attacker attempts to change password without old password
	attackBody, _ := json.Marshal(map[string]string{"password": "hacked-password"})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/enable", bytes.NewReader(attackBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated password reset, got %d", w.Code)
	}

	// Verify password in DB was NOT changed
	stored, _ := storage.GetSetting("auth_password")
	if bcrypt.CompareHashAndPassword([]byte(stored), []byte("original-secret")) != nil {
		t.Fatalf("stored password was altered by unauthorized request!")
	}
}

