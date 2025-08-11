package rrdata

import "testing"

func TestEncodeCNAMEData_Valid(t *testing.T) {
	cname := "alias.example.com"
	want, _ := encodeDomainName(cname)
	got, err := encodeCNAMEData(cname)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !equalBytes(got, want) {
		t.Errorf("EncodeCNAMEData(%q) = %v, want %v", cname, got, want)
	}
}

func TestEncodeCNAMEData_Empty(t *testing.T) {
	got, err := encodeCNAMEData("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want, _ := encodeDomainName("")
	if !equalBytes(got, want) {
		t.Errorf("EncodeCNAMEData(\"\") = %v, want %v", got, want)
	}
}
func TestDecodeCNAMEData_Valid(t *testing.T) {
	cname := "alias.example.com"
	encoded, _ := encodeDomainName(cname)
	got, err := decodeCNAMEData(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != cname {
		t.Errorf("decodeCNAMEData(%v) = %q, want %q", encoded, got, cname)
	}
}

func TestDecodeCNAMEData_Empty(t *testing.T) {
	encoded, _ := encodeDomainName("")
	got, err := decodeCNAMEData(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("decodeCNAMEData(%v) = %q, want \"\"", encoded, got)
	}
}

func TestDecodeCNAMEData_Invalid(t *testing.T) {
	// Invalid encoding: missing length byte, just raw bytes
	invalid := []byte{0x03, 'a', 'b'}
	_, err := decodeCNAMEData(invalid)
	if err == nil {
		t.Errorf("decodeCNAMEData(%v) should have returned error", invalid)
	}
}
