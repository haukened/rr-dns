package rrdata

import (
	"bytes"
	"testing"
)

func TestEncodePTRData_ValidDomain(t *testing.T) {
	input := "ptr.example.com"
	expected, _ := encodeDomainName(input)

	result, err := encodePTRData(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestEncodePTRData_EmptyString(t *testing.T) {
	result, err := encodePTRData("")
	if err != nil {
		t.Fatalf("unexpected error for empty string: %v", err)
	}
	expected, _ := encodeDomainName("")
	if !bytes.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestDecodePTRData_ValidDomain(t *testing.T) {
	input := "ptr.example.com"
	encoded, _ := encodeDomainName(input)

	result, err := decodePTRData(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != input {
		t.Errorf("expected %q, got %q", input, result)
	}
}

func TestDecodePTRData_EmptyString(t *testing.T) {
	encoded, _ := encodeDomainName("")
	result, err := decodePTRData(encoded)
	if err != nil {
		t.Fatalf("unexpected error for empty string: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestDecodePTRData_InvalidData(t *testing.T) {
	invalid := []byte{0xff, 0x00, 0x01}
	_, err := decodePTRData(invalid)
	if err == nil {
		t.Error("expected error for invalid data, got nil")
	}
}
