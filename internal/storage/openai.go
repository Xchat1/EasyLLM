package storage

import (
	"easyllm/internal/models"
	"time"

	"gorm.io/gorm"
)

// --- OpenAI / Codex ---

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
	res := s.db.Where("id = ?", id).Delete(&models.OpenAIAccount{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
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
	var account models.OpenAIAccount
	if err := s.db.Select("id").Where("id = ?", id).First(&account).Error; err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.OpenAIAccount{}).
			Where("1 = 1").Update("is_codex_active", false).Error; err != nil {
			return err
		}
		res := tx.Model(&models.OpenAIAccount{}).
			Where("id = ?", id).Update("is_codex_active", true)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
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

// SetProxyForIDs updates proxy_enabled for a selected set of OAuth account IDs.
func (s *OpenAIStorage) SetProxyForIDs(ids []string, enabled bool) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	res := s.db.Model(&models.OpenAIAccount{}).
		Where("id IN ? AND account_type = ?", ids, models.OpenAIAccountTypeOAuth).
		Update("proxy_enabled", enabled)
	return res.RowsAffected, res.Error
}
