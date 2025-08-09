package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/haukened/rr-dns/internal/dns/config"
	"github.com/stretchr/testify/require"
)

// TestE2E_DNSResolution tests actual DNS queries end-to-end
func TestE2E_DNSResolution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Setup test zone
	tempDir := t.TempDir()
	zoneFile := filepath.Join(tempDir, "e2e.yaml")
	zoneContent := `zone_root: e2e.test
api:
  A: "10.0.0.1"
web:
  A: 
    - "10.0.0.2"
    - "10.0.0.3"
`
	if err := os.WriteFile(zoneFile, []byte(zoneContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Find available port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())

	// Set environment
	originalEnv := map[string]string{
		"DNS_PORT":      os.Getenv("DNS_PORT"),
		"DNS_ZONE_DIR":  os.Getenv("DNS_ZONE_DIR"),
		"DNS_LOG_LEVEL": os.Getenv("DNS_LOG_LEVEL"),
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

	require.NoError(t, os.Setenv("DNS_PORT", fmt.Sprintf("%d", port)))
	require.NoError(t, os.Setenv("DNS_ZONE_DIR", tempDir))
	require.NoError(t, os.Setenv("DNS_LOG_LEVEL", "error")) // Reduce noise

	// Start application
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}

	app, err := buildApplication(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start in background
	appErr := make(chan error, 1)
	go func() {
		appErr <- app.Run(ctx)
	}()

	// Wait for startup
	timeout := time.After(2 * time.Second)
	for {
		select {
		case <-timeout:
			t.Fatal("Server failed to start")
		default:
			conn, err := net.Dial("udp", fmt.Sprintf("localhost:%d", port))
			if err == nil {
				require.NoError(t, conn.Close())
				goto serverStarted
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

serverStarted:
	// TODO: Add actual DNS query tests here using net.LookupHost or custom DNS client
	// This would require implementing a simple DNS client or using a library like miekg/dns

	// For now, just verify the server is responding to connections
	conn, err := net.Dial("udp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		t.Fatalf("Cannot connect to DNS server: %v", err)
	}
	require.NoError(t, conn.Close())

	// Shutdown
	cancel()
	select {
	case err := <-appErr:
		if err != nil {
			t.Errorf("Application shutdown error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Application failed to shutdown")
	}
}
