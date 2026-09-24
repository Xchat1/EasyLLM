package antigravity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"easyllm/internal/models"
)

const (
	agyCredentialFileName = "jetski-standalone-oauth-token"
	agyAppDataDirEnv      = "JETSKI_APP_DATA_DIR"
)

var (
	agyUserHomeDir = os.UserHomeDir
	agyGetenv      = os.Getenv
)

type agyOAuthToken struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
}

type agyStoredToken struct {
	Token      agyOAuthToken `json:"token"`
	AuthMethod string        `json:"auth_method"`
}

func agyCredentialPath(home, appDataDir string) string {
	appDataDir = strings.TrimSpace(appDataDir)
	if appDataDir == "" {
		appDataDir = filepath.Join(home, ".gemini")
	} else if !filepath.IsAbs(appDataDir) {
		appDataDir = filepath.Join(home, appDataDir)
	}
	return filepath.Join(appDataDir, agyCredentialFileName)
}

func writeAgyCredential(path string, payload []byte) (err error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("创建 agy CLI 凭据目录失败: %w", err)
	}

	// Keep the credential private and replace it atomically so a concurrently
	// starting CLI process cannot observe a partially written JSON document.
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+"-*")
	if err != nil {
		return fmt.Errorf("创建 agy CLI 临时凭据失败: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	if err := tmp.Chmod(0o600); err != nil {
		return fmt.Errorf("设置 agy CLI 凭据权限失败: %w", err)
	}
	if _, err := tmp.Write(payload); err != nil {
		return fmt.Errorf("写入 agy CLI 临时凭据失败: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("同步 agy CLI 临时凭据失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("关闭 agy CLI 临时凭据失败: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("替换 agy CLI 凭据失败: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("设置 agy CLI 凭据权限失败: %w", err)
	}
	return nil
}

// SyncAccountToLocalCLI updates the OAuth file read by the Antigravity CLI.
// agy 1.2.x stores this separately from the desktop application's SQLite state
// and does not use the gemini/antigravity system-keyring item.
func SyncAccountToLocalCLI(account *models.AntigravityAccount) error {
	if account == nil {
		return fmt.Errorf("账号不能为空")
	}
	if strings.TrimSpace(account.AccessToken) == "" {
		return fmt.Errorf("账号缺少 access token")
	}

	expiry := time.Unix(account.ExpiryTimestamp, 0).UTC()
	if account.ExpiryTimestamp <= 0 {
		expiresIn := account.ExpiresIn
		if expiresIn <= 0 {
			expiresIn = 3600
		}
		expiry = time.Now().Add(time.Duration(expiresIn) * time.Second).UTC()
	}

	payload, err := json.Marshal(agyStoredToken{
		Token: agyOAuthToken{
			AccessToken:  strings.TrimSpace(account.AccessToken),
			TokenType:    "Bearer",
			RefreshToken: strings.TrimSpace(account.RefreshToken),
			Expiry:       expiry,
		},
		AuthMethod: "consumer",
	})
	if err != nil {
		return fmt.Errorf("生成 agy CLI 凭据失败: %w", err)
	}

	home, err := agyUserHomeDir()
	if err != nil {
		return fmt.Errorf("定位 agy CLI 用户目录失败: %w", err)
	}
	path := agyCredentialPath(home, agyGetenv(agyAppDataDirEnv))
	if err := writeAgyCredential(path, payload); err != nil {
		return err
	}
	return nil
}
