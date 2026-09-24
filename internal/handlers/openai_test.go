package handlers

import (
	"bytes"
	"encoding/base64"
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
	if err := db.AutoMigrate(&models.AppSettings{}, &models.OpenAIAccount{}, &models.CodexAccount{}); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}
	storage.DB = db
	return db
}

func testUnsignedJWT(t *testing.T, payload map[string]interface{}) string {
	t.Helper()
	header, err := json.Marshal(map[string]string{"alg": "none", "typ": "JWT"})
	if err != nil {
		t.Fatalf("marshal jwt header: %v", err)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal jwt payload: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(body)
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

func TestGetServiceConfigReturnsRawAPIKey(t *testing.T) {
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
	if payload["api_key"] != "secret-1234-key" {
		t.Fatalf("expected raw api key in service config response, got %#v", payload["api_key"])
	}
	if payload["api_key_masked"] == "secret-1234-key" {
		t.Fatalf("expected masked api key, got raw key")
	}
}

func TestFilterCodexLocalAccessAccountIDsSkipsMissingAccounts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOpenAIHandlerTestDB(t)
	handler := NewOpenAIHandler(storage.NewOpenAIStorage(db), storage.NewCodexStorage(db))

	statusOK := http.StatusOK
	plan := "plus"
	account := models.OpenAIAccount{
		ID:              "valid-id",
		Email:           "valid@example.com",
		AccountType:     models.OpenAIAccountTypeOAuth,
		AccessToken:     sPtr("access-token"),
		Plan:            &plan,
		QuotaHTTPStatus: &statusOK,
		QuotaVerified:   true,
	}
	if err := handler.storage.Save(&account); err != nil {
		t.Fatalf("seed account: %v", err)
	}

	got, err := handler.filterCodexLocalAccessAccountIDs([]string{"missing-id", "valid-id"}, true)
	if err != nil {
		t.Fatalf("filter local access accounts: %v", err)
	}
	if len(got) != 1 || got[0] != "valid-id" {
		t.Fatalf("expected only valid-id, got %#v", got)
	}
}

func TestListAccountsDoesNotLeakSecrets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOpenAIHandlerTestDB(t)
	handler := NewOpenAIHandler(storage.NewOpenAIStorage(db), storage.NewCodexStorage(db))
	now := time.Now()

	if err := handler.storage.Save(&models.OpenAIAccount{
		ID:             "secret-account",
		Email:          "secret@example.com",
		AccountType:    models.OpenAIAccountTypeOAuth,
		AccessToken:    sPtr("access-secret"),
		RefreshToken:   sPtr("refresh-secret"),
		IDToken:        sPtr(testUnsignedJWT(t, map[string]interface{}{"https://api.openai.com/auth": map[string]interface{}{"chatgpt_plan_type": "plus"}})),
		OpenAIAuthJSON: sPtr(`{"secret":true}`),
		APIKey:         sPtr("api-secret"),
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("seed account: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/openai/accounts", nil)

	handler.ListAccounts(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	var payload []map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload) != 1 {
		t.Fatalf("expected 1 account, got %d", len(payload))
	}
	for _, key := range []string{"access_token", "refresh_token", "id_token", "openai_auth_json", "api_key"} {
		if _, exists := payload[0][key]; exists {
			t.Fatalf("expected %s to be omitted from list response", key)
		}
	}
	if payload[0]["plan"] != "plus" {
		t.Fatalf("expected plan to be inferred before secret fields are omitted, got %#v", payload[0]["plan"])
	}
}

func TestListAccountsFiltersByWorkspaceID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOpenAIHandlerTestDB(t)
	handler := NewOpenAIHandler(storage.NewOpenAIStorage(db), storage.NewCodexStorage(db))
	now := time.Now()

	seeds := []models.OpenAIAccount{
		{ID: "workspace-a-1", Email: "one@example.com", AccountType: models.OpenAIAccountTypeOAuth, ChatGPTAccountID: sPtr("workspace-a"), CreatedAt: now, UpdatedAt: now},
		{ID: "workspace-a-2", Email: "two@example.com", AccountType: models.OpenAIAccountTypeOAuth, ChatGPTAccountID: sPtr("workspace-a"), CreatedAt: now, UpdatedAt: now},
		{ID: "workspace-b-1", Email: "three@example.com", AccountType: models.OpenAIAccountTypeOAuth, ChatGPTAccountID: sPtr("workspace-b"), CreatedAt: now, UpdatedAt: now},
		{ID: "api-account", Email: "api@example.com", AccountType: models.OpenAIAccountTypeAPI, CreatedAt: now, UpdatedAt: now},
	}
	for i := range seeds {
		if err := handler.storage.Save(&seeds[i]); err != nil {
			t.Fatalf("seed account %s: %v", seeds[i].ID, err)
		}
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/openai/accounts?workspace_id=workspace-a", nil)

	handler.ListAccounts(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	var payload []models.OpenAIAccount
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload) != 2 {
		t.Fatalf("expected 2 workspace accounts, got %d", len(payload))
	}
	for _, account := range payload {
		if derefStr(account.ChatGPTAccountID) != "workspace-a" {
			t.Fatalf("unexpected workspace id %q", derefStr(account.ChatGPTAccountID))
		}
	}
}

func TestAPIAccountTestUsesConfiguredWireAPI(t *testing.T) {
	tests := []struct {
		name     string
		wireAPI  string
		wantPath string
	}{
		{name: "responses", wireAPI: "responses", wantPath: "/v1/responses"},
		{name: "chat completions", wireAPI: "chat_completions", wantPath: "/v1/chat/completions"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			db := setupOpenAIHandlerTestDB(t)
			handler := NewOpenAIHandler(storage.NewOpenAIStorage(db), storage.NewCodexStorage(db))

			var gotPath string
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			defer upstream.Close()

			now := time.Now()
			if err := handler.storage.Save(&models.OpenAIAccount{
				ID:          "api-account",
				Email:       "api",
				AccountType: models.OpenAIAccountTypeAPI,
				Model:       sPtr("test-model"),
				BaseURL:     sPtr(upstream.URL + "/v1"),
				APIKey:      sPtr("test-key"),
				WireAPI:     sPtr(tc.wireAPI),
				CreatedAt:   now,
				UpdatedAt:   now,
			}); err != nil {
				t.Fatalf("seed api account: %v", err)
			}

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Params = gin.Params{{Key: "id", Value: "api-account"}}
			ctx.Request = httptest.NewRequest(http.MethodPost, "/openai/api-accounts/api-account/test", nil)

			handler.TestAPIAccount(ctx)

			if recorder.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", recorder.Code)
			}
			if gotPath != tc.wantPath {
				t.Fatalf("upstream path = %q, want %q", gotPath, tc.wantPath)
			}
		})
	}
}

func TestExportAccountsIncludesOAuthPlan(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOpenAIHandlerTestDB(t)
	handler := NewOpenAIHandler(storage.NewOpenAIStorage(db), storage.NewCodexStorage(db))
	now := time.Now()

	if err := handler.storage.Save(&models.OpenAIAccount{
		ID:          "team-export",
		Email:       "team@example.com",
		AccountType: models.OpenAIAccountTypeOAuth,
		Status:      "active",
		Plan:        sPtr("team"),
		AccessToken: sPtr("at"),
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("seed account: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/openai/export", nil)

	handler.ExportAccounts(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		OAuthAccounts []exportedOAuthAccount `json:"oauth_accounts"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.OAuthAccounts) != 1 {
		t.Fatalf("expected one oauth account, got %+v", payload.OAuthAccounts)
	}
	if payload.OAuthAccounts[0].Plan != "team" || payload.OAuthAccounts[0].PlanType != "team" {
		t.Fatalf("expected plan to be exported, got %+v", payload.OAuthAccounts[0])
	}
}

func TestImportFromExportPreservesOAuthPlan(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOpenAIHandlerTestDB(t)
	handler := NewOpenAIHandler(storage.NewOpenAIStorage(db), storage.NewCodexStorage(db))

	body := strings.NewReader(`{
		"oauth_accounts":[
			{
				"id":"team-import",
				"email":"team@example.com",
				"access_token":"at",
				"refresh_token":"rt",
				"account_id":"acct-team",
				"plan_type":"team",
				"type":"codex"
			}
		]
	}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/openai/import/from-export", body)

	handler.ImportFromExport(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	accounts, err := handler.storage.List()
	if err != nil {
		t.Fatalf("list accounts: %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("expected one account, got %+v", accounts)
	}
	if accounts[0].Plan == nil || *accounts[0].Plan != "team" {
		t.Fatalf("expected team plan to be imported, got %#v", accounts[0].Plan)
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

	plusToken := testUnsignedJWT(t, map[string]interface{}{
		"email": "one@example.com",
		"https://api.openai.com/auth": map[string]interface{}{
			"account_id":        "acc-1",
			"chatgpt_plan_type": "chatgpt_plus",
		},
	})
	content := strings.Join([]string{
		`{"email":"one@example.com","access_token":"at1","refresh_token":"rt1","id_token":"` + plusToken + `","account_id":"acc-1","type":"codex"}`,
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
	var plusAccount *models.OpenAIAccount
	for i := range accounts {
		if accounts[i].Email == "one@example.com" {
			plusAccount = &accounts[i]
			break
		}
	}
	if plusAccount == nil || plusAccount.Plan == nil || *plusAccount.Plan != "plus" {
		t.Fatalf("expected plus plan to be persisted from id_token, got %#v", plusAccount)
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

func TestFindMatchingOAuthAccountDoesNotUseEmailWhenScopedIDDiffers(t *testing.T) {
	existingAccounts := []models.OpenAIAccount{
		{
			ID:               "existing-id",
			Email:            "fixture-user@example.com",
			AccountType:      models.OpenAIAccountTypeOAuth,
			ChatGPTAccountID: sPtr("acct-existing"),
			OrganizationID:   sPtr("org-existing"),
		},
	}
	incoming := &models.OpenAIAccount{
		Email:            "fixture-user@example.com",
		AccountType:      models.OpenAIAccountTypeOAuth,
		ChatGPTAccountID: sPtr("acct-incoming"),
		OrganizationID:   sPtr("org-incoming"),
	}
	if idx := findMatchingOAuthAccountIndex(existingAccounts, incoming); idx != -1 {
		t.Fatalf("expected no match for same email with different scoped account, got %d", idx)
	}
}

func TestFindMatchingOAuthAccountFallsBackToEmailWithoutScopedID(t *testing.T) {
	existingAccounts := []models.OpenAIAccount{
		{
			ID:          "existing-id",
			Email:       "legacy@example.com",
			AccountType: models.OpenAIAccountTypeOAuth,
		},
	}
	incoming := &models.OpenAIAccount{
		Email:       "legacy@example.com",
		AccountType: models.OpenAIAccountTypeOAuth,
	}
	if idx := findMatchingOAuthAccountIndex(existingAccounts, incoming); idx != 0 {
		t.Fatalf("expected legacy email fallback match at index 0, got %d", idx)
	}
}

func TestImportCPABytesReimportsSameEmailWithoutDuplicate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOpenAIHandlerTestDB(t)
	handler := NewOpenAIHandler(storage.NewOpenAIStorage(db), storage.NewCodexStorage(db))

	sessionJSON := []byte(`{
		"user": {"email": "fixture-user@example.com"},
		"expires": "2026-10-02T07:11:09.740Z",
		"account": {
			"id": "ff598c4d-ccaf-40c1-bfaa-cb94565764b1",
			"planType": "k12",
			"organizationId": "org-Kkd34iivEq9S3rRMezLxQqGG"
		},
		"accessToken": "at_example",
		"sessionToken": "st_example"
	}`)

	if _, err := handler.ImportCPABytes(sessionJSON); err != nil {
		t.Fatalf("first import: %v", err)
	}
	if _, err := handler.ImportCPABytes(sessionJSON); err != nil {
		t.Fatalf("second import: %v", err)
	}

	accounts, err := handler.storage.List()
	if err != nil {
		t.Fatalf("list accounts: %v", err)
	}
	var oauthCount int
	var imported *models.OpenAIAccount
	for _, account := range accounts {
		if account.AccountType == models.OpenAIAccountTypeOAuth &&
			strings.EqualFold(account.Email, "fixture-user@example.com") {
			oauthCount++
			imported = &account
		}
	}
	if oauthCount != 1 {
		t.Fatalf("expected exactly 1 oauth account for email, got %d", oauthCount)
	}
	if imported == nil {
		t.Fatal("expected imported account")
	}
	if imported.OrganizationID == nil || *imported.OrganizationID != "org-Kkd34iivEq9S3rRMezLxQqGG" {
		t.Fatalf("expected organization id to be saved, got %#v", imported.OrganizationID)
	}
	if imported.ChatGPTAccountID == nil || *imported.ChatGPTAccountID != "ff598c4d-ccaf-40c1-bfaa-cb94565764b1" {
		t.Fatalf("expected chatgpt account id to be saved, got %#v", imported.ChatGPTAccountID)
	}
	if !imported.ProxyEnabled {
		t.Fatalf("expected imported account to join proxy pool")
	}
}

func TestImportCPABytesSameEmailDifferentAccountIDsStayDistinct(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOpenAIHandlerTestDB(t)
	handler := NewOpenAIHandler(storage.NewOpenAIStorage(db), storage.NewCodexStorage(db))

	raw := []byte(`[
		{
			"email": "shared@example.com",
			"account_id": "acct-one",
			"organization_id": "org-one",
			"access_token": "at_one",
			"plan_type": "plus"
		},
		{
			"email": "shared@example.com",
			"account_id": "acct-two",
			"organization_id": "org-two",
			"access_token": "at_two",
			"plan_type": "team"
		}
	]`)

	if _, err := handler.ImportCPABytes(raw); err != nil {
		t.Fatalf("import cpa bytes: %v", err)
	}

	accounts, err := handler.storage.List()
	if err != nil {
		t.Fatalf("list accounts: %v", err)
	}
	if len(accounts) != 2 {
		t.Fatalf("expected 2 accounts for same email with different account IDs, got %d", len(accounts))
	}
	seen := map[string]string{}
	for _, account := range accounts {
		if !strings.EqualFold(account.Email, "shared@example.com") {
			continue
		}
		seen[derefStr(account.ChatGPTAccountID)] = derefStr(account.OrganizationID)
	}
	if seen["acct-one"] != "org-one" || seen["acct-two"] != "org-two" {
		t.Fatalf("expected distinct account/org identities, got %#v", seen)
	}
}

func TestImportCPABytesSameAccountIDWithoutOrgDifferentEmailsStayDistinct(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOpenAIHandlerTestDB(t)
	handler := NewOpenAIHandler(storage.NewOpenAIStorage(db), storage.NewCodexStorage(db))

	raw := []byte(`[
		{
			"email": "one@example.com",
			"account_id": "acct-shared",
			"access_token": "at_one",
			"plan_type": "plus"
		},
		{
			"email": "two@example.com",
			"account_id": "acct-shared",
			"access_token": "at_two",
			"plan_type": "plus"
		},
		{
			"email": "one@example.com",
			"account_id": "acct-shared",
			"access_token": "at_one_refresh",
			"plan_type": "team"
		}
	]`)

	out, err := handler.ImportCPABytes(raw)
	if err != nil {
		t.Fatalf("import cpa bytes: %v", err)
	}
	if out["created"] != 2 || out["updated"] != 1 {
		t.Fatalf("expected 2 created and 1 updated, got %#v", out)
	}

	accounts, err := handler.storage.List()
	if err != nil {
		t.Fatalf("list accounts: %v", err)
	}
	if len(accounts) != 2 {
		t.Fatalf("expected 2 accounts for same account id with different emails, got %d", len(accounts))
	}
	seen := map[string]string{}
	for _, account := range accounts {
		seen[account.Email] = derefStr(account.AccessToken)
	}
	if seen["one@example.com"] != "at_one_refresh" || seen["two@example.com"] != "at_two" {
		t.Fatalf("expected same-email import to update only that account, got %#v", seen)
	}
}

func TestImportCPABytesUsesOuterOrganizationIDWhenIDTokenOrgIsBlank(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOpenAIHandlerTestDB(t)
	handler := NewOpenAIHandler(storage.NewOpenAIStorage(db), storage.NewCodexStorage(db))

	token := testUnsignedJWT(t, map[string]interface{}{
		"email": "shared@example.com",
		"https://api.openai.com/auth": map[string]interface{}{
			"account_id":        "acct-shared",
			"chatgpt_plan_type": "chatgpt_plus",
			"organizations": []map[string]interface{}{
				{"id": "", "is_default": true},
			},
		},
	})
	raw := []byte(`[
		{
			"email": "shared@example.com",
			"account_id": "acct-shared",
			"organization_id": "org-one",
			"id_token": "` + token + `",
			"access_token": "at_one"
		},
		{
			"email": "shared@example.com",
			"account_id": "acct-shared",
			"organization_id": "org-two",
			"id_token": "` + token + `",
			"access_token": "at_two"
		}
	]`)

	if _, err := handler.ImportCPABytes(raw); err != nil {
		t.Fatalf("import cpa bytes: %v", err)
	}

	accounts, err := handler.storage.List()
	if err != nil {
		t.Fatalf("list accounts: %v", err)
	}
	if len(accounts) != 2 {
		t.Fatalf("expected 2 workspace-scoped accounts, got %d", len(accounts))
	}
	seen := map[string]bool{}
	for _, account := range accounts {
		if derefStr(account.ChatGPTAccountID) != "acct-shared" {
			t.Fatalf("expected account id acct-shared, got %q", derefStr(account.ChatGPTAccountID))
		}
		seen[derefStr(account.OrganizationID)] = true
	}
	if !seen["org-one"] || !seen["org-two"] {
		t.Fatalf("expected outer organization ids to be preserved, got %#v", seen)
	}
}

func TestEasyLLMExportImportPreservesOrganizationIDWithoutIDToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOpenAIHandlerTestDB(t)
	handler := NewOpenAIHandler(storage.NewOpenAIStorage(db), storage.NewCodexStorage(db))

	existingAccounts := []models.OpenAIAccount{}
	payload := &easyLLMExportPayload{
		OAuthAccounts: []exportedOAuthAccount{{
			ID:             "old-id",
			Email:          "backup@example.com",
			AccessToken:    "at_backup",
			RefreshToken:   "rt_backup",
			AccountID:      "acct_backup",
			OrganizationID: "org_backup",
			Status:         "active",
			Type:           "codex",
		}},
	}
	if _, err := handler.applyEasyLLMExportPayload(payload, &existingAccounts); err != nil {
		t.Fatalf("import export payload: %v", err)
	}

	accounts, err := handler.storage.List()
	if err != nil {
		t.Fatalf("list accounts: %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}
	account := accounts[0]
	if account.OrganizationID == nil || *account.OrganizationID != "org_backup" {
		t.Fatalf("expected organization id to be restored, got %#v", account.OrganizationID)
	}
	if account.ChatGPTAccountID == nil || *account.ChatGPTAccountID != "acct_backup" {
		t.Fatalf("expected account id to be restored, got %#v", account.ChatGPTAccountID)
	}
	if !account.ProxyEnabled {
		t.Fatalf("expected restored account to join proxy pool")
	}
}

func floatPtr(v float64) *float64 { return &v }

func int64Ptr(v int64) *int64 { return &v }
