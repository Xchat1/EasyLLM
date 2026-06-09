package storage

import (
	"easyllm/internal/models"
	"time"

	"gorm.io/gorm"
)

// CodexStorage handles Codex account CRUD
type CodexStorage struct {
	db *gorm.DB
}

func NewCodexStorage(db *gorm.DB) *CodexStorage {
	return &CodexStorage{db: db}
}

// LoadEnabledAccounts returns all enabled Codex accounts
func (s *CodexStorage) LoadEnabledAccounts() ([]*models.CodexAccount, error) {
	var accounts []*models.CodexAccount
	err := s.db.Where("enabled = ?", true).Order("created_at desc").Find(&accounts).Error
	return accounts, err
}

// LoadAllAccounts returns all Codex accounts
func (s *CodexStorage) LoadAllAccounts() ([]*models.CodexAccount, error) {
	var accounts []*models.CodexAccount
	err := s.db.Order("created_at desc").Find(&accounts).Error
	return accounts, err
}

func (s *CodexStorage) GetAccount(id string) (*models.CodexAccount, error) {
	var account models.CodexAccount
	if err := s.db.Where("id = ?", id).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

// SaveAccount upserts a Codex account
func (s *CodexStorage) SaveAccount(account *models.CodexAccount) error {
	account.UpdatedAt = time.Now()
	return s.db.Save(account).Error
}

// UpdateAccountStats updates request count and last used time
func (s *CodexStorage) UpdateAccountStats(account *models.CodexAccount) error {
	now := time.Now()
	return s.db.Model(account).Updates(map[string]interface{}{
		"request_count": account.RequestCount,
		"last_used_at":  now,
		"updated_at":    now,
	}).Error
}

// IncrementRequestCount atomically increments the request count for a single account by ID.
func (s *CodexStorage) IncrementRequestCount(id string) error {
	now := time.Now()
	return s.db.Model(&models.CodexAccount{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"request_count": gorm.Expr("request_count + 1"),
			"last_used_at":  now,
			"updated_at":    now,
		}).Error
}

// DeleteAccount removes a Codex account
func (s *CodexStorage) DeleteAccount(id string) error {
	res := s.db.Where("id = ?", id).Delete(&models.CodexAccount{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
