package zonecache

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

func TestFind(t *testing.T) {
	cache := New()

	// Create test records
	record1, _ := domain.NewAuthoritativeRecord("www.example.com.", 1, 1, 300, []byte{192, 0, 2, 1})             // A record
	record2, _ := domain.NewAuthoritativeRecord("www.example.com.", 1, 1, 300, []byte{192, 0, 2, 2})             // A record
	record3, _ := domain.NewAuthoritativeRecord("mail.example.com.", 15, 1, 300, []byte("10 mail.example.com.")) // MX record

	records := []*domain.AuthoritativeRecord{record1, record2, record3}
	err := cache.ReplaceZone("example.com", records)
	assert.NoError(t, err)

	tests := []struct {
		name     string
		fqdn     string
		rrType   domain.RRType
		wantLen  int
		wantFind bool
	}{
		{
			name:     "find A records for www.example.com",
			fqdn:     "www.example.com.",
			rrType:   1, // A
			wantLen:  2,
			wantFind: true,
		},
		{
			name:     "find MX record for mail.example.com",
			fqdn:     "mail.example.com.",
			rrType:   15, // MX
			wantLen:  1,
			wantFind: true,
		},
		{
			name:     "find non-existent AAAA record",
			fqdn:     "www.example.com.",
			rrType:   28, // AAAA
			wantLen:  0,
			wantFind: false,
		},
		{
			name:     "find record for non-existent domain",
			fqdn:     "nonexistent.example.com.",
			rrType:   1, // A
			wantLen:  0,
			wantFind: false,
		},
		{
			name:     "find record for domain in different zone",
			fqdn:     "www.other.com.",
			rrType:   1, // A
			wantLen:  0,
			wantFind: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := domain.DNSQuery{Name: tt.fqdn, Type: tt.rrType}
			records, found := cache.Find(query)

			assert.Equal(t, tt.wantFind, found, "unexpected found result")
			assert.Equal(t, tt.wantLen, len(records), "unexpected number of records")

			if tt.wantFind && tt.wantLen > 0 {
				for _, record := range records {
					assert.Equal(t, tt.fqdn, record.Name, "record name should match query")
					assert.Equal(t, tt.rrType, record.Type, "record type should match query")
				}
			}
		})
	}
}

func TestReplaceZone(t *testing.T) {
	cache := New()

	// Create initial records
	record1, _ := domain.NewAuthoritativeRecord("www.example.com.", 1, 1, 300, []byte{192, 0, 2, 1})
	record2, _ := domain.NewAuthoritativeRecord("mail.example.com.", 15, 1, 300, []byte("10 mail.example.com."))

	initialRecords := []*domain.AuthoritativeRecord{record1, record2}
	err := cache.ReplaceZone("example.com", initialRecords)
	assert.NoError(t, err)

	// Verify initial records exist
	records, found := cache.Find(domain.DNSQuery{Name: "www.example.com.", Type: 1})
	assert.True(t, found)
	assert.Len(t, records, 1)

	// Replace with new records
	record3, _ := domain.NewAuthoritativeRecord("api.example.com.", 1, 1, 300, []byte{192, 0, 2, 3})
	record4, _ := domain.NewAuthoritativeRecord("db.example.com.", 1, 1, 300, []byte{192, 0, 2, 4})

	newRecords := []*domain.AuthoritativeRecord{record3, record4}
	err = cache.ReplaceZone("example.com", newRecords)
	assert.NoError(t, err)

	// Verify old records are gone
	records, found = cache.Find(domain.DNSQuery{Name: "www.example.com.", Type: 1})
	assert.False(t, found)
	assert.Len(t, records, 0)

	records, found = cache.Find(domain.DNSQuery{Name: "mail.example.com.", Type: 15})
	assert.False(t, found)

	// Verify new records exist
	records, found = cache.Find(domain.DNSQuery{Name: "api.example.com.", Type: 1})
	assert.True(t, found)
	assert.Len(t, records, 1)

	records, found = cache.Find(domain.DNSQuery{Name: "db.example.com.", Type: 1})
	assert.True(t, found)
	assert.Len(t, records, 1)
}

func TestRemoveZone(t *testing.T) {
	cache := New()

	// Create test records for multiple zones
	record1, _ := domain.NewAuthoritativeRecord("www.example.com.", 1, 1, 300, []byte{192, 0, 2, 1})
	record2, _ := domain.NewAuthoritativeRecord("mail.example.com.", 15, 1, 300, []byte("10 mail.example.com."))
	record3, _ := domain.NewAuthoritativeRecord("www.test.com.", 1, 1, 300, []byte{192, 0, 2, 2})

	exampleRecords := []*domain.AuthoritativeRecord{record1, record2}
	testRecords := []*domain.AuthoritativeRecord{record3}

	err := cache.ReplaceZone("example.com", exampleRecords)
	assert.NoError(t, err)
	err = cache.ReplaceZone("test.com", testRecords)
	assert.NoError(t, err)

	// Verify records exist before removal
	records, found := cache.Find(domain.DNSQuery{Name: "www.example.com.", Type: 1})
	assert.True(t, found)
	assert.Len(t, records, 1)

	records, found = cache.Find(domain.DNSQuery{Name: "www.test.com.", Type: 1})
	assert.True(t, found)
	assert.Len(t, records, 1)

	// Remove one zone
	err = cache.RemoveZone("example.com")
	assert.NoError(t, err)

	// Verify example.com records are gone
	_, found = cache.Find(domain.DNSQuery{Name: "www.example.com.", Type: 1})
	assert.False(t, found)

	_, found = cache.Find(domain.DNSQuery{Name: "mail.example.com.", Type: 15})
	assert.False(t, found)

	// Verify test.com records still exist
	records, found = cache.Find(domain.DNSQuery{Name: "www.test.com.", Type: 1})
	assert.True(t, found)
	assert.Len(t, records, 1)
}

func TestConcurrentAccess(t *testing.T) {
	cache := New()

	// Prepare test data
	record1, _ := domain.NewAuthoritativeRecord("www.example.com.", 1, 1, 300, []byte{192, 0, 2, 1})
	records := []*domain.AuthoritativeRecord{record1}

	var wg sync.WaitGroup

	// Test concurrent reads and writes
	for i := 0; i < 10; i++ {
		wg.Add(3)

		// Concurrent reader
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				cache.Find(domain.DNSQuery{Name: "www.example.com.", Type: 1})
			}
		}()

		// Concurrent writer (replace)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				testRecord, _ := domain.NewAuthoritativeRecord("test.example.com.", 1, 1, 300, []byte{192, 0, 2, byte(id)})
				testRecords := []*domain.AuthoritativeRecord{testRecord}
				cache.ReplaceZone("example.com", testRecords)
			}
		}(i)

		// Concurrent remover
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				cache.RemoveZone("example.com")
				cache.ReplaceZone("example.com", records)
			}
		}()
	}

	wg.Wait()
	// Test should complete without race conditions or panics
}

// Test error paths for ReplaceZone
func TestReplaceZone_ErrorPaths(t *testing.T) {
	cache := New()

	tests := []struct {
		name      string
		zoneRoot  string
		wantError error
	}{
		{
			name:      "empty zone root",
			zoneRoot:  "",
			wantError: ErrInvalidZoneRoot,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cache.ReplaceZone(tt.zoneRoot, nil)
			assert.Equal(t, tt.wantError, err)
		})
	}
}

// Test error paths for RemoveZone
func TestRemoveZone_ErrorPaths(t *testing.T) {
	cache := New()

	tests := []struct {
		name      string
		zoneRoot  string
		wantError error
	}{
		{
			name:      "empty zone root",
			zoneRoot:  "",
			wantError: ErrInvalidZoneRoot,
		},
		{
			name:      "non-existent zone",
			zoneRoot:  "nonexistent.com",
			wantError: ErrZoneNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cache.RemoveZone(tt.zoneRoot)
			assert.Equal(t, tt.wantError, err)
		})
	}
}

// Test zone root normalization (adding trailing dot)
func TestZoneRootNormalization(t *testing.T) {
	cache := New()

	record1, _ := domain.NewAuthoritativeRecord("www.example.com.", 1, 1, 300, []byte{192, 0, 2, 1})
	records := []*domain.AuthoritativeRecord{record1}

	// Test ReplaceZone with zone root without trailing dot
	err := cache.ReplaceZone("example.com", records)
	assert.NoError(t, err)

	// Verify record can be found
	foundRecords, found := cache.Find(domain.DNSQuery{Name: "www.example.com.", Type: 1})
	assert.True(t, found)
	assert.Len(t, foundRecords, 1)

	// Test RemoveZone with zone root without trailing dot
	err = cache.RemoveZone("example.com")
	assert.NoError(t, err)

	// Verify record is gone
	_, found = cache.Find(domain.DNSQuery{Name: "www.example.com.", Type: 1})
	assert.False(t, found)
}

// Test FQDN normalization (adding trailing dot)
func TestFQDNNormalization(t *testing.T) {
	cache := New()

	// Create record with FQDN without trailing dot
	record1, _ := domain.NewAuthoritativeRecord("www.example.com", 1, 1, 300, []byte{192, 0, 2, 1})
	records := []*domain.AuthoritativeRecord{record1}

	err := cache.ReplaceZone("example.com", records)
	assert.NoError(t, err)

	// Should be able to find with trailing dot
	foundRecords, found := cache.Find(domain.DNSQuery{Name: "www.example.com.", Type: 1})
	assert.True(t, found)
	assert.Len(t, foundRecords, 1)
}

// Test empty cache behavior
func TestEmptyCache(t *testing.T) {
	cache := New()

	// Test Find on empty cache
	records, found := cache.Find(domain.DNSQuery{Name: "www.example.com.", Type: 1})
	assert.False(t, found)
	assert.Len(t, records, 0)

	// Test All on empty cache
	allRecords := cache.All()
	assert.Empty(t, allRecords)

	// Test Zones on empty cache
	zones := cache.Zones()
	assert.Empty(t, zones)

	// Test Count on empty cache
	count := cache.Count()
	assert.Equal(t, 0, count)
}

// Test ReplaceZone with empty records slice
func TestReplaceZone_EmptyRecords(t *testing.T) {
	cache := New()

	// First add some records
	record1, _ := domain.NewAuthoritativeRecord("www.example.com.", 1, 1, 300, []byte{192, 0, 2, 1})
	records := []*domain.AuthoritativeRecord{record1}
	err := cache.ReplaceZone("example.com", records)
	assert.NoError(t, err)

	// Verify record exists
	foundRecords, found := cache.Find(domain.DNSQuery{Name: "www.example.com.", Type: 1})
	assert.True(t, found)
	assert.Len(t, foundRecords, 1)

	// Replace with empty slice
	err = cache.ReplaceZone("example.com", []*domain.AuthoritativeRecord{})
	assert.NoError(t, err)

	// Verify records are gone
	foundRecords, found = cache.Find(domain.DNSQuery{Name: "www.example.com.", Type: 1})
	assert.False(t, found)
	assert.Len(t, foundRecords, 0)

	// But zone should still exist (empty)
	zones := cache.Zones()
	assert.Contains(t, zones, "example.com.")
}

// Test All method with multiple zones
func TestAll_MultipleZones(t *testing.T) {
	cache := New()

	// Add records to multiple zones
	record1, _ := domain.NewAuthoritativeRecord("www.example.com.", 1, 1, 300, []byte{192, 0, 2, 1})
	record2, _ := domain.NewAuthoritativeRecord("mail.example.com.", 15, 1, 300, []byte("10 mail.example.com."))
	record3, _ := domain.NewAuthoritativeRecord("www.test.com.", 1, 1, 300, []byte{192, 0, 2, 2})

	err := cache.ReplaceZone("example.com", []*domain.AuthoritativeRecord{record1, record2})
	assert.NoError(t, err)
	err = cache.ReplaceZone("test.com", []*domain.AuthoritativeRecord{record3})
	assert.NoError(t, err)

	allRecords := cache.All()
	assert.Len(t, allRecords, 2)
	assert.Contains(t, allRecords, "example.com.")
	assert.Contains(t, allRecords, "test.com.")
	assert.Len(t, allRecords["example.com."], 2)
	assert.Len(t, allRecords["test.com."], 1)
}

// Test Zones method
func TestZones(t *testing.T) {
	cache := New()

	// Initially empty
	zones := cache.Zones()
	assert.Empty(t, zones)

	// Add zones
	record1, _ := domain.NewAuthoritativeRecord("www.example.com.", 1, 1, 300, []byte{192, 0, 2, 1})
	record2, _ := domain.NewAuthoritativeRecord("www.test.com.", 1, 1, 300, []byte{192, 0, 2, 2})

	err := cache.ReplaceZone("example.com", []*domain.AuthoritativeRecord{record1})
	assert.NoError(t, err)
	err = cache.ReplaceZone("test.com", []*domain.AuthoritativeRecord{record2})
	assert.NoError(t, err)

	zones = cache.Zones()
	assert.Len(t, zones, 2)
	assert.Contains(t, zones, "example.com.")
	assert.Contains(t, zones, "test.com.")
}

// Test Count method
func TestCount(t *testing.T) {
	cache := New()

	// Initially zero
	count := cache.Count()
	assert.Equal(t, 0, count)

	// Add records
	record1, _ := domain.NewAuthoritativeRecord("www.example.com.", 1, 1, 300, []byte{192, 0, 2, 1})
	record2, _ := domain.NewAuthoritativeRecord("mail.example.com.", 15, 1, 300, []byte("10 mail.example.com."))
	record3, _ := domain.NewAuthoritativeRecord("www.test.com.", 1, 1, 300, []byte{192, 0, 2, 2})

	err := cache.ReplaceZone("example.com", []*domain.AuthoritativeRecord{record1, record2})
	assert.NoError(t, err)
	count = cache.Count()
	assert.Equal(t, 2, count)

	err = cache.ReplaceZone("test.com", []*domain.AuthoritativeRecord{record3})
	assert.NoError(t, err)
	count = cache.Count()
	assert.Equal(t, 3, count)

	// Remove zone
	err = cache.RemoveZone("example.com")
	assert.NoError(t, err)
	count = cache.Count()
	assert.Equal(t, 1, count)
}

// Test isInZone helper function edge cases
func TestIsInZone(t *testing.T) {
	tests := []struct {
		name     string
		fqdn     string
		zoneRoot string
		want     bool
	}{
		{
			name:     "exact match with dots",
			fqdn:     "example.com.",
			zoneRoot: "example.com.",
			want:     true,
		},
		{
			name:     "exact match without dots",
			fqdn:     "example.com",
			zoneRoot: "example.com",
			want:     true,
		},
		{
			name:     "subdomain with dots",
			fqdn:     "www.example.com.",
			zoneRoot: "example.com.",
			want:     true,
		},
		{
			name:     "subdomain without dots",
			fqdn:     "www.example.com",
			zoneRoot: "example.com",
			want:     true,
		},
		{
			name:     "mixed dots - fqdn with, zone without",
			fqdn:     "www.example.com.",
			zoneRoot: "example.com",
			want:     true,
		},
		{
			name:     "mixed dots - fqdn without, zone with",
			fqdn:     "www.example.com",
			zoneRoot: "example.com.",
			want:     true,
		},
		{
			name:     "different zone",
			fqdn:     "www.other.com.",
			zoneRoot: "example.com.",
			want:     false,
		},
		{
			name:     "shorter fqdn than zone",
			fqdn:     "com.",
			zoneRoot: "example.com.",
			want:     false,
		},
		{
			name:     "partial match but different zone",
			fqdn:     "notexample.com.",
			zoneRoot: "example.com.",
			want:     false,
		},
		{
			name:     "deep subdomain",
			fqdn:     "deep.sub.www.example.com.",
			zoneRoot: "example.com.",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isInZone(tt.fqdn, tt.zoneRoot)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Test Find with no zone match
func TestFind_NoZoneMatch(t *testing.T) {
	cache := New()

	// Add a zone
	record1, _ := domain.NewAuthoritativeRecord("www.example.com.", 1, 1, 300, []byte{192, 0, 2, 1})
	err := cache.ReplaceZone("example.com", []*domain.AuthoritativeRecord{record1})
	assert.NoError(t, err)

	// Try to find record in different zone
	records, found := cache.Find(domain.DNSQuery{Name: "www.other.com.", Type: 1})
	assert.False(t, found)
	assert.Len(t, records, 0)
}

// Test Find with zone match but no FQDN match
func TestFind_ZoneMatchNoFQDN(t *testing.T) {
	cache := New()

	// Add a zone
	record1, _ := domain.NewAuthoritativeRecord("www.example.com.", 1, 1, 300, []byte{192, 0, 2, 1})
	err := cache.ReplaceZone("example.com", []*domain.AuthoritativeRecord{record1})
	assert.NoError(t, err)

	// Try to find record for FQDN that doesn't exist in zone
	records, found := cache.Find(domain.DNSQuery{Name: "mail.example.com.", Type: 1})
	assert.False(t, found)
	assert.Len(t, records, 0)
}

// Test Find with FQDN match but no RRType match
func TestFind_FQDNMatchNoRRType(t *testing.T) {
	cache := New()

	// Add a zone with A record
	record1, _ := domain.NewAuthoritativeRecord("www.example.com.", 1, 1, 300, []byte{192, 0, 2, 1}) // A record
	err := cache.ReplaceZone("example.com", []*domain.AuthoritativeRecord{record1})
	assert.NoError(t, err)

	// Try to find AAAA record for same FQDN
	records, found := cache.Find(domain.DNSQuery{Name: "www.example.com.", Type: 28}) // AAAA
	assert.False(t, found)
	assert.Len(t, records, 0)
}

// Test multiple records of same type for same FQDN
func TestFind_MultipleRecordsSameType(t *testing.T) {
	cache := New()

	// Create multiple A records for same FQDN
	record1, _ := domain.NewAuthoritativeRecord("www.example.com.", 1, 1, 300, []byte{192, 0, 2, 1})
	record2, _ := domain.NewAuthoritativeRecord("www.example.com.", 1, 1, 300, []byte{192, 0, 2, 2})
	record3, _ := domain.NewAuthoritativeRecord("www.example.com.", 1, 1, 300, []byte{192, 0, 2, 3})

	err := cache.ReplaceZone("example.com", []*domain.AuthoritativeRecord{record1, record2, record3})
	assert.NoError(t, err)

	// Find all A records
	records, found := cache.Find(domain.DNSQuery{Name: "www.example.com.", Type: 1})
	assert.True(t, found)
	assert.Len(t, records, 3)

	// Verify all records have correct type
	for _, record := range records {
		assert.Equal(t, domain.RRType(1), record.Type)
		assert.Equal(t, "www.example.com.", record.Name)
	}
}

// Test Find with most specific zone matching (zone precedence)
func TestFind_MostSpecificZone(t *testing.T) {
	cache := New()

	// Add broader zone first
	record1, _ := domain.NewAuthoritativeRecord("sub.example.com.", 1, 1, 300, []byte{192, 0, 2, 1})
	err := cache.ReplaceZone("example.com", []*domain.AuthoritativeRecord{record1})
	assert.NoError(t, err)

	// Add more specific zone
	record2, _ := domain.NewAuthoritativeRecord("sub.example.com.", 1, 1, 300, []byte{192, 0, 2, 2})
	err = cache.ReplaceZone("sub.example.com", []*domain.AuthoritativeRecord{record2})
	assert.NoError(t, err)

	// Find should match first zone found (order dependent in current implementation)
	// Note: This tests current behavior but might need updating if zone precedence logic changes
	records, found := cache.Find(domain.DNSQuery{Name: "sub.example.com.", Type: 1})
	assert.True(t, found)
	assert.Len(t, records, 1)
}
