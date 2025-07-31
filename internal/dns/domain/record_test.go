package domain

import (
	"testing"
	"time"
)

func TestNewResourceRecord(t *testing.T) {
	rr, err := NewResourceRecord("example.com.", 1, 1, 60, []byte{1, 2, 3, 4})
	if err != nil {
		t.Errorf("NewResourceRecord() error = %v, want nil", err)
	}
	if rr.Name != "example.com." {
		t.Errorf("Name = %v, want %v", rr.Name, "example.com.")
	}
	if rr.Type != 1 {
		t.Errorf("Type = %v, want %v", rr.Type, 1)
	}
	if rr.Class != 1 {
		t.Errorf("Class = %v, want %v", rr.Class, 1)
	}
	if rr.Data == nil || len(rr.Data) != 4 {
		t.Errorf("Data = %v, want length 4", rr.Data)
	}
	if rr.ExpiresAt.Before(time.Now()) {
		t.Errorf("ExpiresAt = %v, want future time", rr.ExpiresAt)
	}
}

func TestNewResourceRecord_Invalid(t *testing.T) {
	cases := []struct {
		name    string
		rrtype  RRType
		class   RRClass
		wantErr bool
	}{
		{"empty name", 1, 1, true},
		{"invalid type", 9999, 1, true},
		{"invalid class", 1, 9999, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewResourceRecord("", tc.rrtype, tc.class, 60, []byte{1, 2, 3, 4})
			if (err != nil) != tc.wantErr {
				t.Errorf("NewResourceRecord() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestResourceRecord_Validate(t *testing.T) {
	rr := ResourceRecord{Name: "example.com.", Type: 1, Class: 1, ExpiresAt: time.Now().Add(60 * time.Second), Data: []byte{1, 2, 3, 4}}
	if err := rr.Validate(); err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
	rr.Name = ""
	if err := rr.Validate(); err == nil {
		t.Errorf("Validate() error = nil, want error for empty name")
	}
	rr.Name = "example.com."
	rr.Type = 9999
	if err := rr.Validate(); err == nil {
		t.Errorf("Validate() error = nil, want error for invalid type")
	}
	rr.Type = 1
	rr.Class = 9999
	if err := rr.Validate(); err == nil {
		t.Errorf("Validate() error = nil, want error for invalid class")
	}
}

func TestResourceRecord_Invalid(t *testing.T) {
	cases := []struct {
		name string
		rr   ResourceRecord
	}{
		{
			name: "empty name",
			rr:   ResourceRecord{Name: "", Type: 1, Class: 1, ExpiresAt: time.Now().Add(60 * time.Second), Data: []byte{1}},
		},
		{
			name: "invalid type",
			rr:   ResourceRecord{Name: "example.com.", Type: 9999, Class: 1, ExpiresAt: time.Now().Add(60 * time.Second), Data: []byte{1}},
		},
		{
			name: "invalid class",
			rr:   ResourceRecord{Name: "example.com.", Type: 1, Class: 9999, ExpiresAt: time.Now().Add(60 * time.Second), Data: []byte{1}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.rr.Validate(); err == nil {
				t.Errorf("Validate() for %s: got nil, want error", tc.name)
			}
		})
	}
}

func TestResourceRecord_TTLRemaining(t *testing.T) {
	rr := ResourceRecord{ExpiresAt: time.Now().Add(60 * time.Second)}
	ttl := rr.TTLRemaining()
	if ttl < 59*time.Second || ttl > 61*time.Second {
		t.Errorf("TTLRemaining() = %v, want ~60s", ttl)
	}
}

func TestResourceRecord_TTL_Underflow(t *testing.T) {
	rr := ResourceRecord{ExpiresAt: time.Now().Add(-60 * time.Second)}
	ttl := rr.TTLRemaining()
	if ttl != 0 {
		t.Errorf("TTLRemaining() = %v, want 0 for expired record", ttl)
	}
}

func TestResourceRecord_CacheKey(t *testing.T) {
	rr := ResourceRecord{Name: "example.com.", Type: 1, Class: 1}
	want := "example.com.|example.com|A|IN"
	if got := rr.CacheKey(); got != want {
		t.Errorf("CacheKey() = %v, want %v", got, want)
	}
}
