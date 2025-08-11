package rrdata

import (
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
)

func TestEncodeSOAData_Valid(t *testing.T) {
	data := "ns.example.com hostmaster.example.com 20240601 3600 600 86400 300"
	got, err := EncodeSOAData(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) == 0 {
		t.Error("expected non-empty result")
	}
}

func TestEncodeSOAData_InvalidFieldCount(t *testing.T) {
	data := "ns.example.com hostmaster.example.com 20240601 3600 600 86400"
	_, err := EncodeSOAData(data)
	if err == nil {
		t.Error("expected error for invalid field count")
	}
}

func TestEncodeSOAData_InvalidSerial(t *testing.T) {
	data := "ns.example.com hostmaster.example.com notanumber 3600 600 86400 300"
	_, err := EncodeSOAData(data)
	if err == nil {
		t.Error("expected error for invalid serial field")
	}
}

func TestEncodeSOAData_FieldsAreEncodedCorrectly(t *testing.T) {
	data := "ns.example.com hostmaster.example.com 1 2 3 4 5"
	got, err := EncodeSOAData(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The last 20 bytes should be the five uint32 values
	if len(got) < 20 {
		t.Fatalf("encoded data too short: %d", len(got))
	}
	u32 := got[len(got)-20:]
	want := []uint32{1, 2, 3, 4, 5}
	for i, v := range want {
		val := binary.BigEndian.Uint32(u32[i*4 : (i+1)*4])
		if val != v {
			t.Errorf("field %d: got %d, want %d", i, val, v)
		}
	}
}

func TestEncodeSOAData_MNameTooLong(t *testing.T) {
	fmtr := "%s hostmaster.example.com 20240601 3600 600 86400 300"
	data := fmt.Sprintf(fmtr, strings.Repeat("a", 256))
	_, err := EncodeSOAData(data)
	if err == nil || !strings.Contains(err.Error(), "invalid SOA mname") {
		t.Errorf("expected error for invalid mname, got: %v", err)
	}
}

func TestEncodeSOAData_RNameTooLong(t *testing.T) {
	fmtr := "ns.example.com %s 20240601 3600 600 86400 300"
	data := fmt.Sprintf(fmtr, strings.Repeat("a", 256))
	_, err := EncodeSOAData(data)
	if err == nil || !strings.Contains(err.Error(), "invalid SOA rname") {
		t.Errorf("expected error for invalid rname, got: %v", err)
	}
}
