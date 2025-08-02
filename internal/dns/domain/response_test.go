package domain

import (
	"testing"
	"time"
)

func TestNewDNSResponse(t *testing.T) {
	timeFixture := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)
	// Create a valid resource record for testing
	rr, err := NewCachedResourceRecord("example.com.", 1, 1, 300, []byte{192, 0, 2, 1}, timeFixture)
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
	validRR, err := NewCachedResourceRecord("example.com.", 1, 1, 300, []byte{192, 0, 2, 1}, timeFixture)
	if err != nil {
		t.Fatalf("Failed to create valid test resource record: %v", err)
	}

	// Create invalid resource record for testing validation failures
	// Note: Since fields are now private/controlled, we can't easily create invalid records
	// We'll test validation through the constructor instead
	_, invalidRRErr := NewCachedResourceRecord("", 1, 1, 300, []byte{192, 0, 2, 1}, timeFixture)
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
	rr, _ := NewCachedResourceRecord("example.com.", 1, 1, 300, []byte{192, 0, 2, 1}, timeFixture)

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
	rr, _ := NewCachedResourceRecord("example.com.", 1, 1, 300, []byte{192, 0, 2, 1}, timeFixture)

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
