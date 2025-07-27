package config

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

// AppConfig holds configuration values parsed from environment variables.
type AppConfig struct {
	CacheSize uint `koanf:"cache_size" validate:"required,gte=1"`

	// Env is the runtime environment, either "dev" or "prod".
	Env string `koanf:"env" validate:"required,oneof=dev prod"`

	// LogLevel controls log verbosity: "debug", "info", "warn", or "error".
	LogLevel string `koanf:"log_level" validate:"required,oneof=debug info warn error"`

	// Port is the network port the DNS server will bind to.
	Port int `koanf:"port" validate:"required,gte=1,lt=65535"`
}

// envLoader is a function that loads environment variables with the prefix "UDNS_".
// It transforms the keys to lowercase and removes the prefix.
// and can be mocked in tests.
var envLoader = func(k *koanf.Koanf) error {
	// Load environment variables with prefix "UDNS_".
	return k.Load(env.Provider(".", env.Opt{
		Prefix: "UDNS_",
		TransformFunc: func(key, value string) (string, any) {
			return strings.ToLower(strings.TrimPrefix(key, "UDNS_")), value
		},
	}), nil)
}

// Load parses environment variables and returns an AppConfig instance.
// It applies default values and runs validation automatically.
func Load() (*AppConfig, error) {
	k := koanf.New(".")

	// Load default values using structs provider.
	k.Load(structs.Provider(AppConfig{
		CacheSize: 1000,
		Env:       "prod",
		LogLevel:  "info",
		Port:      53,
	}, "koanf"), nil)

	// Load environment variables with prefix "UDNS_", using koanf/providers/env/v2 and Opt pattern.
	err := envLoader(k)
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

	err = validate.Struct(&cfg)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &cfg, nil
}
