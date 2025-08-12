package dnscache

import (
	"fmt"
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
		data := []byte{192, 0, 2, byte(i % 256)}
		text := fmt.Sprintf("%d.%d.%d.%d", data[0], data[1], data[2], data[3])
		rr, err := domain.NewCachedResourceRecord(
			"bench.com",
			domain.RRTypeFromString("A"),
			domain.RRClass(1),
			300,
			data,
			text,
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
		err := cache.Set([]domain.ResourceRecord{records[i]})
		if err != nil {
			b.Fatalf("failed to set record: %v", err)
		}
	}
}

func BenchmarkDnsCache_Get(b *testing.B) {
	cache, err := New(1000)
	if err != nil {
		b.Fatalf("failed to create cache: %v", err)
	}

	// Pre-populate cache
	rr, err := domain.NewCachedResourceRecord(
		"bench.com",
		domain.RRTypeFromString("A"),
		domain.RRClass(1),
		300,
		[]byte{192, 0, 2, 1},
		"192.0.2.1",
		time.Now(),
	)
	if err != nil {
		b.Fatalf("failed to create record: %v", err)
	}

	err = cache.Set([]domain.ResourceRecord{rr})
	if err != nil {
		b.Fatalf("failed to set record: %v", err)
	}
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
		data := []byte{192, 0, 2, byte(i + 1)}
		text := fmt.Sprintf("192.0.2.%d", i+1)
		rr, err := domain.NewCachedResourceRecord(
			"multi.com",
			domain.RRTypeFromString("A"),
			domain.RRClass(1),
			300,
			data,
			text,
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
		err = cache.Set(records)
		if err != nil {
			b.Fatalf("failed to set records: %v", err)
		}
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
		data := []byte{192, 0, 2, byte(i + 1)}
		text := fmt.Sprintf("192.0.2.%d", i+1)
		rr, err := domain.NewCachedResourceRecord(
			"multi.com",
			domain.RRTypeFromString("A"),
			domain.RRClass(1),
			300,
			data,
			text,
			time.Now(),
		)
		if err != nil {
			b.Fatalf("failed to create record: %v", err)
		}
		records[i] = rr
	}

	err = cache.Set(records)
	if err != nil {
		b.Fatalf("failed to set records: %v", err)
	}
	key := records[0].CacheKey()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(key)
	}
}
