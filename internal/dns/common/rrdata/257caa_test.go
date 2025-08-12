package rrdata

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestEncodeCAAData_Valid(t *testing.T) {
	tests := []struct {
		input    string
		expected []byte
	}{
		{
			input:    `0 issue "letsencrypt.org"`,
			expected: append([]byte{0, 5}, append([]byte("issue"), []byte("letsencrypt.org")...)...),
		},
		{
			input:    `128 iodef "mailto:security@example.com"`,
			expected: append([]byte{128, 5}, append([]byte("iodef"), []byte("mailto:security@example.com")...)...),
		},
		{
			input:    `0 issuewild "comodoca.com"`,
			expected: append([]byte{0, 9}, append([]byte("issuewild"), []byte("comodoca.com")...)...),
		},
	}

	for _, tt := range tests {
		got, err := encodeCAAData(tt.input)
		if err != nil {
			t.Errorf("EncodeCAAData(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if !bytes.Equal(got, tt.expected) {
			t.Errorf("EncodeCAAData(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestEncodeCAAData_InvalidFormat(t *testing.T) {
	invalidInputs := []string{
		`0 issue`,                 // missing value
		`issue "letsencrypt.org"`, // missing flag
		`0`,                       // missing tag and value
		``,                        // empty string
	}

	for _, input := range invalidInputs {
		_, err := encodeCAAData(input)
		if err == nil {
			t.Errorf("EncodeCAAData(%q) expected error, got nil", input)
		}
	}
}

func TestEncodeCAAData_InvalidFlag(t *testing.T) {
	_, err := encodeCAAData(`foo issue "letsencrypt.org"`)
	if err == nil || !strings.Contains(err.Error(), "invalid CAA flag") {
		t.Errorf("EncodeCAAData with invalid flag did not return expected error: %v", err)
	}
}

func TestEncodeCAAData_TagTooLong(t *testing.T) {
	longTag := strings.Repeat("a", 256)
	input := fmt.Sprintf("0 %s \"value\"", longTag)
	_, err := encodeCAAData(input)
	if err == nil || !strings.Contains(err.Error(), "CAA tag too long") {
		t.Errorf("EncodeCAAData with long tag did not return expected error: %v", err)
	}
}

func TestEncodeCAAData_ValueTooLong(t *testing.T) {
	longValue := strings.Repeat("b", 256)
	input := fmt.Sprintf("0 issue \"%s\"", longValue)
	_, err := encodeCAAData(input)
	if err == nil || !strings.Contains(err.Error(), "CAA value too long") {
		t.Errorf("EncodeCAAData with long value did not return expected error: %v", err)
	}
}

func TestDecodeCAAData_Valid(t *testing.T) {
	tests := []struct {
		input    []byte
		expected string
	}{
		{
			input:    append([]byte{0, 5}, append([]byte("issue"), []byte("letsencrypt.org")...)...),
			expected: `0 issue "letsencrypt.org"`,
		},
		{
			input:    append([]byte{128, 5}, append([]byte("iodef"), []byte("mailto:security@example.com")...)...),
			expected: `128 iodef "mailto:security@example.com"`,
		},
		{
			input:    append([]byte{0, 9}, append([]byte("issuewild"), []byte("comodoca.com")...)...),
			expected: `0 issuewild "comodoca.com"`,
		},
	}

	for _, tt := range tests {
		got, err := decodeCAAData(tt.input)
		if err != nil {
			t.Errorf("decodeCAAData(%v) unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("decodeCAAData(%v) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestDecodeCAAData_InvalidLength(t *testing.T) {
	invalidInputs := [][]byte{
		{},          // empty
		{1},         // too short
		{1, 10},     // tagLen longer than data
		{1, 2, 'a'}, // tagLen longer than data
	}

	for _, input := range invalidInputs {
		_, err := decodeCAAData(input)
		if err == nil {
			t.Errorf("decodeCAAData(%v) expected error, got nil", input)
		}
	}
}

func TestDecodeCAAData_TagLengthMismatch(t *testing.T) {
	// tagLen is 5, but only 3 bytes for tag
	input := []byte{0, 5, 'i', 's', 's'}
	_, err := decodeCAAData(input)
	if err == nil || !strings.Contains(err.Error(), "invalid CAA tag length") {
		t.Errorf("decodeCAAData with mismatched tag length did not return expected error: %v", err)
	}
}
