package domain

import (
	"testing"
	"time"
)

func TestNewDNSResponse(t *testing.T) {
	// Create a valid resource record for testing
	rr, err := NewResourceRecord("example.com.", 1, 1, 300, []byte{192, 0, 2, 1})
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
	// Create a valid resource record for comparison
	validRR, err := NewResourceRecord("example.com.", 1, 1, 300, []byte{192, 0, 2, 1})
	if err != nil {
		t.Fatalf("Failed to create valid test resource record: %v", err)
	}

	// Create invalid resource records for testing validation failures
	invalidRR := ResourceRecord{
		Name:      "", // Invalid: empty name
		Type:      1,
		Class:     1,
		ExpiresAt: time.Now().Add(time.Hour),
		Data:      []byte{192, 0, 2, 1},
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
			name:        "invalid record in answers",
			id:          12345,
			rcode:       0,
			answers:     []ResourceRecord{invalidRR},
			authority:   []ResourceRecord{},
			additional:  []ResourceRecord{},
			expectError: true,
		},
		{
			name:        "invalid record in authority",
			id:          12346,
			rcode:       0,
			answers:     []ResourceRecord{validRR},
			authority:   []ResourceRecord{invalidRR},
			additional:  []ResourceRecord{},
			expectError: true,
		},
		{
			name:        "invalid record in additional",
			id:          12347,
			rcode:       0,
			answers:     []ResourceRecord{validRR},
			authority:   []ResourceRecord{validRR},
			additional:  []ResourceRecord{invalidRR},
			expectError: true,
		},
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
	rr, _ := NewResourceRecord("example.com.", 1, 1, 300, []byte{192, 0, 2, 1})

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
	rr, _ := NewResourceRecord("example.com.", 1, 1, 300, []byte{192, 0, 2, 1})

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
