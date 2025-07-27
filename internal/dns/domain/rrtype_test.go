package domain

import (
	"testing"
)

func TestRRType_IsValid(t *testing.T) {
	cases := []struct {
		value RRType
		want  bool
	}{
		{1, true}, {2, true}, {5, true}, {6, true}, {12, true}, {15, true}, {16, true}, {28, true},
		{33, true}, {35, true}, {41, true}, {43, true}, {46, true}, {47, true}, {48, true}, {52, true},
		{64, true}, {65, true}, {255, true}, {257, true},
		{0, false}, {3, false}, {4, false}, {7, false}, {8, false}, {9, false}, {10, false}, {11, false},
		{13, false}, {14, false}, {17, false}, {18, false}, {19, false}, {20, false}, {100, false}, {999, false}, {9999, false},
	}
	for _, tc := range cases {
		if got := tc.value.IsValid(); got != tc.want {
			t.Errorf("IsValid(%d) = %v, want %v", tc.value, got, tc.want)
		}
	}
}

func TestRRType_String(t *testing.T) {
	cases := []struct {
		t    RRType
		want string
	}{
		{1, "A"}, {2, "NS"}, {5, "CNAME"}, {6, "SOA"}, {12, "PTR"}, {15, "MX"}, {16, "TXT"},
		{28, "AAAA"}, {33, "SRV"}, {35, "NAPTR"}, {41, "OPT"}, {43, "DS"}, {46, "RRSIG"},
		{47, "NSEC"}, {48, "DNSKEY"}, {52, "TLSA"}, {64, "SVCB"}, {65, "HTTPS"}, {255, "ANY"}, {257, "CAA"},
		{0, "UNKNOWN(0)"}, {3, "UNKNOWN(3)"}, {9999, "UNKNOWN(9999)"},
	}
	for _, tc := range cases {
		if got := tc.t.String(); got != tc.want {
			t.Errorf("String(%d) = %v, want %v", tc.t, got, tc.want)
		}
	}
}

func TestRRTypeFromString(t *testing.T) {
	cases := []struct {
		input string
		want  RRType
	}{
		{"A", 1}, {"NS", 2}, {"CNAME", 5}, {"SOA", 6}, {"PTR", 12}, {"MX", 15}, {"TXT", 16},
		{"AAAA", 28}, {"SRV", 33}, {"NAPTR", 35}, {"OPT", 41}, {"DS", 43}, {"RRSIG", 46},
		{"NSEC", 47}, {"DNSKEY", 48}, {"TLSA", 52}, {"SVCB", 64}, {"HTTPS", 65}, {"ANY", 255}, {"CAA", 257},
		{"UNKNOWN", 0}, {"", 0}, {"foo", 0},
	}
	for _, tc := range cases {
		if got := RRTypeFromString(tc.input); got != tc.want {
			t.Errorf("RRTypeFromString(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}
