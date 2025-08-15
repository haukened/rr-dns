package lru

import (
	"testing"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

func TestDecisionCache_HitMissAndPut(t *testing.T) {
	c, err := New(2)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	d := domain.BlockDecision{Blocked: true, MatchedRule: "test"}

	if _, ok := c.Get("example.com."); ok {
		t.Fatalf("expected miss before put")
	}
	// miss increments
	if hits, misses, _ := c.Stats(); hits != 0 || misses != 1 {
		t.Fatalf("stats mismatch: hits=%d misses=%d", hits, misses)
	}

	c.Put("example.com.", d)

	got, ok := c.Get("example.com.")
	if !ok || !got.Blocked || got.MatchedRule != "test" {
		t.Fatalf("unexpected get: ok=%v got=%+v", ok, got)
	}
	// hit increments
	if hits, misses, _ := c.Stats(); hits != 1 || misses != 1 {
		t.Fatalf("stats mismatch after hit: hits=%d misses=%d", hits, misses)
	}
}

func TestDecisionCache_EvictionAndLen(t *testing.T) {
	c, err := New(2)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	c.Put("a.", domain.BlockDecision{Blocked: true})
	c.Put("b.", domain.BlockDecision{Blocked: true})
	if got := c.Len(); got != 2 {
		t.Fatalf("len=%d want=2", got)
	}
	// Adding a third should evict one
	c.Put("c.", domain.BlockDecision{Blocked: true})
	if got := c.Len(); got != 2 {
		t.Fatalf("len=%d want=2 after eviction", got)
	}
	// Give eviction callback time (though it's sync); verify evictions >= 1
	if _, _, ev := c.Stats(); ev < 1 {
		t.Fatalf("expected at least 1 eviction, got %d", ev)
	}
}

func TestDecisionCache_PurgeCountsEvictions(t *testing.T) {
	c, err := New(3)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	c.Put("a.", domain.BlockDecision{Blocked: true})
	c.Put("b.", domain.BlockDecision{Blocked: true})
	c.Put("c.", domain.BlockDecision{Blocked: true})

	c.Purge()
	if got := c.Len(); got != 0 {
		t.Fatalf("len=%d want=0 after purge", got)
	}
	// Purge triggers evictions for each item
	if _, _, ev := c.Stats(); ev < 3 {
		t.Fatalf("expected at least 3 evictions from purge, got %d", ev)
	}
}

func TestDecisionCache_Disabled(t *testing.T) {
	c, err := New(0)
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	// Always miss, no stats tracked
	if _, ok := c.Get("x."); ok {
		t.Fatalf("expected miss in disabled cache")
	}
	c.Put("x.", domain.BlockDecision{Blocked: true})
	if got := c.Len(); got != 0 {
		t.Fatalf("len=%d want=0 for disabled", got)
	}
	if hits, misses, ev := c.Stats(); hits != 0 || misses != 0 || ev != 0 {
		t.Fatalf("disabled stats not zero: %d/%d/%d", hits, misses, ev)
	}
}
