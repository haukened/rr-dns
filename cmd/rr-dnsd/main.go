package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/haukened/rr-dns/internal/dns/common/clock"
	"github.com/haukened/rr-dns/internal/dns/common/log"
	"github.com/haukened/rr-dns/internal/dns/config"
	"github.com/haukened/rr-dns/internal/dns/gateways/transport"
	"github.com/haukened/rr-dns/internal/dns/gateways/upstream"
	"github.com/haukened/rr-dns/internal/dns/gateways/wire"
	"github.com/haukened/rr-dns/internal/dns/repos/blocklist"
	"github.com/haukened/rr-dns/internal/dns/repos/dnscache"
	"github.com/haukened/rr-dns/internal/dns/repos/zone"
	"github.com/haukened/rr-dns/internal/dns/repos/zonecache"
	"github.com/haukened/rr-dns/internal/dns/services/resolver"
)

const (
	// Version information
	version = "0.1.0-dev"
	appName = "rr-dnsd"

	// Default timeouts
	defaultUpstreamTimeout = 5 * time.Second
	defaultShutdownTimeout = 10 * time.Second
)

// Application holds all the components of the DNS server
type Application struct {
	config    *config.AppConfig
	transport *transport.UDPTransport
	resolver  *resolver.Resolver
}

func main() {
	// Load configuration from environment
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Configure global logging
	err = log.Configure(cfg.Env, cfg.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Logging configuration error: %v\n", err)
		os.Exit(1)
	}

	log.Info(map[string]any{
		"version":    version,
		"env":        cfg.Env,
		"log_level":  cfg.LogLevel,
		"port":       cfg.Port,
		"cache_size": cfg.CacheSize,
		"zone_dir":   cfg.ZoneDir,
		"servers":    cfg.Servers,
	}, "Starting RR-DNS server")

	// Build application with all dependencies
	app, err := buildApplication(cfg)
	if err != nil {
		log.Fatal(map[string]any{"error": err}, "Failed to build application")
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Info(map[string]any{"signal": sig.String()}, "Shutdown signal received")
		cancel()
	}()

	// Start the DNS server
	if err := app.Run(ctx); err != nil {
		log.Fatal(map[string]any{"error": err}, "Server failed")
	}

	log.Info(nil, "RR-DNS server stopped gracefully")
}

// buildApplication constructs all components and wires them together
func buildApplication(cfg *config.AppConfig) (*Application, error) {
	// Create shared clock for consistent time across all components
	clk := &clock.RealClock{}

	// Initialize logger (already configured globally)
	logger := log.GetLogger()

	// Create DNS wire codec
	codec := wire.NewUDPCodec(logger)

	// Build repository layer
	repos, err := buildRepositories(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to build repositories: %w", err)
	}

	// Build gateway layer
	gateways, err := buildGateways(cfg, codec, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to build gateways: %w", err)
	}

	// Build service layer
	resolverService := resolver.NewResolver(resolver.ResolverOptions{
		Blocklist:     repos.blocklist,
		Clock:         clk,
		Logger:        logger,
		Upstream:      gateways.upstream,
		UpstreamCache: repos.upstreamCache,
		ZoneCache:     repos.zoneCache,
		MaxRecursion:  8, // TODO: Magic numbers are bad, put this in config.
	})

	// Build transport layer
	addr := fmt.Sprintf(":%d", cfg.Port)
	udpTransport := transport.NewUDPTransport(addr, codec, logger)

	return &Application{
		config:    cfg,
		transport: udpTransport,
		resolver:  resolverService,
	}, nil
}

// repositories holds all repository implementations
type repositories struct {
	blocklist     resolver.Blocklist
	upstreamCache resolver.Cache
	zoneCache     resolver.ZoneCache
}

// gateways holds all gateway implementations
type gateways struct {
	upstream resolver.UpstreamClient
}

// buildRepositories creates and configures all repository implementations
func buildRepositories(cfg *config.AppConfig, logger log.Logger) (*repositories, error) {
	// Create blocklist repository
	blocklistRepo := &blocklist.NoopBlocklist{}

	// Create upstream response cache
	var upstreamCache resolver.Cache
	var err error
	if cfg.DisableCache {
		upstreamCache = nil // No caching
		log.Info(map[string]any{"disabled": true}, "DNS response caching disabled")
	} else {
		// Safely convert uint to int with bounds check
		cacheSize := cfg.CacheSize
		if cacheSize > uint(^uint(0)>>1) { // Check if it exceeds max int
			return nil, fmt.Errorf("cache size too large: %d (max %d)", cacheSize, ^uint(0)>>1)
		}
		upstreamCache, err = dnscache.New(int(cacheSize))
		if err != nil {
			return nil, fmt.Errorf("failed to create upstream cache: %w", err)
		}
		log.Info(map[string]any{
			"type": "LRU",
			"size": cfg.CacheSize,
		}, "DNS response cache configured")
	}

	// Create zone cache
	zoneCache := zonecache.New()

	// load the zone files from the configured directory
	zones, err := zone.LoadZoneDirectory(cfg.ZoneDir, 300*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to load zone directory: %w", err)
	}

	// load each zone into the zone cache
	for zoneRoot, records := range zones {
		zoneCache.PutZone(zoneRoot, records)
	}

	log.Info(map[string]any{
		"zone_dir": cfg.ZoneDir,
		"zones":    len(zoneCache.Zones()),
	}, "Zone cache initialized")

	return &repositories{
		blocklist:     blocklistRepo,
		upstreamCache: upstreamCache,
		zoneCache:     zoneCache,
	}, nil
}

// buildGateways creates and configures all gateway implementations
func buildGateways(cfg *config.AppConfig, codec wire.DNSCodec, logger log.Logger) (*gateways, error) {
	// Create upstream client
	upstreamClient, err := upstream.NewResolver(upstream.Options{
		Servers: cfg.Servers,
		Timeout: defaultUpstreamTimeout,
		Codec:   codec,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create upstream client: %w", err)
	}

	log.Info(map[string]any{
		"servers": cfg.Servers,
		"timeout": defaultUpstreamTimeout,
	}, "Upstream DNS client configured")

	return &gateways{
		upstream: upstreamClient,
	}, nil
}

// Run starts the DNS server and blocks until context is cancelled
func (app *Application) Run(ctx context.Context) error {
	// Start UDP transport
	if err := app.transport.Start(ctx, app.resolver); err != nil {
		return fmt.Errorf("failed to start UDP transport: %w", err)
	}

	log.Info(map[string]any{
		"address":   app.transport.Address(),
		"transport": "UDP",
	}, "DNS server started")

	// Wait for shutdown signal
	<-ctx.Done()

	log.Info(nil, "Shutdown initiated")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer cancel()

	// Stop transport gracefully
	if err := app.transport.Stop(); err != nil {
		log.Warn(map[string]any{"error": err}, "Error during transport shutdown")
	}

	// Wait for shutdown completion or timeout
	done := make(chan struct{})
	go func() {
		// Additional cleanup could go here
		close(done)
	}()

	select {
	case <-done:
		log.Info(nil, "Graceful shutdown completed")
		return nil
	case <-shutdownCtx.Done():
		log.Warn(map[string]any{"timeout": defaultShutdownTimeout}, "Shutdown timeout exceeded")
		return fmt.Errorf("shutdown timeout")
	}
}
