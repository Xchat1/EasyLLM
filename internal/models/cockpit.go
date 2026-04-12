package models

import "time"

type PlatformSupport struct {
	MultiAccount   bool `json:"multi_account"`
	Instances      bool `json:"instances"`
	Wakeup         bool `json:"wakeup"`
	QuotaTracking  bool `json:"quota_tracking"`
	LaunchProfiles bool `json:"launch_profiles"`
}

type PlatformDefinition struct {
	ID             string          `json:"id"`
	Label          string          `json:"label"`
	Icon           string          `json:"icon"`
	Description    string          `json:"description"`
	Category       string          `json:"category"`
	ManagementMode string          `json:"management_mode"`
	Supports       PlatformSupport `json:"supports"`
}

var cockpitPlatformDefinitions = []PlatformDefinition{
	{
		ID:             "antigravity",
		Label:          "Antigravity",
		Icon:           "🚀",
		Description:    "多账号管理、实例编排与唤醒任务的核心平台。",
		Category:       "workspace",
		ManagementMode: "generic",
		Supports: PlatformSupport{
			MultiAccount:   true,
			Instances:      true,
			Wakeup:         true,
			QuotaTracking:  true,
			LaunchProfiles: true,
		},
	},
	{
		ID:             "codex",
		Label:          "Codex",
		Icon:           "🤖",
		Description:    "复用 EasyLLM 现有 OpenAI / Codex 代理池与账号切换能力。",
		Category:       "workspace",
		ManagementMode: "legacy",
		Supports: PlatformSupport{
			MultiAccount:   true,
			Instances:      true,
			Wakeup:         true,
			QuotaTracking:  true,
			LaunchProfiles: true,
		},
	},
	{
		ID:             "github-copilot",
		Label:          "GitHub Copilot",
		Icon:           "🐙",
		Description:    "管理 Copilot 账号、配额视图与本地实例配置。",
		Category:       "ide",
		ManagementMode: "generic",
		Supports: PlatformSupport{
			MultiAccount:   true,
			Instances:      true,
			Wakeup:         false,
			QuotaTracking:  true,
			LaunchProfiles: true,
		},
	},
	{
		ID:             "windsurf",
		Label:          "Windsurf",
		Icon:           "🌊",
		Description:    "围绕账号切换、多开实例和启动路径配置进行管理。",
		Category:       "ide",
		ManagementMode: "generic",
		Supports: PlatformSupport{
			MultiAccount:   true,
			Instances:      true,
			Wakeup:         false,
			QuotaTracking:  true,
			LaunchProfiles: true,
		},
	},
	{
		ID:             "kiro",
		Label:          "Kiro",
		Icon:           "🪐",
		Description:    "对齐 cockpit-tools 的账号、实例与刷新策略配置。",
		Category:       "ide",
		ManagementMode: "generic",
		Supports: PlatformSupport{
			MultiAccount:   true,
			Instances:      true,
			Wakeup:         false,
			QuotaTracking:  true,
			LaunchProfiles: true,
		},
	},
	{
		ID:             "cursor",
		Label:          "Cursor",
		Icon:           "💻",
		Description:    "统一账号、实例和路径配置，兼容 EasyLLM 原有数据。",
		Category:       "ide",
		ManagementMode: "generic",
		Supports: PlatformSupport{
			MultiAccount:   true,
			Instances:      true,
			Wakeup:         false,
			QuotaTracking:  true,
			LaunchProfiles: true,
		},
	},
	{
		ID:             "gemini",
		Label:          "Gemini CLI",
		Icon:           "✨",
		Description:    "管理 Gemini CLI 账号、启动路径和刷新节奏。",
		Category:       "cli",
		ManagementMode: "generic",
		Supports: PlatformSupport{
			MultiAccount:   true,
			Instances:      false,
			Wakeup:         false,
			QuotaTracking:  true,
			LaunchProfiles: false,
		},
	},
	{
		ID:             "codebuddy",
		Label:          "CodeBuddy",
		Icon:           "🧩",
		Description:    "适配 CodeBuddy 系列账号台账、实例和路径管理。",
		Category:       "ide",
		ManagementMode: "generic",
		Supports: PlatformSupport{
			MultiAccount:   true,
			Instances:      true,
			Wakeup:         false,
			QuotaTracking:  true,
			LaunchProfiles: true,
		},
	},
	{
		ID:             "codebuddy-cn",
		Label:          "CodeBuddy CN",
		Icon:           "🀄",
		Description:    "面向国内客户端的账号、实例和配置同步视图。",
		Category:       "ide",
		ManagementMode: "generic",
		Supports: PlatformSupport{
			MultiAccount:   true,
			Instances:      true,
			Wakeup:         false,
			QuotaTracking:  true,
			LaunchProfiles: true,
		},
	},
	{
		ID:             "qoder",
		Label:          "Qoder",
		Icon:           "📐",
		Description:    "支持多账号、多实例与额度记录的统一编排视图。",
		Category:       "ide",
		ManagementMode: "generic",
		Supports: PlatformSupport{
			MultiAccount:   true,
			Instances:      true,
			Wakeup:         false,
			QuotaTracking:  true,
			LaunchProfiles: true,
		},
	},
	{
		ID:             "trae",
		Label:          "Trae",
		Icon:           "🛤️",
		Description:    "提供账户面板、额度备注和实例生命周期规划。",
		Category:       "ide",
		ManagementMode: "generic",
		Supports: PlatformSupport{
			MultiAccount:   true,
			Instances:      true,
			Wakeup:         false,
			QuotaTracking:  true,
			LaunchProfiles: true,
		},
	},
	{
		ID:             "zed",
		Label:          "Zed",
		Icon:           "⚡",
		Description:    "对齐 cockpit-tools 的 Zed 账号台账、路径和额度预警配置。",
		Category:       "editor",
		ManagementMode: "generic",
		Supports: PlatformSupport{
			MultiAccount:   true,
			Instances:      false,
			Wakeup:         false,
			QuotaTracking:  true,
			LaunchProfiles: false,
		},
	},
	{
		ID:             "workbuddy",
		Label:          "Workbuddy",
		Icon:           "🧠",
		Description:    "预留与 cockpit-tools 一致的平台布局与账号视图。",
		Category:       "ide",
		ManagementMode: "generic",
		Supports: PlatformSupport{
			MultiAccount:   true,
			Instances:      true,
			Wakeup:         false,
			QuotaTracking:  true,
			LaunchProfiles: true,
		},
	},
}

func GetCockpitPlatformDefinitions() []PlatformDefinition {
	defs := make([]PlatformDefinition, len(cockpitPlatformDefinitions))
	copy(defs, cockpitPlatformDefinitions)
	return defs
}

func IsSupportedCockpitPlatform(id string) bool {
	for _, def := range cockpitPlatformDefinitions {
		if def.ID == id {
			return true
		}
	}
	return false
}

type PlatformAccount struct {
	ID           string     `json:"id" gorm:"primaryKey"`
	Platform     string     `json:"platform" gorm:"index"`
	Email        string     `json:"email"`
	DisplayName  *string    `json:"display_name,omitempty"`
	AccessToken  *string    `json:"access_token,omitempty"`
	RefreshToken *string    `json:"refresh_token,omitempty"`
	CookieToken  *string    `json:"cookie_token,omitempty"`
	Plan         *string    `json:"plan,omitempty"`
	Status       string     `json:"status" gorm:"default:'active'"`
	Active       bool       `json:"active" gorm:"default:false;index"`
	TagName      *string    `json:"tag_name,omitempty"`
	TagColor     *string    `json:"tag_color,omitempty"`
	QuotaUsed    *float64   `json:"quota_used,omitempty"`
	QuotaLimit   *float64   `json:"quota_limit,omitempty"`
	QuotaUnit    *string    `json:"quota_unit,omitempty"`
	QuotaResetAt *time.Time `json:"quota_reset_at,omitempty"`
	Notes        *string    `json:"notes,omitempty" gorm:"type:text"`
	MetadataJSON *string    `json:"metadata_json,omitempty" gorm:"type:text"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type PlatformInstance struct {
	ID            string     `json:"id" gorm:"primaryKey"`
	Platform      string     `json:"platform" gorm:"index"`
	Name          string     `json:"name"`
	AccountID     *string    `json:"account_id,omitempty" gorm:"index"`
	WorkspaceDir  *string    `json:"workspace_dir,omitempty"`
	UserDataDir   *string    `json:"user_data_dir,omitempty"`
	LaunchArgs    *string    `json:"launch_args,omitempty" gorm:"type:text"`
	State         string     `json:"state" gorm:"default:'stopped'"`
	PID           *int       `json:"pid,omitempty"`
	AutoStart     bool       `json:"auto_start" gorm:"default:false"`
	Notes         *string    `json:"notes,omitempty" gorm:"type:text"`
	LastStartedAt *time.Time `json:"last_started_at,omitempty"`
	LastStoppedAt *time.Time `json:"last_stopped_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type WakeupTask struct {
	ID            string     `json:"id" gorm:"primaryKey"`
	Platform      string     `json:"platform" gorm:"index"`
	Name          string     `json:"name"`
	AccountID     *string    `json:"account_id,omitempty" gorm:"index"`
	Model         *string    `json:"model,omitempty"`
	Prompt        *string    `json:"prompt,omitempty" gorm:"type:text"`
	ScheduleType  string     `json:"schedule_type" gorm:"default:'daily'"`
	ScheduleValue string     `json:"schedule_value"`
	Enabled       bool       `json:"enabled" gorm:"default:true"`
	LastRunAt     *time.Time `json:"last_run_at,omitempty"`
	NextRunAt     *time.Time `json:"next_run_at,omitempty"`
	LastStatus    *string    `json:"last_status,omitempty"`
	LastMessage   *string    `json:"last_message,omitempty" gorm:"type:text"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type CockpitGeneralSettings struct {
	Language           string            `json:"language"`
	Theme              string            `json:"theme"`
	CloseBehavior      string            `json:"close_behavior"`
	PrivacyMode        bool              `json:"privacy_mode"`
	AutoRefreshMinutes int               `json:"auto_refresh_minutes"`
	RefreshIntervals   map[string]int    `json:"refresh_intervals"`
	AppPaths           map[string]string `json:"app_paths"`
	Integrations       map[string]bool   `json:"integrations"`
}

func DefaultCockpitGeneralSettings() CockpitGeneralSettings {
	refresh := map[string]int{
		"antigravity":    5,
		"codex":          5,
		"github-copilot": 10,
		"windsurf":       10,
		"kiro":           10,
		"cursor":         10,
		"gemini":         10,
		"codebuddy":      10,
		"codebuddy-cn":   10,
		"qoder":          10,
		"trae":           10,
		"zed":            10,
		"workbuddy":      10,
	}
	paths := map[string]string{
		"opencode":     "",
		"antigravity":  "",
		"codex":        "",
		"vscode":       "",
		"windsurf":     "",
		"kiro":         "",
		"cursor":       "",
		"gemini":       "",
		"codebuddy":    "",
		"codebuddy-cn": "",
		"qoder":        "",
		"trae":         "",
		"zed":          "",
		"workbuddy":    "",
	}
	return CockpitGeneralSettings{
		Language:           "zh-CN",
		Theme:              "system",
		CloseBehavior:      "ask",
		PrivacyMode:        false,
		AutoRefreshMinutes: 5,
		RefreshIntervals:   refresh,
		AppPaths:           paths,
		Integrations: map[string]bool{
			"codex_launch_on_switch":            true,
			"opencode_sync_on_switch":           false,
			"opencode_auth_overwrite_on_switch": false,
		},
	}
}

func NormalizeCockpitGeneralSettings(in CockpitGeneralSettings) CockpitGeneralSettings {
	normalized := DefaultCockpitGeneralSettings()
	if in.Language != "" {
		normalized.Language = in.Language
	}
	if in.Theme != "" {
		normalized.Theme = in.Theme
	}
	if in.CloseBehavior != "" {
		normalized.CloseBehavior = in.CloseBehavior
	}
	normalized.PrivacyMode = in.PrivacyMode
	if in.AutoRefreshMinutes > 0 {
		normalized.AutoRefreshMinutes = clampRefreshInterval(in.AutoRefreshMinutes)
	}
	if in.RefreshIntervals != nil {
		for key, value := range in.RefreshIntervals {
			if value > 0 {
				normalized.RefreshIntervals[key] = clampRefreshInterval(value)
			}
		}
	}
	if in.AppPaths != nil {
		for key, value := range in.AppPaths {
			normalized.AppPaths[key] = value
		}
	}
	if in.Integrations != nil {
		for key, value := range in.Integrations {
			normalized.Integrations[key] = value
		}
	}
	return normalized
}

func clampRefreshInterval(minutes int) int {
	if minutes < 1 {
		return 1
	}
	if minutes > 720 {
		return 720
	}
	return minutes
}

type CockpitSummary struct {
	TotalPlatforms     int   `json:"total_platforms"`
	EnabledPlatforms   int   `json:"enabled_platforms"`
	TotalAccounts      int64 `json:"total_accounts"`
	ActiveAccounts     int64 `json:"active_accounts"`
	TotalInstances     int64 `json:"total_instances"`
	RunningInstances   int64 `json:"running_instances"`
	TotalWakeupTasks   int64 `json:"total_wakeup_tasks"`
	EnabledWakeupTasks int64 `json:"enabled_wakeup_tasks"`
}

type CodexProxyOverview struct {
	Enabled         bool   `json:"enabled"`
	Strategy        string `json:"strategy"`
	Accounts        int    `json:"accounts"`
	EnabledAccounts int    `json:"enabled_accounts"`
	TotalRequests   int64  `json:"total_requests"`
}

type PlatformOverview struct {
	Definition         PlatformDefinition `json:"definition"`
	Accounts           int64              `json:"accounts"`
	ActiveAccounts     int64              `json:"active_accounts"`
	ActiveAccountEmail *string            `json:"active_account_email,omitempty"`
	Instances          int64              `json:"instances"`
	RunningInstances   int64              `json:"running_instances"`
	WakeupTasks        int64              `json:"wakeup_tasks"`
	EnabledWakeupTasks int64              `json:"enabled_wakeup_tasks"`
}

type CockpitOverviewResponse struct {
	Summary     CockpitSummary     `json:"summary"`
	Platforms   []PlatformOverview `json:"platforms"`
	Proxy       CodexProxyOverview `json:"proxy"`
	GeneratedAt time.Time          `json:"generated_at"`
}
