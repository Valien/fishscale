package config

import (
	"log/slog"
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

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"DEBUG", slog.LevelDebug},  // case insensitive
		{"", slog.LevelInfo},         // default
		{"invalid", slog.LevelInfo},  // fallback
	}
	for _, tt := range tests {
		got := ParseLogLevel(tt.input)
		if got != tt.want {
			t.Errorf("ParseLogLevel(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
