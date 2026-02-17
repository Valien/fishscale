package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	TSAuthKey  string
	TSHostname string
	TSStateDir string
	DBPath     string
	PhotoDir   string
	LogLevel   string
	DevMode    bool
}

func Load() *Config {
	return &Config{
		TSAuthKey:  os.Getenv("TS_AUTHKEY"),
		TSHostname: getEnv("TS_HOSTNAME", "fishscale"),
		TSStateDir: getEnv("TS_STATE_DIR", "/data/tsnet-state"),
		DBPath:     getEnv("FISHSCALE_DB_PATH", "/data/fish.db"),
		PhotoDir:   getEnv("FISHSCALE_PHOTO_DIR", "/data/photos"),
		LogLevel:   getEnv("FISHSCALE_LOG_LEVEL", "info"),
		DevMode:    os.Getenv("FISHSCALE_DEV_MODE") == "true",
	}
}

func (c *Config) Validate() error {
	if !c.DevMode && c.TSAuthKey == "" {
		return fmt.Errorf("TS_AUTHKEY is required in production mode")
	}
	if c.DBPath == "" {
		return fmt.Errorf("FISHSCALE_DB_PATH must not be empty")
	}
	if c.PhotoDir == "" {
		return fmt.Errorf("FISHSCALE_PHOTO_DIR must not be empty")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ParseLogLevel converts a string log level name to a slog.Level.
// Recognized values: "debug", "info", "warn"/"warning", "error" (case-insensitive).
// Unrecognized or empty values default to slog.LevelInfo.
func ParseLogLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
