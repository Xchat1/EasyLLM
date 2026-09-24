package storage

import (
	"easyllm/internal/models"
	"time"

	"gorm.io/gorm"
)

type AntigravityStorage struct {
	db *gorm.DB
}

func NewAntigravityStorage(db *gorm.DB) *AntigravityStorage {
	return &AntigravityStorage{db: db}
}

func (s *AntigravityStorage) Save(account *models.AntigravityAccount) error {
	account.UpdatedAt = time.Now()
	if account.CreatedAt.IsZero() {
		account.CreatedAt = account.UpdatedAt
	}
	return s.db.Save(account).Error
}

func (s *AntigravityStorage) List() ([]models.AntigravityAccount, error) {
	var list []models.AntigravityAccount
	return list, s.db.Order("active desc, created_at desc").Find(&list).Error
}

func (s *AntigravityStorage) Get(id string) (*models.AntigravityAccount, error) {
	var a models.AntigravityAccount
	if err := s.db.Where("id = ?", id).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *AntigravityStorage) GetByEmail(email string) (*models.AntigravityAccount, error) {
	var a models.AntigravityAccount
	if err := s.db.Where("email = ?", email).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *AntigravityStorage) GetActive() (*models.AntigravityAccount, error) {
	var a models.AntigravityAccount
	if err := s.db.Where("active = ?", true).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *AntigravityStorage) Delete(id string) error {
	res := s.db.Where("id = ?", id).Delete(&models.AntigravityAccount{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *AntigravityStorage) DeleteMany(ids []string) error {
	return s.db.Where("id IN ?", ids).Delete(&models.AntigravityAccount{}).Error
}

// SetActive sets one account as active, clearing active from all other accounts.
func (s *AntigravityStorage) SetActive(id string) error {
	var account models.AntigravityAccount
	if err := s.db.Select("id").Where("id = ?", id).First(&account).Error; err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.AntigravityAccount{}).Where("1 = 1").Update("active", false).Error; err != nil {
			return err
		}
		return tx.Model(&models.AntigravityAccount{}).Where("id = ?", id).Update("active", true).Error
	})
}
