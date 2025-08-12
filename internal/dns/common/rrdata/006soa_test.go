package rrdata

import (
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeSOAData_Valid(t *testing.T) {
	data := "ns.example.com hostmaster.example.com 20240601 3600 600 86400 300"
	got, err := encodeSOAData(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) == 0 {
		t.Error("expected non-empty result")
	}
}

func TestEncodeSOAData_InvalidFieldCount(t *testing.T) {
	data := "ns.example.com hostmaster.example.com 20240601 3600 600 86400"
	_, err := encodeSOAData(data)
	if err == nil {
		t.Error("expected error for invalid field count")
	}
}

func TestEncodeSOAData_InvalidSerial(t *testing.T) {
	data := "ns.example.com hostmaster.example.com notanumber 3600 600 86400 300"
	_, err := encodeSOAData(data)
	if err == nil {
		t.Error("expected error for invalid serial field")
	}
}

func TestEncodeSOAData_FieldsAreEncodedCorrectly(t *testing.T) {
	data := "ns.example.com hostmaster.example.com 1 2 3 4 5"
	got, err := encodeSOAData(data)
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
	_, err := encodeSOAData(data)
	if err == nil || !strings.Contains(err.Error(), "invalid SOA mname") {
		t.Errorf("expected error for invalid mname, got: %v", err)
	}
}

func TestEncodeSOAData_RNameTooLong(t *testing.T) {
	fmtr := "ns.example.com %s 20240601 3600 600 86400 300"
	data := fmt.Sprintf(fmtr, strings.Repeat("a", 256))
	_, err := encodeSOAData(data)
	if err == nil || !strings.Contains(err.Error(), "invalid SOA rname") {
		t.Errorf("expected error for invalid rname, got: %v", err)
	}
}

func TestDecodeSOAData_Valid(t *testing.T) {
	// Manually construct: mname=ns.example.com, rname=hostmaster.example.com, numbers: 20240601 3600 600 86400 300
	mname := []byte{2, 'n', 's', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0}
	rname := []byte{10, 'h', 'o', 's', 't', 'm', 'a', 's', 't', 'e', 'r', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0}
	nums := []uint32{20240601, 3600, 600, 86400, 300}
	numBytes := make([]byte, 20)
	for i, v := range nums {
		binary.BigEndian.PutUint32(numBytes[i*4:], v)
	}
	wire := append(append(mname, rname...), numBytes...)
	decoded, err := decodeSOAData(wire)
	if err != nil {
		t.Fatalf("decodeSOAData failed: %v", err)
	}
	want := "ns.example.com hostmaster.example.com 20240601 3600 600 86400 300"
	if decoded != want {
		t.Errorf("decoded SOA mismatch:\n got: %q\nwant: %q", decoded, want)
	}
}

func TestDecodeSOAData_InvalidLength(t *testing.T) {
	b := make([]byte, 10) // too short
	_, err := decodeSOAData(b)
	if err == nil || !strings.Contains(err.Error(), "invalid SOA data length") {
		t.Errorf("expected error for invalid length, got: %v", err)
	}
}

func TestDecodeSOAData_InvalidMName(t *testing.T) {
	// Start with invalid length byte > remaining
	wire := []byte{0xff, 'n', 's', 0} // truncated deliberately before rname/nums
	// append minimal rname + numbers to satisfy length check and reach mname parsing
	rname := []byte{1, 'a', 0}
	nums := make([]byte, 20)
	wire = append(append(wire, rname...), nums...)
	_, err := decodeSOAData(wire)
	if err == nil || !strings.Contains(err.Error(), "invalid SOA mname") {
		t.Errorf("expected error for invalid mname, got: %v", err)
	}
}

func TestDecodeSOAData_InvalidRName(t *testing.T) {
	// Valid mname
	mname := []byte{2, 'n', 's', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0}
	// Corrupt rname: length byte claims 0xff but insufficient data before numbers
	rname := []byte{0xff, 'h', 'o', 's', 't'}
	nums := make([]byte, 20)
	wire := append(append(mname, rname...), nums...)
	_, err := decodeSOAData(wire)
	if err == nil || !strings.Contains(err.Error(), "invalid SOA rname") {
		t.Errorf("expected error for invalid rname, got: %v", err)
	}
}

func TestDecodeSOAData_MissingIntegerFields(t *testing.T) {
	// mname: a., rname: b., but only 19 bytes of integers (need 20) to trigger error path
	wire := append([]byte{1, 'a', 0, 1, 'b', 0}, make([]byte, 19)...)
	_, err := decodeSOAData(wire)
	require.Error(t, err)
	require.Contains(t, err.Error(), "SOA record missing integer fields")
}
