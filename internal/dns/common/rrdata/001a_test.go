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
		got, err := EncodeAData(tt.input)
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
		got, err := EncodeAData(input)
		if err == nil {
			t.Errorf("EncodeAData(%q) expected error, got nil", input)
		}
		if got != nil {
			t.Errorf("EncodeAData(%q) expected nil, got %v", input, got)
		}
	}
}
