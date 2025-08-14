package config

import (
	"errors"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf/v2"
)

func TestLoad_Defaults(t *testing.T) {
	// No env overrides
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Env != "prod" {
		t.Errorf("expected Env=prod, got %q", cfg.Env)
	}
	if cfg.Log.Level != "info" {
		t.Errorf("expected Log.Level=info, got %q", cfg.Log.Level)
	}

	// Resolver defaults
	if cfg.Resolver.ZoneDirectory != "/etc/rr-dns/zone.d/" {
		t.Errorf("expected Resolver.ZoneDirectory=/etc/rr-dns/zone.d/, got %q", cfg.Resolver.ZoneDirectory)
	}
	if cfg.Resolver.Port != 53 {
		t.Errorf("expected Resolver.Port=53, got %d", cfg.Resolver.Port)
	}
	if cfg.Resolver.MaxRecursion != 8 {
		t.Errorf("expected Resolver.MaxRecursion=8, got %d", cfg.Resolver.MaxRecursion)
	}
	if cfg.Resolver.Cache.Size != 1000 {
		t.Errorf("expected Resolver.Cache.Size=1000, got %d", cfg.Resolver.Cache.Size)
	}
	wantUpstream := []string{"1.1.1.1:53", "1.0.0.1:53"}
	if len(cfg.Resolver.Upstream) != len(wantUpstream) {
		t.Errorf("expected Resolver.Upstream length %d, got %d", len(wantUpstream), len(cfg.Resolver.Upstream))
	} else {
		for i, v := range wantUpstream {
			if cfg.Resolver.Upstream[i] != v {
				t.Errorf("expected Resolver.Upstream[%d]=%q, got %q", i, v, cfg.Resolver.Upstream[i])
			}
		}
	}

	// Blocklist defaults
	if cfg.Blocklist.Directory != "/etc/rr-dns/blocklist.d/" {
		t.Errorf("expected Blocklist.Directory=/etc/rr-dns/blocklist.d/, got %q", cfg.Blocklist.Directory)
	}
	if cfg.Blocklist.DB != "/var/lib/rr-dns/blocklist.db" {
		t.Errorf("expected Blocklist.DB=/var/lib/rr-dns/blocklist.db, got %q", cfg.Blocklist.DB)
	}
	if cfg.Blocklist.Strategy != "refused" {
		t.Errorf("expected Blocklist.Strategy=refused, got %q", cfg.Blocklist.Strategy)
	}
	if cfg.Blocklist.Cache.Size != 1000 {
		t.Errorf("expected Blocklist.Cache.Size=1000, got %d", cfg.Blocklist.Cache.Size)
	}
	if len(cfg.Blocklist.URLs) != 0 {
		t.Errorf("expected Blocklist.URLs to be empty by default, got %v", cfg.Blocklist.URLs)
	}
	if cfg.Blocklist.Sinkhole != nil {
		t.Errorf("expected Blocklist.Sinkhole to be nil by default")
	}
}

func TestLoad_ValidOverrides(t *testing.T) {
	t.Setenv("DNS_ENV", "dev")
	t.Setenv("DNS_LOG_LEVEL", "debug")
	t.Setenv("DNS_RESOLVER_ZONES", "/tmp/zone.d/")
	t.Setenv("DNS_RESOLVER_UPSTREAM", "8.8.8.8:53 8.8.4.4:53")
	t.Setenv("DNS_RESOLVER_DEPTH", "12")
	t.Setenv("DNS_RESOLVER_PORT", "9953")
	t.Setenv("DNS_RESOLVER_CACHE_SIZE", "2000")

	t.Setenv("DNS_BLOCKLIST_DIR", "/tmp/blocklist.d/")
	t.Setenv("DNS_BLOCKLIST_URLS", "https://a.example/list.txt,https://b.example/x")
	t.Setenv("DNS_BLOCKLIST_CACHE_SIZE", "5000")
	t.Setenv("DNS_BLOCKLIST_DB", "/tmp/blk.db")
	t.Setenv("DNS_BLOCKLIST_STRATEGY", "sinkhole")
	t.Setenv("DNS_BLOCKLIST_SINKHOLE_TARGET", "0.0.0.0,::")
	t.Setenv("DNS_BLOCKLIST_SINKHOLE_TTL", "60")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Env != "dev" {
		t.Errorf("expected Env=dev, got %q", cfg.Env)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("expected Log.Level=debug, got %q", cfg.Log.Level)
	}
	if cfg.Resolver.ZoneDirectory != "/tmp/zone.d/" {
		t.Errorf("expected Resolver.ZoneDirectory=/tmp/zone.d/, got %q", cfg.Resolver.ZoneDirectory)
	}
	if cfg.Resolver.Port != 9953 {
		t.Errorf("expected Resolver.Port=9953, got %d", cfg.Resolver.Port)
	}
	if cfg.Resolver.MaxRecursion != 12 {
		t.Errorf("expected Resolver.MaxRecursion=12, got %d", cfg.Resolver.MaxRecursion)
	}
	if cfg.Resolver.Cache.Size != 2000 {
		t.Errorf("expected Resolver.Cache.Size=2000, got %d", cfg.Resolver.Cache.Size)
	}
	wantUpstream := []string{"8.8.8.8:53", "8.8.4.4:53"}
	if len(cfg.Resolver.Upstream) != len(wantUpstream) {
		t.Errorf("expected Resolver.Upstream length %d, got %d", len(wantUpstream), len(cfg.Resolver.Upstream))
	} else {
		for i, v := range wantUpstream {
			if cfg.Resolver.Upstream[i] != v {
				t.Errorf("expected Resolver.Upstream[%d]=%q, got %q", i, v, cfg.Resolver.Upstream[i])
			}
		}
	}

	if cfg.Blocklist.Directory != "/tmp/blocklist.d/" {
		t.Errorf("expected Blocklist.BlocklistDirectory=/tmp/blocklist.d/, got %q", cfg.Blocklist.Directory)
	}
	if cfg.Blocklist.DB != "/tmp/blk.db" {
		t.Errorf("expected Blocklist.DB=/tmp/blk.db, got %q", cfg.Blocklist.DB)
	}
	if cfg.Blocklist.Cache.Size != 5000 {
		t.Errorf("expected Blocklist.Cache.Size=5000, got %d", cfg.Blocklist.Cache.Size)
	}
	wantURLs := []string{"https://a.example/list.txt", "https://b.example/x"}
	if len(cfg.Blocklist.URLs) != len(wantURLs) {
		t.Errorf("expected Blocklist.URLs length %d, got %d", len(wantURLs), len(cfg.Blocklist.URLs))
	} else {
		for i, v := range wantURLs {
			if cfg.Blocklist.URLs[i] != v {
				t.Errorf("expected Blocklist.URLs[%d]=%q, got %q", i, v, cfg.Blocklist.URLs[i])
			}
		}
	}

	if cfg.Blocklist.Strategy != "sinkhole" {
		t.Errorf("expected Blocklist.Strategy=sinkhole, got %q", cfg.Blocklist.Strategy)
	}
	if cfg.Blocklist.Sinkhole == nil {
		t.Fatalf("expected Blocklist.Sinkhole to be set")
	}
	wantTargets := []string{"0.0.0.0", "::"}
	if len(cfg.Blocklist.Sinkhole.Target) != len(wantTargets) {
		t.Errorf("expected Sinkhole.Target length %d, got %d", len(wantTargets), len(cfg.Blocklist.Sinkhole.Target))
	} else {
		for i, v := range wantTargets {
			if cfg.Blocklist.Sinkhole.Target[i] != v {
				t.Errorf("expected Sinkhole.Target[%d]=%q, got %q", i, v, cfg.Blocklist.Sinkhole.Target[i])
			}
		}
	}
	if cfg.Blocklist.Sinkhole.TTL != 60 {
		t.Errorf("expected Sinkhole.TTL=60, got %d", cfg.Blocklist.Sinkhole.TTL)
	}
}

func TestLoad_SinkholeRequired(t *testing.T) {
	// Override only the strategy to sinkhole; no sinkhole.* provided
	t.Setenv("DNS_BLOCKLIST_STRATEGY", "sinkhole")
	_, err := Load()
	if err == nil {
		t.Fatal("expected validation error when strategy=sinkhole but sinkhole options are missing")
	}
}

func TestLoad_WhenKoanfDefaultLoadFails(t *testing.T) {
	orig := defaultLoader
	defaultLoader = func(k *koanf.Koanf) error { return errors.New("mocked error") }
	defer func() { defaultLoader = orig }()

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "mocked error") {
		t.Fatal("expected error when loading defaults, got nil")
	}
}

func TestLoad_WhenKoanfEnvLoadFails(t *testing.T) {
	orig := envLoader
	envLoader = func(k *koanf.Koanf) error { return errors.New("mocked error") }
	defer func() { envLoader = orig }()

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "mocked error") {
		t.Fatal("expected error when loading env, got nil")
	}
}

func TestLoad_RegisterValidationFails(t *testing.T) {
	orig := registerValidation
	registerValidation = func(v *validator.Validate) error { return errors.New("mocked validation error") }
	defer func() { registerValidation = orig }()

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "mocked validation error") {
		t.Fatal("expected error when registering validation, got nil")
	}
}

func TestLoad_InvalidEnv(t *testing.T) {
	t.Setenv("DNS_ENV", "staging")
	t.Setenv("DNS_LOG_LEVEL", "info")
	t.Setenv("DNS_RESOLVER_PORT", "53")
	t.Setenv("DNS_RESOLVER_CACHE_SIZE", "1000")
	t.Setenv("DNS_RESOLVER_ZONES", "/tmp/zone.d/")
	t.Setenv("DNS_RESOLVER_UPSTREAM", "8.8.8.8:53,8.8.4.4:53")
	t.Setenv("DNS_RESOLVER_DEPTH", "0")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid DNS_ENV, got nil")
	}
}

func TestLoad_InvalidLogLevel(t *testing.T) {
	t.Setenv("DNS_ENV", "dev")
	t.Setenv("DNS_LOG_LEVEL", "trace")
	t.Setenv("DNS_RESOLVER_PORT", "53")
	t.Setenv("DNS_RESOLVER_CACHE_SIZE", "1000")
	t.Setenv("DNS_RESOLVER_ZONES", "/tmp/zone.d/")
	t.Setenv("DNS_RESOLVER_UPSTREAM", "8.8.8.8:53,8.8.4.4:53")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid LOG_LEVEL, got nil")
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	t.Setenv("DNS_ENV", "dev")
	t.Setenv("DNS_LOG_LEVEL", "info")
	t.Setenv("DNS_RESOLVER_PORT", "99999")
	t.Setenv("DNS_RESOLVER_CACHE_SIZE", "1000")
	t.Setenv("DNS_RESOLVER_ZONES", "/tmp/zone.d/")
	t.Setenv("DNS_RESOLVER_UPSTREAM", "8.8.8.8:53,8.8.4.4:53")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid PORT, got nil")
	}
}

func TestLoad_PortNaN(t *testing.T) {
	t.Setenv("DNS_ENV", "dev")
	t.Setenv("DNS_LOG_LEVEL", "info")
	t.Setenv("DNS_RESOLVER_PORT", "not_a_number")
	t.Setenv("DNS_RESOLVER_ZONES", "/tmp/zone.d/")
	t.Setenv("DNS_RESOLVER_UPSTREAM", "8.8.8.8:53,8.8.4.4:53")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for non-numeric PORT, got nil")
	}
}

func TestLoad_InvalidCacheSize(t *testing.T) {
	t.Setenv("DNS_ENV", "dev")
	t.Setenv("DNS_LOG_LEVEL", "info")
	t.Setenv("DNS_RESOLVER_PORT", "53")
	t.Setenv("DNS_RESOLVER_CACHE_SIZE", "-1")
	t.Setenv("DNS_RESOLVER_ZONES", "/tmp/zone.d/")
	t.Setenv("DNS_RESOLVER_UPSTREAM", "8.8.8.8:53,8.8.4.4:53")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid RESOLVER_CACHE_SIZE, got nil")
	}
}

func TestLoad_InvalidZoneDir(t *testing.T) {
	t.Setenv("DNS_ENV", "dev")
	t.Setenv("DNS_LOG_LEVEL", "info")
	t.Setenv("DNS_RESOLVER_PORT", "53")
	t.Setenv("DNS_RESOLVER_CACHE_SIZE", "1000")
	t.Setenv("DNS_RESOLVER_ZONES", "") // required
	t.Setenv("DNS_RESOLVER_UPSTREAM", "8.8.8.8:53,8.8.4.4:53")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for empty Resolver.ZoneDirectory, got nil")
	}
}

func TestLoad_InvalidUpstream(t *testing.T) {
	t.Setenv("DNS_ENV", "dev")
	t.Setenv("DNS_LOG_LEVEL", "info")
	t.Setenv("DNS_RESOLVER_PORT", "53")
	t.Setenv("DNS_RESOLVER_CACHE_SIZE", "1000")
	t.Setenv("DNS_RESOLVER_ZONES", "/tmp/zone.d/")
	t.Setenv("DNS_RESOLVER_UPSTREAM", "not_a_server") // invalid format

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid Resolver.Upstream, got nil")
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
	if err := defaultLoader(k); err != nil {
		t.Fatalf("defaultLoader returned error: %v", err)
	}

	var cfg AppConfig
	if err := k.Unmarshal("", &cfg); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Compare a subset of defaults
	if cfg.Env != DEFAULT_APP_CONFIG.Env {
		t.Errorf("expected Env=%q, got %q", DEFAULT_APP_CONFIG.Env, cfg.Env)
	}
	if cfg.Log.Level != DEFAULT_APP_CONFIG.Log.Level {
		t.Errorf("expected Log.Level=%q, got %q", DEFAULT_APP_CONFIG.Log.Level, cfg.Log.Level)
	}
	if cfg.Resolver.Port != DEFAULT_APP_CONFIG.Resolver.Port {
		t.Errorf("expected Resolver.Port=%d, got %d", DEFAULT_APP_CONFIG.Resolver.Port, cfg.Resolver.Port)
	}
	if cfg.Resolver.ZoneDirectory != DEFAULT_APP_CONFIG.Resolver.ZoneDirectory {
		t.Errorf("expected Resolver.ZoneDirectory=%q, got %q", DEFAULT_APP_CONFIG.Resolver.ZoneDirectory, cfg.Resolver.ZoneDirectory)
	}
	if cfg.Resolver.Cache.Size != DEFAULT_APP_CONFIG.Resolver.Cache.Size {
		t.Errorf("expected Resolver.Cache.Size=%d, got %d", DEFAULT_APP_CONFIG.Resolver.Cache.Size, cfg.Resolver.Cache.Size)
	}
	if len(cfg.Resolver.Upstream) != len(DEFAULT_APP_CONFIG.Resolver.Upstream) {
		t.Fatalf("expected Resolver.Upstream length %d, got %d", len(DEFAULT_APP_CONFIG.Resolver.Upstream), len(cfg.Resolver.Upstream))
	}
	for i, v := range DEFAULT_APP_CONFIG.Resolver.Upstream {
		if cfg.Resolver.Upstream[i] != v {
			t.Errorf("expected Resolver.Upstream[%d]=%q, got %q", i, v, cfg.Resolver.Upstream[i])
		}
	}
}

func TestDefaultLoader_InvalidDefault_ValidationFails(t *testing.T) {
	orig := DEFAULT_APP_CONFIG
	defer func() { DEFAULT_APP_CONFIG = orig }()

	DEFAULT_APP_CONFIG = AppConfig{
		Env: "prod",
		Log: LoggingConfig{Level: "info"},
		Resolver: ResolverConfig{
			ZoneDirectory: "/etc/rr-dns/zone.d/",
			Upstream:      []string{"not_a_valid_ip_port"},
			MaxRecursion:  8,
			Port:          53,
			Cache:         CacheConfig{Size: 1000},
		},
		Blocklist: BlocklistConfig{
			Directory: "/etc/rr-dns/blocklist.d/",
			URLs:      []string{},
			Cache:     CacheConfig{Size: 1000},
			DB:        "/var/lib/rr-dns/blocklist.db",
			Strategy:  "refused",
			Sinkhole:  nil,
		},
	}

	k := koanf.New(".")
	if err := defaultLoader(k); err != nil {
		t.Fatalf("defaultLoader returned error: %v", err)
	}

	var cfg AppConfig
	if err := k.Unmarshal("", &cfg); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	_ = validate.RegisterValidation("ip_port", validIPPort)
	if err := validate.Struct(&cfg); err == nil {
		t.Fatal("expected validation error for invalid default Resolver.Upstream, got nil")
	}
}
