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
		got, err := encodeAAAAData(tt.input)
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
		_, err := encodeAAAAData(input)
		if err == nil {
			t.Errorf("EncodeAAAAData(%q) expected error, got nil", input)
		}
	}
}

func TestDecodeAAAAData_Valid(t *testing.T) {
	tests := []struct {
		input    []byte
		expected string
	}{
		{
			input:    net.ParseIP("2001:db8::ff00:42:8329").To16(),
			expected: "2001:db8::ff00:42:8329",
		},
		{
			input:    net.ParseIP("::1").To16(),
			expected: "::1",
		},
		{
			input:    net.ParseIP("fe80::1").To16(),
			expected: "fe80::1",
		},
	}

	for _, tt := range tests {
		got, err := decodeAAAAData(tt.input)
		if err != nil {
			t.Errorf("decodeAAAAData(%v) returned error: %v", tt.input, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("decodeAAAAData(%v) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestDecodeAAAAData_InvalidLength(t *testing.T) {
	invalidInputs := [][]byte{
		{},                          // empty
		{0, 1, 2},                   // too short
		make([]byte, net.IPv6len-1), // one less than IPv6 length
		make([]byte, net.IPv6len+1), // one more than IPv6 length
	}

	for _, input := range invalidInputs {
		_, err := decodeAAAAData(input)
		if err == nil {
			t.Errorf("decodeAAAAData(%v) expected error for invalid length, got nil", input)
		}
	}
}

func TestDecodeAAAAData_InvalidIP(t *testing.T) {
	// 16 bytes, but not a valid IPv6 address (all zeros is valid, so use something else)
	input := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	_, err := decodeAAAAData(input)
	if err != nil {
		// Acceptable: decodeAAAAData returns error for invalid IP
	} else {
		// net.IP may treat any 16 bytes as valid, so this test may pass
		// If decodeAAAAData returns a string, check if it's not a valid IPv6 format
		ipStr, _ := decodeAAAAData(input)
		if net.ParseIP(ipStr) == nil || !isIPv6(net.ParseIP(ipStr)) {
			t.Errorf("decodeAAAAData(%v) returned invalid IPv6 string: %q", input, ipStr)
		}
	}
}

func TestDecodeAAAAData_ValidButItsActuallyIPv4(t *testing.T) {
	// Create a valid IPv4 address
	ip := net.ParseIP("192.168.1.1")
	if ip == nil {
		t.Fatal("Failed to parse IPv4 address")
	}
	data, err := decodeAAAAData(ip)
	if err == nil {
		t.Errorf("decodeAAAAData(%v) expected error, got nil", data)
	}
}
