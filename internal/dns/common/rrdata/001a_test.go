package rrdata

import "testing"

func TestEncodeAData_ValidIPv4(t *testing.T) {
	tests := []struct {
		input    string
		expected []byte
	}{
		{"192.168.0.1", []byte{192, 168, 0, 1}},
		{"8.8.8.8", []byte{8, 8, 8, 8}},
		{"127.0.0.1", []byte{127, 0, 0, 1}},
	}

	for _, tt := range tests {
		got, err := encodeAData(tt.input)
		if err != nil {
			t.Errorf("EncodeAData(%q) returned error: %v", tt.input, err)
		}
		if !equalBytes(got, tt.expected) {
			t.Errorf("EncodeAData(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestEncodeAData_InvalidIPv4(t *testing.T) {
	invalidInputs := []string{
		"not.an.ip",
		"256.256.256.256",
		"::1",
		"",
	}

	for _, input := range invalidInputs {
		got, err := encodeAData(input)
		if err == nil {
			t.Errorf("EncodeAData(%q) expected error, got nil", input)
		}
		if got != nil {
			t.Errorf("EncodeAData(%q) expected nil, got %v", input, got)
		}
	}
}
func TestDecodeAData_ValidIPv4(t *testing.T) {
	tests := []struct {
		input    []byte
		expected string
	}{
		{[]byte{192, 168, 0, 1}, "192.168.0.1"},
		{[]byte{8, 8, 8, 8}, "8.8.8.8"},
		{[]byte{127, 0, 0, 1}, "127.0.0.1"},
	}

	for _, tt := range tests {
		got, err := decodeAData(tt.input)
		if err != nil {
			t.Errorf("decodeAData(%v) returned error: %v", tt.input, err)
		}
		if got != tt.expected {
			t.Errorf("decodeAData(%v) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestDecodeAData_InvalidLength(t *testing.T) {
	invalidInputs := [][]byte{
		{},                  // empty
		{192, 168, 0},       // too short
		{192, 168, 0, 1, 2}, // too long
	}

	for _, input := range invalidInputs {
		got, err := decodeAData(input)
		if err == nil {
			t.Errorf("decodeAData(%v) expected error, got nil", input)
		}
		if got != "" {
			t.Errorf("decodeAData(%v) expected empty string, got %q", input, got)
		}
	}
}

func TestDecodeAData_InvalidIP(t *testing.T) {
	// 4 bytes, but not a valid IPv4 address (e.g., all zero)
	input := []byte{0, 0, 0, 0}
	got, err := decodeAData(input)
	if err != nil {
		// Acceptable: decodeAData should return error for invalid IP
	} else if got != "0.0.0.0" {
		t.Errorf("decodeAData(%v) = %q, want %q", input, got, "0.0.0.0")
	}
}

func TestDecodeAData_ButItsActuallyAnIPv6(t *testing.T) {
	// 16 bytes, but not a valid IPv4 address
	input := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	got, err := decodeAData(input)
	if err != nil {
		// Acceptable: decodeAData should return error for invalid IP
	} else if got != "::1" {
		t.Errorf("decodeAData(%v) = %q, want %q", input, got, "::1")
	}
}
