package domain

import (
	"testing"
)

func TestRRClass_IsValid(t *testing.T) {
	cases := []struct {
		class RRClass
		want  bool
	}{
		{1, true},
		{3, true},
		{4, true},
		{254, true},
		{255, true},
		{9999, false},
	}
	for _, tc := range cases {
		if got := tc.class.IsValid(); got != tc.want {
			t.Errorf("IsValid(%d) = %v, want %v", tc.class, got, tc.want)
		}
	}
}

func TestRRClass_String(t *testing.T) {
	cases := []struct {
		class RRClass
		want  string
	}{
		{1, "IN"},
		{3, "CH"},
		{4, "HS"},
		{254, "NONE"},
		{255, "ANY"},
		{9999, "UNKNOWN"},
	}
	for _, tc := range cases {
		if got := tc.class.String(); got != tc.want {
			t.Errorf("String(%d) = %v, want %v", tc.class, got, tc.want)
		}
	}
}

func TestParseRRClass(t *testing.T) {
	cases := []struct {
		input string
		want  RRClass
	}{
		{"IN", 1},
		{"CH", 3},
		{"HS", 4},
		{"NONE", 254},
		{"ANY", 255},
		{"UNKNOWN", 0},
	}
	for _, tc := range cases {
		if got := ParseRRClass(tc.input); got != tc.want {
			t.Errorf("ParseRRClass(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}
