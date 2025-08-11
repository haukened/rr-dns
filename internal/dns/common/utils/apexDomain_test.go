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
			expected: "example.com",
		},
		{
			name:     "simple domain without trailing dot",
			input:    "example.com",
			expected: "example.com",
		},
		{
			name:     "subdomain with trailing dot",
			input:    "www.example.com.",
			expected: "example.com",
		},
		{
			name:     "subdomain without trailing dot",
			input:    "www.example.com",
			expected: "example.com",
		},
		{
			name:     "deep subdomain",
			input:    "api.service.example.com",
			expected: "example.com",
		},
		{
			name:     "co.uk domain",
			input:    "example.co.uk",
			expected: "example.co.uk",
		},
		{
			name:     "subdomain of co.uk",
			input:    "www.example.co.uk",
			expected: "example.co.uk",
		},
		{
			name:     "github.io subdomain",
			input:    "user.github.io",
			expected: "user.github.io",
		},
		{
			name:     "complex subdomain of github.io",
			input:    "subdomain.user.github.io",
			expected: "user.github.io",
		},
		{
			name:     "single label fallback",
			input:    "localhost",
			expected: "localhost",
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
			expected: "invalid..domain",
		},
		{
			name:     "numeric IP fallback",
			input:    "192.168.1.1",
			expected: "1.1",
		},
		{
			name:     "very long subdomain",
			input:    "very.long.subdomain.chain.example.com",
			expected: "example.com",
		},
		{
			name:     "domain with multiple trailing dots",
			input:    "example.com..",
			expected: "example.com",
		},
		{
			name:     "valid custom TLD",
			input:    "foo.domains.google.",
			expected: "domains.google",
		},
		{
			name:     "invalid custom TLD",
			input:    "foo.invalidtld.",
			expected: "foo.invalidtld",
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
