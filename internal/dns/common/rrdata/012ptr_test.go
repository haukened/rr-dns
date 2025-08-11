package rrdata

import (
	"bytes"
	"testing"
)

func TestEncodePTRData_ValidDomain(t *testing.T) {
	input := "ptr.example.com"
	expected, _ := EncodeDomainName(input)

	result, err := EncodePTRData(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestEncodePTRData_EmptyString(t *testing.T) {
	result, err := EncodePTRData("")
	if err != nil {
		t.Fatalf("unexpected error for empty string: %v", err)
	}
	expected, _ := EncodeDomainName("")
	if !bytes.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}
