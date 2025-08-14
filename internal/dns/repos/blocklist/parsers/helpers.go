package parsers

import (
	"strings"
	"unicode"

	"github.com/haukened/rr-dns/internal/dns/common/utils"
	"github.com/haukened/rr-dns/internal/dns/domain"
)

// ruleKindFromRaw decides the BlockRuleKind based on the raw, uncanonicalized input.
// Returns BlockRuleSuffix if the name begins with "*." or ".", otherwise BlockRuleExact.
func ruleKindFromRaw(raw string) domain.BlockRuleKind {
	if strings.HasPrefix(raw, "*.") || strings.HasPrefix(raw, ".") {
		return domain.BlockRuleSuffix
	}
	return domain.BlockRuleExact
}

// isValidFQDN checks whether the provided string is a valid Fully Qualified Domain Name (FQDN).
// It enforces the following rules:
//   - The total length must not exceed 255 characters.
//   - The name must contain at least one label (separated by dots).
//   - Each label must be between 1 and 63 characters long.
//   - The first label must start with a letter, number, or wildcard character.
//
// Returns true if the input meets all FQDN requirements, false otherwise.
func isValidFQDN(name string) bool {
	// the maximum length of an FQDN must not exceed 255 characters
	if len(name) > 255 {
		return false
	}
	// require at least two labels (e.g., example.com)
	// but not localhost or local name resolution
	labels := strings.Split(name, ".")
	if len(labels) < 2 {
		return false
	}
	// each label must be no more than 63 characters
	for _, label := range labels {
		if len(label) > 63 || len(label) == 0 {
			return false
		}
	}
	// it must start only with a letter, number, or wildcard
	firstLabel := labels[0]
	runes := []rune(firstLabel)
	if !isAlphaNumeric(runes[0]) && !isWildcard(runes[0]) {
		return false
	}
	return true
}

// normalizeDomainName takes a domain name string, trims leading and trailing whitespace,
// removes any leading "*." or "." prefixes, and returns the canonical DNS name format
// using utils.CanonicalDNSName. This ensures the domain name is normalized for further processing.
func normalizeDomainName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, "*.")
	name = strings.TrimPrefix(name, ".")
	return utils.CanonicalDNSName(name)
}

// isAlphaNumeric reports whether the given rune is an ASCII letter or digit.
// It returns true if r is a letter (A-Z, a-z) or a digit (0-9), and false otherwise.
func isAlphaNumeric(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

// isWildcard checks if the given rune represents a wildcard character ('*').
// It returns true if the rune is '*', otherwise false.
func isWildcard(r rune) bool {
	return r == '*'
}

// stripLineBOM removes a potential UTF-8 BOM at the start of a line.
// Note: We only handle U+FEFF here as tests/loaders assume UTF-8 input.
func stripLineBOM(s string) string {
	return strings.TrimPrefix(s, "\uFEFF")
}

// classifyLine trims whitespace and classifies the line as empty or comment.
// Returns (isEmpty, isComment) based on the trimmed content.
func classifyLine(s string) (bool, bool) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return true, false
	}
	if strings.HasPrefix(trimmed, "#") {
		return false, true
	}
	return false, false
}

// stripInlineComment removes an inline comment starting with '#'.
// If no '#' is present, the original string is returned.
func stripInlineComment(s string) string {
	if idx := strings.IndexByte(s, '#'); idx >= 0 {
		return s[:idx]
	}
	return s
}
