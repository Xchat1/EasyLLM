package storage

import (
	"errors"
	"testing"

	"easyllm/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newAntigravityTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.AntigravityAccount{}); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}
	return db
}

func TestAntigravityStorageCRUD(t *testing.T) {
	db := newAntigravityTestDB(t)
	store := NewAntigravityStorage(db)

	// 1. Delete on missing returns ErrRecordNotFound
	err := store.Delete("missing")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}

	// 2. Save
	acc := &models.AntigravityAccount{
		ID:           "test-id-1",
		Email:        "user1@example.com",
		AccessToken:  "token-1",
		RefreshToken: "refresh-1",
		Status:       "active",
	}
	if err := store.Save(acc); err != nil {
		t.Fatalf("save account failed: %v", err)
	}

	// 3. Get
	fetched, err := store.Get("test-id-1")
	if err != nil {
		t.Fatalf("get account failed: %v", err)
	}
	if fetched.Email != "user1@example.com" {
		t.Fatalf("expected email user1@example.com, got %s", fetched.Email)
	}

	// 4. SetActive
	acc2 := &models.AntigravityAccount{
		ID:           "test-id-2",
		Email:        "user2@example.com",
		AccessToken:  "token-2",
		RefreshToken: "refresh-2",
		Status:       "active",
	}
	_ = store.Save(acc2)

	if err := store.SetActive("test-id-2"); err != nil {
		t.Fatalf("set active failed: %v", err)
	}

	active, err := store.GetActive()
	if err != nil {
		t.Fatalf("get active failed: %v", err)
	}
	if active.ID != "test-id-2" {
		t.Fatalf("expected active test-id-2, got %s", active.ID)
	}

	// Verify test-id-1 is no longer active
	fetched1, _ := store.Get("test-id-1")
	if fetched1.Active {
		t.Fatalf("expected test-id-1 active=false, got true")
	}

	// 5. Delete
	if err := store.Delete("test-id-1"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	list, err := store.List()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 account, got %d", len(list))
	}
}
