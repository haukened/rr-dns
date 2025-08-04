package resolver

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/haukened/rr-dns/internal/dns/common/clock"
	"github.com/haukened/rr-dns/internal/dns/domain"
)

// Stub implementations for benchmarking (no overhead from mocking framework)
type stubBlocklist struct {
	blocked bool
}

func (s *stubBlocklist) IsBlocked(q domain.DNSQuery) bool {
	return s.blocked
}

type stubCache struct {
	records []domain.ResourceRecord
	found   bool
}

func (s *stubCache) Set(record []domain.ResourceRecord) error {
	return nil
}

func (s *stubCache) Get(key string) ([]domain.ResourceRecord, bool) {
	return s.records, s.found
}

func (s *stubCache) Delete(key string) {}

func (s *stubCache) Len() int {
	return 0
}

func (s *stubCache) Keys() []string {
	return nil
}

type stubUpstreamClient struct {
	response domain.DNSResponse
	err      error
}

func (s *stubUpstreamClient) Resolve(ctx context.Context, query domain.DNSQuery, now time.Time) (domain.DNSResponse, error) {
	return s.response, s.err
}

type stubZoneCache struct {
	records []domain.ResourceRecord
	found   bool
}

func (s *stubZoneCache) FindRecords(query domain.DNSQuery) ([]domain.ResourceRecord, bool) {
	return s.records, s.found
}

func (s *stubZoneCache) PutZone(zoneRoot string, records []domain.ResourceRecord) {}

func (s *stubZoneCache) RemoveZone(zoneRoot string) {}

func (s *stubZoneCache) Zones() []string {
	return nil
}

func (s *stubZoneCache) Count() int {
	return 0
}

type stubLogger struct{}

func (s *stubLogger) Info(map[string]any, string)  {}
func (s *stubLogger) Error(map[string]any, string) {}
func (s *stubLogger) Debug(map[string]any, string) {}
func (s *stubLogger) Warn(map[string]any, string)  {}
func (s *stubLogger) Panic(map[string]any, string) {}
func (s *stubLogger) Fatal(map[string]any, string) {}

func BenchmarkResolver_HandleQuery_AuthoritativeHit(b *testing.B) {
	// Setup authoritative record
	record := createTestRecord("example.com.", domain.RRType(1), []byte{192, 0, 2, 1})

	resolver := NewResolver(ResolverOptions{
		Blocklist:     &stubBlocklist{blocked: false},
		Clock:         &clock.MockClock{CurrentTime: time.Now()},
		Logger:        &stubLogger{},
		Upstream:      &stubUpstreamClient{},
		UpstreamCache: &stubCache{},
		ZoneCache:     &stubZoneCache{records: []domain.ResourceRecord{record}, found: true},
	})

	query := createTestQuery("example.com.", domain.RRType(1))
	ctx := context.Background()
	clientAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := resolver.HandleQuery(ctx, query, clientAddr)
		if err != nil {
			b.Fatalf("HandleQuery failed: %v", err)
		}
	}
}

func BenchmarkResolver_HandleQuery_UpstreamCacheHit(b *testing.B) {
	// Setup cached record
	record := createTestRecord("cached.com.", domain.RRType(1), []byte{192, 0, 2, 1})

	resolver := NewResolver(ResolverOptions{
		Blocklist:     &stubBlocklist{blocked: false},
		Clock:         &clock.MockClock{CurrentTime: time.Now()},
		Logger:        &stubLogger{},
		Upstream:      &stubUpstreamClient{},
		UpstreamCache: &stubCache{records: []domain.ResourceRecord{record}, found: true},
		ZoneCache:     &stubZoneCache{found: false},
	})

	query := createTestQuery("cached.com.", domain.RRType(1))
	ctx := context.Background()
	clientAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := resolver.HandleQuery(ctx, query, clientAddr)
		if err != nil {
			b.Fatalf("HandleQuery failed: %v", err)
		}
	}
}

func BenchmarkResolver_HandleQuery_BlocklistHit(b *testing.B) {
	resolver := NewResolver(ResolverOptions{
		Blocklist:     &stubBlocklist{blocked: true},
		Clock:         &clock.MockClock{CurrentTime: time.Now()},
		Logger:        &stubLogger{},
		Upstream:      &stubUpstreamClient{},
		UpstreamCache: &stubCache{},
		ZoneCache:     &stubZoneCache{found: false},
	})

	query := createTestQuery("blocked.com.", domain.RRType(1))
	ctx := context.Background()
	clientAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := resolver.HandleQuery(ctx, query, clientAddr)
		if err != nil {
			b.Fatalf("HandleQuery failed: %v", err)
		}
	}
}

func BenchmarkResolver_HandleQuery_UpstreamResolution(b *testing.B) {
	// Setup successful upstream response
	record := createTestRecord("upstream.com.", domain.RRType(1), []byte{192, 0, 2, 1})
	upstreamResp := domain.DNSResponse{
		ID:      1,
		RCode:   domain.NOERROR,
		Answers: []domain.ResourceRecord{record},
	}

	resolver := NewResolver(ResolverOptions{
		Blocklist:     &stubBlocklist{blocked: false},
		Clock:         &clock.MockClock{CurrentTime: time.Now()},
		Logger:        &stubLogger{},
		Upstream:      &stubUpstreamClient{response: upstreamResp},
		UpstreamCache: &stubCache{found: false}, // Cache miss
		ZoneCache:     &stubZoneCache{found: false},
	})

	query := createTestQuery("upstream.com.", domain.RRType(1))
	ctx := context.Background()
	clientAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := resolver.HandleQuery(ctx, query, clientAddr)
		if err != nil {
			b.Fatalf("HandleQuery failed: %v", err)
		}
	}
}

func BenchmarkResolver_HandleQuery_ConcurrentQueries(b *testing.B) {
	// Setup authoritative record for fast resolution
	record := createTestRecord("example.com.", domain.RRType(1), []byte{192, 0, 2, 1})

	resolver := NewResolver(ResolverOptions{
		Blocklist:     &stubBlocklist{blocked: false},
		Clock:         &clock.MockClock{CurrentTime: time.Now()},
		Logger:        &stubLogger{},
		Upstream:      &stubUpstreamClient{},
		UpstreamCache: &stubCache{},
		ZoneCache:     &stubZoneCache{records: []domain.ResourceRecord{record}, found: true},
	})

	query := createTestQuery("example.com.", domain.RRType(1))
	ctx := context.Background()
	clientAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := resolver.HandleQuery(ctx, query, clientAddr)
			if err != nil {
				b.Fatalf("HandleQuery failed: %v", err)
			}
		}
	})
}

func BenchmarkBuildResponse(b *testing.B) {
	query := createTestQuery("test.com.", domain.RRType(1))
	records := []domain.ResourceRecord{
		createTestRecord("test.com.", domain.RRType(1), []byte{192, 0, 2, 1}),
		createTestRecord("test.com.", domain.RRType(1), []byte{192, 0, 2, 2}),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = buildResponse(query, domain.NOERROR, records)
	}
}

func BenchmarkResolver_Construction(b *testing.B) {
	opts := ResolverOptions{
		Blocklist:     &stubBlocklist{blocked: false},
		Clock:         &clock.MockClock{CurrentTime: time.Now()},
		Logger:        &stubLogger{},
		Upstream:      &stubUpstreamClient{},
		UpstreamCache: &stubCache{},
		ZoneCache:     &stubZoneCache{},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = NewResolver(opts)
	}
}
