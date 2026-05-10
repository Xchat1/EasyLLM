package storage

import (
	"easyllm/config"
	"easyllm/internal/models"
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/driver/postgres"
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

	switch cfg.Database.Type {
	case "postgres":
		dsn := cfg.Database.DSN
		if dsn == "" {
			return fmt.Errorf("postgres DSN is required when DB_TYPE=postgres")
		}
		DB, err = gorm.Open(postgres.Open(dsn), gormConfig)
		if err != nil {
			return fmt.Errorf("failed to connect to postgres: %w", err)
		}
	default:
		// SQLite (default)
		sqlitePath := cfg.Database.SQLitePath
		if sqlitePath == "" {
			sqlitePath = filepath.Join(cfg.App.DataDir, "easyllm.db")
		}
		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(sqlitePath), 0755); err != nil {
			return fmt.Errorf("failed to create data directory: %w", err)
		}
		DB, err = gorm.Open(sqlite.Open(sqlitePath), gormConfig)
		if err != nil {
			return fmt.Errorf("failed to open sqlite: %w", err)
		}
	}

	return AutoMigrate()
}

// AutoMigrate runs database migrations for all models
func AutoMigrate() error {
	if err := DB.AutoMigrate(
		&models.OpenAIAccount{},
		&models.OpenAIAPIKey{},
		&models.CodexAccount{},
		&models.AntigravityAccount{},
		&models.PlatformAccount{},
		&models.PlatformInstance{},
		&models.WakeupTask{},
		&models.AppSettings{},
	); err != nil {
		return err
	}
	return PurgeCodexLogs()
}

// PurgeCodexLogs removes the legacy per-request log table. EasyLLM keeps account
// configuration and aggregate counters, but it must not retain API call logs.
func PurgeCodexLogs() error {
	if DB == nil {
		return nil
	}
	if !DB.Migrator().HasTable(&models.CodexLog{}) {
		return nil
	}
	return DB.Migrator().DropTable(&models.CodexLog{})
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
