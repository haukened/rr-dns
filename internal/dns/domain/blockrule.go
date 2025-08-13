package domain

import (
	"fmt"
	"strings"
	"time"
)

// BlockRuleKind defines how a rule matches domains.
//
// exact  - matches the apex only (name == queried domain)
// suffix - matches the apex and any subdomain (apex-inclusive suffix)
type BlockRuleKind uint8

const (
	// BlockRuleExact matches only the exact domain.
	BlockRuleExact BlockRuleKind = iota
	// BlockRuleSuffix matches the domain and all its subdomains (apex-inclusive).
	BlockRuleSuffix
)

// String returns a stable string representation of the rule kind.
func (k BlockRuleKind) String() string {
	switch k {
	case BlockRuleExact:
		return "exact"
	case BlockRuleSuffix:
		return "suffix"
	default:
		return fmt.Sprintf("BlockRuleKind(%d)", k)
	}
}

// ParseBlockRuleKind converts a string into a BlockRuleKind.
// Accepts: "exact", "suffix" (case-insensitive).
func ParseBlockRuleKind(s string) (BlockRuleKind, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "exact":
		return BlockRuleExact, nil
	case "suffix":
		return BlockRuleSuffix, nil
	default:
		return 0, fmt.Errorf("unsupported BlockRuleKind: %q", s)
	}
}

// BlockRule represents a single blocking rule sourced from a file or feed.
//
// Notes:
// - Name is expected to be canonical and without a trailing dot (normalization handled elsewhere).
// - Source should identify where the rule came from (file path or feed URL/alias).
// - AddedAt records when the rule was ingested.
type BlockRule struct {
	Name    string        // canonical domain (no trailing dot), e.g., "example.com"
	Kind    BlockRuleKind // exact or suffix (apex-inclusive)
	Source  string        // feed/file identifier
	AddedAt time.Time     // ingestion timestamp
}

// NewBlockRule constructs a BlockRule and validates its fields.
func NewBlockRule(name string, kind BlockRuleKind, source string, addedAt time.Time) (BlockRule, error) {
	r := BlockRule{
		Name:    strings.TrimSpace(name),
		Kind:    kind,
		Source:  strings.TrimSpace(source),
		AddedAt: addedAt,
	}
	if err := r.Validate(); err != nil {
		return BlockRule{}, err
	}
	return r, nil
}

// NewExactBlockRule convenience constructor for an exact rule.
func NewExactBlockRule(name, source string, addedAt time.Time) (BlockRule, error) {
	return NewBlockRule(name, BlockRuleExact, source, addedAt)
}

// NewSuffixBlockRule convenience constructor for a suffix rule (apex-inclusive).
func NewSuffixBlockRule(name, source string, addedAt time.Time) (BlockRule, error) {
	return NewBlockRule(name, BlockRuleSuffix, source, addedAt)
}

// Validate checks the BlockRule for required fields and supported values.
func (r BlockRule) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("rule name must not be empty")
	}
	if r.Source == "" {
		return fmt.Errorf("rule source must not be empty")
	}
	if r.AddedAt.IsZero() {
		return fmt.Errorf("rule addedAt must be set")
	}
	switch r.Kind {
	case BlockRuleExact, BlockRuleSuffix:
		// ok
	default:
		return fmt.Errorf("unsupported BlockRuleKind: %d", r.Kind)
	}
	return nil
}

// IsExact returns true when the rule kind is exact.
func (r BlockRule) IsExact() bool { return r.Kind == BlockRuleExact }

// IsSuffix returns true when the rule kind is suffix (apex-inclusive).
func (r BlockRule) IsSuffix() bool { return r.Kind == BlockRuleSuffix }
