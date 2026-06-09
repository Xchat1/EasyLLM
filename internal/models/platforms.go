package models

import "time"

const AppVersion = "2.0.0"
const AppGitRepo = "https://github.com/libaxuan/EasyLLM"

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
