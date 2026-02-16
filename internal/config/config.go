package config

import "os"

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

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
