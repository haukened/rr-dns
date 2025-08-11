package rrdata

import (
	"net"
	"testing"
)

func TestEncodeAAAAData_ValidIPv6(t *testing.T) {
	tests := []struct {
		input    string
		expected []byte
	}{
		{
			input:    "2001:db8::ff00:42:8329",
			expected: net.ParseIP("2001:db8::ff00:42:8329").To16(),
		},
		{
			input:    "::1",
			expected: net.ParseIP("::1").To16(),
		},
		{
			input:    "fe80::1",
			expected: net.ParseIP("fe80::1").To16(),
		},
	}

	for _, tt := range tests {
		got, err := EncodeAAAAData(tt.input)
		if err != nil {
			t.Errorf("EncodeAAAAData(%q) returned error: %v", tt.input, err)
			continue
		}
		if !equalBytes(got, tt.expected) {
			t.Errorf("EncodeAAAAData(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestEncodeAAAAData_InvalidIPv6(t *testing.T) {
	invalidInputs := []string{
		"not-an-ip",
		"192.168.1.1", // IPv4, not IPv6
		"",
		"2001:db8:::ff00:42:8329", // malformed
	}

	for _, input := range invalidInputs {
		_, err := EncodeAAAAData(input)
		if err == nil {
			t.Errorf("EncodeAAAAData(%q) expected error, got nil", input)
		}
	}
}
