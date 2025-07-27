package domain

import (
	"testing"
)

func TestRCode_IsValid(t *testing.T) {
	cases := []struct {
		code RCode
		want bool
	}{
		{0, true}, {1, true}, {2, true}, {3, true}, {4, true}, {5, true}, {6, true}, {7, true}, {8, true}, {9, true}, {10, true},
		{11, false}, {12, false}, {13, false}, {14, false}, {15, false}, {255, false},
	}
	for _, tc := range cases {
		if got := tc.code.IsValid(); got != tc.want {
			t.Errorf("IsValid(%d) = %v, want %v", tc.code, got, tc.want)
		}
	}
}

func TestRCode_String(t *testing.T) {
	cases := []struct {
		code RCode
		want string
	}{
		{0, "NOERROR"}, {1, "FORMERR"}, {2, "SERVFAIL"}, {3, "NXDOMAIN"}, {4, "NOTIMP"}, {5, "REFUSED"},
		{6, "YXDOMAIN"}, {7, "YXRRSET"}, {8, "NXRRSET"}, {9, "NOTAUTH"}, {10, "NOTZONE"},
		{11, "UNKNOWN(11)"}, {12, "UNKNOWN(12)"}, {255, "UNKNOWN(255)"},
	}
	for _, tc := range cases {
		if got := tc.code.String(); got != tc.want {
			t.Errorf("String(%d) = %v, want %v", tc.code, got, tc.want)
		}
	}
}

func TestParseRCode(t *testing.T) {
	cases := []struct {
		input string
		want  RCode
	}{
		{"NOERROR", 0}, {"FORMERR", 1}, {"SERVFAIL", 2}, {"NXDOMAIN", 3}, {"NOTIMP", 4}, {"REFUSED", 5},
		{"YXDOMAIN", 6}, {"YXRRSET", 7}, {"NXRRSET", 8}, {"NOTAUTH", 9}, {"NOTZONE", 10},
		{"UNKNOWN", 0}, {"", 0}, {"foo", 0},
	}
	for _, tc := range cases {
		if got := ParseRCode(tc.input); got != tc.want {
			t.Errorf("ParseRCode(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}
