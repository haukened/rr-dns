package rrdata

import (
	"bytes"
	"strings"
	"testing"
)

func TestEncodeTXTData_SingleSegment(t *testing.T) {
	data := "hello world"
	expected := append([]byte{byte(len(data))}, []byte(data)...)
	result, err := EncodeTXTData(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestEncodeTXTData_MultipleSegments(t *testing.T) {
	data := "foo;bar;baz"
	expected := []byte{
		3, 'f', 'o', 'o',
		3, 'b', 'a', 'r',
		3, 'b', 'a', 'z',
	}
	result, err := EncodeTXTData(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestEncodeTXTData_EmptySegmentIgnored(t *testing.T) {
	data := "foo;;bar"
	expected := []byte{
		3, 'f', 'o', 'o',
		3, 'b', 'a', 'r',
	}
	result, err := EncodeTXTData(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestEncodeTXTData_SegmentTooLong(t *testing.T) {
	longSegment := strings.Repeat("a", 256)
	_, err := EncodeTXTData(longSegment)
	if err == nil || !strings.Contains(err.Error(), "TXT segment too long") {
		t.Errorf("expected segment too long error, got %v", err)
	}
}

func TestEncodeTXTData_AllSegmentsEmpty(t *testing.T) {
	_, err := EncodeTXTData(" ; ; ")
	if err == nil || !strings.Contains(err.Error(), "must contain at least one segment") {
		t.Errorf("expected error for empty segments, got %v", err)
	}
}
