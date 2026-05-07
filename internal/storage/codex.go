package storage

import (
	"easyllm/internal/models"
	"time"

	"gorm.io/gorm"
)

// CodexStorage handles Codex account and log CRUD
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

// SaveLog records a Codex request log
func (s *CodexStorage) SaveLog(log *models.CodexLog) error {
	return s.db.Create(log).Error
}

// GetLogs returns recent Codex logs with pagination
func (s *CodexStorage) GetLogs(limit, offset int) ([]models.CodexLog, int64, error) {
	var logs []models.CodexLog
	var total int64

	s.db.Model(&models.CodexLog{}).Count(&total)
	err := s.db.Order("created_at desc").Limit(limit).Offset(offset).Find(&logs).Error
	return logs, total, err
}

// GetLogsSince returns all logs created after since.
func (s *CodexStorage) GetLogsSince(since time.Time) ([]models.CodexLog, error) {
	var logs []models.CodexLog
	err := s.db.Where("created_at >= ?", since).Order("created_at desc").Find(&logs).Error
	return logs, err
}

// BackfillPlatform sets platform for all logs that have an empty platform field.
func (s *CodexStorage) BackfillPlatform(platform string) int64 {
	r := s.db.Model(&models.CodexLog{}).Where("platform = '' OR platform IS NULL").Update("platform", platform)
	return r.RowsAffected
}

// GetSessionLogPaths returns all request_path values that start with "session:".
// Used by SessionScanner to avoid re-importing already-seen sessions.
func (s *CodexStorage) GetSessionLogPaths() []string {
	var paths []string
	s.db.Model(&models.CodexLog{}).
		Where("request_path LIKE ?", "session:%").
		Pluck("request_path", &paths)
	return paths
}

// ClearLogs removes all Codex logs
func (s *CodexStorage) ClearLogs() error {
	return s.db.Where("1 = 1").Delete(&models.CodexLog{}).Error
}
