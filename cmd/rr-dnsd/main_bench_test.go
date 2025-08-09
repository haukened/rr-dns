package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/haukened/rr-dns/internal/dns/common/log"
	"github.com/haukened/rr-dns/internal/dns/config"
	"github.com/haukened/rr-dns/internal/dns/domain"
	"github.com/stretchr/testify/require"
)

// BenchmarkBuildApplication measures the time to construct the full application
func BenchmarkBuildApplication(b *testing.B) {
	// Setup noop logger to silence output
	originalLogger := log.GetLogger()
	log.SetLogger(log.NewNoopLogger())
	defer log.SetLogger(originalLogger)

	// Setup test zone directory
	tempDir := b.TempDir()
	for i := 0; i < 10; i++ {
		zoneFile := filepath.Join(tempDir, fmt.Sprintf("zone%d.yaml", i))
		zoneContent := fmt.Sprintf(`zone_root: zone%d.bench
api:
  A: "10.0.%d.1"
web:
  A: 
    - "10.0.%d.2"
    - "10.0.%d.3"
`, i, i, i, i)
		err := os.WriteFile(zoneFile, []byte(zoneContent), 0644)
		require.NoError(b, err)
	}

	// Set environment
	originalZoneDir := os.Getenv("DNS_ZONE_DIR")
	defer func() {
		if originalZoneDir == "" {
			require.NoError(b, os.Unsetenv("DNS_ZONE_DIR"))
		} else {
			require.NoError(b, os.Setenv("DNS_ZONE_DIR", originalZoneDir))
		}
	}()
	require.NoError(b, os.Setenv("DNS_ZONE_DIR", tempDir))

	cfg, err := config.Load()
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app, err := buildApplication(cfg)
		require.NoError(b, err)
		_ = app // Use the app to prevent optimization
	}
}

// BenchmarkApplicationLifecycle measures full startup and shutdown
func BenchmarkApplicationLifecycle(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping lifecycle benchmark in short mode")
	}

	// Setup noop logger to silence output
	originalLogger := log.GetLogger()
	log.SetLogger(log.NewNoopLogger())
	defer log.SetLogger(originalLogger)

	// Setup
	tempDir := b.TempDir()
	zoneFile := filepath.Join(tempDir, "bench.yaml")
	zoneContent := `zone_root: bench.test
api:
  A: "127.0.0.1"
`
	require.NoError(b, os.WriteFile(zoneFile, []byte(zoneContent), 0644))

	originalEnv := map[string]string{
		"DNS_ZONE_DIR": os.Getenv("DNS_ZONE_DIR"),
	}
	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				require.NoError(b, os.Unsetenv(key))
			} else {
				require.NoError(b, os.Setenv(key, value))
			}
		}
	}()

	require.NoError(b, os.Setenv("DNS_ZONE_DIR", tempDir))

	cfg, err := config.Load()
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app, err := buildApplication(cfg)
		require.NoError(b, err)

		ctx, cancel := context.WithCancel(context.Background())

		// Start application in background
		done := make(chan error, 1)
		go func() {
			done <- app.Run(ctx)
		}()

		// Immediately shutdown
		cancel()

		// Wait for completion
		<-done
	}
}

// setupTestServer creates a running DNS server for query benchmarks
func setupTestServer(b *testing.B, zoneContent string) (*Application, func()) {
	// Setup noop logger
	originalLogger := log.GetLogger()
	log.SetLogger(log.NewNoopLogger())

	// Setup test zone
	tempDir := b.TempDir()
	zoneFile := filepath.Join(tempDir, "example.yaml")
	require.NoError(b, os.WriteFile(zoneFile, []byte(zoneContent), 0644))

	// Set environment - no need for actual port since we're testing resolver directly
	originalEnv := map[string]string{
		"DNS_ZONE_DIR":      os.Getenv("DNS_ZONE_DIR"),
		"DNS_CACHE_SIZE":    os.Getenv("DNS_CACHE_SIZE"),
		"DNS_DISABLE_CACHE": os.Getenv("DNS_DISABLE_CACHE"),
	}

	require.NoError(b, os.Setenv("DNS_ZONE_DIR", tempDir))
	require.NoError(b, os.Setenv("DNS_CACHE_SIZE", "1000")) // Larger cache for testing
	require.NoError(b, os.Unsetenv("DNS_DISABLE_CACHE"))    // CRITICAL: Ensure cache is enabled

	// Build application
	cfg, err := config.Load()
	require.NoError(b, err)

	app, err := buildApplication(cfg)
	require.NoError(b, err)

	// Return cleanup function
	cleanup := func() {
		// Restore environment
		for key, value := range originalEnv {
			if value == "" {
				require.NoError(b, os.Unsetenv(key))
			} else {
				require.NoError(b, os.Setenv(key, value))
			}
		}

		// Restore logger
		log.SetLogger(originalLogger)
	}

	return app, cleanup
} // createTestQuery creates a DNS query for benchmarking
func createTestQuery(name string, qtype domain.RRType) domain.Question {
	query, _ := domain.NewQuestion(1, name, qtype, domain.RRClass(1)) // IN class
	return query
}

// queryDNSServer performs a DNS query against the test server's resolver
func queryDNSServer(b *testing.B, app *Application, query domain.Question) {
	// Query directly through the resolver to get real performance
	ctx := context.Background()
	clientAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}

	_, err := app.resolver.HandleQuery(ctx, query, clientAddr)
	if err != nil {
		b.Fatalf("DNS query failed: %v", err)
	}
}

// BenchmarkQuery_AuthoritativeZone tests authoritative query performance
func BenchmarkQuery_AuthoritativeZone(b *testing.B) {
	zoneContent := `zone_root: example.com.
# Load test data with multiple records
www:
  A: 
    - "192.0.2.1"
    - "192.0.2.2"
    - "192.0.2.3"
api:
  A: "192.0.2.10"
  AAAA: "2001:db8::1"
cdn:
  A:
    - "192.0.2.20"
    - "192.0.2.21"
    - "192.0.2.22"
    - "192.0.2.23"
    - "192.0.2.24"
mail:
  A: "192.0.2.30"
  MX: "10 mail.example.com."
blog:
  CNAME: "www.example.com."
shop:
  A:
    - "192.0.2.40"
    - "192.0.2.41"
`

	app, cleanup := setupTestServer(b, zoneContent)
	defer cleanup()
	_ = app // Use the app to prevent optimization

	// Test different query types
	queries := []struct {
		name  string
		qtype domain.RRType
		host  string
	}{
		{"A record single", domain.RRType(1), "api.example.com."},
		{"A record multiple", domain.RRType(1), "www.example.com."},
		{"A record many", domain.RRType(1), "cdn.example.com."},
		{"AAAA record", domain.RRType(28), "api.example.com."},
		{"CNAME record", domain.RRType(5), "blog.example.com."},
		{"MX record", domain.RRType(15), "mail.example.com."},
	}

	for _, q := range queries {
		b.Run(q.name, func(b *testing.B) {
			query := createTestQuery(q.host, q.qtype)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				queryDNSServer(b, app, query)
			}
		})
	}
}

// BenchmarkQuery_UpstreamResolution tests upstream query performance
func BenchmarkQuery_UpstreamResolution(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping upstream benchmark in short mode")
	}

	// Minimal zone for upstream testing
	zoneContent := `zone_root: example.com.
local:
  A: "127.0.0.1"
`

	app, cleanup := setupTestServer(b, zoneContent)
	defer cleanup()

	// Test queries for well-known external domains
	queries := []struct {
		name string
		host string
	}{
		{"Google DNS", "dns.google."},
		{"Cloudflare DNS", "one.one.one.one."},
		{"GitHub", "github.com."},
		{"Stack Overflow", "stackoverflow.com."},
	}

	for _, q := range queries {
		b.Run(q.name, func(b *testing.B) {
			query := createTestQuery(q.host, domain.RRType(1)) // A record

			// Warm-up: measure one cold upstream hit
			firstStart := time.Now()
			queryDNSServer(b, app, query)
			b.Logf("Cold query (%s) took: %s", q.name, time.Since(firstStart))

			// Pause to let cache settle
			time.Sleep(50 * time.Millisecond)

			// Now benchmark warm cache performance (all queries should be cached)
			// This benchmark measures the warm cache behavior after an initial warm-up.
			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				queryDNSServer(b, app, query) // Now all cached
			}
		})
	}
}

// BenchmarkQuery_CachePerformance tests cached query performance
func BenchmarkQuery_CachePerformance(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping cache benchmark in short mode")
	}

	// Minimal zone for cache testing
	zoneContent := `zone_root: example.com.
local:
  A: "127.0.0.1"
`

	app, cleanup := setupTestServer(b, zoneContent)
	defer cleanup()

	// Test one specific domain to ensure consistent caching
	testQuery := createTestQuery("dns.google.", domain.RRType(1)) // A record

	// First, measure the upstream cold query time
	b.Run("Cold upstream query", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		var i int
		for b.Loop() {
			b.StopTimer() // Stop timer during query to avoid timing cache hits
			// Force a fresh query each time by using a different query ID
			freshQuery := createTestQuery("unique"+fmt.Sprintf("%d", i)+".google.", domain.RRType(1))
			b.StartTimer()

			queryDNSServer(b, app, freshQuery)
			i++
		}
	})

	// Second, warm up the cache and measure cached query performance
	b.Run("Warm cache query", func(b *testing.B) {
		// Warm up cache with the test query
		_, err := app.resolver.HandleQuery(context.Background(), testQuery, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345})
		if err != nil {
			b.Fatalf("Failed to warm up cache: %v", err)
		}

		// Small delay to ensure cache is populated
		time.Sleep(50 * time.Millisecond)

		b.ResetTimer()
		b.ReportAllocs()

		for b.Loop() {
			queryDNSServer(b, app, testQuery) // Same query each time = cache hit
		}
	})
}

// BenchmarkQuery_Mixed tests mixed query patterns
func BenchmarkQuery_Mixed(b *testing.B) {
	zoneContent := `zone_root: example.com.
www:
  A: "192.0.2.1"
api:
  A: "192.0.2.10" 
cdn:
  A: "192.0.2.20"
`

	app, cleanup := setupTestServer(b, zoneContent)
	defer cleanup()

	// Mix of authoritative and external queries
	queries := []domain.Question{
		createTestQuery("www.example.com.", domain.RRType(1)), // Authoritative
		createTestQuery("api.example.com.", domain.RRType(1)), // Authoritative
		createTestQuery("dns.google.", domain.RRType(1)),      // External
		createTestQuery("cdn.example.com.", domain.RRType(1)), // Authoritative
		createTestQuery("github.com.", domain.RRType(1)),      // External
	}

	b.ResetTimer()
	b.ReportAllocs()

	var i int
	for b.Loop() {
		query := queries[i%len(queries)]
		queryDNSServer(b, app, query)
		i++
	}
}
