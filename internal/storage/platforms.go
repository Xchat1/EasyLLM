package storage

import (
	"easyllm/internal/models"
	"time"

	"gorm.io/gorm"
)

// --- OpenAI ---

type OpenAIStorage struct{ db *gorm.DB }

func NewOpenAIStorage(db *gorm.DB) *OpenAIStorage { return &OpenAIStorage{db: db} }

func (s *OpenAIStorage) Save(account *models.OpenAIAccount) error {
	account.UpdatedAt = time.Now()
	return s.db.Save(account).Error
}
func (s *OpenAIStorage) List() ([]models.OpenAIAccount, error) {
	var list []models.OpenAIAccount
	return list, s.db.Order("created_at desc").Find(&list).Error
}
func (s *OpenAIStorage) Get(id string) (*models.OpenAIAccount, error) {
	var a models.OpenAIAccount
	return &a, s.db.Where("id = ?", id).First(&a).Error
}

// GetCodexActive returns the account marked as is_codex_active=true.
// This is used as the default upstream configuration for /v1/* proxying.
func (s *OpenAIStorage) GetCodexActive() (*models.OpenAIAccount, error) {
	var a models.OpenAIAccount
	if err := s.db.Where("is_codex_active = ?", true).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}
func (s *OpenAIStorage) Delete(id string) error {
	return s.db.Where("id = ?", id).Delete(&models.OpenAIAccount{}).Error
}
func (s *OpenAIStorage) DeleteMany(ids []string) error {
	return s.db.Where("id IN ?", ids).Delete(&models.OpenAIAccount{}).Error
}

// GetByAccessToken returns a single account matching the given access token.
func (s *OpenAIStorage) GetByAccessToken(token string) (*models.OpenAIAccount, error) {
	var a models.OpenAIAccount
	err := s.db.Where("access_token = ?", token).First(&a).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ListProxyEnabled returns all OAuth accounts with proxy_enabled=true whose token is not expired
func (s *OpenAIStorage) ListProxyEnabled() ([]models.OpenAIAccount, error) {
	var list []models.OpenAIAccount
	err := s.db.Where("proxy_enabled = ? AND account_type = ? AND (expires_at IS NULL OR expires_at > ?)",
		true, models.OpenAIAccountTypeOAuth, time.Now()).
		Order("created_at desc").Find(&list).Error
	return list, err
}

// SetCodexActive marks one account as is_codex_active=true, clears all others
func (s *OpenAIStorage) SetCodexActive(id string) error {
	// Clear all first
	if err := s.db.Model(&models.OpenAIAccount{}).
		Where("1 = 1").Update("is_codex_active", false).Error; err != nil {
		return err
	}
	return s.db.Model(&models.OpenAIAccount{}).
		Where("id = ?", id).Update("is_codex_active", true).Error
}

// ToggleProxy flips proxy_enabled for a single account
func (s *OpenAIStorage) ToggleProxy(id string) (bool, error) {
	var account models.OpenAIAccount
	if err := s.db.Where("id = ?", id).First(&account).Error; err != nil {
		return false, err
	}
	account.ProxyEnabled = !account.ProxyEnabled
	account.UpdatedAt = time.Now()
	if err := s.db.Save(&account).Error; err != nil {
		return false, err
	}
	return account.ProxyEnabled, nil
}

// SetProxyAll sets proxy_enabled for all OAuth accounts (used for one-click enable/disable pool).
func (s *OpenAIStorage) SetProxyAll(enabled bool) (int64, error) {
	res := s.db.Model(&models.OpenAIAccount{}).
		Where("account_type = ?", models.OpenAIAccountTypeOAuth).
		Update("proxy_enabled", enabled)
	return res.RowsAffected, res.Error
}

// CountProxyEnabled returns the number of OAuth accounts with proxy_enabled=true.
func (s *OpenAIStorage) CountProxyEnabled() (int64, error) {
	var count int64
	err := s.db.Model(&models.OpenAIAccount{}).
		Where("proxy_enabled = ? AND account_type = ?", true, models.OpenAIAccountTypeOAuth).
		Count(&count).Error
	return count, err
}

// --- Cursor ---

type CursorStorage struct{ db *gorm.DB }

func NewCursorStorage(db *gorm.DB) *CursorStorage { return &CursorStorage{db: db} }

func (s *CursorStorage) Save(account *models.CursorAccount) error {
	account.UpdatedAt = time.Now()
	return s.db.Save(account).Error
}
func (s *CursorStorage) List() ([]models.CursorAccount, error) {
	var list []models.CursorAccount
	return list, s.db.Order("created_at desc").Find(&list).Error
}
func (s *CursorStorage) Get(id string) (*models.CursorAccount, error) {
	var a models.CursorAccount
	return &a, s.db.Where("id = ?", id).First(&a).Error
}
func (s *CursorStorage) Delete(id string) error {
	return s.db.Where("id = ?", id).Delete(&models.CursorAccount{}).Error
}
func (s *CursorStorage) DeleteMany(ids []string) error {
	return s.db.Where("id IN ?", ids).Delete(&models.CursorAccount{}).Error
}
func (s *CursorStorage) SetActive(id string) error {
	if err := s.db.Model(&models.CursorAccount{}).Where("1 = 1").Update("active", false).Error; err != nil {
		return err
	}
	return s.db.Model(&models.CursorAccount{}).Where("id = ?", id).Update("active", true).Error
}

// --- Windsurf ---

type WindsurfStorage struct{ db *gorm.DB }

func NewWindsurfStorage(db *gorm.DB) *WindsurfStorage { return &WindsurfStorage{db: db} }

func (s *WindsurfStorage) Save(account *models.WindsurfAccount) error {
	account.UpdatedAt = time.Now()
	return s.db.Save(account).Error
}
func (s *WindsurfStorage) List() ([]models.WindsurfAccount, error) {
	var list []models.WindsurfAccount
	return list, s.db.Order("created_at desc").Find(&list).Error
}
func (s *WindsurfStorage) Get(id string) (*models.WindsurfAccount, error) {
	var a models.WindsurfAccount
	return &a, s.db.Where("id = ?", id).First(&a).Error
}
func (s *WindsurfStorage) Delete(id string) error {
	return s.db.Where("id = ?", id).Delete(&models.WindsurfAccount{}).Error
}
func (s *WindsurfStorage) DeleteMany(ids []string) error {
	return s.db.Where("id IN ?", ids).Delete(&models.WindsurfAccount{}).Error
}
func (s *WindsurfStorage) SetActive(id string) error {
	if err := s.db.Model(&models.WindsurfAccount{}).Where("1 = 1").Update("active", false).Error; err != nil {
		return err
	}
	return s.db.Model(&models.WindsurfAccount{}).Where("id = ?", id).Update("active", true).Error
}

// --- Antigravity ---

type AntigravityStorage struct{ db *gorm.DB }

func NewAntigravityStorage(db *gorm.DB) *AntigravityStorage { return &AntigravityStorage{db: db} }

func (s *AntigravityStorage) Save(account *models.AntigravityAccount) error {
	account.UpdatedAt = time.Now()
	return s.db.Save(account).Error
}
func (s *AntigravityStorage) List() ([]models.AntigravityAccount, error) {
	var list []models.AntigravityAccount
	return list, s.db.Order("created_at desc").Find(&list).Error
}
func (s *AntigravityStorage) Get(id string) (*models.AntigravityAccount, error) {
	var a models.AntigravityAccount
	return &a, s.db.Where("id = ?", id).First(&a).Error
}
func (s *AntigravityStorage) Delete(id string) error {
	return s.db.Where("id = ?", id).Delete(&models.AntigravityAccount{}).Error
}
func (s *AntigravityStorage) DeleteMany(ids []string) error {
	return s.db.Where("id IN ?", ids).Delete(&models.AntigravityAccount{}).Error
}
func (s *AntigravityStorage) SetActive(id string) error {
	if err := s.db.Model(&models.AntigravityAccount{}).Where("1 = 1").Update("active", false).Error; err != nil {
		return err
	}
	return s.db.Model(&models.AntigravityAccount{}).Where("id = ?", id).Update("active", true).Error
}

// --- Claude ---

type ClaudeStorage struct{ db *gorm.DB }

func NewClaudeStorage(db *gorm.DB) *ClaudeStorage { return &ClaudeStorage{db: db} }

func (s *ClaudeStorage) Save(account *models.ClaudeAccount) error {
	account.UpdatedAt = time.Now()
	return s.db.Save(account).Error
}
func (s *ClaudeStorage) List() ([]models.ClaudeAccount, error) {
	var list []models.ClaudeAccount
	return list, s.db.Order("created_at desc").Find(&list).Error
}
func (s *ClaudeStorage) Get(id string) (*models.ClaudeAccount, error) {
	var a models.ClaudeAccount
	return &a, s.db.Where("id = ?", id).First(&a).Error
}
func (s *ClaudeStorage) Delete(id string) error {
	return s.db.Where("id = ?", id).Delete(&models.ClaudeAccount{}).Error
}
func (s *ClaudeStorage) DeleteMany(ids []string) error {
	return s.db.Where("id IN ?", ids).Delete(&models.ClaudeAccount{}).Error
}

