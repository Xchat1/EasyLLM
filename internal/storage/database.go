package storage

import (
	"easyllm/config"
	"easyllm/internal/models"
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB initializes the database connection and runs migrations
func InitDB(cfg *config.Config) error {
	var err error

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}
	if cfg.App.Debug {
		gormConfig.Logger = logger.Default.LogMode(logger.Info)
	}

	sqlitePath := cfg.Database.SQLitePath
	if sqlitePath == "" {
		sqlitePath = filepath.Join(cfg.App.DataDir, "easyllm.db")
	}
	if err := os.MkdirAll(filepath.Dir(sqlitePath), 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}
	DB, err = gorm.Open(sqlite.Open(sqlitePath), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to open sqlite: %w", err)
	}

	return AutoMigrate()
}

// AutoMigrate runs database migrations for all models
func AutoMigrate() error {
	if err := DB.AutoMigrate(
		&models.OpenAIAccount{},
		&models.CodexAccount{},
		&models.AppSettings{},
	); err != nil {
		return err
	}
	return PurgeNonOpenAITables()
}

// PurgeNonOpenAITables removes any table outside the current local
// OpenAI/Codex data model.
func PurgeNonOpenAITables() error {
	if DB == nil {
		return nil
	}
	allowed := map[string]bool{
		"open_ai_accounts": true,
		"codex_accounts":   true,
		"app_settings":     true,
	}
	tables, err := DB.Migrator().GetTables()
	if err != nil {
		return err
	}
	for _, table := range tables {
		if allowed[table] || table == "sqlite_sequence" {
			continue
		}
		if err := DB.Migrator().DropTable(table); err != nil {
			return fmt.Errorf("drop non-openai table %s: %w", table, err)
		}
	}
	return nil
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}

// CloseDB closes the underlying database connection
func CloseDB() error {
	if DB == nil {
		return nil
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// SaveSetting saves a key-value setting
func SaveSetting(key, value string) error {
	setting := models.AppSettings{Key: key, Value: value}
	return DB.Save(&setting).Error
}

// GetSetting retrieves a setting value by key
func GetSetting(key string) (string, bool) {
	var setting models.AppSettings
	if err := DB.Where("key = ?", key).First(&setting).Error; err != nil {
		return "", false
	}
	return setting.Value, true
}

// GetAllSettings retrieves all settings as a map
func GetAllSettings() map[string]string {
	var settings []models.AppSettings
	if err := DB.Find(&settings).Error; err != nil {
		return make(map[string]string)
	}
	result := make(map[string]string, len(settings))
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	return result
}
