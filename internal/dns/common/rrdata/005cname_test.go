package rrdata

import "testing"

func TestEncodeCNAMEData_Valid(t *testing.T) {
	cname := "alias.example.com"
	want, _ := EncodeDomainName(cname)
	got, err := EncodeCNAMEData(cname)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !equalBytes(got, want) {
		t.Errorf("EncodeCNAMEData(%q) = %v, want %v", cname, got, want)
	}
}

func TestEncodeCNAMEData_Empty(t *testing.T) {
	got, err := EncodeCNAMEData("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want, _ := EncodeDomainName("")
	if !equalBytes(got, want) {
		t.Errorf("EncodeCNAMEData(\"\") = %v, want %v", got, want)
	}
}
