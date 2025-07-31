package utils

import (
	"testing"
)

func TestGetApexDomain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple domain with trailing dot",
			input:    "example.com.",
			expected: "example.com.",
		},
		{
			name:     "simple domain without trailing dot",
			input:    "example.com",
			expected: "example.com.",
		},
		{
			name:     "subdomain with trailing dot",
			input:    "www.example.com.",
			expected: "example.com.",
		},
		{
			name:     "subdomain without trailing dot",
			input:    "www.example.com",
			expected: "example.com.",
		},
		{
			name:     "deep subdomain",
			input:    "api.service.example.com",
			expected: "example.com.",
		},
		{
			name:     "co.uk domain",
			input:    "example.co.uk",
			expected: "example.co.uk.",
		},
		{
			name:     "subdomain of co.uk",
			input:    "www.example.co.uk",
			expected: "example.co.uk.",
		},
		{
			name:     "github.io subdomain",
			input:    "user.github.io",
			expected: "user.github.io.",
		},
		{
			name:     "complex subdomain of github.io",
			input:    "subdomain.user.github.io",
			expected: "user.github.io.",
		},
		{
			name:     "single label fallback",
			input:    "localhost",
			expected: "localhost.",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "root domain",
			input:    ".",
			expected: "",
		},
		{
			name:     "invalid domain fallback",
			input:    "invalid..domain",
			expected: "invalid..domain.",
		},
		{
			name:     "numeric IP fallback",
			input:    "192.168.1.1",
			expected: "1.1.",
		},
		{
			name:     "very long subdomain",
			input:    "very.long.subdomain.chain.example.com",
			expected: "example.com.",
		},
		{
			name:     "domain with multiple trailing dots",
			input:    "example.com..",
			expected: "example.com.",
		},
		{
			name:     "valid custom TLD",
			input:    "foo.domains.google.",
			expected: "domains.google.",
		},
		{
			name:     "invalid custom TLD",
			input:    "foo.invalidtld.",
			expected: "foo.invalidtld.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetApexDomain(tt.input)
			if got != tt.expected {
				t.Errorf("GetApexDomain(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestRemoveTrailingDot(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "domain with trailing dot",
			input:    "example.com.",
			expected: "example.com",
		},
		{
			name:     "domain without trailing dot",
			input:    "example.com",
			expected: "example.com",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single dot",
			input:    ".",
			expected: "",
		},
		{
			name:     "multiple trailing dots",
			input:    "example.com..",
			expected: "example.com.",
		},
		{
			name:     "domain with internal dots only",
			input:    "sub.example.com",
			expected: "sub.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeTrailingDot(tt.input)
			if got != tt.expected {
				t.Errorf("removeTrailingDot(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestAddTrailingDot(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "domain without trailing dot",
			input:    "example.com",
			expected: "example.com.",
		},
		{
			name:     "domain with trailing dot",
			input:    "example.com.",
			expected: "example.com.",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single dot",
			input:    ".",
			expected: ".",
		},
		{
			name:     "subdomain without trailing dot",
			input:    "www.example.com",
			expected: "www.example.com.",
		},
		{
			name:     "subdomain with trailing dot",
			input:    "www.example.com.",
			expected: "www.example.com.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := addTrailingDot(tt.input)
			if got != tt.expected {
				t.Errorf("addTrailingDot(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGetApexDomain_EdgeCases(t *testing.T) {
	t.Run("handles publicsuffix parsing errors gracefully", func(t *testing.T) {
		// Test with various malformed inputs that might cause publicsuffix to error
		malformedInputs := []string{
			"invalid..domain",
			"domain.with.trailing.spaces ",
			"192.168.1.1", // IP addresses
			"::1",         // IPv6
		}

		for _, input := range malformedInputs {
			got := GetApexDomain(input)
			// Should not panic and should return something sensible
			if got == "" {
				// Empty string is acceptable for some edge cases
				continue
			}
			// Should always end with a dot when non-empty
			if got[len(got)-1] != '.' {
				t.Errorf("GetApexDomain(%q) = %q, expected non-empty result to end with dot", input, got)
			}
		}
	})

	t.Run("consistent behavior with repeated calls", func(t *testing.T) {
		input := "www.example.com"
		first := GetApexDomain(input)
		second := GetApexDomain(input)
		if first != second {
			t.Errorf("GetApexDomain is not deterministic: first=%q, second=%q", first, second)
		}
	})

	t.Run("idempotent with already processed domains", func(t *testing.T) {
		input := "www.example.com"
		first := GetApexDomain(input)
		// Apply again to the result
		second := GetApexDomain(first)
		if first != second {
			t.Errorf("GetApexDomain is not idempotent: first=%q, second=%q", first, second)
		}
	})
}
