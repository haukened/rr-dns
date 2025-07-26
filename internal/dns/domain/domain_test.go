package domain

import (
	"testing"
	"time"
)

func TestNewResourceRecord_ValidInput(t *testing.T) {
	rr, err := NewResourceRecord("example.com.", RRType(1), RRClass(1), 3600, []byte{127, 0, 0, 1})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rr.Name != "example.com." {
		t.Errorf("expected Name 'example.com.', got %q", rr.Name)
	}
	if rr.Type != 1 {
		t.Errorf("expected Type 1, got %d", rr.Type)
	}
	if rr.Class != 1 {
		t.Errorf("expected Class 1, got %d", rr.Class)
	}
	ttl := time.Until(rr.ExpiresAt)
	if ttl < 3599*time.Second || ttl > 3601*time.Second {
		t.Errorf("expected TTL ~3600s, got %v", ttl)
	}
	if len(rr.Data) != 4 || rr.Data[0] != 127 {
		t.Errorf("unexpected Data: %v", rr.Data)
	}
}

func TestNewResourceRecord_EmptyName(t *testing.T) {
	_, err := NewResourceRecord("", RRType(1), RRClass(1), 60, []byte{1, 2, 3, 4})
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
	if want := "record name must not be empty"; err.Error() != want {
		t.Errorf("expected error %q, got %q", want, err.Error())
	}
}

func TestNewResourceRecord_InvalidRRType(t *testing.T) {
	_, err := NewResourceRecord("example.com.", RRType(9999), RRClass(1), 60, []byte{1})
	if err == nil {
		t.Fatal("expected error for invalid RRType, got nil")
	}
	if want := "invalid RRType: 9999"; err.Error() != want {
		t.Errorf("expected error %q, got %q", want, err.Error())
	}
}

func TestNewResourceRecord_InvalidRRClass(t *testing.T) {
	_, err := NewResourceRecord("example.com.", RRType(1), RRClass(9999), 60, []byte{1})
	if err == nil {
		t.Fatal("expected error for invalid RRClass, got nil")
	}
	if want := "invalid RRClass: 9999"; err.Error() != want {
		t.Errorf("expected error %q, got %q", want, err.Error())
	}
}
func TestRRType_IsValid(t *testing.T) {
	validTypes := []RRType{1, 2, 5, 6, 12, 15, 16, 28, 33, 41, 255, 257}
	for _, typ := range validTypes {
		if !typ.IsValid() {
			t.Errorf("expected RRType %d to be valid", typ)
		}
	}
	invalidTypes := []RRType{0, 3, 4, 7, 100, 9999}
	for _, typ := range invalidTypes {
		if typ.IsValid() {
			t.Errorf("expected RRType %d to be invalid", typ)
		}
	}
}

func TestRRClass_IsValid(t *testing.T) {
	validClasses := []RRClass{1, 3, 4, 255}
	for _, class := range validClasses {
		if !class.IsValid() {
			t.Errorf("expected RRClass %d to be valid", class)
		}
	}
	invalidClasses := []RRClass{0, 2, 5, 100, 9999}
	for _, class := range invalidClasses {
		if class.IsValid() {
			t.Errorf("expected RRClass %d to be invalid", class)
		}
	}
}

func TestRCode_IsValid(t *testing.T) {
	for i := 0; i <= 10; i++ {
		rc := RCode(i)
		if !rc.IsValid() {
			t.Errorf("expected RCode %d to be valid", i)
		}
	}
	invalidRCodes := []RCode{11, 12, 15, 100, 255}
	for _, rc := range invalidRCodes {
		if rc.IsValid() {
			t.Errorf("expected RCode %d to be invalid", rc)
		}
	}
}
func TestNewDNSQuery_ValidInput(t *testing.T) {
	q, err := NewDNSQuery(1234, "example.org.", RRType(1), RRClass(1))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if q.ID != 1234 {
		t.Errorf("expected ID 1234, got %d", q.ID)
	}
	if q.Name != "example.org." {
		t.Errorf("expected Name 'example.org.', got %q", q.Name)
	}
	if q.Type != 1 {
		t.Errorf("expected Type 1, got %d", q.Type)
	}
	if q.Class != 1 {
		t.Errorf("expected Class 1, got %d", q.Class)
	}
}

func TestNewDNSQuery_EmptyName(t *testing.T) {
	_, err := NewDNSQuery(1, "", RRType(1), RRClass(1))
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
	if want := "query name must not be empty"; err.Error() != want {
		t.Errorf("expected error %q, got %q", want, err.Error())
	}
}

func TestNewDNSQuery_UnsupportedRRType(t *testing.T) {
	_, err := NewDNSQuery(1, "example.org.", RRType(9999), RRClass(1))
	if err == nil {
		t.Fatal("expected error for unsupported RRType, got nil")
	}
	if want := "unsupported RRType: 9999"; err.Error() != want {
		t.Errorf("expected error %q, got %q", want, err.Error())
	}
}

func TestNewDNSQuery_UnsupportedRRClass(t *testing.T) {
	_, err := NewDNSQuery(1, "example.org.", RRType(1), RRClass(9999))
	if err == nil {
		t.Fatal("expected error for unsupported RRClass, got nil")
	}
	if want := "unsupported RRClass: 9999"; err.Error() != want {
		t.Errorf("expected error %q, got %q", want, err.Error())
	}
}

func TestResourceRecord_TTLRemaining(t *testing.T) {
	rr, err := NewResourceRecord("example.com.", RRType(1), RRClass(1), 5, []byte{127, 0, 0, 1})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	ttl := rr.TTLRemaining()
	if ttl <= 0 || ttl > 5*time.Second {
		t.Errorf("expected TTL remaining between 0 and 5s, got %v", ttl)
	}
}

func TestGenerateCacheKey(t *testing.T) {
	tests := []struct {
		name     string
		rrtype   RRType
		class    RRClass
		expected string
	}{
		{"example.com.", 1, 1, "example.com.:1:1"},
		{"test.org.", 28, 255, "test.org.:28:255"},
		{"", 5, 3, ":5:3"},
		{"zone.local.", 16, 4, "zone.local.:16:4"},
	}
	for _, tt := range tests {
		got := generateCacheKey(tt.name, tt.rrtype, tt.class)
		if got != tt.expected {
			t.Errorf("generateCacheKey(%q, %d, %d) = %q; want %q", tt.name, tt.rrtype, tt.class, got, tt.expected)
		}
	}
}
func TestDNSQuery_CacheKey(t *testing.T) {
	tests := []struct {
		id       uint16
		name     string
		rrtype   RRType
		class    RRClass
		expected string
	}{
		{1, "example.com.", 1, 1, "example.com.:1:1"},
		{2, "test.org.", 28, 255, "test.org.:28:255"},
		{3, "", 5, 3, ":5:3"},
		{4, "zone.local.", 16, 4, "zone.local.:16:4"},
	}
	for _, tt := range tests {
		q := DNSQuery{
			ID:    tt.id,
			Name:  tt.name,
			Type:  tt.rrtype,
			Class: tt.class,
		}
		got := q.CacheKey()
		if got != tt.expected {
			t.Errorf("DNSQuery.CacheKey() = %q; want %q", got, tt.expected)
		}
	}
}

func TestResourceRecord_CacheKey(t *testing.T) {
	tests := []struct {
		name     string
		rrtype   RRType
		class    RRClass
		expected string
	}{
		{"example.com.", 1, 1, "example.com.:1:1"},
		{"test.org.", 28, 255, "test.org.:28:255"},
		{"", 5, 3, ":5:3"},
		{"zone.local.", 16, 4, "zone.local.:16:4"},
	}
	for _, tt := range tests {
		rr := ResourceRecord{
			Name:  tt.name,
			Type:  tt.rrtype,
			Class: tt.class,
		}
		got := rr.CacheKey()
		if got != tt.expected {
			t.Errorf("ResourceRecord.CacheKey() = %q; want %q", got, tt.expected)
		}
	}
}
func TestRRTypeFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected RRType
	}{
		{"A", 1},
		{"NS", 2},
		{"CNAME", 5},
		{"SOA", 6},
		{"PTR", 12},
		{"MX", 15},
		{"TXT", 16},
		{"AAAA", 28},
		{"SRV", 33},
		{"OPT", 41},
		{"ANY", 255},
		{"CAA", 257},
		{"a", 1},
		{"ns", 2},
		{"cname", 5},
		{"soa", 6},
		{"ptr", 12},
		{"mx", 15},
		{"txt", 16},
		{"aaaa", 28},
		{"srv", 33},
		{"opt", 41},
		{"any", 255},
		{"caa", 257},
		{"", 0},
		{"unknown", 0},
		{"ZZZ", 0},
		{"123", 0},
	}
	for _, tt := range tests {
		got := RRTypeFromString(tt.input)
		if got != tt.expected {
			t.Errorf("RRTypeFromString(%q) = %d; want %d", tt.input, got, tt.expected)
		}
	}
}
