package parsers

import (
	"strings"
	"testing"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

func TestNormalizeDomainName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"example.com", "example.com"},
		{" example.com. ", "example.com"},
		{"*.example.com", "example.com"},
		{".example.com", "example.com"},
		{"*.example.com.", "example.com"},
		{".example.com.", "example.com"},
		{"", ""},
		{"   ", ""},
		{"*.", ""},
		{".", ""},
		{"*.sub.domain.example.com.", "sub.domain.example.com"},
	}

	for _, tt := range tests {
		got := normalizeDomainName(tt.in)
		if got != tt.want {
			t.Errorf("normalizeDomainName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRuleKindFromRaw(t *testing.T) {
	tests := []struct {
		raw  string
		want domain.BlockRuleKind
	}{
		{"example.com", domain.BlockRuleExact},
		{" example.com", domain.BlockRuleExact},
		{"*.example.com", domain.BlockRuleSuffix},
		{".example.com", domain.BlockRuleSuffix},
		{"*.example.com.", domain.BlockRuleSuffix},
		{".example.com.", domain.BlockRuleSuffix},
		{"", domain.BlockRuleExact},
	}

	for _, tt := range tests {
		got := ruleKindFromRaw(tt.raw)
		if got != tt.want {
			t.Errorf("ruleKindFromRaw(%q) = %v, want %v", tt.raw, got, tt.want)
		}
	}
}
func TestIsWildcard(t *testing.T) {
	tests := []struct {
		r    rune
		want bool
	}{
		{'*', true},
		{'a', false},
		{'A', false},
		{'1', false},
		{'.', false},
		{'-', false},
		{'_', false},
		{' ', false},
		{0, false},
	}

	for _, tt := range tests {
		got := isWildcard(tt.r)
		if got != tt.want {
			t.Errorf("isWildcard(%q) = %v, want %v", tt.r, got, tt.want)
		}
	}
}
func TestIsAlphaNumeric(t *testing.T) {
	tests := []struct {
		r    rune
		want bool
	}{
		{'a', true},
		{'z', true},
		{'A', true},
		{'Z', true},
		{'0', true},
		{'9', true},
		{'m', true},
		{'M', true},
		{'5', true},
		{'*', false},
		{'-', false},
		{'.', false},
		{'_', false},
		{' ', false},
		{'$', false},
		{0, false},
	}

	for _, tt := range tests {
		got := isAlphaNumeric(tt.r)
		if got != tt.want {
			t.Errorf("isAlphaNumeric(%q) = %v, want %v", tt.r, got, tt.want)
		}
	}
}
func TestIsValidFQDN(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		// Valid FQDNs
		{"example.com", true},
		{"sub.domain.example.com", true},
		{"a.b", true},
		{"a-b.c-d", true},
		{"1a.2b", true},
		{"*.example.com", true},
		{"xn--d1acufc.xn--p1ai", true}, // punycode

		// Invalid: too long (>255)
		{strings.Repeat("a", 64) + "." + strings.Repeat("b", 64) + "." + strings.Repeat("c", 64) + "." + strings.Repeat("d", 64) + ".com", false},

		// Invalid: label too long (>63)
		{strings.Repeat("a", 64) + ".com", false},
		{"com." + strings.Repeat("b", 64), false},

		// Invalid: empty label
		{"example..com", false},
		{".example.com", false},
		{"example.com.", false}, // last label is empty

		// Invalid: less than two labels
		{"localhost", false},
		{"", false},

		// Invalid: first label does not start with alphanum or wildcard
		{"-abc.com", false},
		{".abc.com", false},
		{"_abc.com", false},
		{"#abc.com", false},

		// Valid: first label is wildcard
		{"*.abc.com", true},
	}

	for _, tt := range tests {
		got := isValidFQDN(tt.name)
		if got != tt.want {
			t.Errorf("isValidFQDN(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}
