package models

import "time"

// OpenAIAccountType 账号类型
type OpenAIAccountType string

const (
	OpenAIAccountTypeOAuth OpenAIAccountType = "oauth"
	OpenAIAccountTypeAPI   OpenAIAccountType = "api"
)

// OpenAIAccount represents an OpenAI account (OAuth or API Key type)
type OpenAIAccount struct {
	ID string `json:"id" gorm:"primaryKey"`

	// Common fields
	Email       string            `json:"email"`
	AccountType OpenAIAccountType `json:"account_type" gorm:"default:'oauth'"`
	TagName     *string           `json:"tag_name,omitempty"`
	TagColor    *string           `json:"tag_color,omitempty"`
	Status      string            `json:"status" gorm:"default:'active'"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	LastUsedAt  *time.Time        `json:"last_used_at,omitempty"`

	// OAuth type fields
	AccessToken      *string    `json:"access_token,omitempty"`
	RefreshToken     *string    `json:"refresh_token,omitempty"`
	IDToken          *string    `json:"id_token,omitempty"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	ChatGPTAccountID *string    `json:"chatgpt_account_id,omitempty"`
	ChatGPTUserID    *string    `json:"chatgpt_user_id,omitempty"`
	OrganizationID   *string    `json:"organization_id,omitempty"`
	OpenAIAuthJSON   *string    `json:"openai_auth_json,omitempty"`

	// API type fields
	ModelProvider        *string `json:"model_provider,omitempty"`
	Model                *string `json:"model,omitempty"`
	ModelReasoningEffort *string `json:"model_reasoning_effort,omitempty"`
	WireAPI              *string `json:"wire_api,omitempty"` // "responses" or "chat"
	BaseURL              *string `json:"base_url,omitempty"`
	APIKey               *string `json:"api_key,omitempty"`

	// ProxyEnabled: when true the account's access_token joins the /v1/* proxy pool
	ProxyEnabled bool `json:"proxy_enabled" gorm:"default:false"`

	// IsCodexActive: true for the account currently written to ~/.codex/auth.json
	IsCodexActive bool `json:"is_codex_active" gorm:"default:false"`

	// Quota info (fetched from upstream rate-limit headers)
	QuotaUsed      *int64     `json:"quota_used,omitempty"`
	QuotaTotal     *int64     `json:"quota_total,omitempty"`
	QuotaResetAt   *string    `json:"quota_reset_at,omitempty"`
	QuotaUpdatedAt *time.Time `json:"quota_updated_at,omitempty"`
	Plan           *string    `json:"plan,omitempty"`

	// New percentage-based quota (from x-codex-* headers)
	Quota5hUsedPercent   *float64 `json:"quota_5h_used_percent,omitempty"`
	Quota5hResetSeconds  *int64   `json:"quota_5h_reset_seconds,omitempty"`
	Quota5hWindowMinutes *int64   `json:"quota_5h_window_minutes,omitempty"`
	Quota7dUsedPercent   *float64 `json:"quota_7d_used_percent,omitempty"`
	Quota7dResetSeconds  *int64   `json:"quota_7d_reset_seconds,omitempty"`
	Quota7dWindowMinutes *int64   `json:"quota_7d_window_minutes,omitempty"`
	QuotaIsForbidden     bool     `json:"quota_is_forbidden" gorm:"default:false"`
	QuotaHTTPStatus      *int     `json:"_quota_http_status,omitempty"`
	QuotaError           *string  `json:"_quota_error,omitempty"`
	QuotaVerified        bool     `json:"_verified" gorm:"default:false"`
}

// OpenAIAPIKey represents an OpenAI API key
type OpenAIAPIKey struct {
	ID         string     `json:"id" gorm:"primaryKey"`
	AccountID  string     `json:"account_id"`
	Name       string     `json:"name"`
	Key        string     `json:"key"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

// CodexAccount represents a Codex account in the pool
type CodexAccount struct {
	ID           string     `json:"id" gorm:"primaryKey"`
	AccountID    string     `json:"account_id"`
	Email        string     `json:"email"`
	AccessToken  string     `json:"access_token"`
	Enabled      bool       `json:"enabled" gorm:"default:true"`
	RequestCount int64      `json:"request_count" gorm:"default:0"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// CodexServerConfig holds Codex proxy server configuration
type CodexServerConfig struct {
	Enabled        bool   `json:"enabled"`
	Port           int    `json:"port"`
	Strategy       string `json:"strategy"` // "round_robin", "random", "least_used"
	MaxConcurrent  int    `json:"max_concurrent"`
	RequestTimeout int    `json:"request_timeout"`
}

// CodexLog represents a Codex request log entry
type CodexLog struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	AccountID    string    `json:"account_id"`
	AccountEmail string    `json:"account_email"`
	RequestPath  string    `json:"request_path"`
	Model        string    `json:"model"`
	Platform     string    `json:"platform"`
	InputTokens  int64     `json:"input_tokens"`
	OutputTokens int64     `json:"output_tokens"`
	Duration     int64     `json:"duration_ms"`
	StatusCode   int       `json:"status_code"`
	Error        *string   `json:"error,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// CodexPoolStatus represents the current status of the Codex account pool
type CodexPoolStatus struct {
	TotalAccounts   int            `json:"total_accounts"`
	EnabledAccounts int            `json:"enabled_accounts"`
	TotalRequests   int64          `json:"total_requests"`
	Accounts        []CodexAccount `json:"accounts"`
}
