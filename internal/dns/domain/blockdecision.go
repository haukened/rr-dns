package domain

// BlockDecision represents the outcome of evaluating a domain against the blocklist.
// Pure value type, no external dependencies.
type BlockDecision struct {
	Blocked     bool   // true if the name is blocked by any rule
	MatchedRule string // rule name that matched (canonical root for suffix, exact domain for exact)
	Source      string // optional: source identifier of the matched rule
	Kind        BlockRuleKind
}

// IsBlocked is a convenience accessor.
func (d BlockDecision) IsBlocked() bool { return d.Blocked }

// EmptyDecision returns a not-blocked decision.
func EmptyDecision() BlockDecision { return BlockDecision{Blocked: false} }
