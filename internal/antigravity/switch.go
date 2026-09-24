package antigravity

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"easyllm/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

// ---- Protobuf Varint and Field Encoding Helpers ----

func encodeVarint(value uint64) []byte {
	var buf []byte
	for value >= 0x80 {
		buf = append(buf, byte(value&0x7F|0x80))
		value >>= 7
	}
	buf = append(buf, byte(value))
	return buf
}

func encodeLenDelimField(fieldNum uint32, data []byte) []byte {
	tag := (fieldNum << 3) | 2
	f := encodeVarint(uint64(tag))
	f = append(f, encodeVarint(uint64(len(data)))...)
	f = append(f, data...)
	return f
}

func encodeStringField(fieldNum uint32, value string) []byte {
	return encodeLenDelimField(fieldNum, []byte(value))
}

func encodeVarintField(fieldNum uint32, value uint64) []byte {
	tag := (fieldNum << 3) | 0
	f := encodeVarint(uint64(tag))
	return append(f, encodeVarint(value)...)
}

func readVarint(data []byte, offset int) (uint64, int, error) {
	var result uint64
	var shift uint
	pos := offset
	for {
		if pos >= len(data) {
			return 0, pos, fmt.Errorf("数据不完整")
		}
		b := data[pos]
		result |= uint64(b&0x7F) << shift
		pos++
		if b&0x80 == 0 {
			break
		}
		shift += 7
	}
	return result, pos, nil
}

func skipField(data []byte, offset int, wireType byte) (int, error) {
	switch wireType {
	case 0:
		_, newOffset, err := readVarint(data, offset)
		return newOffset, err
	case 1:
		return offset + 8, nil
	case 2:
		length, contentOffset, err := readVarint(data, offset)
		if err != nil {
			return 0, err
		}
		return contentOffset + int(length), nil
	case 5:
		return offset + 4, nil
	default:
		return 0, fmt.Errorf("未知 wire_type: %d", wireType)
	}
}

// createOAuthInfo encodes the OAuthTokenInfo protobuf message
func createOAuthInfo(accessToken, refreshToken string, expiry int64, isGcpTos bool, idToken, email string) []byte {
	if email != "" {
		lower := strings.ToLower(email)
		if strings.HasSuffix(lower, "@gmail.com") || strings.HasSuffix(lower, "@googlemail.com") {
			isGcpTos = false
		}
	}

	field1 := encodeStringField(1, accessToken)
	field2 := encodeStringField(2, "Bearer")
	field3 := encodeStringField(3, refreshToken)

	// Field 4: expiry Timestamp message (seconds = tag (1<<3)|0)
	timestampTag := (1 << 3) | 0
	timestampMsg := encodeVarint(uint64(timestampTag))
	timestampMsg = append(timestampMsg, encodeVarint(uint64(expiry))...)
	timestampMsg = append(timestampMsg, encodeVarintField(2, 0)...)
	field4 := encodeLenDelimField(4, timestampMsg)

	var oauthInfo []byte
	oauthInfo = append(oauthInfo, field1...)
	oauthInfo = append(oauthInfo, field2...)
	oauthInfo = append(oauthInfo, field3...)
	oauthInfo = append(oauthInfo, field4...)

	if idToken != "" {
		oauthInfo = append(oauthInfo, encodeStringField(5, idToken)...)
	}
	if isGcpTos {
		oauthInfo = append(oauthInfo, encodeVarintField(6, 1)...)
	}
	return oauthInfo
}

// createUnifiedTopicEntry encodes a map entry into Topic.data
func createUnifiedTopicEntry(sentinelKey string, payload []byte) []byte {
	b64Payload := base64.StdEncoding.EncodeToString(payload)
	row := encodeStringField(1, b64Payload)
	entry := append(encodeStringField(1, sentinelKey), encodeLenDelimField(2, row)...)
	return encodeLenDelimField(1, entry)
}

func createMinimalUserStatusPayload(email string) []byte {
	return append(encodeStringField(3, email), encodeStringField(7, email)...)
}

func removeUnifiedTopicEntry(data []byte, targetKey string) ([]byte, error) {
	var result []byte
	offset := 0

	for offset < len(data) {
		startOffset := offset
		tag, newOffset, err := readVarint(data, offset)
		if err != nil {
			return nil, err
		}
		wireType := byte(tag & 7)
		fieldNum := uint32(tag >> 3)
		nextOffset, err := skipField(data, newOffset, wireType)
		if err != nil {
			return nil, err
		}

		shouldRemove := false
		if fieldNum == 1 && wireType == 2 {
			length, contentOffset, err := readVarint(data, newOffset)
			if err == nil && contentOffset+int(length) <= len(data) {
				entry := data[contentOffset : contentOffset+int(length)]
				if getEntryKey(entry) == targetKey {
					shouldRemove = true
				}
			}
		}

		if !shouldRemove {
			result = append(result, data[startOffset:nextOffset]...)
		}
		offset = nextOffset
	}
	return result, nil
}

func getEntryKey(data []byte) string {
	offset := 0
	for offset < len(data) {
		tag, newOffset, err := readVarint(data, offset)
		if err != nil {
			return ""
		}
		wireType := byte(tag & 7)
		fieldNum := uint32(tag >> 3)

		if fieldNum == 1 && wireType == 2 {
			length, contentOffset, err := readVarint(data, newOffset)
			if err != nil || contentOffset+int(length) > len(data) {
				return ""
			}
			return string(data[contentOffset : contentOffset+int(length)])
		}

		nextOffset, err := skipField(data, newOffset, wireType)
		if err != nil {
			return ""
		}
		offset = nextOffset
	}
	return ""
}

// ---- Local Antigravity Desktop Discovery and Injection ----

func antigravityStateDBPathCandidates(goos, home, appdata string) (ide, antigravity2 []string) {
	switch goos {
	case "darwin":
		ide = append(ide,
			filepath.Join(home, "Library/Application Support/Antigravity IDE/User/globalStorage/state.vscdb"),
		)
		antigravity2 = append(antigravity2,
			filepath.Join(home, "Library/Application Support/Antigravity/User/globalStorage/state.vscdb"),
			filepath.Join(home, "Library/Application Support/Antigravity 2/User/globalStorage/state.vscdb"),
			filepath.Join(home, "Library/Application Support/Antigravity 2.0/User/globalStorage/state.vscdb"),
		)
	case "windows":
		if appdata != "" {
			ide = append(ide, filepath.Join(appdata, "Antigravity IDE/User/globalStorage/state.vscdb"))
			antigravity2 = append(antigravity2,
				filepath.Join(appdata, "Antigravity/User/globalStorage/state.vscdb"),
				filepath.Join(appdata, "Antigravity 2/User/globalStorage/state.vscdb"),
				filepath.Join(appdata, "Antigravity 2.0/User/globalStorage/state.vscdb"),
			)
		}
	case "linux":
		ide = append(ide, filepath.Join(home, ".config/Antigravity IDE/User/globalStorage/state.vscdb"))
		antigravity2 = append(antigravity2,
			filepath.Join(home, ".config/Antigravity/User/globalStorage/state.vscdb"),
			filepath.Join(home, ".config/Antigravity 2/User/globalStorage/state.vscdb"),
			filepath.Join(home, ".config/Antigravity 2.0/User/globalStorage/state.vscdb"),
		)
	}
	return ide, antigravity2
}

func existingStateDBPaths(candidates []string) []string {
	var existing []string
	for _, path := range candidates {
		if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
			existing = append(existing, path)
		}
	}
	return existing
}

func getAntigravityStateDBPaths() (ide, antigravity2 []string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil
	}
	ideCandidates, antigravity2Candidates := antigravityStateDBPathCandidates(runtime.GOOS, home, os.Getenv("APPDATA"))
	return existingStateDBPaths(ideCandidates), existingStateDBPaths(antigravity2Candidates)
}

// InjectAccountToLocalIDE writes credentials into local Antigravity IDE state.vscdb if found.
func InjectAccountToLocalIDE(account *models.AntigravityAccount) ([]string, error) {
	paths, _ := getAntigravityStateDBPaths()
	return injectAccountToStateDBPaths(account, paths)
}

// InjectAccountToLocalAntigravity2 writes credentials into the Antigravity 2.0
// desktop application's state database. Antigravity 2.0 uses the
// com.google.antigravity bundle and the Antigravity user-data directory, while
// the legacy IDE uses com.google.antigravity-ide / Antigravity IDE.
func InjectAccountToLocalAntigravity2(account *models.AntigravityAccount) ([]string, error) {
	_, paths := getAntigravityStateDBPaths()
	return injectAccountToStateDBPaths(account, paths)
}

func injectAccountToStateDBPaths(account *models.AntigravityAccount, paths []string) (injectedPaths []string, err error) {
	if account == nil {
		return nil, fmt.Errorf("账号不能为空")
	}
	if len(paths) == 0 {
		return nil, nil
	}

	var injectErrors []error
	for _, dbPath := range paths {
		if err := injectAccountToStateDB(account, dbPath); err != nil {
			injectErrors = append(injectErrors, fmt.Errorf("%s: %w", dbPath, err))
			continue
		}
		injectedPaths = append(injectedPaths, dbPath)
	}
	return injectedPaths, errors.Join(injectErrors...)
}

func injectAccountToStateDB(account *models.AntigravityAccount, dbPath string) error {
	idToken := ""
	if account.IDToken != nil {
		idToken = *account.IDToken
	}

	oauthInfo := createOAuthInfo(
		account.AccessToken,
		account.RefreshToken,
		account.ExpiryTimestamp,
		account.IsGcpTos,
		idToken,
		account.Email,
	)

	entry := createUnifiedTopicEntry("oauthTokenInfoSentinelKey", oauthInfo)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.Exec("PRAGMA busy_timeout = 2000"); err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Preserve other unified-state entries while replacing the active OAuth token.
	var currentB64 string
	rowErr := tx.QueryRow("SELECT value FROM ItemTable WHERE key = ?", "antigravityUnifiedStateSync.oauthToken").Scan(&currentB64)
	if rowErr != nil && !errors.Is(rowErr, sql.ErrNoRows) {
		return rowErr
	}
	var topicData []byte
	if currentB64 != "" {
		decoded, decodeErr := base64.StdEncoding.DecodeString(currentB64)
		if decodeErr != nil {
			return fmt.Errorf("解析现有 OAuth 状态失败: %w", decodeErr)
		}
		cleaned, cleanErr := removeUnifiedTopicEntry(decoded, "oauthTokenInfoSentinelKey")
		if cleanErr != nil {
			return fmt.Errorf("更新现有 OAuth 状态失败: %w", cleanErr)
		}
		topicData = cleaned
	}
	topicData = append(topicData, entry...)
	topicB64 := base64.StdEncoding.EncodeToString(topicData)
	if _, err := tx.Exec("INSERT OR REPLACE INTO ItemTable (key, value) VALUES (?, ?)",
		"antigravityUnifiedStateSync.oauthToken", topicB64); err != nil {
		return err
	}

	userStatusPayload := createMinimalUserStatusPayload(account.Email)
	userStatusEntry := createUnifiedTopicEntry("userStatusSentinelKey", userStatusPayload)
	userStatusB64 := base64.StdEncoding.EncodeToString(userStatusEntry)
	if _, err := tx.Exec("INSERT OR REPLACE INTO ItemTable (key, value) VALUES (?, ?)",
		"antigravityUnifiedStateSync.userStatus", userStatusB64); err != nil {
		return err
	}
	if _, err := tx.Exec("INSERT OR REPLACE INTO ItemTable (key, value) VALUES (?, ?)",
		"antigravityOnboarding", "true"); err != nil {
		return err
	}

	return tx.Commit()
}
