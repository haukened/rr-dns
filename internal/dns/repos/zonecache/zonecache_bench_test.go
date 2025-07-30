package zonecache

import (
	"testing"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

// Benchmark tests
func BenchmarkFind(b *testing.B) {
	cache := New()

	// Setup test data
	var records []*domain.AuthoritativeRecord
	for i := 0; i < 1000; i++ {
		record, _ := domain.NewAuthoritativeRecord("www.example.com.", 1, 1, 300, []byte{192, 0, 2, byte(i)})
		records = append(records, record)
	}
	cache.ReplaceZone("example.com", records)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Find("www.example.com.", 1)
	}
}

func BenchmarkReplaceZone(b *testing.B) {
	cache := New()

	// Setup test data
	var records []*domain.AuthoritativeRecord
	for i := 0; i < 100; i++ {
		record, _ := domain.NewAuthoritativeRecord("www.example.com.", 1, 1, 300, []byte{192, 0, 2, byte(i)})
		records = append(records, record)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.ReplaceZone("example.com", records)
	}
}

func BenchmarkCount(b *testing.B) {
	cache := New()

	// Setup test data
	var records []*domain.AuthoritativeRecord
	for i := 0; i < 1000; i++ {
		record, _ := domain.NewAuthoritativeRecord("www.example.com.", 1, 1, 300, []byte{192, 0, 2, byte(i)})
		records = append(records, record)
	}
	cache.ReplaceZone("example.com", records)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Count()
	}
}

func BenchmarkFind_Concurrent(b *testing.B) {
	cache := New()

	// Setup test data
	var records []*domain.AuthoritativeRecord
	for i := 0; i < 100; i++ {
		record, _ := domain.NewAuthoritativeRecord("www.example.com.", 1, 1, 300, []byte{192, 0, 2, byte(i)})
		records = append(records, record)
	}
	cache.ReplaceZone("example.com", records)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache.Find("www.example.com.", 1)
		}
	})
}

func BenchmarkReplaceZone_Concurrent(b *testing.B) {
	cache := New()

	// Setup test data
	var records []*domain.AuthoritativeRecord
	for i := 0; i < 10; i++ {
		record, _ := domain.NewAuthoritativeRecord("www.example.com.", 1, 1, 300, []byte{192, 0, 2, byte(i)})
		records = append(records, record)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		zoneCounter := 0
		for pb.Next() {
			zoneCounter++
			zoneName := "example" + string(rune(zoneCounter%10)) + ".com"
			cache.ReplaceZone(zoneName, records)
		}
	})
}
