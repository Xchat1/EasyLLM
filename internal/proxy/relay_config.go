package proxy

import (
	"easyllm/internal/storage"
	"encoding/json"
	"fmt"
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

type relayConfigUpdateRequest struct {
	Upstreams        []RelayUpstream
	UpstreamStrategy string
	UpstreamURL      string
	APIKey           string
	AuthHeader       string
	AuthValuePrefix  string
	DefaultModel     string
	ModelMapJSON     string
	ToolDenylistStr  string
	MaxSessions      int
	MaxSessionBytes  int
	SessionTTLHours  int
}

func applyRelayConfigUpdate(config *RelayConfig, req relayConfigUpdateRequest) *RelayConfig {
	if config == nil {
		config = DefaultRelayConfig()
	}

	if req.Upstreams != nil {
		config.Upstreams = normalizeRelayUpstreams(req.Upstreams)
	}
	if strategy := strings.TrimSpace(req.UpstreamStrategy); strategy != "" {
		config.UpstreamStrategy = strategy
	} else if strings.TrimSpace(config.UpstreamStrategy) == "" {
		config.UpstreamStrategy = "round_robin"
	}

	config.UpstreamURL = strings.TrimRight(strings.TrimSpace(req.UpstreamURL), "/")
	config.APIKey = strings.TrimSpace(req.APIKey)
	config.AuthHeader = strings.TrimSpace(req.AuthHeader)
	config.AuthValuePrefix = strings.TrimSpace(req.AuthValuePrefix)
	config.DefaultModel = req.DefaultModel
	config.ModelMapJSON = req.ModelMapJSON
	config.ToolDenylistStr = req.ToolDenylistStr
	config.ModelMap = ParseModelMap(req.ModelMapJSON)
	config.ToolDenylist = ParseToolDenylist(req.ToolDenylistStr)

	if req.MaxSessions > 0 {
		config.MaxSessions = req.MaxSessions
	}
	if req.MaxSessionBytes > 0 {
		config.MaxSessionBytes = req.MaxSessionBytes
	}
	if req.SessionTTLHours > 0 {
		config.SessionTTLHours = req.SessionTTLHours
	}
	return config
}

func normalizeRelayUpstreams(upstreams []RelayUpstream) []RelayUpstream {
	out := make([]RelayUpstream, 0, len(upstreams))
	for i, upstream := range upstreams {
		upstream.ID = strings.TrimSpace(upstream.ID)
		upstream.Name = strings.TrimSpace(upstream.Name)
		upstream.UpstreamURL = strings.TrimRight(strings.TrimSpace(upstream.UpstreamURL), "/")
		upstream.APIKey = strings.TrimSpace(upstream.APIKey)
		upstream.AuthHeader = strings.TrimSpace(upstream.AuthHeader)
		upstream.AuthValuePrefix = strings.TrimSpace(upstream.AuthValuePrefix)
		if upstream.UpstreamURL == "" {
			continue
		}
		if upstream.ID == "" {
			upstream.ID = fmt.Sprintf("upstream-%d", i+1)
		}
		if upstream.Name == "" {
			upstream.Name = relayUpstreamNameFromURL(upstream.UpstreamURL)
		}
		out = append(out, upstream)
	}
	return out
}

func relayUpstreamNameFromURL(rawURL string) string {
	name := strings.TrimPrefix(strings.TrimPrefix(rawURL, "https://"), "http://")
	if idx := strings.Index(name, "/"); idx >= 0 {
		name = name[:idx]
	}
	if strings.TrimSpace(name) == "" {
		return "上游渠道"
	}
	return name
}
