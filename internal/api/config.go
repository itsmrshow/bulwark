package api

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config holds API/UI server configuration.
type Config struct {
	Addr           string
	Root           string
	StateDB        string
	UIEnabled      bool
	ReadOnly       bool
	WebToken       string
	DistDir        string
	DataDir        string
	ConfigPath     string
	WriteRateRPS   float64
	WriteRateBurst int
	PlanCacheTTL   time.Duration
}

// LoadConfig loads configuration from environment variables.
func LoadConfig() Config {
	return Config{
		Addr:           getEnv("BULWARK_UI_ADDR", ":8080"),
		Root:           getEnv("BULWARK_ROOT", "/docker_data"),
		StateDB:        os.Getenv("BULWARK_STATE_DB"),
		UIEnabled:      getEnvBool("BULWARK_UI_ENABLED", true),
		ReadOnly:       getEnvBool("BULWARK_UI_READONLY", true),
		WebToken:       os.Getenv("BULWARK_WEB_TOKEN"),
		DistDir:        getEnv("BULWARK_UI_DIST", "web/dist"),
		DataDir:        getEnv("BULWARK_DATA_DIR", "/data"),
		WriteRateRPS:   getEnvFloat("BULWARK_WEB_WRITE_RPS", 1.0),
		WriteRateBurst: getEnvInt("BULWARK_WEB_WRITE_BURST", 3),
		PlanCacheTTL:   getEnvDuration("BULWARK_PLAN_CACHE_TTL", 15*time.Second),
	}
}

func (c Config) WithDefaults() Config {
	if c.ConfigPath == "" {
		c.ConfigPath = getEnv("BULWARK_CONFIG_PATH", filepath.Join(c.DataDir, "bulwark.json"))
	}
	return c
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return defaultValue
	}
	switch value {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return defaultValue
	}
}

func getEnvInt(key string, defaultValue int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func getEnvFloat(key string, defaultValue float64) float64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}
