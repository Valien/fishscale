package config

import (
	"fmt"
	"os"
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
