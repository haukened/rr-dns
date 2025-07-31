package domain

import (
	"testing"
)

func TestGenerateCacheKey(t *testing.T) {
	cases := []struct {
		name string
		fqdn string
		t    RRType
		c    RRClass
		want string
	}{
		{
			name: "A record in example.com zone",
			fqdn: "www.example.com.",
			t:    1, // A
			c:    1, // IN
			want: "example.com.|www.example.com|A|IN",
		},
		{
			name: "AAAA record in example.org zone",
			fqdn: "foo.example.org.",
			t:    28, // AAAA
			c:    1,  // IN
			want: "example.org.|foo.example.org|AAAA|IN",
		},
		{
			name: "CNAME record in github.io zone",
			fqdn: "pages.github.io.",
			t:    5, // CNAME
			c:    1, // IN
			want: "pages.github.io.|pages.github.io|CNAME|IN",
		},
		{
			name: "subdomain in same zone",
			fqdn: "sub.www.example.com.",
			t:    1, // A
			c:    1, // IN
			want: "example.com.|sub.www.example.com|A|IN",
		},
		{
			name: "fallback for unknown TLD",
			fqdn: "foo.unknowntld.",
			t:    1, // A
			c:    1, // IN
			want: "foo.unknowntld.|foo.unknowntld|A|IN",
		},
		{
			name: "name without trailing dot",
			fqdn: "www.example.com",
			t:    1, // A
			c:    1, // IN
			want: "example.com.|www.example.com|A|IN",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := generateCacheKey(tc.fqdn, tc.t, tc.c)
			if got != tc.want {
				t.Errorf("generateCacheKey(%q, %d, %d) = %q, want %q",
					tc.fqdn, tc.t, tc.c, got, tc.want)
			}
		})
	}
}

func TestGenerateCacheKey_PipeSeparatorSafety(t *testing.T) {
	// Test that pipe separators work correctly with IPv6 and URIs
	cases := []struct {
		name string
		fqdn string
		t    RRType
		c    RRClass
		want string
	}{
		{
			name: "IPv6 addresses with colons",
			fqdn: "ipv6.example.com.",
			t:    28, // AAAA (would contain 2001:db8::1 in rdata)
			c:    1,  // IN
			want: "example.com.|ipv6.example.com|AAAA|IN",
		},
		{
			name: "URI records with colons and ports",
			fqdn: "uri.example.com.",
			t:    256, // URI (would contain https://example.com:8080 in rdata)
			c:    1,   // IN
			want: "example.com.|uri.example.com|UNKNOWN(256)|IN",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := generateCacheKey(tc.fqdn, tc.t, tc.c)
			if got != tc.want {
				t.Errorf("generateCacheKey(%q, %d, %d) = %q, want %q",
					tc.fqdn, tc.t, tc.c, got, tc.want)
			}
		})
	}
}

func TestRemoveTrailingDot(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "with trailing dot",
			input: "example.com.",
			want:  "example.com",
		},
		{
			name:  "without trailing dot",
			input: "example.com",
			want:  "example.com",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "single dot",
			input: ".",
			want:  "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := removeTrailingDot(tc.input)
			if got != tc.want {
				t.Errorf("removeTrailingDot(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestAddTrailingDot(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "without trailing dot",
			input: "example.com",
			want:  "example.com.",
		},
		{
			name:  "with trailing dot",
			input: "example.com.",
			want:  "example.com.",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "single dot",
			input: ".",
			want:  ".",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := addTrailingDot(tc.input)
			if got != tc.want {
				t.Errorf("addTrailingDot(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
