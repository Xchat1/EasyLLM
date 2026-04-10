package config

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"strconv"
	"sync"
)

type Config struct {
	mu          sync.RWMutex `json:"-"`
	Server      ServerConfig
	Database    DatabaseConfig
	Proxy       ProxyConfig
	App         AppConfig
	Log         LogConfig
	IPBlacklist IPBlacklistConfig
}

type ServerConfig struct {
	Port    int
	Host    string
	APIPort int
}

type DatabaseConfig struct {
	Type     string // "sqlite" or "postgres"
	DSN      string // postgres: "host=... user=... password=... dbname=... port=5432 sslmode=disable"
	SQLitePath string
}

type ProxyConfig struct {
	Enabled  bool
	Host     string
	Port     int
	Username string
	Password string
}

type AppConfig struct {
	DataDir        string
	SecretKey      string
	Debug          bool
	DefaultPassword string
}

type LogConfig struct {
	Enabled bool
}

type IPBlacklistConfig struct {
	Enabled bool
	IPs     []string
}

var (
	instance *Config
	once     sync.Once
)

func Load() *Config {
	once.Do(func() {
		instance = &Config{
			Server: ServerConfig{
				Port:    getEnvInt("SERVER_PORT", 8022),
				Host:    getEnv("SERVER_HOST", "0.0.0.0"),
				APIPort: getEnvInt("SERVER_PORT", 8022), // same port; APIPort is legacy, kept for struct compat
			},
			Database: DatabaseConfig{
				Type:       getEnv("DB_TYPE", "sqlite"),
				DSN:        getEnv("DB_DSN", ""),
				SQLitePath: getEnv("DB_SQLITE_PATH", "./data/easyllm.db"),
			},
			Proxy: ProxyConfig{
				Enabled:  getEnvBool("PROXY_ENABLED", false),
				Host:     getEnv("PROXY_HOST", ""),
				Port:     getEnvInt("PROXY_PORT", 0),
				Username: getEnv("PROXY_USERNAME", ""),
				Password: getEnv("PROXY_PASSWORD", ""),
			},
			App: AppConfig{
				DataDir:         getEnv("DATA_DIR", "./data"),
				SecretKey:       getEnv("SECRET_KEY", ""),
				Debug:           getEnvBool("DEBUG", false),
				DefaultPassword: getEnv("DEFAULT_PASSWORD", ""),
			},
			Log: LogConfig{
				Enabled: getEnvBool("LOG_ENABLED", true),
			},
			IPBlacklist: IPBlacklistConfig{
				Enabled: getEnvBool("IP_BLACKLIST_ENABLED", false),
				IPs:     []string{},
			},
		}
	})
	if instance.App.SecretKey == "" {
		instance.App.SecretKey = generateRandomSecretKey()
		log.Println("[WARNING] No SECRET_KEY set — generated a random one for this session. Set SECRET_KEY env var for persistent sessions.")
	}
	return instance
}

// generateRandomSecretKey generates a 32-byte random hex string.
func generateRandomSecretKey() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func Get() *Config {
	if instance == nil {
		return Load()
	}
	return instance
}

// Update updates config fields at runtime. Values from JSON are type-safe
// checked to avoid panics (JSON numbers arrive as float64).
func (c *Config) Update(updates map[string]interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if v, ok := updates["proxy_enabled"].(bool); ok {
		c.Proxy.Enabled = v
	}
	if v, ok := updates["proxy_host"].(string); ok {
		c.Proxy.Host = v
	}
	if v, ok := updates["proxy_port"]; ok {
		switch n := v.(type) {
		case float64:
			c.Proxy.Port = int(n)
		case int:
			c.Proxy.Port = n
		}
	}
	if v, ok := updates["proxy_username"].(string); ok {
		c.Proxy.Username = v
	}
	if v, ok := updates["proxy_password"].(string); ok {
		c.Proxy.Password = v
	}
	if v, ok := updates["db_type"].(string); ok {
		c.Database.Type = v
	}
	if v, ok := updates["db_dsn"].(string); ok {
		c.Database.DSN = v
	}
	if v, ok := updates["log_enabled"].(bool); ok {
		c.Log.Enabled = v
	}
	if v, ok := updates["ip_blacklist_enabled"].(bool); ok {
		c.IPBlacklist.Enabled = v
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultVal
}
