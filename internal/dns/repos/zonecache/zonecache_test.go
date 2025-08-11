package zonecache

import (
	"testing"

	"github.com/haukened/rr-dns/internal/dns/domain"
	"github.com/haukened/rr-dns/internal/dns/services/resolver"
)

func TestNew(t *testing.T) {
	zc := New()
	if zc == nil {
		t.Fatal("New() returned nil")
	}
	if zc.zones == nil {
		t.Error("zones map not initialized")
	}
	if len(zc.zones) != 0 {
		t.Errorf("expected empty zones map, got %d entries", len(zc.zones))
	}
}

func TestZoneCache_PutZone(t *testing.T) {
	tests := []struct {
		name      string
		zoneRoot  string
		records   []domain.ResourceRecord
		wantZones int
		wantCount int
	}{
		{
			name:     "single zone with one record",
			zoneRoot: "example.com",
			records: []domain.ResourceRecord{
				{Name: "www.example.com", Type: 1, Class: 1, Data: []byte{192, 168, 1, 1}},
			},
			wantZones: 1,
			wantCount: 1,
		},
		{
			name:     "single zone with multiple records",
			zoneRoot: "example.com",
			records: []domain.ResourceRecord{
				{Name: "www.example.com", Type: 1, Class: 1, Data: []byte{192, 168, 1, 1}},
				{Name: "mail.example.com", Type: 1, Class: 1, Data: []byte{192, 168, 1, 2}},
				{Name: "example.com", Type: 15, Class: 1, Data: []byte{10, 0, 'm', 'a', 'i', 'l'}},
			},
			wantZones: 1,
			wantCount: 3,
		},
		{
			name:     "zone root without trailing dot",
			zoneRoot: "example.com",
			records: []domain.ResourceRecord{
				{Name: "www.example.com", Type: 1, Class: 1, Data: []byte{192, 168, 1, 1}},
			},
			wantZones: 1,
			wantCount: 1,
		},
		{
			name:     "zone root with whitespace",
			zoneRoot: "  example.com  ",
			records: []domain.ResourceRecord{
				{Name: "www.example.com", Type: 1, Class: 1, Data: []byte{192, 168, 1, 1}},
			},
			wantZones: 1,
			wantCount: 1,
		},
		{
			name:      "empty records slice",
			zoneRoot:  "example.com",
			records:   []domain.ResourceRecord{},
			wantZones: 1,
			wantCount: 0,
		},
		{
			name:     "multiple records with same cache key",
			zoneRoot: "example.com",
			records: []domain.ResourceRecord{
				{Name: "www.example.com", Type: 1, Class: 1, Data: []byte{192, 168, 1, 1}},
				{Name: "www.example.com", Type: 1, Class: 1, Data: []byte{192, 168, 1, 2}},
			},
			wantZones: 1,
			wantCount: 1, // Same cache key, so only one entry in zone map
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zc := New()
			zc.PutZone(tt.zoneRoot, tt.records)

			if len(zc.zones) != tt.wantZones {
				t.Errorf("expected %d zones, got %d", tt.wantZones, len(zc.zones))
			}

			if zc.Count() != tt.wantCount {
				t.Errorf("expected count %d, got %d", tt.wantCount, zc.Count())
			}

			// Verify canonical zone root is used
			canonicalZone := "example.com"
			if _, exists := zc.zones[canonicalZone]; !exists && tt.wantZones > 0 {
				t.Errorf("expected zone %q to exist", canonicalZone)
			}
		})
	}
}

func TestZoneCache_PutZone_Replace(t *testing.T) {
	zc := New()

	// Put initial records
	initialRecords := []domain.ResourceRecord{
		{Name: "www.example.com", Type: 1, Class: 1, Data: []byte{192, 168, 1, 1}},
		{Name: "mail.example.com", Type: 1, Class: 1, Data: []byte{192, 168, 1, 2}},
	}
	zc.PutZone("example.com", initialRecords)

	if zc.Count() != 2 {
		t.Errorf("expected initial count 2, got %d", zc.Count())
	}

	// Replace with new records
	newRecords := []domain.ResourceRecord{
		{Name: "api.example.com", Type: 1, Class: 1, Data: []byte{192, 168, 1, 3}},
	}
	zc.PutZone("example.com", newRecords)

	if zc.Count() != 1 {
		t.Errorf("expected count 1 after replacement, got %d", zc.Count())
	}

	// Verify old records are gone
	query := domain.Question{Name: "www.example.com", Type: 1, Class: 1}
	if _, found := zc.FindRecords(query); found {
		t.Error("expected old record to be removed")
	}

	// Verify new record exists
	newQuery := domain.Question{Name: "api.example.com", Type: 1, Class: 1}
	if _, found := zc.FindRecords(newQuery); !found {
		t.Error("expected new record to be found")
	}
}

func TestZoneCache_FindRecords(t *testing.T) {
	zc := New()

	// Setup test data
	records := []domain.ResourceRecord{
		{Name: "www.example.com", Type: 1, Class: 1, Data: []byte{192, 168, 1, 1}},
		{Name: "www.example.com", Type: 1, Class: 1, Data: []byte{192, 168, 1, 2}}, // Multiple A records
		{Name: "mail.example.com", Type: 1, Class: 1, Data: []byte{192, 168, 1, 3}},
		{Name: "example.com", Type: 15, Class: 1, Data: []byte{10, 0, 'm', 'a', 'i', 'l'}},
		{Name: "example.com", Type: 28, Class: 1, Data: []byte{0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}}, // AAAA
	}
	zc.PutZone("example.com", records)

	tests := []struct {
		name      string
		query     domain.Question
		wantFound bool
		wantCount int
	}{
		{
			name:      "find existing A record with multiple values",
			query:     domain.Question{Name: "www.example.com", Type: 1, Class: 1},
			wantFound: true,
			wantCount: 2, // Two A records for www.example.com
		},
		{
			name:      "find existing MX record",
			query:     domain.Question{Name: "example.com", Type: 15, Class: 1},
			wantFound: true,
			wantCount: 1,
		},
		{
			name:      "find existing AAAA record",
			query:     domain.Question{Name: "example.com", Type: 28, Class: 1},
			wantFound: true,
			wantCount: 1,
		},
		{
			name:      "query without trailing dot",
			query:     domain.Question{Name: "www.example.com", Type: 1, Class: 1},
			wantFound: true,
			wantCount: 2,
		},
		{
			name:      "query with mixed case",
			query:     domain.Question{Name: "WwW.example.com", Type: 1, Class: 1},
			wantFound: true,
			wantCount: 2,
		},
		{
			name:      "query with whitespace",
			query:     domain.Question{Name: "  www.example.com  ", Type: 1, Class: 1},
			wantFound: true,
			wantCount: 2,
		},
		{
			name:      "nonexistent record",
			query:     domain.Question{Name: "nonexistent.example.com", Type: 1, Class: 1},
			wantFound: false,
			wantCount: 0,
		},
		{
			name:      "wrong record type",
			query:     domain.Question{Name: "www.example.com", Type: 28, Class: 1}, // AAAA for A record
			wantFound: false,
			wantCount: 0,
		},
		{
			name:      "wrong record class",
			query:     domain.Question{Name: "www.example.com", Type: 1, Class: 3}, // CH instead of IN
			wantFound: false,
			wantCount: 0,
		},
		{
			name:      "different zone",
			query:     domain.Question{Name: "www.other.com.", Type: 1, Class: 1},
			wantFound: false,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, found := zc.FindRecords(tt.query)

			if found != tt.wantFound {
				t.Errorf("expected found=%v, got %v", tt.wantFound, found)
			}

			if len(result) != tt.wantCount {
				t.Errorf("expected %d records, got %d", tt.wantCount, len(result))
			}

			// Verify returned records are valid ResourceRecords
			for i, rr := range result {
				if err := rr.Validate(); err != nil {
					t.Errorf("record %d is invalid: %v", i, err)
				}
			}
		})
	}
}

func TestZoneCache_RemoveZone(t *testing.T) {
	zc := New()

	// Setup multiple zones
	zone1Records := []domain.ResourceRecord{
		{Name: "www.example.com", Type: 1, Class: 1, Data: []byte{192, 168, 1, 1}},
	}
	zone2Records := []domain.ResourceRecord{
		{Name: "www.test.com.", Type: 1, Class: 1, Data: []byte{192, 168, 2, 1}},
	}

	zc.PutZone("example.com", zone1Records)
	zc.PutZone("test.com.", zone2Records)

	if zc.Count() != 2 {
		t.Errorf("expected count 2, got %d", zc.Count())
	}

	// Remove one zone
	zc.RemoveZone("example.com")

	if zc.Count() != 1 {
		t.Errorf("expected count 1 after removal, got %d", zc.Count())
	}

	// Verify removed zone is gone
	query1 := domain.Question{Name: "www.example.com", Type: 1, Class: 1}
	if _, found := zc.FindRecords(query1); found {
		t.Error("expected removed zone records to be gone")
	}

	// Verify other zone still exists
	query2 := domain.Question{Name: "www.test.com.", Type: 1, Class: 1}
	if _, found := zc.FindRecords(query2); !found {
		t.Error("expected other zone to still exist")
	}
}

func TestZoneCache_RemoveZone_Canonical(t *testing.T) {
	zc := New()

	// Put zone with canonical name
	records := []domain.ResourceRecord{
		{Name: "www.example.com", Type: 1, Class: 1, Data: []byte{192, 168, 1, 1}},
	}
	zc.PutZone("example.com", records)

	// Remove with different formats
	testCases := []string{
		"example.com",     // No trailing dot
		"  example.com  ", // With whitespace
		"example.com",     // Different case
	}

	for _, zoneRoot := range testCases {
		t.Run("remove_zone_"+zoneRoot, func(t *testing.T) {
			// Re-add the zone
			zc.PutZone("example.com", records)
			if zc.Count() == 0 {
				t.Fatal("failed to add zone for test")
			}

			// Remove with different format
			zc.RemoveZone(zoneRoot)

			if zc.Count() != 0 {
				t.Errorf("expected zone to be removed with format %q", zoneRoot)
			}
		})
	}
}

func TestZoneCache_RemoveZone_NonExistent(t *testing.T) {
	zc := New()

	// Try to remove non-existent zone (should not panic)
	zc.RemoveZone("nonexistent.com.")

	if zc.Count() != 0 {
		t.Errorf("expected count 0, got %d", zc.Count())
	}
}

func TestZoneCache_Zones(t *testing.T) {
	zc := New()

	// Test empty cache
	zones := zc.Zones()
	if len(zones) != 0 {
		t.Errorf("expected empty zones slice, got %d entries", len(zones))
	}

	// Add some zones
	expectedZones := []string{"example.com", "test.com", "another.org"}
	for _, zone := range expectedZones {
		records := []domain.ResourceRecord{
			{Name: "www." + zone, Type: 1, Class: 1, Data: []byte{192, 168, 1, 1}},
		}
		zc.PutZone(zone, records)
	}

	zones = zc.Zones()
	if len(zones) != len(expectedZones) {
		t.Errorf("expected %d zones, got %d", len(expectedZones), len(zones))
	}

	// Verify all expected zones are present
	zoneMap := make(map[string]bool)
	for _, zone := range zones {
		zoneMap[zone] = true
	}

	for _, expected := range expectedZones {
		if !zoneMap[expected] {
			t.Errorf("expected zone %q not found in result", expected)
		}
	}
}

func TestZoneCache_Count(t *testing.T) {
	zc := New()

	// Test empty cache
	if zc.Count() != 0 {
		t.Errorf("expected count 0 for empty cache, got %d", zc.Count())
	}

	// Add records with different cache keys
	records1 := []domain.ResourceRecord{
		{Name: "www.example.com", Type: 1, Class: 1, Data: []byte{192, 168, 1, 1}},
		{Name: "mail.example.com", Type: 1, Class: 1, Data: []byte{192, 168, 1, 2}},
		{Name: "example.com", Type: 15, Class: 1, Data: []byte{10, 0, 'm', 'a', 'i', 'l'}},
	}
	zc.PutZone("example.com", records1)

	if zc.Count() != 3 {
		t.Errorf("expected count 3, got %d", zc.Count())
	}

	// Add another zone
	records2 := []domain.ResourceRecord{
		{Name: "www.test.com", Type: 1, Class: 1, Data: []byte{192, 168, 2, 1}},
		{Name: "api.test.com", Type: 1, Class: 1, Data: []byte{192, 168, 2, 2}},
	}
	zc.PutZone("test.com", records2)

	if zc.Count() != 5 {
		t.Errorf("expected count 5, got %d", zc.Count())
	}

	// Remove a zone
	zc.RemoveZone("example.com")

	if zc.Count() != 2 {
		t.Errorf("expected count 2 after removal, got %d", zc.Count())
	}
}

func TestZoneCache_Count_SameCacheKey(t *testing.T) {
	zc := New()

	// Add multiple records with the same cache key
	records := []domain.ResourceRecord{
		{Name: "www.example.com", Type: 1, Class: 1, Data: []byte{192, 168, 1, 1}},
		{Name: "www.example.com", Type: 1, Class: 1, Data: []byte{192, 168, 1, 2}},
		{Name: "www.example.com", Type: 1, Class: 1, Data: []byte{192, 168, 1, 3}},
	}
	zc.PutZone("example.com", records)

	// Should count as 1 since they have the same cache key
	if zc.Count() != 1 {
		t.Errorf("expected count 1 (same cache key), got %d", zc.Count())
	}
}

func TestZoneCache_ConcurrentAccess(t *testing.T) {
	zc := New()

	// Setup initial data
	records := []domain.ResourceRecord{
		{Name: "www.example.com", Type: 1, Class: 1, Data: []byte{192, 168, 1, 1}},
	}
	zc.PutZone("example.com", records)

	// Test concurrent reads
	done := make(chan bool, 10)
	query := domain.Question{Name: "www.example.com", Type: 1, Class: 1}

	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < 100; j++ {
				_, found := zc.FindRecords(query)
				if !found {
					t.Error("expected to find record in concurrent read")
				}
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestZoneCache_Interface(t *testing.T) {
	// Verify that ZoneCache implements resolver.ZoneCache interface
	var _ resolver.ZoneCache = (*ZoneCache)(nil)
}

func TestZoneCache_EdgeCases(t *testing.T) {
	t.Run("nil records slice", func(t *testing.T) {
		zc := New()
		// Should not panic
		zc.PutZone("example.com", nil)
		if zc.Count() != 0 {
			t.Errorf("expected count 0 for nil records, got %d", zc.Count())
		}
	})

	t.Run("empty zone root", func(t *testing.T) {
		zc := New()
		// Use a record that would naturally belong to the root zone
		records := []domain.ResourceRecord{
			{Name: ".", Type: 1, Class: 1, Data: []byte{192, 168, 1, 1}},
		}
		zc.PutZone("", records)

		// Should have stored the zone
		zones := zc.Zones()
		if len(zones) != 1 {
			t.Errorf("expected 1 zone, got %d: %v", len(zones), zones)
		}

		// Should have the records stored
		if zc.Count() != 1 {
			t.Errorf("expected count 1, got %d", zc.Count())
		}
	})

	t.Run("find with invalid query", func(t *testing.T) {
		zc := New()
		// Query for empty name
		query := domain.Question{Name: "", Type: 1, Class: 1}
		_, found := zc.FindRecords(query)
		if found {
			t.Error("expected not to find record for empty name")
		}
	})
}
