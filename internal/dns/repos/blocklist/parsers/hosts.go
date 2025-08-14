package parsers

import (
	"bufio"
	"io"
	"strings"
	"time"

	logpkg "github.com/haukened/rr-dns/internal/dns/common/log"
	"github.com/haukened/rr-dns/internal/dns/common/utils"
	"github.com/haukened/rr-dns/internal/dns/domain"
)

// ParseHostsFile parses /etc/hosts-style files and returns exact BlockRules for valid hostnames.
//
// Rules:
// - Ignore the IP field; extract one or more hostnames following it
// - Skip comments (whole-line or inline after '#') and blank lines
// - Skip invalid tokens (wildcards like "*." or any '*' present, or names starting with '.')
// - Normalize via CanonicalDNSName; validate with isValidFQDN; require exact match kind only
// - De-duplicate by canonical name (exact only), preserving first-seen order
func ParseHostsFile(r io.Reader, source string, logger logpkg.Logger, now time.Time) ([]domain.BlockRule, error) {
	scanner := bufio.NewScanner(r)

	seen := make(map[string]struct{})
	out := make([]domain.BlockRule, 0, 256)

	logger.Debug(map[string]any{"source": source}, "parse_hosts_start")

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := stripLineBOM(scanner.Text())

		// Check for empties or full-line comments before stripping inline comments
		if isEmpty, isComment := classifyLine(line); isEmpty || isComment {
			if isEmpty {
				logger.Debug(map[string]any{"line": lineNum}, "hosts_skip_empty")
			} else {
				logger.Debug(map[string]any{"line": lineNum}, "hosts_skip_comment")
			}
			continue
		}

		// Remove inline comments
		line = stripInlineComment(line)

		fields := strings.Fields(line)
		if len(fields) < 2 {
			// No hostnames present after IP
			logger.Debug(map[string]any{"line": lineNum}, "hosts_no_hostnames")
			continue
		}

		// fields[0] is IP (ignored)
		for _, raw := range fields[1:] {
			// Fast reject invalid hostfile tokens
			// no domains, no wildcards, per standard host file syntax
			if raw == "" || strings.HasPrefix(raw, ".") || strings.Contains(raw, "*") {
				logger.Debug(map[string]any{"line": lineNum, "raw": raw}, "hosts_skip_invalid_token")
				continue
			}

			name := utils.CanonicalDNSName(raw)

			if !isValidFQDN(name) {
				logger.Debug(map[string]any{"line": lineNum, "name": name}, "hosts_skip_invalid_fqdn")
				continue
			}

			if _, ok := seen[name]; ok {
				logger.Debug(map[string]any{"line": lineNum, "name": name}, "hosts_skip_duplicate")
				continue
			}

			rule, err := domain.NewExactBlockRule(name, source, now)
			if err != nil {
				logger.Debug(map[string]any{"line": lineNum, "name": name, "error": err.Error()}, "hosts_skip_constructor_error")
				continue
			}

			out = append(out, rule)
			seen[name] = struct{}{}
			logger.Debug(map[string]any{"line": lineNum, "name": rule.Name}, "hosts_emit_rule")
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Debug(map[string]any{"source": source, "error": err.Error()}, "parse_hosts_scan_error")
		return nil, err
	}

	logger.Debug(map[string]any{"source": source, "count": len(out)}, "parse_hosts_done")
	return out, nil
}
