package antigravity

import (
	"database/sql"
	"encoding/base64"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"easyllm/internal/models"
)

func TestAntigravityStateDBPathCandidates(t *testing.T) {
	tests := []struct {
		name       string
		goos       string
		home       string
		appdata    string
		wantIDE    string
		wantAG2    string
		wantCounts [2]int
	}{
		{
			name:       "macOS",
			goos:       "darwin",
			home:       "/Users/test",
			wantIDE:    filepath.Join("/Users/test", "Library/Application Support/Antigravity IDE/User/globalStorage/state.vscdb"),
			wantAG2:    filepath.Join("/Users/test", "Library/Application Support/Antigravity/User/globalStorage/state.vscdb"),
			wantCounts: [2]int{1, 3},
		},
		{
			name:       "Windows",
			goos:       "windows",
			home:       `C:\Users\test`,
			appdata:    `C:\Users\test\AppData\Roaming`,
			wantIDE:    filepath.Join(`C:\Users\test\AppData\Roaming`, "Antigravity IDE/User/globalStorage/state.vscdb"),
			wantAG2:    filepath.Join(`C:\Users\test\AppData\Roaming`, "Antigravity/User/globalStorage/state.vscdb"),
			wantCounts: [2]int{1, 3},
		},
		{
			name:       "Linux",
			goos:       "linux",
			home:       "/home/test",
			wantIDE:    filepath.Join("/home/test", ".config/Antigravity IDE/User/globalStorage/state.vscdb"),
			wantAG2:    filepath.Join("/home/test", ".config/Antigravity/User/globalStorage/state.vscdb"),
			wantCounts: [2]int{1, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ide, ag2 := antigravityStateDBPathCandidates(tt.goos, tt.home, tt.appdata)
			if len(ide) != tt.wantCounts[0] || len(ag2) != tt.wantCounts[1] {
				t.Fatalf("unexpected candidate counts: IDE=%d AG2=%d", len(ide), len(ag2))
			}
			if ide[0] != tt.wantIDE || ag2[0] != tt.wantAG2 {
				t.Fatalf("unexpected primary paths: IDE=%q AG2=%q", ide[0], ag2[0])
			}
		})
	}
}

func TestInjectAccountToStateDBPaths(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "state.vscdb")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if _, err := db.Exec("CREATE TABLE ItemTable (key TEXT PRIMARY KEY, value TEXT)"); err != nil {
		t.Fatalf("create ItemTable: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close sqlite: %v", err)
	}

	idToken := "id-token"
	account := &models.AntigravityAccount{
		Email:           "test@example.com",
		AccessToken:     "access-token",
		RefreshToken:    "refresh-token",
		IDToken:         &idToken,
		ExpiryTimestamp: time.Now().Add(time.Hour).Unix(),
	}
	paths, err := injectAccountToStateDBPaths(account, []string{dbPath})
	if err != nil {
		t.Fatalf("inject account: %v", err)
	}
	if len(paths) != 1 || paths[0] != dbPath {
		t.Fatalf("unexpected injected paths: %v", paths)
	}

	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("reopen sqlite: %v", err)
	}
	defer db.Close()
	values := map[string]string{}
	rows, err := db.Query("SELECT key, value FROM ItemTable")
	if err != nil {
		t.Fatalf("query ItemTable: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			t.Fatalf("scan ItemTable: %v", err)
		}
		values[key] = value
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate ItemTable: %v", err)
	}
	if values["antigravityOnboarding"] != "true" {
		t.Fatalf("onboarding flag not injected: %q", values["antigravityOnboarding"])
	}
	for _, key := range []string{"antigravityUnifiedStateSync.oauthToken", "antigravityUnifiedStateSync.userStatus"} {
		decoded, err := base64.StdEncoding.DecodeString(values[key])
		if err != nil || len(decoded) == 0 {
			t.Fatalf("%s was not injected as base64 protobuf: len=%d err=%v", key, len(decoded), err)
		}
	}
}

func TestInjectAccountToStateDBPathsReportsFailure(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "state.vscdb")
	paths, err := injectAccountToStateDBPaths(&models.AntigravityAccount{AccessToken: "token"}, []string{dbPath})
	if err == nil || !strings.Contains(err.Error(), dbPath) {
		t.Fatalf("expected path-specific injection error, got paths=%v err=%v", paths, err)
	}
	if len(paths) != 0 {
		t.Fatalf("failed database must not be reported as injected: %v", paths)
	}
}
