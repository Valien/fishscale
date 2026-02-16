package config

import (
	"os"
	"testing"
)

func TestValidate_RequiresAuthKeyInProduction(t *testing.T) {
	cfg := &Config{
		DevMode:   false,
		TSAuthKey: "",
		DBPath:    "/data/fish.db",
		PhotoDir:  "/data/photos",
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing TS_AUTHKEY in production mode, got nil")
	}
}

func TestValidate_AllowsMissingAuthKeyInDevMode(t *testing.T) {
	cfg := &Config{
		DevMode:  true,
		DBPath:   "/data/fish.db",
		PhotoDir: "/data/photos",
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected error in dev mode: %v", err)
	}
}

func TestLoad_SetsDefaults(t *testing.T) {
	os.Unsetenv("TS_HOSTNAME")
	os.Unsetenv("FISHSCALE_LOG_LEVEL")

	cfg := Load()
	if cfg.TSHostname != "fishscale" {
		t.Errorf("expected default hostname 'fishscale', got %q", cfg.TSHostname)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected default log level 'info', got %q", cfg.LogLevel)
	}
}
