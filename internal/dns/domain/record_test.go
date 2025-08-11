package domain

import (
	"testing"
	"time"
)

func TestNewAuthoritativeResourceRecord(t *testing.T) {
	tests := []struct {
		name         string
		recordName   string
		rrtype       RRType
		class        RRClass
		ttl          uint32
		data         []byte
		expectError  bool
		expectedName string
	}{
		{
			name:         "valid A record",
			recordName:   "example.com.",
			rrtype:       1, // A
			class:        1, // IN
			ttl:          300,
			data:         []byte{192, 0, 2, 1},
			expectError:  false,
			expectedName: "example.com",
		},
		{
			name:         "name gets canonicalized",
			recordName:   "EXAMPLE.COM",
			rrtype:       1, // A
			class:        1, // IN
			ttl:          300,
			data:         []byte{192, 0, 2, 1},
			expectError:  false,
			expectedName: "example.com",
		},
		{
			name:         "name with whitespace gets canonicalized",
			recordName:   "  example.com  ",
			rrtype:       1, // A
			class:        1, // IN
			ttl:          300,
			data:         []byte{192, 0, 2, 1},
			expectError:  false,
			expectedName: "example.com",
		},
		{
			name:        "empty name",
			recordName:  "",
			rrtype:      1, // A
			class:       1, // IN
			ttl:         300,
			data:        []byte{192, 0, 2, 1},
			expectError: true,
		},
		{
			name:        "invalid RRType",
			recordName:  "example.com.",
			rrtype:      0, // Invalid
			class:       1, // IN
			ttl:         300,
			data:        []byte{192, 0, 2, 1},
			expectError: true,
		},
		{
			name:        "invalid RRClass",
			recordName:  "example.com.",
			rrtype:      1, // A
			class:       0, // Invalid
			ttl:         300,
			data:        []byte{192, 0, 2, 1},
			expectError: true,
		},
		{
			name:         "zero TTL is valid",
			recordName:   "example.com.",
			rrtype:       1, // A
			class:        1, // IN
			ttl:          0,
			data:         []byte{192, 0, 2, 1},
			expectError:  false,
			expectedName: "example.com",
		},
		{
			name:         "empty data is valid",
			recordName:   "example.com.",
			rrtype:       1, // A
			class:        1, // IN
			ttl:          300,
			data:         []byte{},
			expectError:  false,
			expectedName: "example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr, err := NewAuthoritativeResourceRecord(tt.recordName, tt.rrtype, tt.class, tt.ttl, tt.data)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify the record fields
			if rr.Name != tt.expectedName {
				t.Errorf("Expected name %q, got %q", tt.expectedName, rr.Name)
			}
			if rr.Type != tt.rrtype {
				t.Errorf("Expected type %d, got %d", tt.rrtype, rr.Type)
			}
			if rr.Class != tt.class {
				t.Errorf("Expected class %d, got %d", tt.class, rr.Class)
			}
			if rr.ttl != tt.ttl {
				t.Errorf("Expected TTL %d, got %d", tt.ttl, rr.ttl)
			}
			if rr.expiresAt != nil {
				t.Errorf("Expected expiresAt to be nil for authoritative record, got %v", rr.expiresAt)
			}
			if !equalBytes(rr.Data, tt.data) {
				t.Errorf("Expected data %v, got %v", tt.data, rr.Data)
			}
		})
	}
}

func TestNewCachedResourceRecord(t *testing.T) {
	timeFixture := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name         string
		recordName   string
		rrtype       RRType
		class        RRClass
		ttl          uint32
		data         []byte
		now          time.Time
		expectError  bool
		expectedName string
	}{
		{
			name:         "valid cached record",
			recordName:   "example.com.",
			rrtype:       1, // A
			class:        1, // IN
			ttl:          300,
			data:         []byte{192, 0, 2, 1},
			now:          timeFixture,
			expectError:  false,
			expectedName: "example.com",
		},
		{
			name:         "name gets canonicalized",
			recordName:   "EXAMPLE.COM",
			rrtype:       1, // A
			class:        1, // IN
			ttl:          300,
			data:         []byte{192, 0, 2, 1},
			now:          timeFixture,
			expectError:  false,
			expectedName: "example.com",
		},
		{
			name:        "empty name",
			recordName:  "",
			rrtype:      1, // A
			class:       1, // IN
			ttl:         300,
			data:        []byte{192, 0, 2, 1},
			now:         timeFixture,
			expectError: true,
		},
		{
			name:         "zero TTL cached record",
			recordName:   "example.com.",
			rrtype:       1, // A
			class:        1, // IN
			ttl:          0,
			data:         []byte{192, 0, 2, 1},
			now:          timeFixture,
			expectError:  false,
			expectedName: "example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr, err := NewCachedResourceRecord(tt.recordName, tt.rrtype, tt.class, tt.ttl, tt.data, tt.now)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify the record fields
			if rr.Name != tt.expectedName {
				t.Errorf("Expected name %q, got %q", tt.expectedName, rr.Name)
			}
			if rr.Type != tt.rrtype {
				t.Errorf("Expected type %d, got %d", tt.rrtype, rr.Type)
			}
			if rr.Class != tt.class {
				t.Errorf("Expected class %d, got %d", tt.class, rr.Class)
			}
			if rr.ttl != tt.ttl {
				t.Errorf("Expected TTL %d, got %d", tt.ttl, rr.ttl)
			}
			if rr.expiresAt == nil {
				t.Errorf("Expected expiresAt to be set for cached record, got nil")
			} else {
				expectedExpiration := tt.now.Add(time.Duration(tt.ttl) * time.Second)
				if !rr.expiresAt.Equal(expectedExpiration) {
					t.Errorf("Expected expiresAt %v, got %v", expectedExpiration, *rr.expiresAt)
				}
			}
			if !equalBytes(rr.Data, tt.data) {
				t.Errorf("Expected data %v, got %v", tt.data, rr.Data)
			}
		})
	}
}

func TestResourceRecord_TTL(t *testing.T) {
	tests := []struct {
		name        string
		record      ResourceRecord
		expectedTTL uint32
	}{
		{
			name: "authoritative record returns original TTL",
			record: ResourceRecord{
				Name:      "example.com.",
				Type:      1,
				Class:     1,
				ttl:       300,
				expiresAt: nil,
				Data:      []byte{192, 0, 2, 1},
			},
			expectedTTL: 300,
		},
		{
			name: "authoritative record with zero TTL",
			record: ResourceRecord{
				Name:      "example.com.",
				Type:      1,
				Class:     1,
				ttl:       0,
				expiresAt: nil,
				Data:      []byte{192, 0, 2, 1},
			},
			expectedTTL: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualTTL := tt.record.TTL()

			// For authoritative records, TTL should be the original value
			if tt.record.expiresAt == nil && actualTTL != tt.expectedTTL {
				t.Errorf("Expected TTL %d, got %d", tt.expectedTTL, actualTTL)
			}
		})
	}
}

func TestResourceRecord_TTL_CachedRecords(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		record      ResourceRecord
		expectedTTL uint32
	}{
		{
			name: "cached record with future expiration",
			record: ResourceRecord{
				Name:      "example.com.",
				Type:      1,
				Class:     1,
				ttl:       300,
				expiresAt: func() *time.Time { exp := now.Add(100 * time.Second); return &exp }(),
				Data:      []byte{192, 0, 2, 1},
			},
			expectedTTL: 100, // Approximately 100 seconds remaining
		},
		{
			name: "cached record with short remaining TTL",
			record: ResourceRecord{
				Name:      "example.com.",
				Type:      1,
				Class:     1,
				ttl:       300,
				expiresAt: func() *time.Time { exp := now.Add(5 * time.Second); return &exp }(),
				Data:      []byte{192, 0, 2, 1},
			},
			expectedTTL: 5, // Approximately 5 seconds remaining
		},
		{
			name: "cached record exactly at expiration",
			record: ResourceRecord{
				Name:      "example.com.",
				Type:      1,
				Class:     1,
				ttl:       300,
				expiresAt: &now, // Expires exactly now
				Data:      []byte{192, 0, 2, 1},
			},
			expectedTTL: 0, // Should return 0 when expired
		},
		{
			name: "cached record past expiration",
			record: ResourceRecord{
				Name:      "example.com.",
				Type:      1,
				Class:     1,
				ttl:       300,
				expiresAt: func() *time.Time { exp := now.Add(-10 * time.Second); return &exp }(),
				Data:      []byte{192, 0, 2, 1},
			},
			expectedTTL: 0, // Should return 0 when expired
		},
		{
			name: "cached record with zero original TTL but future expiration",
			record: ResourceRecord{
				Name:      "example.com.",
				Type:      1,
				Class:     1,
				ttl:       0,
				expiresAt: func() *time.Time { exp := now.Add(50 * time.Second); return &exp }(),
				Data:      []byte{192, 0, 2, 1},
			},
			expectedTTL: 50, // Should return remaining time, not original TTL
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualTTL := tt.record.TTL()

			// For cached records, we need to be more flexible with timing
			if tt.record.expiresAt != nil {
				// Allow for small timing differences (±2 seconds)
				tolerance := uint32(2)
				if actualTTL > tt.expectedTTL+tolerance {
					t.Errorf("Expected TTL around %d, got %d (too high)", tt.expectedTTL, actualTTL)
				}
				// For expired records, TTL should be exactly 0
				if tt.expectedTTL == 0 && actualTTL != 0 {
					t.Errorf("Expected TTL 0 for expired record, got %d", actualTTL)
				}
				// For non-expired records, TTL should be close to expected
				if tt.expectedTTL > 0 && actualTTL == 0 {
					t.Errorf("Expected TTL around %d, got 0 (unexpectedly expired)", tt.expectedTTL)
				}
			} else {
				// For authoritative records, TTL should be exact
				if actualTTL != tt.expectedTTL {
					t.Errorf("Expected TTL %d, got %d", tt.expectedTTL, actualTTL)
				}
			}
		})
	}
}

func TestResourceRecord_TTLRemaining(t *testing.T) {
	tests := []struct {
		name             string
		record           ResourceRecord
		expectedDuration time.Duration
	}{
		{
			name: "authoritative record returns original TTL as duration",
			record: ResourceRecord{
				Name:      "example.com.",
				Type:      1,
				Class:     1,
				ttl:       300,
				expiresAt: nil,
				Data:      []byte{192, 0, 2, 1},
			},
			expectedDuration: 300 * time.Second,
		},
		{
			name: "cached record with future expiration",
			record: ResourceRecord{
				Name:      "example.com.",
				Type:      1,
				Class:     1,
				ttl:       300,
				expiresAt: &[]time.Time{time.Now().Add(200 * time.Second)}[0],
				Data:      []byte{192, 0, 2, 1},
			},
			expectedDuration: 200 * time.Second, // Approximate
		},
		{
			name: "cached record expired",
			record: ResourceRecord{
				Name:      "example.com.",
				Type:      1,
				Class:     1,
				ttl:       300,
				expiresAt: &[]time.Time{time.Now().Add(-100 * time.Second)}[0],
				Data:      []byte{192, 0, 2, 1},
			},
			expectedDuration: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remaining := tt.record.TTLRemaining()

			if tt.record.expiresAt == nil {
				// Authoritative record should return original TTL
				if remaining != tt.expectedDuration {
					t.Errorf("Expected TTL remaining %v, got %v", tt.expectedDuration, remaining)
				}
			} else {
				// Cached record - test within reasonable bounds due to time precision
				if tt.expectedDuration == 0 {
					if remaining != 0 {
						t.Errorf("Expected TTL remaining %v, got %v", tt.expectedDuration, remaining)
					}
				} else {
					// Allow some tolerance for time passage during test execution
					tolerance := 2 * time.Second
					if remaining < tt.expectedDuration-tolerance || remaining > tt.expectedDuration+tolerance {
						t.Errorf("Expected TTL remaining ~%v, got %v", tt.expectedDuration, remaining)
					}
				}
			}
		})
	}
}

func TestResourceRecord_IsAuthoritative(t *testing.T) {
	timeFixture := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)
	authRecord := ResourceRecord{
		Name:      "example.com.",
		Type:      1,
		Class:     1,
		ttl:       300,
		expiresAt: nil,
		Data:      []byte{192, 0, 2, 1},
	}

	cachedRecord := ResourceRecord{
		Name:      "example.com.",
		Type:      1,
		Class:     1,
		ttl:       300,
		expiresAt: &timeFixture,
		Data:      []byte{192, 0, 2, 1},
	}

	if !authRecord.IsAuthoritative() {
		t.Error("Expected authoritative record to return true for IsAuthoritative()")
	}

	if cachedRecord.IsAuthoritative() {
		t.Error("Expected cached record to return false for IsAuthoritative()")
	}
}

func TestResourceRecord_IsExpired(t *testing.T) {
	futureTime := time.Now().Add(300 * time.Second)
	pastTime := time.Now().Add(-300 * time.Second)

	tests := []struct {
		name            string
		record          ResourceRecord
		expectedExpired bool
	}{
		{
			name: "authoritative record never expires",
			record: ResourceRecord{
				Name:      "example.com.",
				Type:      1,
				Class:     1,
				ttl:       300,
				expiresAt: nil,
				Data:      []byte{192, 0, 2, 1},
			},
			expectedExpired: false,
		},
		{
			name: "cached record not yet expired",
			record: ResourceRecord{
				Name:      "example.com.",
				Type:      1,
				Class:     1,
				ttl:       300,
				expiresAt: &futureTime,
				Data:      []byte{192, 0, 2, 1},
			},
			expectedExpired: false,
		},
		{
			name: "cached record expired",
			record: ResourceRecord{
				Name:      "example.com.",
				Type:      1,
				Class:     1,
				ttl:       300,
				expiresAt: &pastTime,
				Data:      []byte{192, 0, 2, 1},
			},
			expectedExpired: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.record.IsExpired() != tt.expectedExpired {
				t.Errorf("Expected IsExpired() = %v, got %v", tt.expectedExpired, tt.record.IsExpired())
			}
		})
	}
}

func TestResourceRecord_CacheKey(t *testing.T) {
	rr1 := ResourceRecord{
		Name:  "example.com.",
		Type:  1, // A
		Class: 1, // IN
		ttl:   300,
		Data:  []byte{192, 0, 2, 1},
	}

	rr2 := ResourceRecord{
		Name:  "example.com.",
		Type:  1,                    // A
		Class: 1,                    // IN
		ttl:   600,                  // Different TTL
		Data:  []byte{192, 0, 2, 2}, // Different data
	}

	rr3 := ResourceRecord{
		Name:  "example.com.",
		Type:  28, // AAAA - different type
		Class: 1,  // IN
		ttl:   300,
		Data:  []byte{1, 2, 3, 4},
	}

	// Records with same name, type, class should have same cache key
	key1 := rr1.CacheKey()
	key2 := rr2.CacheKey()
	if key1 != key2 {
		t.Errorf("Expected same cache key for records with same name/type/class, got %q vs %q", key1, key2)
	}

	// Records with different type should have different cache key
	key3 := rr3.CacheKey()
	if key1 == key3 {
		t.Errorf("Expected different cache keys for records with different types, both got %q", key1)
	}

	// Cache key should not be empty
	if key1 == "" {
		t.Error("Cache key should not be empty")
	}
}

func TestResourceRecord_Validate(t *testing.T) {
	tests := []struct {
		name        string
		record      ResourceRecord
		expectError bool
	}{
		{
			name: "valid record",
			record: ResourceRecord{
				Name:  "example.com.",
				Type:  1, // A
				Class: 1, // IN
				ttl:   300,
				Data:  []byte{192, 0, 2, 1},
			},
			expectError: false,
		},
		{
			name: "empty name",
			record: ResourceRecord{
				Name:  "",
				Type:  1, // A
				Class: 1, // IN
				ttl:   300,
				Data:  []byte{192, 0, 2, 1},
			},
			expectError: true,
		},
		{
			name: "invalid type",
			record: ResourceRecord{
				Name:  "example.com.",
				Type:  0, // Invalid
				Class: 1, // IN
				ttl:   300,
				Data:  []byte{192, 0, 2, 1},
			},
			expectError: true,
		},
		{
			name: "invalid class",
			record: ResourceRecord{
				Name:  "example.com.",
				Type:  1, // A
				Class: 0, // Invalid
				ttl:   300,
				Data:  []byte{192, 0, 2, 1},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.record.Validate()
			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

// Helper function to compare byte slices
func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Benchmark tests
func BenchmarkNewAuthoritativeResourceRecord(b *testing.B) {
	name := "example.com."
	rrtype := RRType(1)
	class := RRClass(1)
	ttl := uint32(300)
	data := []byte{192, 0, 2, 1}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewAuthoritativeResourceRecord(name, rrtype, class, ttl, data)
	}
}

func BenchmarkNewCachedResourceRecord(b *testing.B) {
	timeFixture := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)
	name := "example.com."
	rrtype := RRType(1)
	class := RRClass(1)
	ttl := uint32(300)
	data := []byte{192, 0, 2, 1}
	now := timeFixture

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewCachedResourceRecord(name, rrtype, class, ttl, data, now)
	}
}

func BenchmarkResourceRecord_TTL(b *testing.B) {
	rr := ResourceRecord{
		Name:      "example.com.",
		Type:      1,
		Class:     1,
		ttl:       300,
		expiresAt: nil,
		Data:      []byte{192, 0, 2, 1},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rr.TTL()
	}
}

func BenchmarkResourceRecord_CacheKey(b *testing.B) {
	rr := ResourceRecord{
		Name:  "example.com.",
		Type:  1,
		Class: 1,
		ttl:   300,
		Data:  []byte{192, 0, 2, 1},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rr.CacheKey()
	}
}
