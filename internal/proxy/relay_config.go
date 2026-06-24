package proxy

import (
	"encoding/json"
	"easyllm/internal/storage"
	"strconv"
	"strings"
)

const (
	settingRelayUpstreamURL      = "relay_upstream_url"
	settingRelayAPIKey           = "relay_api_key"
	settingRelayAuthHeader       = "relay_auth_header"
	settingRelayAuthPrefix       = "relay_auth_value_prefix"
	settingRelayDefaultModel     = "relay_default_model"
	settingRelayModelMap         = "relay_model_map"
	settingRelayToolDenylist     = "relay_tool_denylist"
	settingRelayMaxSessions      = "relay_max_sessions"
	settingRelayMaxSessionBytes  = "relay_max_session_bytes"
	settingRelaySessionTTLHours  = "relay_session_ttl_hours"
	settingRelayDiskCacheDir     = "relay_disk_cache_dir"
	settingRelayUpstreams        = "relay_upstreams"
	settingRelayUpstreamStrategy = "relay_upstream_strategy"
)

// LoadRelayConfigFromSettings loads relay config from the settings table.
func LoadRelayConfigFromSettings() *RelayConfig {
	config := DefaultRelayConfig()
	settings := storage.GetAllSettings()

	// ── Multi-upstream pool ──────────────────────────────────
	if v, ok := settings[settingRelayUpstreams]; ok && v != "" {
		var upstreams []RelayUpstream
		if err := json.Unmarshal([]byte(v), &upstreams); err == nil {
			config.Upstreams = upstreams
		}
	}
	if v, ok := settings[settingRelayUpstreamStrategy]; ok && v != "" {
		config.UpstreamStrategy = v
	}

	// ── Legacy single-upstream fields (fallback / migration) ─
	if v, ok := settings[settingRelayUpstreamURL]; ok && v != "" {
		config.UpstreamURL = v
		// Auto-migrate: if the pool is empty and a legacy URL exists, seed it.
		if len(config.Upstreams) == 0 {
			apiKey, _ := settings[settingRelayAPIKey]
			authHeader, _ := settings[settingRelayAuthHeader]
			authPrefix, _ := settings[settingRelayAuthPrefix]
			config.Upstreams = []RelayUpstream{
				{
					ID:              "default",
					Name:            "默认",
					Enabled:         true,
					UpstreamURL:     v,
					APIKey:          apiKey,
					AuthHeader:      authHeader,
					AuthValuePrefix: authPrefix,
				},
			}
		}
	}
	if v, ok := settings[settingRelayAPIKey]; ok {
		config.APIKey = v
	}
	if v, ok := settings[settingRelayAuthHeader]; ok {
		config.AuthHeader = v
	}
	if v, ok := settings[settingRelayAuthPrefix]; ok {
		config.AuthValuePrefix = v
	}

	// ── Global options ───────────────────────────────────────
	if v, ok := settings[settingRelayDefaultModel]; ok {
		config.DefaultModel = v
	}
	if v, ok := settings[settingRelayModelMap]; ok && v != "" {
		config.ModelMapJSON = v
		config.ModelMap = ParseModelMap(v)
	}
	if v, ok := settings[settingRelayToolDenylist]; ok && v != "" {
		config.ToolDenylistStr = v
		config.ToolDenylist = ParseToolDenylist(v)
	}
	if v, ok := settings[settingRelayMaxSessions]; ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			config.MaxSessions = n
		}
	}
	if v, ok := settings[settingRelayMaxSessionBytes]; ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			config.MaxSessionBytes = n
		}
	}
	if v, ok := settings[settingRelaySessionTTLHours]; ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			config.SessionTTLHours = n
		}
	}
	if v, ok := settings[settingRelayDiskCacheDir]; ok {
		config.DiskCacheDir = v
	}
	return config
}

func saveRelayConfigToSettings(config *RelayConfig) {
	if config == nil {
		return
	}

	// ── Multi-upstream pool ──────────────────────────────────
	if config.Upstreams != nil {
		if b, err := json.Marshal(config.Upstreams); err == nil {
			_ = storage.SaveSetting(settingRelayUpstreams, string(b))
		}
	}
	_ = storage.SaveSetting(settingRelayUpstreamStrategy, config.UpstreamStrategy)

	// ── Legacy single-upstream fields ─────────────────────────
	_ = storage.SaveSetting(settingRelayUpstreamURL, config.UpstreamURL)
	_ = storage.SaveSetting(settingRelayAPIKey, config.APIKey)
	_ = storage.SaveSetting(settingRelayAuthHeader, config.AuthHeader)
	_ = storage.SaveSetting(settingRelayAuthPrefix, config.AuthValuePrefix)

	// ── Global options ───────────────────────────────────────
	_ = storage.SaveSetting(settingRelayDefaultModel, config.DefaultModel)

	modelMapJSON := config.ModelMapJSON
	if modelMapJSON == "" && len(config.ModelMap) > 0 {
		if b, err := json.Marshal(config.ModelMap); err == nil {
			modelMapJSON = string(b)
		}
	}
	_ = storage.SaveSetting(settingRelayModelMap, modelMapJSON)

	toolDenylistStr := config.ToolDenylistStr
	if toolDenylistStr == "" && len(config.ToolDenylist) > 0 {
		var names []string
		for name := range config.ToolDenylist {
			names = append(names, name)
		}
		toolDenylistStr = strings.Join(names, ",")
	}
	_ = storage.SaveSetting(settingRelayToolDenylist, toolDenylistStr)

	_ = storage.SaveSetting(settingRelayMaxSessions, strconv.Itoa(config.MaxSessions))
	_ = storage.SaveSetting(settingRelayMaxSessionBytes, strconv.Itoa(config.MaxSessionBytes))
	_ = storage.SaveSetting(settingRelaySessionTTLHours, strconv.Itoa(config.SessionTTLHours))
	_ = storage.SaveSetting(settingRelayDiskCacheDir, config.DiskCacheDir)
}
