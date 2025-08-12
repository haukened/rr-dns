package rrdata

import (
	"bytes"
	"strings"
	"testing"
)

func TestEncodeMXData_Valid(t *testing.T) {
	tests := []struct {
		input    string
		expected []byte
	}{
		{
			input:    "10 mail.example.com",
			expected: append([]byte{0, 10}, []byte{4, 'm', 'a', 'i', 'l', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0}...),
		},
		{
			input:    "0 mx.example.org",
			expected: append([]byte{0, 0}, []byte{2, 'm', 'x', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'o', 'r', 'g', 0}...),
		},
		{
			input:    "65535 mail.test.net",
			expected: append([]byte{255, 255}, []byte{4, 'm', 'a', 'i', 'l', 4, 't', 'e', 's', 't', 3, 'n', 'e', 't', 0}...),
		},
	}

	for _, tt := range tests {
		got, err := encodeMXData(tt.input)
		if err != nil {
			t.Errorf("EncodeMXData(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if !bytes.Equal(got, tt.expected) {
			t.Errorf("EncodeMXData(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestEncodeMXData_InvalidFormat(t *testing.T) {
	invalidInputs := []string{
		"",
		"10",
		"mail.example.com",
		"10 mail.example.com extra",
		"10mail.example.com",
	}

	for _, input := range invalidInputs {
		_, err := encodeMXData(input)
		if err == nil {
			t.Errorf("EncodeMXData(%q) expected error, got nil", input)
		}
	}
}

func TestEncodeMXData_InvalidPreference(t *testing.T) {
	invalidPrefs := []string{
		"-1 mail.example.com",
		"65536 mail.example.com",
		"notanumber mail.example.com",
	}

	for _, input := range invalidPrefs {
		_, err := encodeMXData(input)
		if err == nil {
			t.Errorf("EncodeMXData(%q) expected error for invalid preference, got nil", input)
		}
	}
}

func TestEncodeMXData_DomainTooLong(t *testing.T) {
	longDomain := "10 " + strings.Repeat("a", 256) + ".example.com"
	_, err := encodeMXData(longDomain)
	if err == nil {
		t.Errorf("EncodeMXData(%q) expected error for domain too long, got nil", longDomain)
	}
}
func TestDecodeMXData_Valid(t *testing.T) {
	tests := []struct {
		input    []byte
		expected string
	}{
		{
			input:    append([]byte{0, 10}, []byte{4, 'm', 'a', 'i', 'l', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0}...),
			expected: "10 mail.example.com",
		},
		{
			input:    append([]byte{0, 0}, []byte{2, 'm', 'x', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'o', 'r', 'g', 0}...),
			expected: "0 mx.example.org",
		},
		{
			input:    append([]byte{255, 255}, []byte{4, 'm', 'a', 'i', 'l', 4, 't', 'e', 's', 't', 3, 'n', 'e', 't', 0}...),
			expected: "65535 mail.test.net",
		},
	}

	for _, tt := range tests {
		got, err := decodeMXData(tt.input)
		if err != nil {
			t.Errorf("decodeMXData(%v) unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("decodeMXData(%v) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestDecodeMXData_InvalidLength(t *testing.T) {
	invalidInputs := [][]byte{
		{},
		{0},
	}

	for _, input := range invalidInputs {
		_, err := decodeMXData(input)
		if err == nil {
			t.Errorf("decodeMXData(%v) expected error for invalid length, got nil", input)
		}
	}
}

func TestDecodeMXData_TargetTooLong(t *testing.T) {
	var target []byte
	target = append(target, 0, 10)
	target = append(target, []byte(strings.Repeat("a", 256))...)
	data, err := decodeMXData(target)
	if err == nil {
		t.Errorf("decodeMXData(%v) expected error, got %v", target, data)
	}
}
