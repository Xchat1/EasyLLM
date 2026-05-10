package models

import "time"

const AppVersion = "2.0.0"
const AppGitRepo = "https://github.com/libaxuan/EasyLLM"

// AugmentToken 占位模型，用于 GORM 迁移与既有 augment 表兼容（字段随 AutoMigrate 补齐）.
type AugmentToken struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AntigravityAccount represents an Antigravity account
type AntigravityAccount struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	Email       string    `json:"email"`
	AccessToken string    `json:"access_token"`
	Name        *string   `json:"name,omitempty"`
	Active      bool      `json:"active" gorm:"default:false"`
	Plan        *string   `json:"plan,omitempty"`
	Quota       *int64    `json:"quota,omitempty"`
	UsedQuota   *int64    `json:"used_quota,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	TagName     *string   `json:"tag_name,omitempty"`
	TagColor    *string   `json:"tag_color,omitempty"`
}

// ClaudeAccount represents a Claude/Anthropic account
type ClaudeAccount struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	Email      string    `json:"email"`
	SessionKey string    `json:"session_key"`
	Name       *string   `json:"name,omitempty"`
	Plan       *string   `json:"plan,omitempty"`
	Active     bool      `json:"active" gorm:"default:false"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	TagName    *string   `json:"tag_name,omitempty"`
	TagColor   *string   `json:"tag_color,omitempty"`
}

// AppSettings stores application settings in the database
type AppSettings struct {
	Key       string    `json:"key" gorm:"primaryKey"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// HealthResponse for health check endpoint
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Port    int    `json:"port"`
}

// APIError standard API error response
type APIError struct {
	Error   string  `json:"error"`
	Code    string  `json:"code"`
	Details *string `json:"details,omitempty"`
}

// PagedResult for paginated responses
type PagedResult[T any] struct {
	Items   []T   `json:"items"`
	Total   int64 `json:"total"`
	Page    int   `json:"page"`
	PerPage int   `json:"per_page"`
}
