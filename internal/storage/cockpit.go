package storage

import (
	"easyllm/internal/models"
	"time"

	"gorm.io/gorm"
)

type CockpitStorage struct {
	db *gorm.DB
}

func NewCockpitStorage(db *gorm.DB) *CockpitStorage {
	return &CockpitStorage{db: db}
}

func (s *CockpitStorage) SaveAccount(account *models.PlatformAccount) error {
	account.UpdatedAt = time.Now()
	if account.CreatedAt.IsZero() {
		account.CreatedAt = time.Now()
	}
	return s.db.Save(account).Error
}

func (s *CockpitStorage) ListAccounts(platform string) ([]models.PlatformAccount, error) {
	var accounts []models.PlatformAccount
	query := s.db.Order("active desc, updated_at desc")
	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	return accounts, query.Find(&accounts).Error
}

func (s *CockpitStorage) GetAccount(id string) (*models.PlatformAccount, error) {
	var account models.PlatformAccount
	if err := s.db.Where("id = ?", id).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (s *CockpitStorage) DeleteAccount(platform, id string) error {
	res := s.db.Where("platform = ? AND id = ?", platform, id).Delete(&models.PlatformAccount{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *CockpitStorage) DeleteManyAccounts(platform string, ids []string) error {
	return s.db.Where("platform = ? AND id IN ?", platform, ids).Delete(&models.PlatformAccount{}).Error
}

func (s *CockpitStorage) SetActiveAccount(platform, id string) error {
	var account models.PlatformAccount
	if err := s.db.Select("id").Where("platform = ? AND id = ?", platform, id).First(&account).Error; err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.PlatformAccount{}).
			Where("platform = ?", platform).
			Update("active", false).Error; err != nil {
			return err
		}
		res := tx.Model(&models.PlatformAccount{}).
			Where("platform = ? AND id = ?", platform, id).
			Update("active", true)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (s *CockpitStorage) CountAccounts(platform string) (int64, error) {
	var count int64
	query := s.db.Model(&models.PlatformAccount{})
	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	return count, query.Count(&count).Error
}

func (s *CockpitStorage) CountActiveAccounts(platform string) (int64, error) {
	var count int64
	query := s.db.Model(&models.PlatformAccount{}).Where("active = ?", true)
	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	return count, query.Count(&count).Error
}

func (s *CockpitStorage) GetActiveAccount(platform string) (*models.PlatformAccount, error) {
	var account models.PlatformAccount
	if err := s.db.Where("platform = ? AND active = ?", platform, true).First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func (s *CockpitStorage) SaveInstance(instance *models.PlatformInstance) error {
	instance.UpdatedAt = time.Now()
	if instance.CreatedAt.IsZero() {
		instance.CreatedAt = time.Now()
	}
	return s.db.Save(instance).Error
}

func (s *CockpitStorage) ListInstances(platform string) ([]models.PlatformInstance, error) {
	var instances []models.PlatformInstance
	query := s.db.Order("updated_at desc")
	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	return instances, query.Find(&instances).Error
}

func (s *CockpitStorage) GetInstance(id string) (*models.PlatformInstance, error) {
	var instance models.PlatformInstance
	if err := s.db.Where("id = ?", id).First(&instance).Error; err != nil {
		return nil, err
	}
	return &instance, nil
}

func (s *CockpitStorage) DeleteInstance(platform, id string) error {
	res := s.db.Where("platform = ? AND id = ?", platform, id).Delete(&models.PlatformInstance{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *CockpitStorage) SetInstanceState(platform, id, state string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"state":      state,
		"updated_at": now,
	}
	if state == "running" {
		updates["last_started_at"] = now
	} else {
		updates["last_stopped_at"] = now
	}
	res := s.db.Model(&models.PlatformInstance{}).
		Where("platform = ? AND id = ?", platform, id).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *CockpitStorage) CountInstances(platform string, runningOnly bool) (int64, error) {
	var count int64
	query := s.db.Model(&models.PlatformInstance{})
	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	if runningOnly {
		query = query.Where("state = ?", "running")
	}
	return count, query.Count(&count).Error
}

func (s *CockpitStorage) SaveWakeupTask(task *models.WakeupTask) error {
	task.UpdatedAt = time.Now()
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}
	return s.db.Save(task).Error
}

func (s *CockpitStorage) ListWakeupTasks(platform string) ([]models.WakeupTask, error) {
	var tasks []models.WakeupTask
	query := s.db.Order("enabled desc, updated_at desc")
	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	return tasks, query.Find(&tasks).Error
}

func (s *CockpitStorage) GetWakeupTask(id string) (*models.WakeupTask, error) {
	var task models.WakeupTask
	if err := s.db.Where("id = ?", id).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *CockpitStorage) DeleteWakeupTask(id string) error {
	res := s.db.Where("id = ?", id).Delete(&models.WakeupTask{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *CockpitStorage) ToggleWakeupTask(id string) (bool, error) {
	task, err := s.GetWakeupTask(id)
	if err != nil {
		return false, err
	}
	task.Enabled = !task.Enabled
	task.UpdatedAt = time.Now()
	return task.Enabled, s.db.Save(task).Error
}

func (s *CockpitStorage) CountWakeupTasks(platform string, enabledOnly bool) (int64, error) {
	var count int64
	query := s.db.Model(&models.WakeupTask{})
	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	if enabledOnly {
		query = query.Where("enabled = ?", true)
	}
	return count, query.Count(&count).Error
}

func (s *CockpitStorage) MigrateLegacyPlatformAccounts() error {
	return s.migrateAntigravityAccounts()
}

func (s *CockpitStorage) migrateAntigravityAccounts() error {
	if !s.db.Migrator().HasTable(&models.AntigravityAccount{}) {
		return nil
	}
	var existing int64
	if err := s.db.Model(&models.PlatformAccount{}).Where("platform = ?", "antigravity").Count(&existing).Error; err != nil {
		return err
	}
	if existing > 0 {
		return nil
	}

	var legacy []models.AntigravityAccount
	if err := s.db.Order("created_at desc").Find(&legacy).Error; err != nil {
		return err
	}
	for _, item := range legacy {
		accessToken := item.AccessToken
		account := models.PlatformAccount{
			ID:          item.ID,
			Platform:    "antigravity",
			Email:       item.Email,
			DisplayName: item.Name,
			AccessToken: &accessToken,
			Plan:        item.Plan,
			Status:      "active",
			Active:      item.Active,
			TagName:     item.TagName,
			TagColor:    item.TagColor,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		}
		if item.Quota != nil {
			quotaLimit := float64(*item.Quota)
			account.QuotaLimit = &quotaLimit
			quotaUnit := "credits"
			account.QuotaUnit = &quotaUnit
		}
		if item.UsedQuota != nil {
			quotaUsed := float64(*item.UsedQuota)
			account.QuotaUsed = &quotaUsed
		}
		if err := s.db.Save(&account).Error; err != nil {
			return err
		}
	}
	return nil
}
