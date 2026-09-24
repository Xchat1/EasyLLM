package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"easyllm/internal/models"
	"easyllm/internal/storage"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestAntigravityDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.AntigravityAccount{}); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}
	return db
}

func setupAntigravityTestRouter(store *storage.AntigravityStorage) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api/v1")
	h := NewAntigravityHandler(store)
	// Activation tests must never write mock credentials into the developer's
	// real IDE database or CLI OAuth file.
	h.injectAccountToLocalIDE = func(*models.AntigravityAccount) ([]string, error) {
		return []string{"test-state.vscdb"}, nil
	}
	h.injectAccountToAG2 = func(*models.AntigravityAccount) ([]string, error) {
		return []string{"test-antigravity-2-state.vscdb"}, nil
	}
	h.syncAccountToLocalCLI = func(*models.AntigravityAccount) error { return nil }
	h.RegisterRoutes(api)
	return r
}

func TestAntigravityHandlerAccounts(t *testing.T) {
	db := newTestAntigravityDB(t)
	store := storage.NewAntigravityStorage(db)
	router := setupAntigravityTestRouter(store)

	// 1. Initial list should be empty
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/antigravity/accounts", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var list []models.AntigravityAccount
	_ = json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d", len(list))
	}

	// 2. Create account (using access_token so it doesn't fail external refresh token exchange in test)
	in := CreateAccountInput{
		Email:       "test@example.com",
		AccessToken: "mock-access-token",
	}
	body, _ := json.Marshal(in)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/antigravity/accounts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var created models.AntigravityAccount
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	if created.Email != "test@example.com" {
		t.Fatalf("expected test@example.com, got %s", created.Email)
	}

	// 3. Get created account
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/antigravity/accounts/"+created.ID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 4. Activate created account
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/antigravity/accounts/"+created.ID+"/activate", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var activated struct {
		CLISynced            bool `json:"cli_synced"`
		Antigravity2Detected bool `json:"antigravity_2_detected"`
		Antigravity2Synced   bool `json:"antigravity_2_synced"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &activated); err != nil {
		t.Fatalf("decode activation response: %v", err)
	}
	if !activated.CLISynced || !activated.Antigravity2Detected || !activated.Antigravity2Synced {
		t.Fatalf("expected activation to report CLI and Antigravity 2.0 sync: %+v", activated)
	}

	// 5. Delete account
	req, _ = http.NewRequest(http.MethodDelete, "/api/v1/antigravity/accounts/"+created.ID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAntigravityHandlerOAuthStartCancel(t *testing.T) {
	t.Setenv("ANTIGRAVITY_OAUTH_CLIENT_SECRET", "test-client-secret")
	db := newTestAntigravityDB(t)
	store := storage.NewAntigravityStorage(db)
	router := setupAntigravityTestRouter(store)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/antigravity/oauth/start", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var res map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	loginID, ok := res["login_id"].(string)
	if !ok || loginID == "" {
		t.Fatalf("expected login_id in response")
	}

	// Cancel flow
	cancelBody, _ := json.Marshal(map[string]string{"login_id": loginID})
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/antigravity/oauth/cancel", bytes.NewReader(cancelBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAntigravityActivateReportsSyncFailures(t *testing.T) {
	db := newTestAntigravityDB(t)
	store := storage.NewAntigravityStorage(db)
	account := &models.AntigravityAccount{
		ID:          "account-1",
		Email:       "test@example.com",
		AccessToken: "mock-access-token",
		Status:      "active",
	}
	if err := store.Save(account); err != nil {
		t.Fatalf("save account: %v", err)
	}

	router := gin.New()
	h := NewAntigravityHandler(store)
	h.injectAccountToLocalIDE = func(*models.AntigravityAccount) ([]string, error) { return nil, nil }
	h.injectAccountToAG2 = func(*models.AntigravityAccount) ([]string, error) {
		return nil, errors.New("database locked")
	}
	h.syncAccountToLocalCLI = func(*models.AntigravityAccount) error {
		return errors.New("credential file unavailable")
	}
	h.RegisterRoutes(router.Group("/api/v1"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/antigravity/accounts/account-1/activate", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result struct {
		Success      bool   `json:"success"`
		CLISynced    bool   `json:"cli_synced"`
		CLISyncError string `json:"cli_sync_error"`
		AG2Detected  bool   `json:"antigravity_2_detected"`
		AG2Synced    bool   `json:"antigravity_2_synced"`
		AG2SyncError string `json:"antigravity_2_sync_error"`
		SyncWarning  bool   `json:"sync_warning"`
		Message      string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !result.Success || result.CLISynced || !result.AG2Detected || result.AG2Synced || !result.SyncWarning {
		t.Fatalf("unexpected partial sync result: %+v", result)
	}
	if !strings.Contains(result.CLISyncError, "credential file unavailable") ||
		!strings.Contains(result.AG2SyncError, "database locked") ||
		!strings.Contains(result.Message, "agy CLI 同步失败") ||
		!strings.Contains(result.Message, "Antigravity 2.0 同步失败") {
		t.Fatalf("expected visible CLI sync warning, got %+v", result)
	}
}
