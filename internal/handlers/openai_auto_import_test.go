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

	"easyllm/internal/storage"

	"github.com/gin-gonic/gin"
)

func TestDetectAutoImportFormat(t *testing.T) {
	cases := []struct {
		name     string
		filename string
		raw      string
		want     string
	}{
		{
			name: "easyllm export",
			raw:  `{"oauth_accounts":[{"email":"a@b.com","access_token":"at"}],"api_accounts":[]}`,
			want: scanFormatEasyLLMExport,
		},
		{
			name:     "cpa by filename",
			filename: "user@example.com-cpa.json",
			raw:      `{"type":"codex","email":"u@example.com","access_token":"at","refresh_token":"rt"}`,
			want:     scanFormatCPA,
		},
		{
			name: "token flat",
			raw:  `{"email":"a@b.com","access_token":"at","refresh_token":"rt","id_token":"id"}`,
			want: scanFormatToken,
		},
		{
			name: "sub2api current v1",
			raw:  `{"type":"sub2api-data","version":1,"exported_at":"2026-07-16T00:00:00Z","proxies":[],"accounts":[{"name":"a@b.com","platform":"openai","type":"oauth","credentials":{"access_token":"at"}}]}`,
			want: scanFormatSub2API,
		},
		{
			name: "sub2api legacy unversioned",
			raw:  `{"exported_at":"2026-07-16T00:00:00Z","accounts":[{"name":"a@b.com","platform":"openai","type":"oauth","credentials":{"access_token":"at"}}]}`,
			want: scanFormatSub2API,
		},
		{
			name: "sub2api api envelope",
			raw:  `{"data":{"type":"sub2api-bundle","version":1,"proxies":[],"accounts":[{"name":"a@b.com","platform":"openai","type":"oauth","credentials":{"access_token":"at"}}]}}`,
			want: scanFormatSub2API,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := detectAutoImportFormat([]byte(tc.raw), tc.filename)
			if got != tc.want {
				t.Fatalf("detectAutoImportFormat() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestImportAutoJSONFileSupportsSub2API(t *testing.T) {
	db := setupOpenAIHandlerTestDB(t)
	handler := NewOpenAIHandler(storage.NewOpenAIStorage(db), storage.NewCodexStorage(db))
	existingAccounts, err := handler.storage.List()
	if err != nil {
		t.Fatalf("list existing accounts: %v", err)
	}

	raw := []byte(`{
		"type":"sub2api-data",
		"version":1,
		"exported_at":"2026-07-16T00:00:00Z",
		"proxies":[],
		"accounts":[
			{
				"name":"sub@example.com",
				"platform":"openai",
				"type":"oauth",
				"credentials":{
					"access_token":"access-1",
					"refresh_token":"refresh-1",
					"id_token":"id-1",
					"email":"sub@example.com",
					"chatgpt_account_id":"acct-1",
					"chatgpt_user_id":"user-1",
					"organization_id":"org-1",
					"plan_type":"chatgpt_plus",
					"expires_at":1893456000000
				}
			},
			{
				"name":"claude@example.com",
				"platform":"anthropic",
				"type":"oauth",
				"credentials":{"access_token":"claude-access"}
			}
		]
	}`)

	results := handler.importAutoJSONFile("sub2api-export.json", raw, &existingAccounts)
	if len(results) != 2 {
		t.Fatalf("result count = %d, want 2", len(results))
	}
	if !results[0].Success || results[0].Action != "created" || results[0].Format != scanFormatSub2API {
		t.Fatalf("unexpected OpenAI import result: %+v", results[0])
	}
	if !results[1].Skipped || results[1].Success {
		t.Fatalf("expected non-OpenAI account to be skipped: %+v", results[1])
	}

	accounts, err := handler.storage.List()
	if err != nil {
		t.Fatalf("list imported accounts: %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("account count = %d, want 1", len(accounts))
	}
	account := accounts[0]
	assertStringPointerValue(t, "access token", account.AccessToken, "access-1")
	assertStringPointerValue(t, "refresh token", account.RefreshToken, "refresh-1")
	assertStringPointerValue(t, "id token", account.IDToken, "id-1")
	assertStringPointerValue(t, "account id", account.ChatGPTAccountID, "acct-1")
	assertStringPointerValue(t, "user id", account.ChatGPTUserID, "user-1")
	assertStringPointerValue(t, "organization id", account.OrganizationID, "org-1")
	assertStringPointerValue(t, "plan", account.Plan, "plus")
	wantExpiry := time.Date(2030, time.January, 1, 0, 0, 0, 0, time.UTC)
	if account.ExpiresAt == nil || !account.ExpiresAt.Equal(wantExpiry) {
		t.Fatalf("expires_at = %v, want %v", account.ExpiresAt, wantExpiry)
	}

	updateRaw := []byte(`{"data":{"type":"sub2api-bundle","version":1,"proxies":[],"accounts":[{
		"name":"sub@example.com",
		"platform":"openai",
		"type":"oauth",
		"credentials":{
			"access_token":"access-2",
			"refresh_token":"refresh-2",
			"email":"sub@example.com",
			"chatgpt_account_id":"acct-1",
			"organization_id":"org-1"
		}
	}]}}`)
	updateResults := handler.importAutoJSONFile("sub2api-update.json", updateRaw, &existingAccounts)
	if len(updateResults) != 1 || !updateResults[0].Success || updateResults[0].Action != "updated" {
		t.Fatalf("unexpected update result: %+v", updateResults)
	}

	accounts, err = handler.storage.List()
	if err != nil {
		t.Fatalf("list updated accounts: %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("updated account count = %d, want 1", len(accounts))
	}
	assertStringPointerValue(t, "updated access token", accounts[0].AccessToken, "access-2")
	assertStringPointerValue(t, "updated refresh token", accounts[0].RefreshToken, "refresh-2")
}

func TestImportAutoJSONFileSupportsUnversionedSub2APIBatch(t *testing.T) {
	db := setupOpenAIHandlerTestDB(t)
	handler := NewOpenAIHandler(storage.NewOpenAIStorage(db), storage.NewCodexStorage(db))
	existingAccounts, err := handler.storage.List()
	if err != nil {
		t.Fatalf("list existing accounts: %v", err)
	}

	raw := []byte(`{
		"exported_at":"2026-07-18T08:34:13Z",
		"proxies":[],
		"accounts":[{
			"name":"batch@example.com",
			"platform":"openai",
			"type":"oauth",
			"credentials":{
				"access_token":"access-batch",
				"chatgpt_account_id":"acct-batch",
				"chatgpt_user_id":"user-batch",
				"email":"batch@example.com",
				"expires_at":"2026-07-28T07:36:46.000Z",
				"expires_in":863999,
				"plan_type":"k12"
			},
			"extra":{
				"email":"batch@example.com",
				"last_refresh":"2026-07-18T08:34:13Z",
				"source":"batch-export"
			},
			"concurrency":1,
			"priority":0,
			"rate_multiplier":1,
			"auto_pause_on_expired":true
		}]
	}`)

	if got := detectAutoImportFormat(raw, "batch-0119.json"); got != scanFormatSub2API {
		t.Fatalf("detectAutoImportFormat() = %q, want %q", got, scanFormatSub2API)
	}
	results := handler.importAutoJSONFile("batch-0119.json", raw, &existingAccounts)
	if len(results) != 1 || !results[0].Success || results[0].Action != "created" {
		t.Fatalf("unexpected batch import result: %+v", results)
	}

	accounts, err := handler.storage.List()
	if err != nil {
		t.Fatalf("list imported accounts: %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("account count = %d, want 1", len(accounts))
	}
	account := accounts[0]
	assertStringPointerValue(t, "access token", account.AccessToken, "access-batch")
	assertStringPointerValue(t, "account id", account.ChatGPTAccountID, "acct-batch")
	assertStringPointerValue(t, "user id", account.ChatGPTUserID, "user-batch")
	assertStringPointerValue(t, "plan", account.Plan, "k12")
	wantExpiry := time.Date(2026, time.July, 28, 7, 36, 46, 0, time.UTC)
	if account.ExpiresAt == nil || !account.ExpiresAt.Equal(wantExpiry) {
		t.Fatalf("expires_at = %v, want %v", account.ExpiresAt, wantExpiry)
	}
}

func TestImportByAutoJSONDetectsSub2API(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOpenAIHandlerTestDB(t)
	handler := NewOpenAIHandler(storage.NewOpenAIStorage(db), storage.NewCodexStorage(db))
	body := strings.NewReader(`{
		"accounts":[{
			"name":"manual@example.com",
			"platform":"openai",
			"type":"oauth",
			"credentials":{
				"access_token":"manual-access",
				"email":"manual@example.com",
				"chatgpt_account_id":"manual-account"
			}
		}]
	}`)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/openai/import/auto-json", body)

	handler.ImportByAutoJSON(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Success int                 `json:"success"`
		Failed  int                 `json:"failed"`
		Results []tokenImportResult `json:"results"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Success != 1 || response.Failed != 0 || len(response.Results) != 1 {
		t.Fatalf("unexpected response: %+v", response)
	}
	if response.Results[0].Format != scanFormatSub2API || response.Results[0].Action != "created" {
		t.Fatalf("unexpected import result: %+v", response.Results[0])
	}
}

func TestImportByAutoFilesSupportsMixedFormats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOpenAIHandlerTestDB(t)
	handler := NewOpenAIHandler(storage.NewOpenAIStorage(db), storage.NewCodexStorage(db))

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	files := map[string]string{
		"account-cpa.json": `{
			"type":"codex",
			"email":"cpa@example.com",
			"access_token":"cpa-access"
		}`,
		"sub2api-batch.json": `{
			"accounts":[{
				"name":"sub2api@example.com",
				"platform":"openai",
				"type":"oauth",
				"credentials":{"access_token":"sub2api-access","email":"sub2api@example.com"}
			}]
		}`,
		"easyllm-backup.json": `{
			"oauth_accounts":[{"email":"easyllm@example.com","access_token":"easyllm-access"}],
			"api_accounts":[]
		}`,
	}
	for name, content := range files {
		part, err := writer.CreateFormFile("files", name)
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if _, err := part.Write([]byte(content)); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(http.MethodPost, "/openai/import/auto-files", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	ctx.Request = request
	handler.ImportByAutoFiles(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Success int                 `json:"success"`
		Failed  int                 `json:"failed"`
		Results []tokenImportResult `json:"results"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Success != 3 || response.Failed != 0 || len(response.Results) != 3 {
		t.Fatalf("unexpected response: %+v", response)
	}
	formats := make(map[string]bool, len(response.Results))
	for _, result := range response.Results {
		formats[result.Format] = result.Success
	}
	for _, format := range []string{scanFormatCPA, scanFormatSub2API, scanFormatEasyLLMExport} {
		if !formats[format] {
			t.Fatalf("missing successful %s result: %+v", format, response.Results)
		}
	}
}

func TestParseSub2APIExportRejectsUnsupportedVersion(t *testing.T) {
	raw := []byte(`{"type":"sub2api-data","version":2,"accounts":[]}`)
	if detectAutoImportFormat(raw, "future.json") != scanFormatSub2API {
		t.Fatal("expected future Sub2API payload to be recognized before validation")
	}
	if _, err := parseSub2APIExport(raw); err == nil {
		t.Fatal("expected unsupported Sub2API version to be rejected")
	}
}

func TestSub2APIExpiryAcceptsFractionalMicroseconds(t *testing.T) {
	credentials := map[string]json.RawMessage{
		"expires_at": json.RawMessage(`1893456000000000.0`),
	}
	if got := sub2APIExpiry(credentials); got != "2030-01-01T00:00:00Z" {
		t.Fatalf("sub2APIExpiry() = %q, want 2030-01-01T00:00:00Z", got)
	}
}

func assertStringPointerValue(t *testing.T, field string, got *string, want string) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v, want %q", field, got, want)
	}
}
