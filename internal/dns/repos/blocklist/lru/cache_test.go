package lru

import (
	"errors"
	"testing"

	lru "github.com/hashicorp/golang-lru/v2"
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
	// miss path exercised; no stats asserted in MVP

	c.Put("example.com.", d)

	got, ok := c.Get("example.com.")
	if !ok || !got.Blocked || got.MatchedRule != "test" {
		t.Fatalf("unexpected get: ok=%v got=%+v", ok, got)
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
	// We don't expose eviction counts in MVP; capacity remained fixed at 2
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
	// Purge empties the cache; no eviction stats in MVP
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
	// Disabled cache has no stats in MVP
}

func TestNewLRU_Error(t *testing.T) {
	originalLRU := newLRU
	newLRU = func(size int) (*lru.Cache[string, domain.BlockDecision], error) {
		return nil, errors.New("cache creation error") // Simulate an error
	}
	_, err := New(1)
	if err == nil {
		t.Fatalf("expected error but got nil")
	}
	newLRU = originalLRU
}
