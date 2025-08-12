package rrdata

import (
	"bytes"
	"strings"
	"testing"
)

func TestEncodeTXTData_SingleSegment(t *testing.T) {
	data := "hello world"
	expected := append([]byte{byte(len(data))}, []byte(data)...)
	result, err := encodeTXTData(data)
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
	result, err := encodeTXTData(data)
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
	result, err := encodeTXTData(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestEncodeTXTData_SegmentTooLong(t *testing.T) {
	longSegment := strings.Repeat("a", 256)
	_, err := encodeTXTData(longSegment)
	if err == nil || !strings.Contains(err.Error(), "TXT segment too long") {
		t.Errorf("expected segment too long error, got %v", err)
	}
}

func TestEncodeTXTData_AllSegmentsEmpty(t *testing.T) {
	_, err := encodeTXTData(" ; ; ")
	if err == nil || !strings.Contains(err.Error(), "must contain at least one segment") {
		t.Errorf("expected error for empty segments, got %v", err)
	}
}

func TestDecodeTXTData_SingleSegment(t *testing.T) {
	input := []byte{11, 'h', 'e', 'l', 'l', 'o', ' ', 'w', 'o', 'r', 'l', 'd'}
	expected := "hello world"
	result, err := decodeTXTData(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestDecodeTXTData_MultipleSegments(t *testing.T) {
	input := []byte{
		3, 'f', 'o', 'o',
		3, 'b', 'a', 'r',
		3, 'b', 'a', 'z',
	}
	expected := "foo; bar; baz"
	result, err := decodeTXTData(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestDecodeTXTData_EmptyInput(t *testing.T) {
	input := []byte{}
	expected := ""
	result, err := decodeTXTData(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestDecodeTXTData_ZeroLengthSegment(t *testing.T) {
	input := []byte{0}
	expected := ""
	result, err := decodeTXTData(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestDecodeTXTData_InvalidSegmentLength(t *testing.T) {
	input := []byte{5, 'a', 'b'}
	_, err := decodeTXTData(input)
	if err == nil || !strings.Contains(err.Error(), "segment length exceeds remaining data") {
		t.Errorf("expected segment length error, got %v", err)
	}
}
