package domain

import (
	"testing"
)

func TestNewDNSQuery(t *testing.T) {
	cases := []struct {
		name    string
		id      uint16
		qname   string
		qtype   RRType
		qclass  RRClass
		wantErr bool
	}{
		{"valid query", 1, "example.com.", 1, 1, false},
		{"empty name", 1, "", 1, 1, true},
		{"invalid type", 1, "example.com.", 9999, 1, true},
		{"invalid class", 1, "example.com.", 1, 9999, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewDNSQuery(tc.id, tc.qname, tc.qtype, tc.qclass)
			if (err != nil) != tc.wantErr {
				t.Errorf("NewDNSQuery() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestDNSQuery_Validate(t *testing.T) {
	q := DNSQuery{ID: 1, Name: "example.com.", Type: 1, Class: 1}
	if err := q.Validate(); err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
	q.Name = ""
	if err := q.Validate(); err == nil {
		t.Errorf("Validate() error = nil, want error for empty name")
	}
	q.Name = "example.com."
	q.Type = 9999
	if err := q.Validate(); err == nil {
		t.Errorf("Validate() error = nil, want error for invalid type")
	}
	q.Type = 1
	q.Class = 9999
	if err := q.Validate(); err == nil {
		t.Errorf("Validate() error = nil, want error for invalid class")
	}
}

func TestDNSQuery_CacheKey(t *testing.T) {
	q := DNSQuery{ID: 1, Name: "example.com.", Type: 1, Class: 1}
	want := "example.com.:1:1"
	if got := q.CacheKey(); got != want {
		t.Errorf("CacheKey() = %v, want %v", got, want)
	}
}
