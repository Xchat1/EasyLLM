package storage

import "gorm.io/gorm"

// AugmentStorage 占位：完整 Augment 功能见上游仓库；此处仅满足编译与路由注册。
type AugmentStorage struct {
	db      *gorm.DB
	dataDir string
}

// NewAugmentStorage creates augment storage (stub).
func NewAugmentStorage(db *gorm.DB, dataDir string) *AugmentStorage {
	return &AugmentStorage{db: db, dataDir: dataDir}
}
