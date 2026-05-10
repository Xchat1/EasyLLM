package handlers

import (
	"easyllm/config"
	"easyllm/internal/models"
	"easyllm/internal/storage"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var startTime = time.Now()

// SettingsHandler manages application settings
type SettingsHandler struct{}

func NewSettingsHandler() *SettingsHandler {
	return &SettingsHandler{}
}

func (h *SettingsHandler) RegisterRoutes(rg *gin.RouterGroup) {
	// Settings
	s := rg.Group("/settings")
	s.GET("", h.GetSettings)
	s.PUT("", h.UpdateSettings)

	// Switches (toggles)
	s.GET("/switches", h.GetSwitches)
	s.PUT("/switches", h.UpdateSwitches)

	// IP blacklist management
	s.GET("/ip-blacklist", h.GetIPBlacklist)
	s.PUT("/ip-blacklist", h.UpdateIPBlacklist)

	// Proxy config
	s.GET("/proxy", h.GetProxy)
	s.PUT("/proxy", h.UpdateProxy)

	// Database config
	s.GET("/database", h.GetDatabase)
	s.PUT("/database", h.UpdateDatabase)

	// System info
	rg.GET("/health", h.Health)
	rg.GET("/system/info", h.SystemInfo)

	// API server status 已移至 server.go 作为公开接口注册，避免鉴权导致轮询 401
}

func (h *SettingsHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, models.HealthResponse{
		Status:  "ok",
		Version: models.AppVersion,
		Port:    config.Get().Server.Port,
	})
}

func (h *SettingsHandler) SystemInfo(c *gin.Context) {
	cfg := config.Get()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := time.Since(startTime)
	uptimeStr := formatUptime(uptime)

	hostname, _ := os.Hostname()
	cwd, _ := os.Getwd()

	// Account counts per platform
	counts := getAccountCounts()

	c.JSON(http.StatusOK, gin.H{
		"version":              models.AppVersion,
		"git_repo":             models.AppGitRepo,
		"go_version":           runtime.Version(),
		"os":                   runtime.GOOS,
		"arch":                 runtime.GOARCH,
		"hostname":             hostname,
		"pid":                  os.Getpid(),
		"cwd":                  cwd,
		"data_dir":             cfg.App.DataDir,
		"uptime":               uptimeStr,
		"uptime_seconds":       int64(uptime.Seconds()),
		"goroutines":           runtime.NumGoroutine(),
		"memory_alloc_mb":      fmt.Sprintf("%.1f", float64(m.Alloc)/1024/1024),
		"memory_sys_mb":        fmt.Sprintf("%.1f", float64(m.Sys)/1024/1024),
		"memory_gc_cycles":     m.NumGC,
		"db_type":              cfg.Database.Type,
		"server_port":          cfg.Server.Port,
		"server_host":          cfg.Server.Host,
		"proxy_enabled":        cfg.Proxy.Enabled,
		"ip_blacklist_enabled": cfg.IPBlacklist.Enabled,
		"debug":                cfg.App.Debug,
		"accounts":             counts,
	})
}

func formatUptime(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%d天 %d小时 %d分钟", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%d小时 %d分钟", hours, minutes)
	}
	return fmt.Sprintf("%d分钟", minutes)
}

func getAccountCounts() gin.H {
	db := storage.GetDB()
	if db == nil {
		return gin.H{}
	}
	counts := gin.H{}
	tables := map[string]string{
		"openai":      "open_ai_accounts",
		"antigravity": "antigravity_accounts",
		"codex_pool":  "codex_accounts",
		"instances":   "platform_instances",
		"wakeup":      "wakeup_tasks",
	}
	for key, table := range tables {
		var count int64
		if err := db.Table(table).Count(&count).Error; err == nil {
			counts[key] = count
		}
	}
	if db.Migrator().HasTable("platform_accounts") {
		for _, def := range models.GetCockpitPlatformDefinitions() {
			var count int64
			if err := db.Table("platform_accounts").Where("platform = ?", def.ID).Count(&count).Error; err == nil {
				counts[def.ID] = count
			}
		}
	}
	return counts
}

func (h *SettingsHandler) GetAPIServerStatus(c *gin.Context) {
	cfg := config.Get()
	c.JSON(http.StatusOK, gin.H{
		"running": true,
		"port":    cfg.Server.Port,
		"address": "http://" + cfg.Server.Host + ":" + strconv.Itoa(cfg.Server.Port),
	})
}

func (h *SettingsHandler) GetSettings(c *gin.Context) {
	cfg := config.Get()
	c.JSON(http.StatusOK, gin.H{
		"proxy": gin.H{
			"enabled":  cfg.Proxy.Enabled,
			"host":     cfg.Proxy.Host,
			"port":     cfg.Proxy.Port,
			"username": cfg.Proxy.Username,
		},
		"database": gin.H{
			"type": cfg.Database.Type,
			"dsn":  maskDSN(cfg.Database.DSN),
		},
		"ip_blacklist": gin.H{
			"enabled": cfg.IPBlacklist.Enabled,
			"ips":     cfg.IPBlacklist.IPs,
		},
	})
}

func (h *SettingsHandler) UpdateSettings(c *gin.Context) {
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}

	sanitized := make(map[string]interface{}, len(updates))
	for k, v := range updates {
		if !isSupportedSettingsUpdateKey(k) {
			c.JSON(http.StatusBadRequest, models.APIError{Error: "unsupported settings key: " + k, Code: "INVALID_REQUEST"})
			return
		}
		if !isSupportedSettingsUpdateValue(v) {
			c.JSON(http.StatusBadRequest, models.APIError{Error: "unsupported value type for key: " + k, Code: "INVALID_REQUEST"})
			return
		}
		sanitized[k] = v
	}
	config.Get().Update(sanitized)

	// Persist to database
	for k, v := range sanitized {
		storage.SaveSetting(k, toString(v))
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *SettingsHandler) GetSwitches(c *gin.Context) {
	cfg := config.Get()
	c.JSON(http.StatusOK, gin.H{
		"ip_blacklist_enabled": cfg.IPBlacklist.Enabled,
		"proxy_enabled":        cfg.Proxy.Enabled,
	})
}

func (h *SettingsHandler) UpdateSwitches(c *gin.Context) {
	var req struct {
		IPBlacklistEnabled *bool `json:"ip_blacklist_enabled"`
		ProxyEnabled       *bool `json:"proxy_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}

	cfg := config.Get()
	if req.IPBlacklistEnabled != nil {
		cfg.IPBlacklist.Enabled = *req.IPBlacklistEnabled
		storage.SaveSetting("ip_blacklist_enabled", strconv.FormatBool(*req.IPBlacklistEnabled))
	}
	if req.ProxyEnabled != nil {
		cfg.Proxy.Enabled = *req.ProxyEnabled
		storage.SaveSetting("proxy_enabled", strconv.FormatBool(*req.ProxyEnabled))
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *SettingsHandler) GetIPBlacklist(c *gin.Context) {
	cfg := config.Get()
	c.JSON(http.StatusOK, gin.H{
		"enabled": cfg.IPBlacklist.Enabled,
		"ips":     cfg.IPBlacklist.IPs,
	})
}

func (h *SettingsHandler) UpdateIPBlacklist(c *gin.Context) {
	var req struct {
		Enabled *bool    `json:"enabled"`
		IPs     []string `json:"ips"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}

	cfg := config.Get()
	if req.Enabled != nil {
		cfg.IPBlacklist.Enabled = *req.Enabled
		storage.SaveSetting("ip_blacklist_enabled", strconv.FormatBool(*req.Enabled))
	}
	if req.IPs != nil {
		cleaned := make([]string, 0, len(req.IPs))
		for _, ip := range req.IPs {
			ip = strings.TrimSpace(ip)
			if ip != "" {
				cleaned = append(cleaned, ip)
			}
		}
		cfg.IPBlacklist.IPs = cleaned
		storage.SaveSetting("ip_blacklist", strings.Join(cleaned, ","))
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *SettingsHandler) GetProxy(c *gin.Context) {
	cfg := config.Get()
	c.JSON(http.StatusOK, gin.H{
		"enabled":  cfg.Proxy.Enabled,
		"host":     cfg.Proxy.Host,
		"port":     cfg.Proxy.Port,
		"username": cfg.Proxy.Username,
	})
}

func (h *SettingsHandler) UpdateProxy(c *gin.Context) {
	var req struct {
		Enabled  bool   `json:"enabled"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}

	cfg := config.Get()
	cfg.Proxy.Enabled = req.Enabled
	cfg.Proxy.Host = req.Host
	cfg.Proxy.Port = req.Port
	cfg.Proxy.Username = req.Username
	if req.Password != "" {
		cfg.Proxy.Password = req.Password
	}

	// Persist
	storage.SaveSetting("proxy_enabled", strconv.FormatBool(req.Enabled))
	storage.SaveSetting("proxy_host", req.Host)
	storage.SaveSetting("proxy_port", strconv.Itoa(req.Port))
	storage.SaveSetting("proxy_username", req.Username)
	if req.Password != "" {
		storage.SaveSetting("proxy_password", req.Password)
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *SettingsHandler) GetDatabase(c *gin.Context) {
	cfg := config.Get()
	c.JSON(http.StatusOK, gin.H{
		"type":        cfg.Database.Type,
		"dsn":         maskDSN(cfg.Database.DSN),
		"sqlite_path": cfg.Database.SQLitePath,
	})
}

func (h *SettingsHandler) UpdateDatabase(c *gin.Context) {
	var req struct {
		Type       string `json:"type"`
		DSN        string `json:"dsn"`
		SQLitePath string `json:"sqlite_path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIError{Error: err.Error(), Code: "INVALID_REQUEST"})
		return
	}

	// Only persist - actual DB switch requires restart
	storage.SaveSetting("db_type", req.Type)
	if req.DSN != "" {
		storage.SaveSetting("db_dsn", req.DSN)
	}
	if req.SQLitePath != "" {
		storage.SaveSetting("db_sqlite_path", req.SQLitePath)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Database settings saved. Restart required to take effect.",
	})
}

// Helper functions
func maskSecret(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return "***"
	}
	return s[:4] + "***"
}

func maskDSN(dsn string) string {
	if dsn == "" {
		return ""
	}
	return "[configured]"
}

func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case bool:
		return strconv.FormatBool(val)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return strconv.Itoa(val)
	default:
		return ""
	}
}

func isSupportedSettingsUpdateKey(key string) bool {
	switch key {
	case "proxy_enabled", "proxy_host", "proxy_port", "proxy_username", "proxy_password", "db_type", "db_dsn", "ip_blacklist_enabled":
		return true
	default:
		return false
	}
}

func isSupportedSettingsUpdateValue(value interface{}) bool {
	switch value.(type) {
	case string, bool, float64, int:
		return true
	default:
		return false
	}
}
