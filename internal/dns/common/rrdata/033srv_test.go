package rrdata

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
)

func TestEncodeSRVData_Valid(t *testing.T) {
	tests := []struct {
		input    string
		expected []byte
	}{
		{
			input: "10 20 80 example.com.",
			expected: func() []byte {
				b := make([]byte, 6)
				binary.BigEndian.PutUint16(b[0:], 10)
				binary.BigEndian.PutUint16(b[2:], 20)
				binary.BigEndian.PutUint16(b[4:], 80)
				target, _ := EncodeDomainName("example.com.")
				return append(b, target...)
			}(),
		},
		{
			input: "0 0 443 _sip._tcp.example.com.",
			expected: func() []byte {
				b := make([]byte, 6)
				binary.BigEndian.PutUint16(b[0:], 0)
				binary.BigEndian.PutUint16(b[2:], 0)
				binary.BigEndian.PutUint16(b[4:], 443)
				target, _ := EncodeDomainName("_sip._tcp.example.com.")
				return append(b, target...)
			}(),
		},
	}

	for _, tt := range tests {
		got, err := EncodeSRVData(tt.input)
		if err != nil {
			t.Errorf("EncodeSRVData(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if !bytes.Equal(got, tt.expected) {
			t.Errorf("EncodeSRVData(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestEncodeSRVData_InvalidFormat(t *testing.T) {
	invalidInputs := []string{
		"10 20 80",                  // Too few fields
		"10 20 80 extra field test", // Too many fields
		"",                          // Empty string
	}

	for _, input := range invalidInputs {
		_, err := EncodeSRVData(input)
		if err == nil {
			t.Errorf("EncodeSRVData(%q) expected error, got nil", input)
		}
	}
}

func TestEncodeSRVData_InvalidNumbers(t *testing.T) {
	invalidInputs := []string{
		"abc 20 80 example.com.",
		"10 xyz 80 example.com.",
		"10 20 port example.com.",
		"-1 20 80 example.com.",
		"10 65536 80 example.com.",
	}

	for _, input := range invalidInputs {
		_, err := EncodeSRVData(input)
		if err == nil {
			t.Errorf("EncodeSRVData(%q) expected error, got nil", input)
		}
	}
}

func TestEncodeSRVData_InvalidTarget(t *testing.T) {
	fmtr := "10 20 80 %s"
	data := fmt.Sprintf(fmtr, strings.Repeat("a", 256))
	_, err := EncodeSRVData(data)
	if err == nil {
		t.Error("EncodeSRVData with invalid target expected error, got nil")
	}
}
