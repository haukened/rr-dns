package lru

import (
	"strconv"
	"testing"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

// Benchmark cache hit performance (Get on existing key).
func BenchmarkCache_PositiveHit(b *testing.B) {
	c, err := New(1024)
	if err != nil {
		b.Fatalf("New: %v", err)
	}
	key := "example.com"
	c.Put(key, domain.BlockDecision{Blocked: true, MatchedRule: key})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := c.Get(key); !ok {
			b.Fatalf("unexpected miss for key %q", key)
		}
	}
}

// Benchmark cache miss performance (Get on absent key).
func BenchmarkCache_NegativeMiss(b *testing.B) {
	c, err := New(1024)
	if err != nil {
		b.Fatalf("New: %v", err)
	}
	key := "absent.example"

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := c.Get(key); ok {
			b.Fatalf("unexpected hit for key %q", key)
		}
	}
}

// Validate LRU behavior under pressure: least recently used entries should be evicted.
func BenchmarkCache_LRUEviction(b *testing.B) {
	// Small cache to force evictions
	const cap = 3
	mkDecision := func(k string) domain.BlockDecision { return domain.BlockDecision{Blocked: true, MatchedRule: k} }

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		c, err := New(cap)
		if err != nil {
			b.Fatalf("New: %v", err)
		}
		// Fill A, B, C
		c.Put("A", mkDecision("A"))
		c.Put("B", mkDecision("B"))
		c.Put("C", mkDecision("C"))
		// Touch A and B to make C the least-recently-used
		if _, ok := c.Get("A"); !ok {
			b.Fatalf("miss on A")
		}
		if _, ok := c.Get("B"); !ok {
			b.Fatalf("miss on B")
		}
		// Insert D; expect C evicted
		c.Put("D", mkDecision("D"))

		if _, ok := c.Get("C"); ok {
			b.Fatalf("expected C to be evicted")
		}
		// A, B, D should be present
		if _, ok := c.Get("A"); !ok {
			b.Fatalf("A should be present")
		}
		if _, ok := c.Get("B"); !ok {
			b.Fatalf("B should be present")
		}
		if _, ok := c.Get("D"); !ok {
			b.Fatalf("D should be present")
		}
	}
}

// Optional: throughput for mixed workload (80% hits, 20% misses)
func BenchmarkCache_MixedHitRatio(b *testing.B) {
	c, err := New(10_000)
	if err != nil {
		b.Fatalf("New: %v", err)
	}
	// Preload 8k keys
	for i := 0; i < 8_000; i++ {
		k := "k" + strconv.Itoa(i)
		c.Put(k, domain.BlockDecision{Blocked: i%2 == 0, MatchedRule: k})
	}
	// Prepare miss keys outside the loaded range
	hitKey := func(i int) string { return "k" + strconv.Itoa(i%8_000) }
	missKey := func(i int) string { return "m" + strconv.Itoa(i) }

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%5 == 0 { // ~20% misses
			_, _ = c.Get(missKey(i))
		} else {
			_, _ = c.Get(hitKey(i))
		}
	}
}
