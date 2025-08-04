package dnscache

import (
	"testing"
	"time"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

func BenchmarkDnsCache_Set(b *testing.B) {
	cache, err := New(1000)
	if err != nil {
		b.Fatalf("failed to create cache: %v", err)
	}

	// Pre-create records for benchmarking
	records := make([]domain.ResourceRecord, b.N)
	for i := 0; i < b.N; i++ {
		rr, err := domain.NewCachedResourceRecord(
			"bench.com.",
			domain.RRTypeFromString("A"),
			domain.RRClass(1),
			300,
			[]byte{192, 0, 2, byte(i % 256)},
			time.Now(),
		)
		if err != nil {
			b.Fatalf("failed to create record: %v", err)
		}
		records[i] = rr
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache.Set([]domain.ResourceRecord{records[i]})
	}
}

func BenchmarkDnsCache_Get(b *testing.B) {
	cache, err := New(1000)
	if err != nil {
		b.Fatalf("failed to create cache: %v", err)
	}

	// Pre-populate cache
	rr, err := domain.NewCachedResourceRecord(
		"bench.com.",
		domain.RRTypeFromString("A"),
		domain.RRClass(1),
		300,
		[]byte{192, 0, 2, 1},
		time.Now(),
	)
	if err != nil {
		b.Fatalf("failed to create record: %v", err)
	}

	cache.Set([]domain.ResourceRecord{rr})
	key := rr.CacheKey()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(key)
	}
}

func BenchmarkDnsCache_SetMultiple(b *testing.B) {
	cache, err := New(100)
	if err != nil {
		b.Fatalf("failed to create cache: %v", err)
	}

	// Create multiple records with same key
	records := make([]domain.ResourceRecord, 5)
	for i := 0; i < 5; i++ {
		rr, err := domain.NewCachedResourceRecord(
			"multi.com.",
			domain.RRTypeFromString("A"),
			domain.RRClass(1),
			300,
			[]byte{192, 0, 2, byte(i + 1)},
			time.Now(),
		)
		if err != nil {
			b.Fatalf("failed to create record: %v", err)
		}
		records[i] = rr
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cache.Set(records)
	}
}

func BenchmarkDnsCache_GetMultiple(b *testing.B) {
	cache, err := New(100)
	if err != nil {
		b.Fatalf("failed to create cache: %v", err)
	}

	// Pre-populate with multiple records
	records := make([]domain.ResourceRecord, 5)
	for i := 0; i < 5; i++ {
		rr, err := domain.NewCachedResourceRecord(
			"multi.com.",
			domain.RRTypeFromString("A"),
			domain.RRClass(1),
			300,
			[]byte{192, 0, 2, byte(i + 1)},
			time.Now(),
		)
		if err != nil {
			b.Fatalf("failed to create record: %v", err)
		}
		records[i] = rr
	}

	cache.Set(records)
	key := records[0].CacheKey()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(key)
	}
}
