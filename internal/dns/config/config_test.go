package config

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf/v2"
)

func TestLoad_Defaults(t *testing.T) {
	os.Unsetenv("DNS_ENV")
	os.Unsetenv("DNS_LOG_LEVEL")
	os.Unsetenv("DNS_PORT")
	os.Unsetenv("DNS_CACHE_SIZE")
	os.Unsetenv("DNS_ZONE_DIR")
	os.Unsetenv("DNS_SERVERS")

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
	if cfg.ZoneDir != "/etc/rr-dns/zones/" {
		t.Errorf("expected ZoneDir=/etc/rr-dns/zones/, got %q", cfg.ZoneDir)
	}
	wantUpstream := []string{"1.1.1.1:53", "1.0.0.1:53"}
	if len(cfg.Servers) != len(wantUpstream) {
		t.Errorf("expected Upstream length %d, got %d", len(wantUpstream), len(cfg.Servers))
	} else {
		for i, v := range wantUpstream {
			if cfg.Servers[i] != v {
				t.Errorf("expected Upstream[%d]=%q, got %q", i, v, cfg.Servers[i])
			}
		}
	}
}

func TestLoad_ValidOverrides(t *testing.T) {
	t.Setenv("DNS_ENV", "prod")
	t.Setenv("DNS_LOG_LEVEL", "info")
	t.Setenv("DNS_PORT", "9953")
	t.Setenv("DNS_CACHE_SIZE", "2000")
	t.Setenv("DNS_ZONE_DIR", "/tmp/zones/")
	t.Setenv("DNS_SERVERS", "8.8.8.8:53,8.8.4.4:53")

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
	if cfg.ZoneDir != "/tmp/zones/" {
		t.Errorf("expected ZoneDir=/tmp/zones/, got %q", cfg.ZoneDir)
	}
	wantUpstream := []string{"8.8.8.8:53", "8.8.4.4:53"}
	if len(cfg.Servers) != len(wantUpstream) {
		t.Errorf("expected Upstream length %d, got %d", len(wantUpstream), len(cfg.Servers))
	} else {
		for i, v := range wantUpstream {
			if cfg.Servers[i] != v {
				t.Errorf("expected Upstream[%d]=%q, got %q", i, v, cfg.Servers[i])
			}
		}
	}
}

func TestLoad_WhenKoanfDefaultLoadFails(t *testing.T) {
	orig := defaultLoader
	defaultLoader = func(k *koanf.Koanf) error {
		return errors.New("mocked error")
	}
	defer func() { defaultLoader = orig }()

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "mocked error") {
		t.Fatal("expected error when loading defaults, got nil")
	}
}

func TestLoad_WhenKoanfEnvLoadFails(t *testing.T) {
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

func TestLoad_RegisterValidationFails(t *testing.T) {
	orig := registerValidation
	registerValidation = func(v *validator.Validate) error {
		return errors.New("mocked validation error")
	}
	defer func() { registerValidation = orig }()

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "mocked validation error") {
		t.Fatal("expected error when registering validation, got nil")
	}
}

func TestLoad_InvalidEnv(t *testing.T) {
	t.Setenv("DNS_ENV", "staging")
	t.Setenv("DNS_LOG_LEVEL", "info")
	t.Setenv("DNS_PORT", "53")
	t.Setenv("DNS_CACHE_SIZE", "1000")
	t.Setenv("DNS_ZONE_DIR", "/tmp/zones/")
	t.Setenv("DNS_UPSTREAM", "8.8.8.8:53,8.8.4.4:53")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid DNS_ENV, got nil")
	}
}

func TestLoad_InvalidLogLevel(t *testing.T) {
	t.Setenv("DNS_ENV", "dev")
	t.Setenv("DNS_LOG_LEVEL", "trace")
	t.Setenv("DNS_PORT", "53")
	t.Setenv("DNS_CACHE_SIZE", "1000")
	t.Setenv("DNS_ZONE_DIR", "/tmp/zones/")
	t.Setenv("DNS_UPSTREAM", "8.8.8.8:53,8.8.4.4:53")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid LOG_LEVEL, got nil")
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	t.Setenv("DNS_ENV", "dev")
	t.Setenv("DNS_LOG_LEVEL", "info")
	t.Setenv("DNS_PORT", "99999")
	t.Setenv("DNS_CACHE_SIZE", "1000")
	t.Setenv("DNS_ZONE_DIR", "/tmp/zones/")
	t.Setenv("DNS_UPSTREAM", "8.8.8.8:53,8.8.4.4:53")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid PORT, got nil")
	}
}

func TestLoad_PortNaN(t *testing.T) {
	t.Setenv("DNS_ENV", "dev")
	t.Setenv("DNS_LOG_LEVEL", "info")
	t.Setenv("DNS_PORT", "not_a_number")
	t.Setenv("DNS_ZONE_DIR", "/tmp/zones/")
	t.Setenv("DNS_UPSTREAM", "8.8.8.8:53,8.8.4.4:53")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for non-numeric PORT, got nil")
	}
}

func TestLoad_InvalidCacheSize(t *testing.T) {
	t.Setenv("DNS_ENV", "dev")
	t.Setenv("DNS_LOG_LEVEL", "info")
	t.Setenv("DNS_PORT", "53")
	t.Setenv("DNS_CACHE_SIZE", "-1")
	t.Setenv("DNS_ZONE_DIR", "/tmp/zones/")
	t.Setenv("DNS_UPSTREAM", "8.8.8.8:53,8.8.4.4:53")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid CACHE_SIZE, got nil")
	}
}

func TestLoad_InvalidZoneDir(t *testing.T) {
	t.Setenv("DNS_ENV", "dev")
	t.Setenv("DNS_LOG_LEVEL", "info")
	t.Setenv("DNS_PORT", "53")
	t.Setenv("DNS_CACHE_SIZE", "1000")
	t.Setenv("DNS_ZONE_DIR", "") // required
	t.Setenv("DNS_UPSTREAM", "8.8.8.8:53,8.8.4.4:53")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for empty ZoneDir, got nil")
	}
}

func TestLoad_InvalidUpstream(t *testing.T) {
	t.Setenv("DNS_ENV", "dev")
	t.Setenv("DNS_LOG_LEVEL", "info")
	t.Setenv("DNS_PORT", "53")
	t.Setenv("DNS_CACHE_SIZE", "1000")
	t.Setenv("DNS_ZONE_DIR", "/tmp/zones/")
	t.Setenv("DNS_SERVERS", "not_a_server") // invalid format

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid Servers, got nil")
	}
}

func TestValidIPPort(t *testing.T) {
	type testCase struct {
		input    string
		expected bool
	}

	cases := []testCase{
		{"1.2.3.4:53", true},
		{"127.0.0.1:5353", true},
		{"::1:53", false}, // missing brackets for IPv6
		{"[::1]:53", true},
		{"192.168.1.1:", false},
		{":53", false},
		{"not_an_ip:53", false},
		{"1.2.3.4:notaport", false},
		{"", false},
		{"1.2.3.4", false},
		{"[::1]", false},
	}

	validate := validator.New()
	_ = validate.RegisterValidation("ip_port", validIPPort)

	for _, tc := range cases {
		// Use a struct to test the validator
		type S struct {
			Addr string `validate:"ip_port"`
		}
		s := S{Addr: tc.input}
		err := validate.Struct(s)
		if tc.expected && err != nil {
			t.Errorf("validIPPort(%q) = false, want true", tc.input)
		}
		if !tc.expected && err == nil {
			t.Errorf("validIPPort(%q) = true, want false", tc.input)
		}
	}
}
func TestDefaultLoader_LoadsDefaults(t *testing.T) {
	k := koanf.New(".")
	err := defaultLoader(k)
	if err != nil {
		t.Fatalf("defaultLoader returned error: %v", err)
	}

	var cfg AppConfig
	if err := k.Unmarshal("", &cfg); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if cfg.CacheSize != DEFAULT_APP_CONFIG.CacheSize {
		t.Errorf("expected CacheSize=%d, got %d", DEFAULT_APP_CONFIG.CacheSize, cfg.CacheSize)
	}
	if cfg.DisableCache != DEFAULT_APP_CONFIG.DisableCache {
		t.Errorf("expected DisableCache=%v, got %v", DEFAULT_APP_CONFIG.DisableCache, cfg.DisableCache)
	}
	if cfg.Env != DEFAULT_APP_CONFIG.Env {
		t.Errorf("expected Env=%q, got %q", DEFAULT_APP_CONFIG.Env, cfg.Env)
	}
	if cfg.LogLevel != DEFAULT_APP_CONFIG.LogLevel {
		t.Errorf("expected LogLevel=%q, got %q", DEFAULT_APP_CONFIG.LogLevel, cfg.LogLevel)
	}
	if cfg.Port != DEFAULT_APP_CONFIG.Port {
		t.Errorf("expected Port=%d, got %d", DEFAULT_APP_CONFIG.Port, cfg.Port)
	}
	if cfg.ZoneDir != DEFAULT_APP_CONFIG.ZoneDir {
		t.Errorf("expected ZoneDir=%q, got %q", DEFAULT_APP_CONFIG.ZoneDir, cfg.ZoneDir)
	}
	if len(cfg.Servers) != len(DEFAULT_APP_CONFIG.Servers) {
		t.Errorf("expected Servers length %d, got %d", len(DEFAULT_APP_CONFIG.Servers), len(cfg.Servers))
	} else {
		for i, v := range DEFAULT_APP_CONFIG.Servers {
			if cfg.Servers[i] != v {
				t.Errorf("expected Servers[%d]=%q, got %q", i, v, cfg.Servers[i])
			}
		}
	}
}

func TestDefaultLoader_ErrorPropagation(t *testing.T) {
	orig := DEFAULT_APP_CONFIG
	defer func() { DEFAULT_APP_CONFIG = orig }()

	// Simulate an invalid default config that cannot be unmarshalled (e.g., invalid type)
	DEFAULT_APP_CONFIG = AppConfig{
		Servers:   []string{"not_a_valid_ip_port"},
		Env:       "prod",
		LogLevel:  "info",
		Port:      53,
		ZoneDir:   "/etc/rr-dns/zones/",
		CacheSize: 1000,
	}

	k := koanf.New(".")
	err := defaultLoader(k)
	if err != nil {
		t.Fatalf("defaultLoader returned error: %v", err)
	}

	var cfg AppConfig
	err = k.Unmarshal("", &cfg)
	if err != nil {
		// Should fail validation, not unmarshalling
		return
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	_ = validate.RegisterValidation("ip_port", validIPPort)
	err = validate.Struct(&cfg)
	if err == nil {
		t.Fatal("expected validation error for invalid default Servers, got nil")
	}
}
