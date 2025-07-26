package config

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/knadh/koanf"
)

func TestLoad_Defaults(t *testing.T) {
	os.Unsetenv("UDNS_ENV")
	os.Unsetenv("UDNS_LOG_LEVEL")
	os.Unsetenv("UDNS_PORT")
	os.Unsetenv("UDNS_CACHE_SIZE")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.Env != "prod" {
		t.Errorf("expected Env=prod, got %q", cfg.Env)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected LogLevel=info, got %q", cfg.LogLevel)
	}
	if cfg.Port != 53 {
		t.Errorf("expected Port=53, got %d", cfg.Port)
	}
}

func TestLoad_ValidOverrides(t *testing.T) {
	t.Setenv("UDNS_ENV", "prod")
	t.Setenv("UDNS_LOG_LEVEL", "info")
	t.Setenv("UDNS_PORT", "9953")
	t.Setenv("UDNS_CACHE_SIZE", "2000")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.Env != "prod" {
		t.Errorf("expected Env=prod, got %q", cfg.Env)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected Log_Level=info, got %q", cfg.LogLevel)
	}
	if cfg.Port != 9953 {
		t.Errorf("expected Port=9953, got %d", cfg.Port)
	}
}
func TestLoad_WhenKoanfLoadFails(t *testing.T) {
	orig := envLoader
	envLoader = func(k *koanf.Koanf) error {
		return errors.New("mocked error")
	}
	defer func() { envLoader = orig }()

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "mocked error") {
		t.Fatal("expected error when loading env, got nil")
	}
}

func TestLoad_InvalidEnv(t *testing.T) {
	t.Setenv("UDNS_ENV", "staging")
	t.Setenv("UDNS_LOG_LEVEL", "info")
	t.Setenv("UDNS_PORT", "53")
	t.Setenv("UDNS_CACHE_SIZE", "1000")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid UDNS_ENV, got nil")
	}
}

func TestLoad_InvalidLogLevel(t *testing.T) {
	t.Setenv("UDNS_ENV", "dev")
	t.Setenv("UDNS_LOG_LEVEL", "trace")
	t.Setenv("UDNS_PORT", "53")
	t.Setenv("UDNS_CACHE_SIZE", "1000")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid LOG_LEVEL, got nil")
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	t.Setenv("UDNS_ENV", "dev")
	t.Setenv("UDNS_LOG_LEVEL", "info")
	t.Setenv("UDNS_PORT", "99999")
	t.Setenv("UDNS_CACHE_SIZE", "1000")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid PORT, got nil")
	}
}

func TestLoad_PortNaN(t *testing.T) {
	t.Setenv("UDNS_ENV", "dev")
	t.Setenv("UDNS_LOG_LEVEL", "info")
	t.Setenv("UDNS_PORT", "not_a_number")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for non-numeric PORT, got nil")
	}
}

func TestLoad_InvalidCacheSize(t *testing.T) {
	t.Setenv("UDNS_ENV", "dev")
	t.Setenv("UDNS_LOG_LEVEL", "info")
	t.Setenv("UDNS_PORT", "53")
	t.Setenv("UDNS_CACHE_SIZE", "-1")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid CACHE_SIZE, got nil")
	}
}
