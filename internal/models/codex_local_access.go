package models

// CodexLocalAccessCollection describes the local Codex API service selection.
type CodexLocalAccessCollection struct {
	Enabled              bool     `json:"enabled"`
	Port                 int      `json:"port"`
	APIKeyMasked         string   `json:"api_key_masked,omitempty"`
	RoutingStrategy      string   `json:"routing_strategy"`
	RestrictFreeAccounts bool     `json:"restrict_free_accounts"`
	AccountIDs           []string `json:"account_ids"`
	CreatedAt            string   `json:"created_at,omitempty"`
	UpdatedAt            string   `json:"updated_at,omitempty"`
}

// CodexLocalAccessUsageStats is an aggregate view of local Codex API usage.
type CodexLocalAccessUsageStats struct {
	RequestCount   int64 `json:"request_count"`
	SuccessCount   int64 `json:"success_count"`
	FailureCount   int64 `json:"failure_count"`
	TotalLatencyMs int64 `json:"total_latency_ms"`
	InputTokens    int64 `json:"input_tokens"`
	OutputTokens   int64 `json:"output_tokens"`
	TotalTokens    int64 `json:"total_tokens"`
}

// CodexLocalAccessAccountStats is a per-account usage aggregate.
type CodexLocalAccessAccountStats struct {
	AccountID string                     `json:"account_id"`
	Email     string                     `json:"email"`
	Usage     CodexLocalAccessUsageStats `json:"usage"`
	UpdatedAt string                     `json:"updated_at,omitempty"`
}

// CodexLocalAccessStatsWindow is one time-bounded stats window.
type CodexLocalAccessStatsWindow struct {
	Since     string                         `json:"since"`
	UpdatedAt string                         `json:"updated_at"`
	Totals    CodexLocalAccessUsageStats     `json:"totals"`
	Accounts  []CodexLocalAccessAccountStats `json:"accounts"`
}

// CodexLocalAccessStats groups the stats windows shown in the service panel.
type CodexLocalAccessStats struct {
	Daily   CodexLocalAccessStatsWindow `json:"daily"`
	Weekly  CodexLocalAccessStatsWindow `json:"weekly"`
	Monthly CodexLocalAccessStatsWindow `json:"monthly"`
}

// CodexLocalAccessState is the full state returned to the UI.
type CodexLocalAccessState struct {
	Collection  *CodexLocalAccessCollection `json:"collection"`
	Running     bool                        `json:"running"`
	BaseURL     string                      `json:"base_url"`
	APIPortURL  string                      `json:"api_port_url"`
	ModelIDs    []string                    `json:"model_ids"`
	LastError   string                      `json:"last_error,omitempty"`
	MemberCount int                         `json:"member_count"`
	Stats       CodexLocalAccessStats       `json:"stats"`
}
