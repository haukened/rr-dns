package memcache

import (
	"testing"
	"time"

	"github.com/haukened/udns/internal/dns/domain"
)

func TestDnsCache_Get_ReturnsRecordIfNotExpired(t *testing.T) {
	cache, err := New(2)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	key := "example.com:A"
	rr := &domain.ResourceRecord{Name: "example.com.", Type: 1, Class: 1, TTL: 60}
	cache.Set(key, rr, 1*time.Minute)

	got, ok := cache.Get(key)
	if !ok {
		t.Fatalf("expected record to be found")
	}
	if got != rr {
		t.Errorf("expected %v, got %v", rr, got)
	}
}

func TestDnsCache_Get_ReturnsFalseIfExpired(t *testing.T) {
	cache, err := New(2)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	key := "expired.com:A"
	rr := &domain.ResourceRecord{Name: "expired.com.", Type: 1, Class: 1, TTL: 1}
	cache.Set(key, rr, -1*time.Second) // already expired

	got, ok := cache.Get(key)
	if ok {
		t.Errorf("expected not found for expired record, got %v", got)
	}
	// Should be evicted after Get
	if cache.Len() != 0 {
		t.Errorf("expected cache to be empty after expired Get, got %d", cache.Len())
	}
}

func TestDnsCache_Get_ReturnsFalseIfNotPresent(t *testing.T) {
	cache, err := New(2)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	got, ok := cache.Get("missing.com:A")
	if ok {
		t.Errorf("expected not found for missing key, got %v", got)
	}
}

func TestDnsCache_Keys_ReturnsAllKeys(t *testing.T) {
	cache, err := New(3)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	rr1 := &domain.ResourceRecord{Name: "a.com.", Type: 1, Class: 1, TTL: 60}
	rr2 := &domain.ResourceRecord{Name: "b.com.", Type: 1, Class: 1, TTL: 60}
	rr3 := &domain.ResourceRecord{Name: "c.com.", Type: 1, Class: 1, TTL: 60}

	cache.Set("a.com:A", rr1, 1*time.Minute)
	cache.Set("b.com:A", rr2, 1*time.Minute)
	cache.Set("c.com:A", rr3, 1*time.Minute)

	keys := cache.Keys()
	want := map[string]bool{
		"a.com:A": true,
		"b.com:A": true,
		"c.com:A": true,
	}
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}
	for _, k := range keys {
		if !want[k] {
			t.Errorf("unexpected key: %s", k)
		}
	}
}

func TestDnsCache_Keys_ExcludesExpiredEntries(t *testing.T) {
	cache, err := New(2)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	rr1 := &domain.ResourceRecord{Name: "expired.com.", Type: 1, Class: 1, TTL: 1}
	rr2 := &domain.ResourceRecord{Name: "valid.com.", Type: 1, Class: 1, TTL: 60}

	cache.Set("expired.com:A", rr1, -1*time.Second) // expired
	cache.Set("valid.com:A", rr2, 1*time.Minute)

	// Trigger eviction of expired by accessing it
	cache.Get("expired.com:A")

	keys := cache.Keys()
	if len(keys) != 1 || keys[0] != "valid.com:A" {
		t.Errorf("expected only 'valid.com:A' in keys, got %v", keys)
	}
}

func TestDnsCache_Keys_EmptyWhenNoEntries(t *testing.T) {
	cache, err := New(2)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	keys := cache.Keys()
	if len(keys) != 0 {
		t.Errorf("expected no keys, got %v", keys)
	}
}

func TestDnsCache_Delete_RemovesEntry(t *testing.T) {
	cache, err := New(2)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	key := "delete.com:A"
	rr := &domain.ResourceRecord{Name: "delete.com.", Type: 1, Class: 1, TTL: 60}
	cache.Set(key, rr, 1*time.Minute)

	cache.Delete(key)

	got, ok := cache.Get(key)
	if ok {
		t.Errorf("expected record to be deleted, got %v", got)
	}
	if cache.Len() != 0 {
		t.Errorf("expected cache to be empty after delete, got %d", cache.Len())
	}
}

func TestDnsCache_Delete_NonExistentKey_NoPanic(t *testing.T) {
	cache, err := New(2)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	// Should not panic or error
	cache.Delete("nonexistent.com:A")
	// Cache should still be empty
	if cache.Len() != 0 {
		t.Errorf("expected cache to be empty, got %d", cache.Len())
	}
}

func TestDnsCache_Delete_OnlyDeletesSpecifiedKey(t *testing.T) {
	cache, err := New(3)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	rr1 := &domain.ResourceRecord{Name: "a.com.", Type: 1, Class: 1, TTL: 60}
	rr2 := &domain.ResourceRecord{Name: "b.com.", Type: 1, Class: 1, TTL: 60}
	cache.Set("a.com:A", rr1, 1*time.Minute)
	cache.Set("b.com:A", rr2, 1*time.Minute)

	cache.Delete("a.com:A")

	if _, ok := cache.Get("a.com:A"); ok {
		t.Errorf("expected 'a.com:A' to be deleted")
	}
	if _, ok := cache.Get("b.com:A"); !ok {
		t.Errorf("expected 'b.com:A' to remain")
	}
	if cache.Len() != 1 {
		t.Errorf("expected cache length 1, got %d", cache.Len())
	}
}
