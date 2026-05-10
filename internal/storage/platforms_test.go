package storage

import (
	"errors"
	"testing"

	"easyllm/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newPlatformTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.OpenAIAccount{}, &models.AntigravityAccount{}, &models.CodexAccount{}); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}
	return db
}

func TestOpenAIStorageDeleteReturnsNotFound(t *testing.T) {
	store := NewOpenAIStorage(newPlatformTestDB(t))
	err := store.Delete("missing")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestAntigravityStorageDeleteReturnsNotFound(t *testing.T) {
	store := NewAntigravityStorage(newPlatformTestDB(t))
	err := store.Delete("missing")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestCodexStorageDeleteReturnsNotFound(t *testing.T) {
	store := NewCodexStorage(newPlatformTestDB(t))
	err := store.DeleteAccount("missing")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}
