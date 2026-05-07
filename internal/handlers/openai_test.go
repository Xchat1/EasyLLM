package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"easyllm/internal/models"
	"easyllm/internal/storage"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupOpenAIHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.AppSettings{}, &models.OpenAIAccount{}, &models.CodexLog{}); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}
	storage.DB = db
	return db
}

func TestMergeOpenAIAccountUpdatePreservesManagedFields(t *testing.T) {
	now := time.Now()
	quotaStatus := 429
	quotaError := "rate limited"

	existing := &models.OpenAIAccount{
		IsCodexActive:        true,
		ProxyEnabled:         true,
		LastUsedAt:           &now,
		QuotaHTTPStatus:      &quotaStatus,
		QuotaError:           &quotaError,
		QuotaVerified:        true,
		QuotaIsForbidden:     true,
		Quota5hUsedPercent:   floatPtr(61.5),
		Quota7dUsedPercent:   floatPtr(18.0),
		Quota5hResetSeconds:  int64Ptr(120),
		Quota7dResetSeconds:  int64Ptr(3600),
		Quota5hWindowMinutes: int64Ptr(300),
		Quota7dWindowMinutes: int64Ptr(10080),
		QuotaUpdatedAt:       &now,
	}

	incoming := models.OpenAIAccount{Email: "updated@example.com"}
	merged := mergeOpenAIAccountUpdate(existing, incoming)

	if !merged.IsCodexActive || !merged.ProxyEnabled {
		t.Fatalf("expected codex active and proxy enabled flags to be preserved")
	}
	if merged.LastUsedAt == nil || !merged.LastUsedAt.Equal(now) {
		t.Fatalf("expected last used time to be preserved")
	}
	if merged.QuotaHTTPStatus == nil || *merged.QuotaHTTPStatus != quotaStatus {
		t.Fatalf("expected quota http status to be preserved")
	}
	if merged.QuotaError == nil || *merged.QuotaError != quotaError {
		t.Fatalf("expected quota error to be preserved")
	}
	if !merged.QuotaVerified || !merged.QuotaIsForbidden {
		t.Fatalf("expected quota state flags to be preserved")
	}
}

func TestGetServiceConfigDoesNotLeakRawAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOpenAIHandlerTestDB(t)
	handler := NewOpenAIHandler(storage.NewOpenAIStorage(db), storage.NewCodexStorage(db))

	if err := storage.SaveSetting("proxy_api_key", "secret-1234-key"); err != nil {
		t.Fatalf("save proxy_api_key: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "/openai/service-config", nil)

	handler.GetServiceConfig(ctx)

	if recorder.Code != 200 {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, exists := payload["api_key"]; exists {
		t.Fatalf("expected raw api_key to be omitted from response")
	}
	if payload["api_key_masked"] == "secret-1234-key" {
		t.Fatalf("expected masked api key, got raw key")
	}
}

func TestExtractOpenAIOAuthCodeFromCallbackURL(t *testing.T) {
	code, err := extractOpenAIOAuthCodeFromCallbackURL("expected", "http://localhost:1455/auth/callback?code=abc123&state=expected")
	if err != nil {
		t.Fatalf("expected callback URL to parse, got error: %v", err)
	}
	if code != "abc123" {
		t.Fatalf("expected code abc123, got %q", code)
	}

	if _, err := extractOpenAIOAuthCodeFromCallbackURL("expected", "http://localhost:1455/auth/callback?code=abc123&state=wrong"); err == nil {
		t.Fatalf("expected state mismatch error")
	}
}

func TestRecordOAuthCallbackStoresAuthorizationResult(t *testing.T) {
	handler := &OpenAIHandler{
		oauthSessions: map[string]*openaiOAuthSession{
			"ready": {State: "state-ready", CreatedAt: time.Now()},
			"fail":  {State: "state-fail", CreatedAt: time.Now()},
		},
	}

	if err := handler.recordOAuthCallback("state-ready", "code-123", "", ""); err != nil {
		t.Fatalf("record ready callback: %v", err)
	}
	ready := handler.oauthSessions["ready"]
	if ready.AuthorizationCode != "code-123" {
		t.Fatalf("expected authorization code to be stored, got %q", ready.AuthorizationCode)
	}
	if ready.LastError != "" || ready.CallbackReceivedAt == nil {
		t.Fatalf("expected ready callback metadata to be stored")
	}

	if err := handler.recordOAuthCallback("state-fail", "", "access_denied", "user denied"); err != nil {
		t.Fatalf("record failed callback: %v", err)
	}
	failed := handler.oauthSessions["fail"]
	if failed.AuthorizationCode != "" {
		t.Fatalf("expected failed callback to clear authorization code")
	}
	if failed.LastError == "" || failed.CallbackReceivedAt == nil {
		t.Fatalf("expected failed callback error to be stored")
	}
}

func TestGetOAuthSessionReturnsCallbackReceivedStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &OpenAIHandler{
		oauthSessions: map[string]*openaiOAuthSession{
			"session-1": {
				State:             "state-1",
				CreatedAt:         time.Now(),
				AuthorizationCode: "auth-code",
			},
		},
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "id", Value: "session-1"}}
	ctx.Request = httptest.NewRequest("GET", "/openai/oauth/sessions/session-1", nil)

	handler.GetOAuthSession(ctx)

	if recorder.Code != 200 {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["status"] != "callback_received" {
		t.Fatalf("expected callback_received status, got %#v", payload["status"])
	}
	if payload["authorization_code_ready"] != true {
		t.Fatalf("expected authorization_code_ready=true, got %#v", payload["authorization_code_ready"])
	}
}

func TestParseTokenFileEntriesSupportsNDJSON(t *testing.T) {
	raw := []byte("{\"email\":\"a@example.com\",\"access_token\":\"at1\",\"refresh_token\":\"rt1\"}\n{\"email\":\"b@example.com\",\"access_token\":\"at2\",\"refresh_token\":\"rt2\"}")

	entries, err := parseTokenFileEntries(raw)
	if err != nil {
		t.Fatalf("parse ndjson: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Email != "a@example.com" || entries[1].Email != "b@example.com" {
		t.Fatalf("unexpected parsed emails: %#v", entries)
	}
}

func TestImportByTokenFilesSupportsBundledNDJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOpenAIHandlerTestDB(t)
	handler := NewOpenAIHandler(storage.NewOpenAIStorage(db), storage.NewCodexStorage(db))

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("files", "codex_tokens.json")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}

	content := strings.Join([]string{
		`{"email":"one@example.com","access_token":"at1","refresh_token":"rt1","account_id":"acc-1","type":"codex"}`,
		`{"email":"two@example.com","access_token":"at2","refresh_token":"rt2","account_id":"acc-2","type":"codex"}`,
	}, "\n")
	if _, err := part.Write([]byte(content)); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/openai/import/token-files", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	ctx.Request = req

	handler.ImportByTokenFiles(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Total   int `json:"total"`
		Success int `json:"success"`
		Failed  int `json:"failed"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Total != 2 || payload.Success != 2 || payload.Failed != 0 {
		t.Fatalf("unexpected payload: %+v", payload)
	}

	accounts, err := handler.storage.List()
	if err != nil {
		t.Fatalf("list accounts: %v", err)
	}
	if len(accounts) != 2 {
		t.Fatalf("expected 2 accounts imported, got %d", len(accounts))
	}
}

func TestUpsertImportedOAuthAccountCanEnableProxyForExistingAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOpenAIHandlerTestDB(t)
	handler := NewOpenAIHandler(storage.NewOpenAIStorage(db), storage.NewCodexStorage(db))

	existing := models.OpenAIAccount{
		ID:               "existing-id",
		Email:            "oauth@example.com",
		AccountType:      models.OpenAIAccountTypeOAuth,
		AccessToken:      sPtr("old-access"),
		RefreshToken:     sPtr("old-refresh"),
		ChatGPTAccountID: sPtr("acct-1"),
		ProxyEnabled:     false,
		CreatedAt:        time.Now().Add(-time.Hour),
		UpdatedAt:        time.Now().Add(-time.Hour),
	}
	if err := handler.storage.Save(&existing); err != nil {
		t.Fatalf("seed existing account: %v", err)
	}

	existingAccounts, err := handler.storage.List()
	if err != nil {
		t.Fatalf("list existing accounts: %v", err)
	}

	incoming := &models.OpenAIAccount{
		Email:            "oauth@example.com",
		AccountType:      models.OpenAIAccountTypeOAuth,
		AccessToken:      sPtr("new-access"),
		RefreshToken:     sPtr("new-refresh"),
		ChatGPTAccountID: sPtr("acct-1"),
		ProxyEnabled:     true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	account, _, err := handler.upsertImportedOAuthAccount(incoming, &existingAccounts)
	if err != nil {
		t.Fatalf("upsert incoming oauth account: %v", err)
	}
	if account.ID != existing.ID {
		t.Fatalf("expected existing account to be updated, got id=%s", account.ID)
	}
	if !account.ProxyEnabled {
		t.Fatalf("expected proxy_enabled to be turned on for re-authorized account")
	}
	if account.AccessToken == nil || *account.AccessToken != "new-access" {
		t.Fatalf("expected access token to be refreshed, got %#v", account.AccessToken)
	}

	stored, err := handler.storage.Get(existing.ID)
	if err != nil {
		t.Fatalf("reload stored account: %v", err)
	}
	if !stored.ProxyEnabled {
		t.Fatalf("expected stored account to persist proxy_enabled=true")
	}
}

func TestParseCockpitToolsAccountsSupportsArrayAndTransferBundle(t *testing.T) {
	arrayPayload := []byte(`[
		{
			"id":"codex-1",
			"email":"one@example.com",
			"auth_mode":"oauth",
			"account_id":"acct-1",
			"tokens":{"access_token":"at1","refresh_token":"rt1","id_token":""}
		}
	]`)
	accounts, err := parseCockpitToolsAccounts(arrayPayload)
	if err != nil {
		t.Fatalf("parse account array: %v", err)
	}
	if len(accounts) != 1 || accounts[0].Email != "one@example.com" || accounts[0].Tokens.AccessToken != "at1" {
		t.Fatalf("unexpected parsed accounts: %+v", accounts)
	}

	topLevelPayload := []byte(`{
		"id_token":"id-top",
		"access_token":"at-top",
		"refresh_token":"rt-top",
		"account_id":"acct-top",
		"last_refresh":"2026-05-07T01:29:06.000Z",
		"email":"top@example.com",
		"type":"codex",
		"expired":"2026-05-16T06:20:07.000Z"
	}`)
	accounts, err = parseCockpitToolsAccounts(topLevelPayload)
	if err != nil {
		t.Fatalf("parse top-level token object: %v", err)
	}
	if len(accounts) != 1 || accounts[0].Email != "top@example.com" || accounts[0].AccessToken != "at-top" || accounts[0].RefreshToken != "rt-top" {
		t.Fatalf("unexpected parsed top-level account: %+v", accounts)
	}

	bundlePayload := []byte(`{
		"schema":"account-transfer",
		"platforms":{
			"codex":{
				"account_count":1,
				"exported_data":[
					{
						"id":"codex-2",
						"email":"two@example.com",
						"auth_mode":"oauth",
						"account_id":"acct-2",
						"tokens":{"access_token":"at2","refresh_token":"rt2","id_token":""}
					}
				]
			}
		}
	}`)
	accounts, err = parseCockpitToolsAccounts(bundlePayload)
	if err != nil {
		t.Fatalf("parse transfer bundle: %v", err)
	}
	if len(accounts) != 1 || accounts[0].Email != "two@example.com" || accounts[0].Tokens.RefreshToken != "rt2" {
		t.Fatalf("unexpected parsed bundle accounts: %+v", accounts)
	}
}

func TestImportFromCockpitToolsImportsOAuthAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOpenAIHandlerTestDB(t)
	handler := NewOpenAIHandler(storage.NewOpenAIStorage(db), storage.NewCodexStorage(db))

	body := strings.NewReader(`[
		{
			"id":"codex-1",
			"email":"one@example.com",
			"id_token":"id1",
			"access_token":"at1",
			"refresh_token":"rt1",
			"account_id":"acct-1",
			"expired":"2026-05-16T06:20:07.000Z",
			"plan_type":"plus",
			"type":"codex"
		}
	]`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/openai/import/cockpit-tools", body)

	handler.ImportFromCockpitTools(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Total   int `json:"total"`
		Success int `json:"success"`
		Failed  int `json:"failed"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Total != 1 || payload.Success != 1 || payload.Failed != 0 {
		t.Fatalf("unexpected payload: %+v", payload)
	}

	accounts, err := handler.storage.List()
	if err != nil {
		t.Fatalf("list accounts: %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account imported, got %d", len(accounts))
	}
	if accounts[0].Email != "one@example.com" ||
		derefStr(accounts[0].AccessToken) != "at1" ||
		derefStr(accounts[0].RefreshToken) != "rt1" ||
		derefStr(accounts[0].IDToken) != "id1" ||
		derefStr(accounts[0].ChatGPTAccountID) != "acct-1" ||
		accounts[0].ExpiresAt == nil {
		t.Fatalf("unexpected imported account: %+v", accounts[0])
	}
}

func floatPtr(v float64) *float64 { return &v }

func int64Ptr(v int64) *int64 { return &v }
