package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

// AppConfig holds configuration values parsed from environment variables.
type AppConfig struct {
	// Env is the runtime environment, either "dev" or "prod".
	Env string `koanf:"env" validate:"required,oneof=dev prod"`

	Log LoggingConfig `koanf:"log" validate:"required"`

	Resolver ResolverConfig `koanf:"resolver" validate:"required"`

	Blocklist BlocklistConfig `koanf:"blocklist" validate:"required"`
}

type CacheConfig struct {
	// LRU cache size, 0 to disable
	Size int `koanf:"size" validate:"gte=0"`
}

type LoggingConfig struct {
	// Level defines the logging level: "debug", "info", "warn", or "error".
	Level string `koanf:"level" validate:"required,oneof=debug info warn error"`
}

type ResolverConfig struct {
	// ZoneDirectory is the directory where zone files are located.
	// default: /etc/rr-dns/zone.d/
	ZoneDirectory string `koanf:"zones" validate:"required"`

	// Upstream is a list of upstream DNS servers in ip:port format.
	// default: 1.1.1.1:53, 1.0.0.1:53
	Upstream []string `koanf:"upstream" validate:"required,dive,ip_port"`

	// MaxRecursion limits in-zone CNAME (or future alias) chase depth.
	// Prevents infinite loops; 0 or negative will be rejected by validation (must be >=1).
	// default: 8
	MaxRecursion int `koanf:"depth" validate:"required,gte=1"`

	// Port is the network port the DNS server will bind to.
	// default: 53
	Port int `koanf:"port" validate:"required,gte=1,lte=65535"`

	// Upstream cache configuration (size 0 disables cache)
	Cache CacheConfig `koanf:"cache"`
}

type BlocklistConfig struct {
	// the local path to the directory holding blocklist files
	// default: /etc/rr-dns/blocklist.d/
	Directory string `koanf:"dir" validate:"required"`

	// URLs is a list of blocklist URLs to fetch and update the local blocklist.
	// default: none
	URLs []string `koanf:"urls" validate:"omitempty,dive,url"`

	// LRU Cache Config (size 0 disables cache)
	Cache CacheConfig `koanf:"cache"`

	// Database file path for badgerDB
	DB string `koanf:"db" validate:"required"`

	// Blocking strategy
	Strategy string `koanf:"strategy" validate:"required,oneof=refused nxdomain sinkhole"`

	// required sinkhole target if strategy is sinkhole
	Sinkhole *SinkholeOptions `koanf:"sinkhole" validate:"required_if=Strategy sinkhole"`
}

type SinkholeOptions struct {
	// a list of IP addresses to resolve for the sinkhole, IPv4 or IPv6
	Target []string `koanf:"target" validate:"required,dive,ip"`
	// Time-to-live for sinkhole DNS records
	TTL int `koanf:"ttl" validate:"required,gte=0"`
}

// DEFAULT_APP_CONFIG defines the default application configuration settings for the DNS service.
// It includes default values for cache size, environment, log level, listening port, zone directory,
// and upstream DNS servers.
var DEFAULT_APP_CONFIG = AppConfig{
	Env: "prod",
	Log: LoggingConfig{
		Level: "info",
	},
	Resolver: ResolverConfig{
		ZoneDirectory: "/etc/rr-dns/zone.d/",
		Upstream:      []string{"1.1.1.1:53", "1.0.0.1:53"},
		MaxRecursion:  8,
		Port:          53,
		Cache: CacheConfig{
			Size: 1000,
		},
	},
	Blocklist: BlocklistConfig{
		Directory: "/etc/rr-dns/blocklist.d/",
		URLs:      []string{},
		Cache: CacheConfig{
			Size: 1000,
		},
		DB:       "/var/lib/rr-dns/blocklist.db",
		Strategy: "refused",
		Sinkhole: nil,
	},
}

// validIPPort validates whether the provided field value is a valid IP address and port combination.
// It expects the value to be in the format "IP:Port". The function returns true if the IP address
// is valid and both the IP and port are non-empty; otherwise, it returns false.
func validIPPort(fl validator.FieldLevel) bool {
	// stringify the field value to get the IP:Port format.
	addr := fl.Field().String()
	// Split the address into IP and port.
	ip, port, err := net.SplitHostPort(addr)
	if err != nil || ip == "" || port == "" {
		return false
	}
	// Check if the IP address is valid.
	if net.ParseIP(ip) == nil {
		return false
	}
	// Check if the port is a valid number between 1 and 65535.
	portNum, err := strconv.ParseUint(port, 10, 16)
	return err == nil && portNum > 0 && portNum < 65536
}

// envLoader is a function that loads environment variables with the prefix "DNS_".
// It transforms the keys to lowercase and removes the prefix, and replaces _ with .
// and can be mocked in tests.
var envLoader = func(k *koanf.Koanf) error {
	// Load environment variables with prefix "DNS_".
	return k.Load(env.Provider(".", env.Opt{
		Prefix: "DNS_",
		TransformFunc: func(key, value string) (string, any) {
			key = strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(key, "DNS_")), "_", ".")
			value = strings.TrimSpace(value)

			if value == "" {
				return key, value
			}

			if strings.Contains(value, " ") || strings.Contains(value, ",") {
				parts := strings.FieldsFunc(value, func(r rune) bool {
					return r == ' ' || r == ','
				})
				return key, parts
			}

			return key, value
		},
	}), nil)
}

// defaultLoader loads default configuration values into the provided Koanf instance
// using the structs provider and the DEFAULT_APP_CONFIG struct. It returns an error
// if loading fails.
var defaultLoader = func(k *koanf.Koanf) error {
	// Load default values using structs provider.
	return k.Load(structs.Provider(DEFAULT_APP_CONFIG, "koanf"), nil)
}

// registerValidation registers a custom validation function "ip_port" with the provided validator.
// It associates the "ip_port" tag with the validIPPort validation logic.
// Returns an error if registration fails.
var registerValidation = func(v *validator.Validate) error {
	return v.RegisterValidation("ip_port", validIPPort)
}

// Load parses environment variables and returns an AppConfig instance.
// It applies default values and runs validation automatically.
func Load() (*AppConfig, error) {
	k := koanf.New(".")

	// Load default values using structs provider.
	err := defaultLoader(k)
	if err != nil {
		return nil, fmt.Errorf("error loading default config: %w", err)
	}

	// Load environment variables with prefix "DNS_", using koanf/providers/env/v2 and Opt pattern.
	err = envLoader(k)
	if err != nil {
		return nil, fmt.Errorf("error loading env: %w", err)
	}

	var cfg AppConfig

	// Unmarshal the loaded configuration into AppConfig struct.
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	// Validate the configuration.
	validate := validator.New(validator.WithRequiredStructEnabled())

	// Register the custom validation function for IP:Port format.
	err = registerValidation(validate)
	if err != nil {
		return nil, fmt.Errorf("error registering validation: %w", err)
	}

	err = validate.Struct(&cfg)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &cfg, nil
}
