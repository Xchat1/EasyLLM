package storage

import (
	"errors"
	"testing"

	"easyllm/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newCockpitTestStorage(t *testing.T) *CockpitStorage {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.PlatformAccount{}, &models.PlatformInstance{}, &models.WakeupTask{}); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}
	return NewCockpitStorage(db)
}

func TestSetActiveAccountKeepsExistingActiveWhenTargetMissing(t *testing.T) {
	store := newCockpitTestStorage(t)
	active := models.PlatformAccount{
		ID:       "existing",
		Platform: "gemini",
		Email:    "active@example.com",
		Active:   true,
		Status:   "active",
	}
	if err := store.SaveAccount(&active); err != nil {
		t.Fatalf("seed active account: %v", err)
	}

	err := store.SetActiveAccount("gemini", "missing")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}

	current, err := store.GetActiveAccount("gemini")
	if err != nil {
		t.Fatalf("load active account: %v", err)
	}
	if current.ID != active.ID {
		t.Fatalf("expected original active account to remain active, got %s", current.ID)
	}
}

func TestDeleteAccountReturnsNotFoundForMissingRecord(t *testing.T) {
	store := newCockpitTestStorage(t)
	err := store.DeleteAccount("gemini", "missing")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}
