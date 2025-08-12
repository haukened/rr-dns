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
			want: "example.com|www.example.com|A|IN",
		},
		{
			name: "AAAA record in example.org zone",
			fqdn: "foo.example.org.",
			t:    28, // AAAA
			c:    1,  // IN
			want: "example.org|foo.example.org|AAAA|IN",
		},
		{
			name: "CNAME record in github.io zone",
			fqdn: "pages.github.io.",
			t:    5, // CNAME
			c:    1, // IN
			want: "pages.github.io|pages.github.io|CNAME|IN",
		},
		{
			name: "subdomain in same zone",
			fqdn: "sub.www.example.com.",
			t:    1, // A
			c:    1, // IN
			want: "example.com|sub.www.example.com|A|IN",
		},
		{
			name: "fallback for unknown TLD",
			fqdn: "foo.unknowntld.",
			t:    1, // A
			c:    1, // IN
			want: "foo.unknowntld|foo.unknowntld|A|IN",
		},
		{
			name: "name without trailing dot",
			fqdn: "www.example.com",
			t:    1, // A
			c:    1, // IN
			want: "example.com|www.example.com|A|IN",
		},
		{
			name: "mixed case domain name",
			fqdn: "WwW.ExAmPlE.CoM",
			t:    1, // A
			c:    1, // IN
			want: "example.com|www.example.com|A|IN",
		},
		{
			name: "domain with whitespace",
			fqdn: "  www.example.com  ",
			t:    1, // A
			c:    1, // IN
			want: "example.com|www.example.com|A|IN",
		},
		{
			name: "empty string input",
			fqdn: "",
			t:    1, // A
			c:    1, // IN
			want: "||A|IN",
		},
		{
			name: "whitespace only input",
			fqdn: "   ",
			t:    1, // A
			c:    1, // IN
			want: "||A|IN",
		},
		{
			name: "root domain",
			fqdn: ".",
			t:    1, // A
			c:    1, // IN
			want: "||A|IN",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := GenerateCacheKey(tc.fqdn, tc.t, tc.c)
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
			want: "example.com|ipv6.example.com|AAAA|IN",
		},
		{
			name: "URI records with colons and ports",
			fqdn: "uri.example.com.",
			t:    256, // URI (would contain https://example.com:8080 in rdata)
			c:    1,   // IN
			want: "example.com|uri.example.com|UNKNOWN(256)|IN",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := GenerateCacheKey(tc.fqdn, tc.t, tc.c)
			if got != tc.want {
				t.Errorf("generateCacheKey(%q, %d, %d) = %q, want %q",
					tc.fqdn, tc.t, tc.c, got, tc.want)
			}
		})
	}
}
