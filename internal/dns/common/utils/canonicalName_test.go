package utils

import (
	"strings"
	"testing"
)

func TestCanonicalDNSName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple domain without trailing dot",
			input:    "example.com",
			expected: "example.com.",
		},
		{
			name:     "simple domain with trailing dot",
			input:    "example.com.",
			expected: "example.com.",
		},
		{
			name:     "uppercase domain",
			input:    "EXAMPLE.COM",
			expected: "example.com.",
		},
		{
			name:     "mixed case domain",
			input:    "ExAmPlE.CoM",
			expected: "example.com.",
		},
		{
			name:     "domain with leading whitespace",
			input:    "  example.com",
			expected: "example.com.",
		},
		{
			name:     "domain with trailing whitespace",
			input:    "example.com  ",
			expected: "example.com.",
		},
		{
			name:     "domain with leading and trailing whitespace",
			input:    "  example.com  ",
			expected: "example.com.",
		},
		{
			name:     "domain with tabs and spaces",
			input:    "\t example.com \t",
			expected: "example.com.",
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
		{
			name:     "deep subdomain with mixed case",
			input:    "API.Service.EXAMPLE.com",
			expected: "api.service.example.com.",
		},
		{
			name:     "root domain",
			input:    ".",
			expected: ".",
		},
		{
			name:     "root domain with whitespace",
			input:    " . ",
			expected: ".",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: "",
		},
		{
			name:     "tab only",
			input:    "\t",
			expected: "",
		},
		{
			name:     "newline and whitespace",
			input:    " \n \t ",
			expected: "",
		},
		{
			name:     "single label domain",
			input:    "localhost",
			expected: "localhost.",
		},
		{
			name:     "single label with case and whitespace",
			input:    " LOCALHOST ",
			expected: "localhost.",
		},
		{
			name:     "IDN domain (ASCII form)",
			input:    "xn--nxasmq6b.xn--j6w193g",
			expected: "xn--nxasmq6b.xn--j6w193g.",
		},
		{
			name:     "domain with numbers",
			input:    "test123.example.com",
			expected: "test123.example.com.",
		},
		{
			name:     "domain with hyphens",
			input:    "sub-domain.example-site.com",
			expected: "sub-domain.example-site.com.",
		},
		{
			name:     "very long domain name",
			input:    "very.long.subdomain.chain.with.many.labels.example.com",
			expected: "very.long.subdomain.chain.with.many.labels.example.com.",
		},
		{
			name:     "domain with mixed case and whitespace and dot",
			input:    "  WwW.ExAmPlE.CoM.  ",
			expected: "www.example.com.",
		},
		{
			name:     "special characters in domain (valid DNS)",
			input:    "test-123.example_site.com",
			expected: "test-123.example_site.com.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanonicalDNSName(tt.input)
			if got != tt.expected {
				t.Errorf("CanonicalDNSName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCanonicalDNSName_Properties(t *testing.T) {
	t.Run("idempotent behavior", func(t *testing.T) {
		testCases := []string{
			"example.com",
			"EXAMPLE.COM",
			"  www.example.com  ",
			"localhost",
			".",
		}

		for _, input := range testCases {
			first := CanonicalDNSName(input)
			second := CanonicalDNSName(first)
			if first != second {
				t.Errorf("CanonicalDNSName is not idempotent for input %q: first=%q, second=%q", input, first, second)
			}
		}
	})

	t.Run("consistent behavior with repeated calls", func(t *testing.T) {
		input := "  ExAmPlE.CoM  "
		first := CanonicalDNSName(input)
		second := CanonicalDNSName(input)
		if first != second {
			t.Errorf("CanonicalDNSName is not deterministic: first=%q, second=%q", first, second)
		}
	})

	t.Run("always lowercase output", func(t *testing.T) {
		inputs := []string{
			"EXAMPLE.COM",
			"WwW.ExAmPlE.CoM",
			"API.SERVICE.EXAMPLE.COM",
			"LOCALHOST",
		}

		for _, input := range inputs {
			got := CanonicalDNSName(input)
			if got != "" && got != strings.ToLower(got) {
				t.Errorf("CanonicalDNSName(%q) = %q, expected lowercase output", input, got)
			}
		}
	})

	t.Run("no leading or trailing whitespace in output", func(t *testing.T) {
		inputs := []string{
			"  example.com  ",
			"\texample.com\t",
			" \n example.com \n ",
			"   localhost   ",
		}

		for _, input := range inputs {
			got := CanonicalDNSName(input)
			if got != "" && (strings.HasPrefix(got, " ") || strings.HasSuffix(got, " ") ||
				strings.HasPrefix(got, "\t") || strings.HasSuffix(got, "\t")) {
				t.Errorf("CanonicalDNSName(%q) = %q, output contains leading/trailing whitespace", input, got)
			}
		}
	})

	t.Run("non-empty input produces output ending with dot", func(t *testing.T) {
		inputs := []string{
			"example.com",
			"www.example.com",
			"localhost",
			"EXAMPLE.COM",
			"  example.com  ",
		}

		for _, input := range inputs {
			got := CanonicalDNSName(input)
			if got != "" && !strings.HasSuffix(got, ".") {
				t.Errorf("CanonicalDNSName(%q) = %q, expected non-empty output to end with dot", input, got)
			}
		}
	})

	t.Run("empty or whitespace-only input produces empty output", func(t *testing.T) {
		inputs := []string{
			"",
			" ",
			"  ",
			"\t",
			"\n",
			" \t \n ",
		}

		for _, input := range inputs {
			got := CanonicalDNSName(input)
			if got != "" {
				t.Errorf("CanonicalDNSName(%q) = %q, expected empty output for whitespace-only input", input, got)
			}
		}
	})
}
