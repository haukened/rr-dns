package domain

import (
	"strings"
	"testing"
	"time"
)

func TestNewDNSResponse(t *testing.T) {
	timeFixture := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)
	// Create a valid resource record for testing
	rr, err := NewCachedResourceRecord("example.com.", 1, 1, 300, []byte{192, 0, 2, 1}, "192.0.2.1", timeFixture)
	if err != nil {
		t.Fatalf("Failed to create test resource record: %v", err)
	}

	tests := []struct {
		name        string
		id          uint16
		rcode       RCode
		answers     []ResourceRecord
		authority   []ResourceRecord
		additional  []ResourceRecord
		expectError bool
	}{
		{
			name:        "valid response with answers",
			id:          12345,
			rcode:       0, // NOERROR
			answers:     []ResourceRecord{rr},
			authority:   []ResourceRecord{},
			additional:  []ResourceRecord{},
			expectError: false,
		},
		{
			name:        "valid NXDOMAIN response",
			id:          12346,
			rcode:       3, // NXDOMAIN
			answers:     []ResourceRecord{},
			authority:   []ResourceRecord{},
			additional:  []ResourceRecord{},
			expectError: false,
		},
		{
			name:        "invalid RCode",
			id:          12347,
			rcode:       255, // Invalid RCode
			answers:     []ResourceRecord{},
			authority:   []ResourceRecord{},
			additional:  []ResourceRecord{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := NewDNSResponse(tt.id, tt.rcode, tt.answers, tt.authority, tt.additional)

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

			if resp.ID != tt.id {
				t.Errorf("Expected ID %d, got %d", tt.id, resp.ID)
			}
			if resp.RCode != tt.rcode {
				t.Errorf("Expected RCode %d, got %d", tt.rcode, resp.RCode)
			}
		})
	}
}

func TestNewDNSResponse_ValidationFailures(t *testing.T) {
	timeFixture := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)
	// Create a valid resource record for comparison
	validRR, err := NewCachedResourceRecord("example.com.", 1, 1, 300, []byte{192, 0, 2, 1}, "192.0.2.1", timeFixture)
	if err != nil {
		t.Fatalf("Failed to create valid test resource record: %v", err)
	}

	// Create invalid resource record for testing validation failures
	// Note: Since fields are now private/controlled, we can't easily create invalid records
	// We'll test validation through the constructor instead
	_, invalidRRErr := NewCachedResourceRecord("", 1, 1, 300, []byte{192, 0, 2, 1}, "192.0.2.1", timeFixture)
	if invalidRRErr == nil {
		t.Fatal("Expected error when creating record with empty name")
	}

	tests := []struct {
		name        string
		id          uint16
		rcode       RCode
		answers     []ResourceRecord
		authority   []ResourceRecord
		additional  []ResourceRecord
		expectError bool
	}{
		{
			name:        "valid records in all sections",
			id:          12345,
			rcode:       0,
			answers:     []ResourceRecord{validRR},
			authority:   []ResourceRecord{validRR},
			additional:  []ResourceRecord{validRR},
			expectError: false,
		},
		// Note: With the new constructor-based approach, it's harder to create invalid records
		// for testing validation failures. The validation now happens at construction time.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewDNSResponse(tt.id, tt.rcode, tt.answers, tt.authority, tt.additional)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestDNSResponse_IsError(t *testing.T) {
	tests := []struct {
		name     string
		rcode    RCode
		expected bool
	}{
		{"NOERROR is not error", 0, false},
		{"FORMERR is error", 1, true},
		{"SERVFAIL is error", 2, true},
		{"NXDOMAIN is error", 3, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := DNSResponse{RCode: tt.rcode}
			if resp.IsError() != tt.expected {
				t.Errorf("Expected IsError() = %v for RCode %d", tt.expected, tt.rcode)
			}
		})
	}
}

func TestDNSResponse_HasAnswers(t *testing.T) {
	timeFixture := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)
	rr, _ := NewCachedResourceRecord("example.com.", 1, 1, 300, []byte{192, 0, 2, 1}, "192.0.2.1", timeFixture)

	tests := []struct {
		name     string
		answers  []ResourceRecord
		expected bool
	}{
		{"no answers", []ResourceRecord{}, false},
		{"has answers", []ResourceRecord{rr}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := DNSResponse{Answers: tt.answers}
			if resp.HasAnswers() != tt.expected {
				t.Errorf("Expected HasAnswers() = %v", tt.expected)
			}
		})
	}
}

func TestDNSResponse_Counts(t *testing.T) {
	timeFixture := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)
	rr, _ := NewCachedResourceRecord("example.com.", 1, 1, 300, []byte{192, 0, 2, 1}, "192.0.2.1", timeFixture)

	resp := DNSResponse{
		Answers:    []ResourceRecord{rr, rr},
		Authority:  []ResourceRecord{rr},
		Additional: []ResourceRecord{rr, rr, rr},
	}

	if resp.AnswerCount() != 2 {
		t.Errorf("Expected AnswerCount() = 2, got %d", resp.AnswerCount())
	}
	if resp.AuthorityCount() != 1 {
		t.Errorf("Expected AuthorityCount() = 1, got %d", resp.AuthorityCount())
	}
	if resp.AdditionalCount() != 3 {
		t.Errorf("Expected AdditionalCount() = 3, got %d", resp.AdditionalCount())
	}
}

func TestNewDNSErrorResponse(t *testing.T) {
	tests := []struct {
		name          string
		id            uint16
		rcode         RCode
		expectedID    uint16
		expectedRCode RCode
	}{
		{
			name:          "SERVFAIL error response",
			id:            12345,
			rcode:         2, // SERVFAIL
			expectedID:    12345,
			expectedRCode: 2,
		},
		{
			name:          "NXDOMAIN error response",
			id:            54321,
			rcode:         3, // NXDOMAIN
			expectedID:    54321,
			expectedRCode: 3,
		},
		{
			name:          "REFUSED error response",
			id:            65535, // Maximum uint16 value
			rcode:         5,     // REFUSED
			expectedID:    65535,
			expectedRCode: 5,
		},
		{
			name:          "zero ID with FORMERR",
			id:            0,
			rcode:         1, // FORMERR
			expectedID:    0,
			expectedRCode: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := NewDNSErrorResponse(tt.id, tt.rcode)

			// Verify ID and RCode are set correctly
			if resp.ID != tt.expectedID {
				t.Errorf("Expected ID %d, got %d", tt.expectedID, resp.ID)
			}
			if resp.RCode != tt.expectedRCode {
				t.Errorf("Expected RCode %d, got %d", tt.expectedRCode, resp.RCode)
			}

			// Verify all sections are empty (nil)
			if resp.Answers != nil {
				t.Errorf("Expected Answers to be nil, got %v", resp.Answers)
			}
			if resp.Authority != nil {
				t.Errorf("Expected Authority to be nil, got %v", resp.Authority)
			}
			if resp.Additional != nil {
				t.Errorf("Expected Additional to be nil, got %v", resp.Additional)
			}

			// Verify response is considered an error (except for NOERROR)
			if tt.rcode == 0 {
				if resp.IsError() {
					t.Errorf("Expected NOERROR response to not be considered an error")
				}
			} else {
				if !resp.IsError() {
					t.Errorf("Expected error response to be considered an error")
				}
			}

			// Verify response has no answers
			if resp.HasAnswers() {
				t.Errorf("Expected error response to have no answers")
			}

			// Verify all counts are zero
			if resp.AnswerCount() != 0 {
				t.Errorf("Expected AnswerCount() = 0, got %d", resp.AnswerCount())
			}
			if resp.AuthorityCount() != 0 {
				t.Errorf("Expected AuthorityCount() = 0, got %d", resp.AuthorityCount())
			}
			if resp.AdditionalCount() != 0 {
				t.Errorf("Expected AdditionalCount() = 0, got %d", resp.AdditionalCount())
			}
		})
	}
}

func TestNewDNSErrorResponse_EdgeCases(t *testing.T) {
	// Test with NOERROR (technically not an error, but should still work)
	resp := NewDNSErrorResponse(12345, 0) // NOERROR
	if resp.ID != 12345 {
		t.Errorf("Expected ID 12345, got %d", resp.ID)
	}
	if resp.RCode != 0 {
		t.Errorf("Expected RCode 0 (NOERROR), got %d", resp.RCode)
	}
	if resp.IsError() {
		t.Errorf("NOERROR response should not be considered an error")
	}

	// Test with maximum uint16 ID
	maxIDResp := NewDNSErrorResponse(65535, 2) // SERVFAIL
	if maxIDResp.ID != 65535 {
		t.Errorf("Expected ID 65535, got %d", maxIDResp.ID)
	}

	// Test with invalid RCode (function should still work, validation happens elsewhere)
	invalidResp := NewDNSErrorResponse(1234, 255) // Invalid RCode
	if invalidResp.RCode != 255 {
		t.Errorf("Expected RCode 255, got %d", invalidResp.RCode)
	}
}

// TestDNSResponse_Validate tests all validation error paths
func TestDNSResponse_Validate(t *testing.T) {
	timeFixture := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)

	// Create a valid resource record for testing
	validRR, err := NewCachedResourceRecord("example.com.", 1, 1, 300, []byte{192, 0, 2, 1}, "192.0.2.1", timeFixture)
	if err != nil {
		t.Fatalf("Failed to create valid test resource record: %v", err)
	}

	// Create invalid resource records by manipulating the struct directly
	// (since we can't create them through constructors)
	invalidNameRR := ResourceRecord{
		Name:  "", // Invalid: empty name
		Type:  1,  // A record
		Class: 1,  // IN
		ttl:   300,
	}

	invalidTypeRR := ResourceRecord{
		Name:  "example.com.",
		Type:  999, // Invalid: unsupported RRType
		Class: 1,   // IN
		ttl:   300,
	}

	invalidClassRR := ResourceRecord{
		Name:  "example.com.",
		Type:  1,   // A record
		Class: 999, // Invalid: unsupported RRClass
		ttl:   300,
	}

	tests := []struct {
		name        string
		response    DNSResponse
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid response",
			response: DNSResponse{
				ID:         12345,
				RCode:      0, // NOERROR
				Answers:    []ResourceRecord{validRR},
				Authority:  []ResourceRecord{validRR},
				Additional: []ResourceRecord{validRR},
			},
			expectError: false,
		},
		{
			name: "invalid RCode",
			response: DNSResponse{
				ID:         12345,
				RCode:      255, // Invalid RCode
				Answers:    []ResourceRecord{validRR},
				Authority:  []ResourceRecord{},
				Additional: []ResourceRecord{},
			},
			expectError: true,
			errorMsg:    "invalid RCode: 255",
		},
		{
			name: "invalid answer record at index 0",
			response: DNSResponse{
				ID:         12345,
				RCode:      0, // NOERROR
				Answers:    []ResourceRecord{invalidNameRR},
				Authority:  []ResourceRecord{},
				Additional: []ResourceRecord{},
			},
			expectError: true,
			errorMsg:    "invalid answer record at index 0",
		},
		{
			name: "invalid answer record at index 1",
			response: DNSResponse{
				ID:         12345,
				RCode:      0, // NOERROR
				Answers:    []ResourceRecord{validRR, invalidTypeRR},
				Authority:  []ResourceRecord{},
				Additional: []ResourceRecord{},
			},
			expectError: true,
			errorMsg:    "invalid answer record at index 1",
		},
		{
			name: "invalid authority record at index 0",
			response: DNSResponse{
				ID:         12345,
				RCode:      0, // NOERROR
				Answers:    []ResourceRecord{validRR},
				Authority:  []ResourceRecord{invalidClassRR},
				Additional: []ResourceRecord{},
			},
			expectError: true,
			errorMsg:    "invalid authority record at index 0",
		},
		{
			name: "invalid authority record at index 2",
			response: DNSResponse{
				ID:         12345,
				RCode:      0, // NOERROR
				Answers:    []ResourceRecord{},
				Authority:  []ResourceRecord{validRR, validRR, invalidNameRR},
				Additional: []ResourceRecord{},
			},
			expectError: true,
			errorMsg:    "invalid authority record at index 2",
		},
		{
			name: "invalid additional record at index 0",
			response: DNSResponse{
				ID:         12345,
				RCode:      0, // NOERROR
				Answers:    []ResourceRecord{},
				Authority:  []ResourceRecord{},
				Additional: []ResourceRecord{invalidTypeRR},
			},
			expectError: true,
			errorMsg:    "invalid additional record at index 0",
		},
		{
			name: "invalid additional record at index 3",
			response: DNSResponse{
				ID:         12345,
				RCode:      0, // NOERROR
				Answers:    []ResourceRecord{},
				Authority:  []ResourceRecord{},
				Additional: []ResourceRecord{validRR, validRR, validRR, invalidClassRR},
			},
			expectError: true,
			errorMsg:    "invalid additional record at index 3",
		},
		{
			name: "multiple invalid records - should catch first error",
			response: DNSResponse{
				ID:         12345,
				RCode:      0, // NOERROR
				Answers:    []ResourceRecord{invalidNameRR, invalidTypeRR},
				Authority:  []ResourceRecord{invalidClassRR},
				Additional: []ResourceRecord{invalidNameRR},
			},
			expectError: true,
			errorMsg:    "invalid answer record at index 0", // Should catch the first error
		},
		{
			name: "empty sections are valid",
			response: DNSResponse{
				ID:         12345,
				RCode:      0, // NOERROR
				Answers:    []ResourceRecord{},
				Authority:  []ResourceRecord{},
				Additional: []ResourceRecord{},
			},
			expectError: false,
		},
		{
			name: "nil sections are valid",
			response: DNSResponse{
				ID:         12345,
				RCode:      0, // NOERROR
				Answers:    nil,
				Authority:  nil,
				Additional: nil,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.response.Validate()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected validation error but got none")
					return
				}
				if tt.errorMsg != "" {
					if !strings.Contains(err.Error(), tt.errorMsg) {
						t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no validation error but got: %v", err)
				}
			}
		})
	}
}
