package antigravity

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"easyllm/internal/models"
)

func mockAgyPaths(t *testing.T, home, appDataDir string) {
	t.Helper()
	originalHomeDir := agyUserHomeDir
	originalGetenv := agyGetenv
	t.Cleanup(func() {
		agyUserHomeDir = originalHomeDir
		agyGetenv = originalGetenv
	})
	agyUserHomeDir = func() (string, error) { return home, nil }
	agyGetenv = func(key string) string {
		if key == agyAppDataDirEnv {
			return appDataDir
		}
		return ""
	}
}

func TestSyncAccountToLocalCLI(t *testing.T) {
	home := t.TempDir()
	mockAgyPaths(t, home, "")

	idToken := "id-token"
	expiry := time.Date(2026, time.September, 23, 8, 30, 0, 0, time.UTC)
	account := &models.AntigravityAccount{
		AccessToken:     " access-token ",
		RefreshToken:    " refresh-token ",
		IDToken:         &idToken,
		ExpiryTimestamp: expiry.Unix(),
	}
	if err := SyncAccountToLocalCLI(account); err != nil {
		t.Fatalf("SyncAccountToLocalCLI() error = %v", err)
	}

	credentialPath := filepath.Join(home, ".gemini", agyCredentialFileName)
	payload, err := os.ReadFile(credentialPath)
	if err != nil {
		t.Fatalf("read CLI credential: %v", err)
	}
	var stored agyStoredToken
	if err := json.Unmarshal(payload, &stored); err != nil {
		t.Fatalf("unmarshal stored token: %v", err)
	}
	if stored.AuthMethod != "consumer" {
		t.Fatalf("unexpected auth method: %q", stored.AuthMethod)
	}
	if stored.Token.AccessToken != "access-token" || stored.Token.RefreshToken != "refresh-token" {
		t.Fatalf("unexpected stored OAuth token: %+v", stored.Token)
	}
	if stored.Token.TokenType != "Bearer" || !stored.Token.Expiry.Equal(expiry) {
		t.Fatalf("unexpected token type or expiry: %+v", stored.Token)
	}
	if strings.Contains(string(payload), "id_token") {
		t.Fatalf("agy credential must match the CLI schema and omit id_token: %s", payload)
	}
	if info, err := os.Stat(credentialPath); err != nil {
		t.Fatalf("stat CLI credential: %v", err)
	} else if info.Mode().Perm() != 0o600 {
		t.Fatalf("credential mode = %o, want 600", info.Mode().Perm())
	}
}

func TestSyncAccountToLocalCLIUsesConfiguredAppDataDir(t *testing.T) {
	home := t.TempDir()
	mockAgyPaths(t, home, "custom-gemini")

	if err := SyncAccountToLocalCLI(&models.AntigravityAccount{AccessToken: "token", ExpiresIn: 60}); err != nil {
		t.Fatalf("SyncAccountToLocalCLI() error = %v", err)
	}
	path := filepath.Join(home, "custom-gemini", agyCredentialFileName)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("configured credential path was not written: %v", err)
	}
}

func TestAgyCredentialPathAbsoluteAppDataDir(t *testing.T) {
	customDir := filepath.Join(t.TempDir(), "agy-data")
	got := agyCredentialPath("/ignored", customDir)
	want := filepath.Join(customDir, agyCredentialFileName)
	if got != want {
		t.Fatalf("agyCredentialPath() = %q, want %q", got, want)
	}
}

func TestSyncAccountToLocalCLIValidationAndWriteError(t *testing.T) {
	if err := SyncAccountToLocalCLI(nil); err == nil {
		t.Fatal("expected nil account error")
	}
	if err := SyncAccountToLocalCLI(&models.AntigravityAccount{}); err == nil {
		t.Fatal("expected missing access token error")
	}

	blockedHome := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blockedHome, []byte("blocked"), 0o600); err != nil {
		t.Fatalf("create blocking file: %v", err)
	}
	mockAgyPaths(t, blockedHome, "")
	err := SyncAccountToLocalCLI(&models.AntigravityAccount{AccessToken: "token", ExpiresIn: 60})
	if err == nil || !strings.Contains(err.Error(), "创建 agy CLI 凭据目录失败") {
		t.Fatalf("expected wrapped credential write error, got %v", err)
	}
}

func TestSyncAccountToLocalCLIHomeDirError(t *testing.T) {
	originalHomeDir := agyUserHomeDir
	t.Cleanup(func() { agyUserHomeDir = originalHomeDir })
	agyUserHomeDir = func() (string, error) { return "", errors.New("home unavailable") }

	err := SyncAccountToLocalCLI(&models.AntigravityAccount{AccessToken: "token", ExpiresIn: 60})
	if err == nil || !strings.Contains(err.Error(), "home unavailable") {
		t.Fatalf("expected wrapped home directory error, got %v", err)
	}
}
