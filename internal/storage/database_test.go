package storage

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newDatabaseTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	DB = db
	return db
}

func TestAutoMigrateDropsNonOpenAITables(t *testing.T) {
	db := newDatabaseTestDB(t)
	extraTables := []string{
		"open_ai_api_keys",
		"codex_logs",
		"legacy_accounts",
		"unrelated_runtime_state",
	}

	for _, table := range extraTables {
		if err := db.Exec("CREATE TABLE " + table + " (id text primary key)").Error; err != nil {
			t.Fatalf("create extra table %s: %v", table, err)
		}
	}

	if err := AutoMigrate(); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	for _, table := range extraTables {
		if db.Migrator().HasTable(table) {
			t.Fatalf("expected non-openai table %s to be dropped", table)
		}
	}

	for _, table := range []string{"open_ai_accounts", "codex_accounts", "app_settings"} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("expected current table %s to exist", table)
		}
	}
}
