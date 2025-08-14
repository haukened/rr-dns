package parsers

import (
	"bufio"
	"io"
	"strings"
	"time"

	logpkg "github.com/haukened/rr-dns/internal/dns/common/log"
	"github.com/haukened/rr-dns/internal/dns/domain"
)

// ParsePlainList parses a simple newline-delimited list of domains into BlockRule values.
// Default is exact; leading "*." or "." indicates suffix (apex-inclusive).
//
// Behavior:
// - Supports comments starting with '#' (inline or whole-line)
// - Trims surrounding whitespace and removes trailing dots via CanonicalDNSName
// - Skips empty lines after trimming/stripping comments
// - De-duplicates by canonical name while preserving first-seen order
// - Each rule is attributed to the provided source and timestamped with now
func ParsePlainList(r io.Reader, source string, logger logpkg.Logger, now time.Time) ([]domain.BlockRule, error) {
	scanner := bufio.NewScanner(r)
	// Default scanner buffer should suffice for typical lines; adjust if needed later.

	// seen key must include kind to allow both exact and suffix for same name
	seen := make(map[string]struct{})
	out := make([]domain.BlockRule, 0, 256)
	logger.Debug(map[string]any{"source": source}, "parse_plain_list_start")
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		// Remove potential BOM at start of first token
		line = strings.TrimPrefix(line, "\uFEFF")

		// Detect empty or full-line comment before stripping inline comments
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			logger.Debug(map[string]any{"line": lineNum}, "skip_empty")
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			logger.Debug(map[string]any{"line": lineNum}, "skip_comment")
			continue
		}

		// Strip inline comments
		if idx := strings.IndexByte(line, '#'); idx >= 0 {
			line = line[:idx]
		}

		// Trim and canonicalize base string
		s := strings.TrimSpace(line)
		// Remove potential BOM at start of first token
		s = strings.TrimPrefix(s, "\uFEFF")
		// Determine kind by marker and strip marker if suffix.
		kind := ruleKindFromRaw(s)

		name := normalizeDomainName(s)

		if !isValidFQDN(name) {
			// skip obviously invalid tokens (e.g., "\\t\\n")
			// skip email addresses and such
			logger.Debug(map[string]any{"line": lineNum, "raw": s, "name": name}, "skip_invalid_fqdn")
			continue
		}

		// seen key combines name and kind to allow both for same domain
		seenKey := name + "|" + kind.String()
		if _, ok := seen[seenKey]; ok {
			logger.Debug(map[string]any{"line": lineNum, "name": name, "kind": kind.String()}, "skip_duplicate")
			continue
		}

		var (
			rule domain.BlockRule
			err  error
		)
		if kind == domain.BlockRuleSuffix {
			rule, err = domain.NewSuffixBlockRule(name, source, now)
		} else {
			rule, err = domain.NewExactBlockRule(name, source, now)
		}
		if err != nil {
			// Skip invalid entries rather than failing the entire parse.
			// Source or time should be valid from caller; name can still fail if empty.
			logger.Debug(map[string]any{"line": lineNum, "name": name, "kind": kind.String(), "error": err.Error()}, "skip_constructor_error")
			continue
		}
		out = append(out, rule)
		seen[seenKey] = struct{}{}
		logger.Debug(map[string]any{"line": lineNum, "name": rule.Name, "kind": rule.Kind.String()}, "emit_rule")
	}

	if err := scanner.Err(); err != nil {
		logger.Debug(map[string]any{"source": source, "error": err.Error()}, "parse_plain_list_scan_error")
		return nil, err
	}
	logger.Debug(map[string]any{"source": source, "count": len(out)}, "parse_plain_list_done")
	return out, nil
}
