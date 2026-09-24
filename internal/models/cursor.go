package models

import "time"

// CursorAccount represents a Cursor IDE account with usage and quota info
type CursorAccount struct {
	ID                 string     `json:"id" gorm:"primaryKey"`
	Email              string     `json:"email" gorm:"index"`
	DisplayName        *string    `json:"display_name,omitempty"`
	Picture            *string    `json:"picture,omitempty"`
	SessionToken       string     `json:"session_token"`
	AccessToken        string     `json:"access_token"`
	RefreshToken       string     `json:"refresh_token,omitempty"`
	MembershipType     string     `json:"membership_type" gorm:"default:'free'"` // free, pro, business, enterprise
	SubscriptionStatus string     `json:"subscription_status" gorm:"default:'active'"` // active, canceled, etc.
	BillingCycleStart  string     `json:"billing_cycle_start,omitempty"`
	BillingCycleEnd    string     `json:"billing_cycle_end,omitempty"`

	// Fast / Included Plan Requests
	PlanEnabled        bool       `json:"plan_enabled"`
	PlanUsed           int        `json:"plan_used"`
	PlanLimit          int        `json:"plan_limit"`
	PlanRemaining      int        `json:"plan_remaining"`
	PlanPercentage     int        `json:"plan_percentage"` // remaining % (0-100)

	// On-Demand Spending (in cents)
	OnDemandEnabled    bool       `json:"on_demand_enabled"`
	OnDemandUsedCents  int        `json:"on_demand_used_cents"`
	OnDemandLimitCents *int       `json:"on_demand_limit_cents,omitempty"`

	// Sub metrics
	AutoPercentUsed    int        `json:"auto_percent_used"`
	ApiPercentUsed     int        `json:"api_percent_used"`
	TotalPercentUsed   int        `json:"total_percent_used"`

	RawUsageJSON       *string    `json:"raw_usage_json,omitempty" gorm:"type:text"`
	Active             bool       `json:"active" gorm:"default:false;index"`
	Status             string     `json:"status" gorm:"default:'active'"` // active, forbidden, expired, error
	StatusMessage      *string    `json:"status_message,omitempty"`
	TagName            *string    `json:"tag_name,omitempty"`
	TagColor           *string    `json:"tag_color,omitempty"`
	Notes              *string    `json:"notes,omitempty" gorm:"type:text"`
	LastRefreshAt      *time.Time `json:"last_refresh_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// CursorWakeupResponse defines output from a Cursor wakeup test
type CursorWakeupResponse struct {
	Success            bool   `json:"success"`
	Email              string `json:"email"`
	MembershipType     string `json:"membership_type"`
	SubscriptionStatus string `json:"subscription_status"`
	PlanRemaining      int    `json:"plan_remaining"`
	PlanLimit          int    `json:"plan_limit"`
	DurationMS         int64  `json:"duration_ms"`
	Message            string `json:"message"`
}
