package codexconfig

import (
	"strconv"
	"strings"
)

const (
	ModeDefault    = "default"
	ModePreset516K = "preset_516k"
	ModePreset1M   = "preset_1m"
	ModeCustom     = "custom"

	SettingMode                  = "codex_context_mode"
	SettingModelContextWindow    = "codex_model_context_window"
	SettingAutoCompactTokenLimit = "codex_model_auto_compact_token_limit"

	ModelContextWindowKey         = "model_context_window"
	ModelAutoCompactTokenLimitKey = "model_auto_compact_token_limit"
)

type ContextConfig struct {
	Mode                       string `json:"codex_context_mode"`
	ModelContextWindow         int64  `json:"model_context_window"`
	ModelAutoCompactTokenLimit int64  `json:"model_auto_compact_token_limit"`
}

func Default() ContextConfig {
	return ContextConfig{Mode: ModeDefault}
}

func FromSettings(settings map[string]string) ContextConfig {
	if settings == nil {
		return Default()
	}
	cfg := ContextConfig{
		Mode:                       strings.TrimSpace(settings[SettingMode]),
		ModelContextWindow:         parseInt64(settings[SettingModelContextWindow]),
		ModelAutoCompactTokenLimit: parseInt64(settings[SettingAutoCompactTokenLimit]),
	}
	return Normalize(cfg)
}

func Normalize(cfg ContextConfig) ContextConfig {
	switch strings.TrimSpace(cfg.Mode) {
	case ModePreset516K:
		return ContextConfig{
			Mode:                       ModePreset516K,
			ModelContextWindow:         516000,
			ModelAutoCompactTokenLimit: 460000,
		}
	case ModePreset1M:
		return ContextConfig{
			Mode:                       ModePreset1M,
			ModelContextWindow:         1000000,
			ModelAutoCompactTokenLimit: 900000,
		}
	case ModeCustom:
		cfg.Mode = ModeCustom
		if cfg.ModelContextWindow <= 0 || cfg.ModelAutoCompactTokenLimit <= 0 {
			return Default()
		}
		return cfg
	default:
		return Default()
	}
}

func ValidMode(mode string) bool {
	switch strings.TrimSpace(mode) {
	case "", ModeDefault, ModePreset516K, ModePreset1M, ModeCustom:
		return true
	default:
		return false
	}
}

func (cfg ContextConfig) Enabled() bool {
	cfg = Normalize(cfg)
	return cfg.Mode != ModeDefault &&
		cfg.ModelContextWindow > 0 &&
		cfg.ModelAutoCompactTokenLimit > 0
}

func (cfg ContextConfig) Settings() map[string]string {
	cfg = Normalize(cfg)
	return map[string]string{
		SettingMode:                  cfg.Mode,
		SettingModelContextWindow:    formatOptionalInt64(cfg.ModelContextWindow),
		SettingAutoCompactTokenLimit: formatOptionalInt64(cfg.ModelAutoCompactTokenLimit),
	}
}

func parseInt64(value string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func formatOptionalInt64(value int64) string {
	if value <= 0 {
		return ""
	}
	return strconv.FormatInt(value, 10)
}
