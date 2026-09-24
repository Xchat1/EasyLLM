package storage

import (
	"easyllm/internal/models"
	"time"

	"gorm.io/gorm"
)

type CursorStorage struct {
	db *gorm.DB
}

func NewCursorStorage(db *gorm.DB) *CursorStorage {
	return &CursorStorage{db: db}
}

func (s *CursorStorage) Save(account *models.CursorAccount) error {
	account.UpdatedAt = time.Now()
	if account.CreatedAt.IsZero() {
		account.CreatedAt = account.UpdatedAt
	}
	return s.db.Save(account).Error
}

func (s *CursorStorage) List() ([]models.CursorAccount, error) {
	var list []models.CursorAccount
	return list, s.db.Order("active desc, created_at desc").Find(&list).Error
}

func (s *CursorStorage) Get(id string) (*models.CursorAccount, error) {
	var a models.CursorAccount
	if err := s.db.Where("id = ?", id).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *CursorStorage) GetByEmail(email string) (*models.CursorAccount, error) {
	var a models.CursorAccount
	if err := s.db.Where("email = ?", email).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *CursorStorage) GetActive() (*models.CursorAccount, error) {
	var a models.CursorAccount
	if err := s.db.Where("active = ?", true).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *CursorStorage) Delete(id string) error {
	res := s.db.Where("id = ?", id).Delete(&models.CursorAccount{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *CursorStorage) DeleteMany(ids []string) error {
	return s.db.Where("id IN ?", ids).Delete(&models.CursorAccount{}).Error
}

// SetActive sets one account as active, clearing active from all other accounts.
func (s *CursorStorage) SetActive(id string) error {
	var account models.CursorAccount
	if err := s.db.Select("id").Where("id = ?", id).First(&account).Error; err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.CursorAccount{}).Where("1 = 1").Update("active", false).Error; err != nil {
			return err
		}
		return tx.Model(&models.CursorAccount{}).Where("id = ?", id).Update("active", true).Error
	})
}
