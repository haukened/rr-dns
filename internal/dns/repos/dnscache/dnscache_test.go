package dnscache

import (
	"testing"
	"time"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

func TestInvalidCacheSize(t *testing.T) {
	_, err := New(-1)
	if err == nil {
		t.Errorf("expected error for negative cache size, got nil")
	}
}

func TestDnsCache_Get_ReturnsRecordIfNotExpired(t *testing.T) {
	cache, err := New(2)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	rr := &domain.ResourceRecord{Name: "example.com.", Type: 1, Class: 1, ExpiresAt: time.Now().Add(10 * time.Second)}
	err = cache.Set([]*domain.ResourceRecord{rr})
	if err != nil {
		t.Fatalf("failed to set record: %v", err)
	}

	got, ok := cache.Get(rr.CacheKey())
	if !ok {
		t.Fatalf("expected record to be found")
	}
	if len(got) != 1 || got[0] != rr {
		t.Errorf("expected [%v], got %v", rr, got)
	}
}

func TestDnsCache_Get_ReturnsFalseIfExpired(t *testing.T) {
	cache, err := New(2)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	rr := &domain.ResourceRecord{Name: "expired.com.", Type: 1, Class: 1, ExpiresAt: time.Now().Add(-1 * time.Second)}
	err = cache.Set([]*domain.ResourceRecord{rr}) // already expired
	if err != nil {
		t.Fatalf("failed to set record: %v", err)
	}

	got, ok := cache.Get(rr.CacheKey())
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
	rr1 := &domain.ResourceRecord{Name: "a.com.", Type: 1, Class: 1, ExpiresAt: time.Now().Add(1 * time.Minute)}
	rr2 := &domain.ResourceRecord{Name: "b.com.", Type: 1, Class: 1, ExpiresAt: time.Now().Add(1 * time.Minute)}
	rr3 := &domain.ResourceRecord{Name: "c.com.", Type: 1, Class: 1, ExpiresAt: time.Now().Add(1 * time.Minute)}

	err = cache.Set([]*domain.ResourceRecord{rr1})
	if err != nil {
		t.Fatalf("failed to set rr1: %v", err)
	}
	err = cache.Set([]*domain.ResourceRecord{rr2})
	if err != nil {
		t.Fatalf("failed to set rr2: %v", err)
	}
	err = cache.Set([]*domain.ResourceRecord{rr3})
	if err != nil {
		t.Fatalf("failed to set rr3: %v", err)
	}

	keys := cache.Keys()
	want := map[string]bool{
		"a.com.|a.com|A|IN": true,
		"b.com.|b.com|A|IN": true,
		"c.com.|c.com|A|IN": true,
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
	rr1 := &domain.ResourceRecord{Name: "expired.com.", Type: 1, Class: 1, ExpiresAt: time.Now().Add(-1 * time.Second)}
	rr2 := &domain.ResourceRecord{Name: "valid.com.", Type: 1, Class: 1, ExpiresAt: time.Now().Add(1 * time.Minute)}

	err = cache.Set([]*domain.ResourceRecord{rr1}) // expired
	if err != nil {
		t.Fatalf("failed to set rr1: %v", err)
	}
	err = cache.Set([]*domain.ResourceRecord{rr2}) // valid
	if err != nil {
		t.Fatalf("failed to set rr2: %v", err)
	}

	// Trigger eviction of expired by accessing it
	cache.Get(rr1.CacheKey())

	keys := cache.Keys()
	if len(keys) != 1 || keys[0] != "valid.com.|valid.com|A|IN" {
		t.Errorf("expected only 'valid.com.|valid.com|A|IN' in keys, got %v", keys)
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
	rr := &domain.ResourceRecord{Name: "delete.com.", Type: 1, Class: 1, ExpiresAt: time.Now().Add(1 * time.Minute)}
	err = cache.Set([]*domain.ResourceRecord{rr})
	if err != nil {
		t.Fatalf("failed to set record: %v", err)
	}

	cache.Delete(rr.CacheKey())

	got, ok := cache.Get(rr.CacheKey())
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
	rr1 := &domain.ResourceRecord{Name: "a.com.", Type: 1, Class: 1, ExpiresAt: time.Now().Add(1 * time.Minute)}
	rr2 := &domain.ResourceRecord{Name: "b.com.", Type: 1, Class: 1, ExpiresAt: time.Now().Add(1 * time.Minute)}
	err = cache.Set([]*domain.ResourceRecord{rr1})
	if err != nil {
		t.Fatalf("failed to set rr1: %v", err)
	}
	err = cache.Set([]*domain.ResourceRecord{rr2})
	if err != nil {
		t.Fatalf("failed to set rr2: %v", err)
	}

	cache.Delete(rr1.CacheKey())

	if _, ok := cache.Get(rr1.CacheKey()); ok {
		t.Errorf("expected 'a.com:A' to be deleted")
	}
	if _, ok := cache.Get(rr2.CacheKey()); !ok {
		t.Errorf("expected 'b.com:A' to remain")
	}
	if cache.Len() != 1 {
		t.Errorf("expected cache length 1, got %d", cache.Len())
	}
}

func TestDnsCache_SetZeroRecords(t *testing.T) {
	cache, err := New(2)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	err = cache.Set([]*domain.ResourceRecord{})
	if err != nil {
		t.Fatalf("failed to set zero records: %v", err)
	}
	if cache.Len() != 0 {
		t.Errorf("expected cache length 0, got %d", cache.Len())
	}
}

func TestDnsCache_SetWithDifferentKeys(t *testing.T) {
	cache, err := New(2)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	records := []*domain.ResourceRecord{
		{Name: "a.com.", Type: 1, Class: 1, ExpiresAt: time.Now().Add(1 * time.Minute)},
		{Name: "b.com.", Type: 1, Class: 1, ExpiresAt: time.Now().Add(1 * time.Minute)},
	}

	err = cache.Set(records)
	if err == nil {
		t.Errorf("expected error for multiple records with different keys, got nil")
	}
}
