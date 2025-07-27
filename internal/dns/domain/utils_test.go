package domain

import (
	"testing"
)

func TestGenerateCacheKey(t *testing.T) {
	cases := []struct {
		name string
		t    RRType
		c    RRClass
		want string
	}{
		{"example.com.", 1, 1, "example.com.:1:1"},
		{"foo.local.", 28, 255, "foo.local.:28:255"},
		{"bar.", 5, 3, "bar.:5:3"},
	}
	for _, tc := range cases {
		got := generateCacheKey(tc.name, tc.t, tc.c)
		if got != tc.want {
			t.Errorf("generateCacheKey(%q, %d, %d) = %q, want %q", tc.name, tc.t, tc.c, got, tc.want)
		}
	}
}
