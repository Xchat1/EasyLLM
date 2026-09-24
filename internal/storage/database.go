package storage

import (
	"easyllm/config"
	"easyllm/internal/models"
	"fmt"
	"os"
	"path/filepath"
	"time"

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

	if sqlDB, err := DB.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetConnMaxIdleTime(30 * time.Second)
		_, _ = sqlDB.Exec("PRAGMA journal_mode=WAL;")
		_, _ = sqlDB.Exec("PRAGMA busy_timeout=5000;")
		_, _ = sqlDB.Exec("PRAGMA synchronous=NORMAL;")
		_, _ = sqlDB.Exec("PRAGMA cache_size=-2000;") // 2MB cache
		_, _ = sqlDB.Exec("PRAGMA temp_store=MEMORY;")
	}

	return AutoMigrate()
}

// AutoMigrate runs database migrations for all models
func AutoMigrate() error {
	return DB.AutoMigrate(
		&models.OpenAIAccount{},
		&models.CodexAccount{},
		&models.AntigravityAccount{},
		&models.CursorAccount{},
		&models.AppSettings{},
	)
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
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}
	setting := models.AppSettings{Key: key, Value: value}
	return DB.Save(&setting).Error
}

// GetSetting retrieves a setting value by key
func GetSetting(key string) (string, bool) {
	if DB == nil {
		return "", false
	}
	var setting models.AppSettings
	if err := DB.Where("key = ?", key).First(&setting).Error; err != nil {
		return "", false
	}
	return setting.Value, true
}

// GetAllSettings retrieves all settings as a map
func GetAllSettings() map[string]string {
	if DB == nil {
		return make(map[string]string)
	}
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
