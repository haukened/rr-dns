package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/haukened/rr-dns/internal/dns/config"
)

// TestApplication_Integration tests the full application lifecycle
func TestApplication_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary zone directory with test zone
	tempDir := t.TempDir()
	zoneFile := filepath.Join(tempDir, "test.yaml")
	zoneContent := `zone_root: test.local
www:
  A: "127.0.0.1"
`
	require.NoError(t, os.WriteFile(zoneFile, []byte(zoneContent), 0644))

	// Set environment variables for test configuration
	originalEnv := map[string]string{
		"DNS_PORT":       os.Getenv("DNS_PORT"),
		"DNS_ZONE_DIR":   os.Getenv("DNS_ZONE_DIR"),
		"DNS_LOG_LEVEL":  os.Getenv("DNS_LOG_LEVEL"),
		"DNS_CACHE_SIZE": os.Getenv("DNS_CACHE_SIZE"),
	}
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				require.NoError(t, os.Unsetenv(key))
			} else {
				require.NoError(t, os.Setenv(key, value))
			}
		}
	}()

	// Find available port
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())

	require.NoError(t, os.Setenv("DNS_PORT", fmt.Sprintf("%d", port)))
	require.NoError(t, os.Setenv("DNS_ZONE_DIR", tempDir))
	require.NoError(t, os.Setenv("DNS_LOG_LEVEL", "debug"))
	require.NoError(t, os.Setenv("DNS_CACHE_SIZE", "100"))

	// Build application
	cfg, err := config.Load()
	require.NoError(t, err)

	app, err := buildApplication(cfg)
	require.NoError(t, err)
	assert.NotNil(t, app)

	// Test application startup and shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start application in goroutine
	appErr := make(chan error, 1)
	go func() {
		appErr <- app.Run(ctx)
	}()

	// Wait for server to start (or timeout)
	timeout := time.After(2 * time.Second)
	for {
		select {
		case <-timeout:
			t.Fatal("Server failed to start within timeout")
		case err := <-appErr:
			if err != nil {
				t.Fatalf("Server failed to start: %v", err)
			}
		default:
			// Check if server is listening
			conn, err := net.Dial("udp", fmt.Sprintf("localhost:%d", port))
			if err == nil {
				require.NoError(t, conn.Close())
				goto serverStarted
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

serverStarted:
	// Test graceful shutdown
	cancel()

	select {
	case err := <-appErr:
		assert.NoError(t, err, "Application should shutdown gracefully")
	case <-time.After(5 * time.Second):
		t.Fatal("Application failed to shutdown within timeout")
	}
}

// TestBuildApplication_ConfigurationVariations tests different configurations
func TestBuildApplication_ConfigurationVariations(t *testing.T) {
	tests := []struct {
		name          string
		setupEnv      func()
		wantErr       bool
		errorContains string
	}{
		{
			name: "minimal valid config",
			setupEnv: func() {
				require.NoError(t, os.Setenv("DNS_ZONE_DIR", t.TempDir()))
			},
			wantErr: false,
		},
		{
			name: "invalid zone directory",
			setupEnv: func() {
				require.NoError(t, os.Setenv("DNS_ZONE_DIR", "/nonexistent/path"))
			},
			wantErr:       true,
			errorContains: "failed to load zone directory",
		},
		{
			name: "cache disabled",
			setupEnv: func() {
				require.NoError(t, os.Setenv("DNS_ZONE_DIR", t.TempDir()))
				require.NoError(t, os.Setenv("DNS_DISABLE_CACHE", "true"))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean environment
			for _, key := range []string{"DNS_PORT", "DNS_ZONE_DIR", "DNS_DISABLE_CACHE"} {
				_ = os.Unsetenv(key)
			}

			tt.setupEnv()

			cfg, err := config.Load()
			if err != nil {
				if tt.wantErr {
					return // Configuration error is expected
				}
				t.Fatalf("Config load failed: %v", err)
			}

			app, err := buildApplication(cfg)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, app)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, app)
			}
		})
	}
}

// TestApplication_ComponentIntegration tests that all components work together
func TestApplication_ComponentIntegration(t *testing.T) {
	// Create test zone
	tempDir := t.TempDir()
	zoneFile := filepath.Join(tempDir, "integration.yaml")
	zoneContent := `zone_root: integration.test
api:
  A: "10.0.0.1"
web:
  A: 
    - "10.0.0.2"
    - "10.0.0.3"
`
	require.NoError(t, os.WriteFile(zoneFile, []byte(zoneContent), 0644))

	// Set test environment
	require.NoError(t, os.Setenv("DNS_ZONE_DIR", tempDir))
	require.NoError(t, os.Setenv("DNS_CACHE_SIZE", "50"))
	defer func() {
		_ = os.Unsetenv("DNS_ZONE_DIR")
		_ = os.Unsetenv("DNS_CACHE_SIZE")
	}()

	cfg, err := config.Load()
	require.NoError(t, err)

	app, err := buildApplication(cfg)
	require.NoError(t, err)

	// Verify components are wired correctly
	assert.NotNil(t, app.config)
	assert.NotNil(t, app.transport)
	assert.NotNil(t, app.resolver)

	// Verify zone loading worked
	assert.Equal(t, tempDir, app.config.ZoneDir)
	assert.Equal(t, uint(50), app.config.CacheSize)
}
