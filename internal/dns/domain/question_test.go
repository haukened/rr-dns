package domain

import (
	"testing"
)

func TestNewQuestion(t *testing.T) {
	tests := []struct {
		name        string
		id          uint16
		queryName   string
		rrtype      RRType
		class       RRClass
		expectError bool
	}{
		{
			name:        "valid A record query",
			id:          12345,
			queryName:   "example.com.",
			rrtype:      1, // A record
			class:       1, // IN class
			expectError: false,
		},
		{
			name:        "valid AAAA record query",
			id:          12346,
			queryName:   "test.example.com.",
			rrtype:      28, // AAAA record
			class:       1,  // IN class
			expectError: false,
		},
		{
			name:        "valid CNAME record query",
			id:          12347,
			queryName:   "www.example.com.",
			rrtype:      5, // CNAME record
			class:       1, // IN class
			expectError: false,
		},
		{
			name:        "empty name should fail",
			id:          12348,
			queryName:   "",
			rrtype:      1, // A record
			class:       1, // IN class
			expectError: true,
		},
		{
			name:        "invalid RRType should fail",
			id:          12349,
			queryName:   "example.com.",
			rrtype:      999, // Invalid RRType
			class:       1,   // IN class
			expectError: true,
		},
		{
			name:        "invalid RRClass should fail",
			id:          12350,
			queryName:   "example.com.",
			rrtype:      1,   // A record
			class:       999, // Invalid RRClass
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := NewQuestion(tt.id, tt.queryName, tt.rrtype, tt.class)

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

			// Verify all fields are set correctly
			if query.ID != tt.id {
				t.Errorf("Expected ID %d, got %d", tt.id, query.ID)
			}
			if query.Name != tt.queryName {
				t.Errorf("Expected Name %q, got %q", tt.queryName, query.Name)
			}
			if query.Type != tt.rrtype {
				t.Errorf("Expected Type %d, got %d", tt.rrtype, query.Type)
			}
			if query.Class != tt.class {
				t.Errorf("Expected Class %d, got %d", tt.class, query.Class)
			}
		})
	}
}

func TestQuestion_Validate(t *testing.T) {
	tests := []struct {
		name        string
		query       Question
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid query",
			query: Question{
				ID:    12345,
				Name:  "example.com.",
				Type:  1, // A record
				Class: 1, // IN class
			},
			expectError: false,
		},
		{
			name: "empty name should fail",
			query: Question{
				ID:    12346,
				Name:  "",
				Type:  1, // A record
				Class: 1, // IN class
			},
			expectError: true,
			errorMsg:    "query name must not be empty",
		},
		{
			name: "invalid RRType should fail",
			query: Question{
				ID:    12347,
				Name:  "example.com.",
				Type:  999, // Invalid RRType
				Class: 1,   // IN class
			},
			expectError: true,
			errorMsg:    "unsupported RRType: 999",
		},
		{
			name: "invalid RRClass should fail",
			query: Question{
				ID:    12348,
				Name:  "example.com.",
				Type:  1,   // A record
				Class: 999, // Invalid RRClass
			},
			expectError: true,
			errorMsg:    "unsupported RRClass: 999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.query.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("Expected error message %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestQuestion_CacheKey(t *testing.T) {
	tests := []struct {
		name     string
		query1   Question
		query2   Question
		expected bool // true if cache keys should be equal
	}{
		{
			name: "identical queries should have same cache key",
			query1: Question{
				ID:    12345,
				Name:  "example.com.",
				Type:  1, // A record
				Class: 1, // IN class
			},
			query2: Question{
				ID:    54321, // Different ID should not affect cache key
				Name:  "example.com.",
				Type:  1, // A record
				Class: 1, // IN class
			},
			expected: true,
		},
		{
			name: "different names should have different cache keys",
			query1: Question{
				ID:    12345,
				Name:  "example.com.",
				Type:  1, // A record
				Class: 1, // IN class
			},
			query2: Question{
				ID:    12345,
				Name:  "different.com.",
				Type:  1, // A record
				Class: 1, // IN class
			},
			expected: false,
		},
		{
			name: "different types should have different cache keys",
			query1: Question{
				ID:    12345,
				Name:  "example.com.",
				Type:  1, // A record
				Class: 1, // IN class
			},
			query2: Question{
				ID:    12345,
				Name:  "example.com.",
				Type:  28, // AAAA record
				Class: 1,  // IN class
			},
			expected: false,
		},
		{
			name: "different classes should have different cache keys",
			query1: Question{
				ID:    12345,
				Name:  "example.com.",
				Type:  1, // A record
				Class: 1, // IN class
			},
			query2: Question{
				ID:    12345,
				Name:  "example.com.",
				Type:  1, // A record
				Class: 3, // CH class
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key1 := tt.query1.CacheKey()
			key2 := tt.query2.CacheKey()

			// Verify cache keys are non-empty strings
			if key1 == "" {
				t.Errorf("query1.CacheKey() returned empty string")
			}
			if key2 == "" {
				t.Errorf("query2.CacheKey() returned empty string")
			}

			keysEqual := key1 == key2
			if keysEqual != tt.expected {
				t.Errorf("Expected cache keys equal = %v, but key1=%q, key2=%q", tt.expected, key1, key2)
			}
		})
	}
}

func TestQuestion_CacheKey_Consistency(t *testing.T) {
	// Test that the same query always generates the same cache key
	query := Question{
		ID:    12345,
		Name:  "example.com.",
		Type:  1, // A record
		Class: 1, // IN class
	}

	key1 := query.CacheKey()
	key2 := query.CacheKey()
	key3 := query.CacheKey()

	if key1 != key2 || key2 != key3 {
		t.Errorf("CacheKey() should be consistent. Got: %q, %q, %q", key1, key2, key3)
	}

	// Verify the cache key format (should be non-empty)
	if key1 == "" {
		t.Errorf("CacheKey() should not return empty string")
	}
}
