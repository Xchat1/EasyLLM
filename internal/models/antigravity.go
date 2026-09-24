package models

import "time"

// AntigravityQuotaModel represents the quota status for a specific model or bucket
type AntigravityQuotaModel struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`
	Percentage  int    `json:"percentage"`
	ResetTime   string `json:"reset_time,omitempty"`
}

// AntigravityCreditInfo represents credit details for paid tiers
type AntigravityCreditInfo struct {
	CreditType                  string  `json:"credit_type"`
	CreditAmount                *string `json:"credit_amount,omitempty"`
	MinimumCreditAmountForUsage *string `json:"minimum_credit_amount_for_usage,omitempty"`
}

// AntigravityAccount represents an Antigravity IDE account
type AntigravityAccount struct {
	ID               string     `json:"id" gorm:"primaryKey"`
	Email            string     `json:"email" gorm:"index"`
	DisplayName      *string    `json:"display_name,omitempty"`
	Picture          *string    `json:"picture,omitempty"`
	AccessToken      string     `json:"access_token"`
	RefreshToken     string     `json:"refresh_token"`
	IDToken          *string    `json:"id_token,omitempty"`
	ExpiresIn        int64      `json:"expires_in"`
	ExpiryTimestamp  int64      `json:"expiry_timestamp"`
	ProjectID        *string    `json:"project_id,omitempty"`
	SessionID        *string    `json:"session_id,omitempty"`
	IsGcpTos         bool       `json:"is_gcp_tos"`
	SubscriptionTier *string    `json:"subscription_tier,omitempty"`
	CreditsJSON      *string    `json:"credits_json,omitempty" gorm:"type:text"`
	QuotaModelsJSON  *string    `json:"quota_models_json,omitempty" gorm:"type:text"`
	Active           bool       `json:"active" gorm:"default:false;index"`
	Status           string     `json:"status" gorm:"default:'active'"` // active, forbidden, expired, error
	StatusMessage    *string    `json:"status_message,omitempty"`
	TagName          *string    `json:"tag_name,omitempty"`
	TagColor         *string    `json:"tag_color,omitempty"`
	Notes            *string    `json:"notes,omitempty" gorm:"type:text"`
	LastRefreshAt    *time.Time `json:"last_refresh_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// AntigravityWakeupRequest defines parameters for a wakeup ping/test
type AntigravityWakeupRequest struct {
	Model           string `json:"model,omitempty"`
	Prompt          string `json:"prompt,omitempty"`
	MaxOutputTokens uint32 `json:"max_output_tokens,omitempty"`
}

// AntigravityWakeupResponse defines output from a wakeup execution
type AntigravityWakeupResponse struct {
	Reply            string  `json:"reply"`
	PromptTokens     *uint32 `json:"prompt_tokens,omitempty"`
	CompletionTokens *uint32 `json:"completion_tokens,omitempty"`
	TotalTokens      *uint32 `json:"total_tokens,omitempty"`
	TraceID          *string `json:"trace_id,omitempty"`
	ResponseID       *string `json:"response_id,omitempty"`
	DurationMS       int64   `json:"duration_ms"`
}

// AntigravitySummaryResponse represents a lightweight quota summary for menu bar / status items
type AntigravitySummaryResponse struct {
	HasAccount                 bool   `json:"has_account"`
	AccountID                  string `json:"account_id,omitempty"`
	Email                      string `json:"email,omitempty"`
	DisplayName                string `json:"display_name,omitempty"`
	SubscriptionTier           string `json:"subscription_tier,omitempty"`
	Status                     string `json:"status,omitempty"`
	Claude5h                   *int   `json:"claude_5h,omitempty"`
	Claude5hResetFormatted     string `json:"claude_5h_reset_formatted,omitempty"`
	ClaudeWeekly               *int   `json:"claude_weekly,omitempty"`
	ClaudeWeeklyResetFormatted string `json:"claude_weekly_reset_formatted,omitempty"`
	Gemini5h                   *int   `json:"gemini_5h,omitempty"`
	Gemini5hResetFormatted     string `json:"gemini_5h_reset_formatted,omitempty"`
	GeminiWeekly               *int   `json:"gemini_weekly,omitempty"`
	GeminiWeeklyResetFormatted string `json:"gemini_weekly_reset_formatted,omitempty"`
	StatusText                 string `json:"status_text"`
	TooltipText                string `json:"tooltip_text"`
	LastRefreshAt              string `json:"last_refresh_at,omitempty"`
}
