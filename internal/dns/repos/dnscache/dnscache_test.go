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

	rr, err := domain.NewCachedResourceRecord(
		"example.com",
		domain.RRTypeFromString("A"),
		domain.RRClass(1),
		10, // 10 second TTL
		[]byte{192, 0, 2, 1},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to create resource record: %v", err)
	}

	err = cache.Set([]domain.ResourceRecord{rr})
	if err != nil {
		t.Fatalf("failed to set record: %v", err)
	}

	got, ok := cache.Get(rr.CacheKey())
	if !ok {
		t.Fatalf("expected record to be found")
	}
	if len(got) != 1 {
		t.Errorf("expected 1 record, got %d", len(got))
	}
	if got[0].Name != rr.Name || got[0].Type != rr.Type {
		t.Errorf("expected record %+v, got %+v", rr, got[0])
	}
}

func TestDnsCache_Get_ReturnsFalseIfExpired(t *testing.T) {
	cache, err := New(2)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	// Create expired record by setting timestamp in the past
	rr, err := domain.NewCachedResourceRecord(
		"expired.com",
		domain.RRTypeFromString("A"),
		domain.RRClass(1),
		1, // 1 second TTL
		[]byte{192, 0, 2, 1},
		time.Now().Add(-2*time.Second), // created 2 seconds ago, so already expired
	)
	if err != nil {
		t.Fatalf("failed to create resource record: %v", err)
	}

	err = cache.Set([]domain.ResourceRecord{rr}) // already expired
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
	got, ok := cache.Get("missing.com|missing.com|A|IN")
	if ok {
		t.Errorf("expected not found for missing key, got %v", got)
	}
}

func TestDnsCache_Keys_ReturnsAllKeys(t *testing.T) {
	cache, err := New(3)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	rr1, err := domain.NewCachedResourceRecord(
		"a.com",
		domain.RRTypeFromString("A"),
		domain.RRClass(1),
		60, // 1 minute TTL
		[]byte{192, 0, 2, 1},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to create rr1: %v", err)
	}

	rr2, err := domain.NewCachedResourceRecord(
		"b.com",
		domain.RRTypeFromString("A"),
		domain.RRClass(1),
		60, // 1 minute TTL
		[]byte{192, 0, 2, 2},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to create rr2: %v", err)
	}

	rr3, err := domain.NewCachedResourceRecord(
		"c.com",
		domain.RRTypeFromString("A"),
		domain.RRClass(1),
		60, // 1 minute TTL
		[]byte{192, 0, 2, 3},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to create rr3: %v", err)
	}

	err = cache.Set([]domain.ResourceRecord{rr1})
	if err != nil {
		t.Fatalf("failed to set rr1: %v", err)
	}
	err = cache.Set([]domain.ResourceRecord{rr2})
	if err != nil {
		t.Fatalf("failed to set rr2: %v", err)
	}
	err = cache.Set([]domain.ResourceRecord{rr3})
	if err != nil {
		t.Fatalf("failed to set rr3: %v", err)
	}

	keys := cache.Keys()
	want := map[string]bool{
		"a.com|a.com|A|IN": true,
		"b.com|b.com|A|IN": true,
		"c.com|c.com|A|IN": true,
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

	// Create expired record
	rr1, err := domain.NewCachedResourceRecord(
		"expired.com",
		domain.RRTypeFromString("A"),
		domain.RRClass(1),
		1, // 1 second TTL
		[]byte{192, 0, 2, 1},
		time.Now().Add(-2*time.Second), // created 2 seconds ago, so already expired
	)
	if err != nil {
		t.Fatalf("failed to create rr1: %v", err)
	}

	// Create valid record
	rr2, err := domain.NewCachedResourceRecord(
		"valid.com",
		domain.RRTypeFromString("A"),
		domain.RRClass(1),
		60, // 1 minute TTL
		[]byte{192, 0, 2, 2},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to create rr2: %v", err)
	}

	err = cache.Set([]domain.ResourceRecord{rr1}) // expired
	if err != nil {
		t.Fatalf("failed to set rr1: %v", err)
	}
	err = cache.Set([]domain.ResourceRecord{rr2}) // valid
	if err != nil {
		t.Fatalf("failed to set rr2: %v", err)
	}

	// Trigger eviction of expired by accessing it
	cache.Get(rr1.CacheKey())

	keys := cache.Keys()
	if len(keys) != 1 || keys[0] != "valid.com|valid.com|A|IN" {
		t.Errorf("expected only 'valid.com|valid.com|A|IN' in keys, got %v", keys)
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

	rr, err := domain.NewCachedResourceRecord(
		"delete.com",
		domain.RRTypeFromString("A"),
		domain.RRClass(1),
		60, // 1 minute TTL
		[]byte{192, 0, 2, 1},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to create resource record: %v", err)
	}

	err = cache.Set([]domain.ResourceRecord{rr})
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
	cache.Delete("nonexistent.com|nonexistent.com|A|IN")
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

	rr1, err := domain.NewCachedResourceRecord(
		"a.com",
		domain.RRTypeFromString("A"),
		domain.RRClass(1),
		60, // 1 minute TTL
		[]byte{192, 0, 2, 1},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to create rr1: %v", err)
	}

	rr2, err := domain.NewCachedResourceRecord(
		"b.com",
		domain.RRTypeFromString("A"),
		domain.RRClass(1),
		60, // 1 minute TTL
		[]byte{192, 0, 2, 2},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to create rr2: %v", err)
	}

	err = cache.Set([]domain.ResourceRecord{rr1})
	if err != nil {
		t.Fatalf("failed to set rr1: %v", err)
	}
	err = cache.Set([]domain.ResourceRecord{rr2})
	if err != nil {
		t.Fatalf("failed to set rr2: %v", err)
	}

	cache.Delete(rr1.CacheKey())

	if _, ok := cache.Get(rr1.CacheKey()); ok {
		t.Errorf("expected rr1 to be deleted")
	}
	if _, ok := cache.Get(rr2.CacheKey()); !ok {
		t.Errorf("expected rr2 to remain")
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
	err = cache.Set([]domain.ResourceRecord{})
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

	rr1, err := domain.NewCachedResourceRecord(
		"a.com",
		domain.RRTypeFromString("A"),
		domain.RRClass(1),
		60, // 1 minute TTL
		[]byte{192, 0, 2, 1},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to create rr1: %v", err)
	}

	rr2, err := domain.NewCachedResourceRecord(
		"b.com",
		domain.RRTypeFromString("A"),
		domain.RRClass(1),
		60, // 1 minute TTL
		[]byte{192, 0, 2, 2},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to create rr2: %v", err)
	}

	records := []domain.ResourceRecord{rr1, rr2}

	err = cache.Set(records)
	if err == nil {
		t.Errorf("expected error for multiple records with different keys, got nil")
	}
	if err != ErrMultipleKeys {
		t.Errorf("expected ErrMultipleKeys, got %v", err)
	}
}

func TestDnsCache_SetMultipleRecordsSameKey(t *testing.T) {
	cache, err := New(2)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	// Create multiple A records for the same domain
	rr1, err := domain.NewCachedResourceRecord(
		"multi.com",
		domain.RRTypeFromString("A"),
		domain.RRClass(1),
		60, // 1 minute TTL
		[]byte{192, 0, 2, 1},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to create rr1: %v", err)
	}

	rr2, err := domain.NewCachedResourceRecord(
		"multi.com",
		domain.RRTypeFromString("A"),
		domain.RRClass(1),
		60, // 1 minute TTL
		[]byte{192, 0, 2, 2},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to create rr2: %v", err)
	}

	records := []domain.ResourceRecord{rr1, rr2}

	err = cache.Set(records)
	if err != nil {
		t.Fatalf("failed to set multiple records with same key: %v", err)
	}

	got, ok := cache.Get(rr1.CacheKey())
	if !ok {
		t.Fatalf("expected records to be found")
	}
	if len(got) != 2 {
		t.Errorf("expected 2 records, got %d", len(got))
	}
}

func TestDnsCache_Len(t *testing.T) {
	cache, err := New(3)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	if cache.Len() != 0 {
		t.Errorf("expected empty cache length 0, got %d", cache.Len())
	}

	rr1, err := domain.NewCachedResourceRecord(
		"test1.com",
		domain.RRTypeFromString("A"),
		domain.RRClass(1),
		60,
		[]byte{192, 0, 2, 1},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to create rr1: %v", err)
	}

	err = cache.Set([]domain.ResourceRecord{rr1})
	if err != nil {
		t.Fatalf("failed to set rr1: %v", err)
	}

	// After setting one record, cache length should be 1
	if cache.Len() != 1 {
		t.Errorf("expected cache length 1, got %d", cache.Len())
	}

	rr2, err := domain.NewCachedResourceRecord(
		"test2.com",
		domain.RRTypeFromString("A"),
		domain.RRClass(1),
		60,
		[]byte{192, 0, 2, 2},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("failed to create rr2: %v", err)
	}

	err = cache.Set([]domain.ResourceRecord{rr2})
	if err != nil {
		t.Fatalf("failed to set rr2: %v", err)
	}
	if cache.Len() != 2 {
		t.Errorf("expected cache length 2, got %d", cache.Len())
	}
}

func TestDnsCache_FilterExpiredRecords(t *testing.T) {
	cache, err := New(2)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	now := time.Now()

	// Create mix of expired and valid records with same cache key
	rr1, err := domain.NewCachedResourceRecord(
		"mixed.com",
		domain.RRTypeFromString("A"),
		domain.RRClass(1),
		1, // 1 second TTL
		[]byte{192, 0, 2, 1},
		now.Add(-2*time.Second), // expired
	)
	if err != nil {
		t.Fatalf("failed to create rr1: %v", err)
	}

	rr2, err := domain.NewCachedResourceRecord(
		"mixed.com",
		domain.RRTypeFromString("A"),
		domain.RRClass(1),
		60, // 1 minute TTL
		[]byte{192, 0, 2, 2},
		now, // valid
	)
	if err != nil {
		t.Fatalf("failed to create rr2: %v", err)
	}

	records := []domain.ResourceRecord{rr1, rr2}
	err = cache.Set(records)
	if err != nil {
		t.Fatalf("failed to set records: %v", err)
	}

	// Get should filter out expired records
	got, ok := cache.Get(rr1.CacheKey())
	if !ok {
		t.Fatalf("expected to find valid records")
	}
	if len(got) != 1 {
		t.Errorf("expected 1 valid record after filtering, got %d", len(got))
	}
	// The remaining record should be the valid one
	if len(got[0].Data) == 0 || got[0].Data[3] != 2 {
		t.Errorf("expected valid record with IP ending in .2, got %v", got[0].Data)
	}
}
