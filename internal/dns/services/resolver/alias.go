// Package resolver contains the core DNS resolution orchestration including
// alias (CNAME) chaining logic. The alias resolution helpers in this file are
// intentionally factored for readability, testability, and potential future
// extraction into their own package without changing behavior.
package resolver

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/haukened/rr-dns/internal/dns/common/clock"
	"github.com/haukened/rr-dns/internal/dns/common/log"
	"github.com/haukened/rr-dns/internal/dns/domain"
)

// aliasErrors provides sentinel errors for policy decisions (Issue #11 will
// map these to RCODE behaviors if required).
var (
	// ErrAliasDepthExceeded is returned when the number of CNAME indirections
	// encountered during a chase exceeds the configured maximum depth.
	ErrAliasDepthExceeded = errors.New("alias resolution max depth exceeded")
	// ErrAliasLoopDetected is returned when a loop is detected (a previously
	// visited owner name reappears in the CNAME chain).
	ErrAliasLoopDetected = errors.New("alias loop detected")
	// ErrAliasTargetInvalid indicates the CNAME target was missing / invalid.
	ErrAliasTargetInvalid = errors.New("alias target invalid")
	// ErrAliasQuestionBuild indicates constructing the synthetic follow-up question failed.
	ErrAliasQuestionBuild = errors.New("alias question build failed")
)

// aliasChaser is the concrete implementation of AliasResolver. It is designed
// to be easily extractable into its own package later: all dependencies are
// injected via narrow interfaces already defined in resolver.
// aliasChaser implements AliasResolver. It performs CNAME (alias) expansion
// across authoritative data first, then falls back to upstream lookups. All
// collaborators are injected so that the component adheres to dependency
// inversion and can be cleanly unit tested.
type aliasChaser struct {
	// zone provides authoritative value-based record lookup.
	zone ZoneCache
	// up performs upstream DNS resolution when authoritative data is absent.
	up UpstreamClient
	// cache reserved for future optimizations (not currently used in Chase).
	cache Cache
	// clock supplies current time for upstream calls (and future TTL logic).
	clock clock.Clock
	// logger records structured diagnostic and policy events.
	logger log.Logger
	// maxDepth sets an upper bound on the number of CNAME hops.
	maxDepth int
}

// NewAliasChaser constructs an AliasResolver with the required dependencies.
// cache may be nil (only affects potential future optimization of upstream lookups).
// NewAliasChaser constructs an AliasResolver implementation bound to the
// supplied collaborators. A maxDepth <= 0 disables depth limiting (not
// recommended in production). The returned value satisfies AliasResolver.
func NewAliasChaser(zone ZoneCache, upstream UpstreamClient, cache Cache, clk clock.Clock, logger log.Logger, maxDepth int) AliasResolver {
	return &aliasChaser{zone: zone, up: upstream, cache: cache, clock: clk, logger: logger, maxDepth: maxDepth}
}

// NewNoOpAliasResolver retained for compatibility / testing fallback.
// NewNoOpAliasResolver returns an AliasResolver that performs no chasing and
// simply echoes the provided records. Useful for tests or disabled feature flags.
func NewNoOpAliasResolver() AliasResolver { return &noOpAliasResolver{} }

// noOpAliasResolver implements a no-op strategy (used only if explicitly injected).
// noOpAliasResolver is a trivial AliasResolver used when alias chasing is
// intentionally disabled.
type noOpAliasResolver struct{}

// Chase for the no-op implementation returns the input unmodified.
func (n *noOpAliasResolver) Chase(query domain.Question, initial []domain.ResourceRecord) ([]domain.ResourceRecord, error) {
	return initial, nil
}

// Chase implements recursive CNAME expansion per RFC 1034 §3.6.2.
// Algorithm (authoritative-first, then upstream if necessary):
//  1. Start with initial records (first must be a CNAME) already fetched from zone cache.
//  2. While current head is CNAME and depth < maxDepth and not loop:
//     a. Append the CNAME to answer chain (maintained in 'chain').
//     b. Extract target from the CNAME's RDATA (already stored in Text field or decode Data fallback).
//     c. Form a synthetic Question for the target using original qtype (unless qtype == CNAME).
//     d. Attempt authoritative lookup for target & qtype.
//     - If empty authoritative answer, check if another CNAME exists; if so continue.
//     e. If authoritative miss, try upstream resolution (when upstream client available).
//     f. If terminal RRset (non-CNAME answers for original qtype) found, append and stop.
//  3. On depth exceed or loop detection, return current chain + error (caller may decide SERVFAIL policy later).
//  4. If we never encountered a CNAME (should not happen based on precondition), return initial unchanged.
//
// Notes:
// - We rely on ResourceRecord.Text for the CNAME target to avoid decoding overhead. If Text empty, we fallback.
// - We maintain a visited set of canonical (lowercase) names to detect loops.
// Chase expands a CNAME chain beginning with the provided initial records.
// It returns the ordered list of CNAME hops followed (if any) plus the
// terminal RRset that answers the original query type. If neither a loop nor
// a depth violation occur, err will be nil. The bool indicates whether any
// alias chasing was actually performed.

func (a *aliasChaser) Chase(query domain.Question, initial []domain.ResourceRecord) ([]domain.ResourceRecord, error) {
	// Fast path: nothing to chase (either empty, not CNAME head, or query type explicitly CNAME)
	if !a.shouldChase(query, initial) {
		return initial, nil
	}
	st := newChaseState(query, initial)
	// Main expansion loop: iterate until terminal RRset, missing data, loop, or depth exceed.
	for {
		head, _ := st.currentHead()
		// PROCESS: CNAME hop encountered (loop only entered when initial head is a CNAME)
		st.chased = true
		if err := a.guardDepth(&st, head); err != nil {
			// TERMINATION (error): depth exceeded policy
			st.chain = append(st.chain, head)
			return st.chain, err
		}
		if err := a.guardLoop(&st, head); err != nil {
			// TERMINATION (error): loop detected
			st.chain = append(st.chain, head)
			return st.chain, err
		}
		st.chain = append(st.chain, head)
		target, err := a.extractTarget(head)
		if err != nil {
			// TERMINATION (error): invalid / missing target
			return st.chain, err
		}
		nextQ, err := a.buildNextQuestion(&st, target)
		if err != nil {
			// TERMINATION (error): question synthesis failure
			return st.chain, err
		}
		nextRecords, found := a.authoritativeLookup(nextQ)
		if !found || len(nextRecords) == 0 {
			// Attempt authoritative CNAME lookup for target when original type not found; enables multi-hop chains
			if st.originalType != domain.RRTypeCNAME { // avoid redundant CNAME query if original was CNAME (we don't chase in that case anyway)
				cnameQ, qErr := domain.NewQuestion(st.query.ID, target, domain.RRTypeCNAME, st.query.Class)
				if qErr == nil { // safe guard; should not fail
					cnameRecords, cnameFound := a.authoritativeLookup(cnameQ)
					if cnameFound && len(cnameRecords) > 0 {
						nextRecords, found = cnameRecords, true
					}
				}
			}
			if !found || len(nextRecords) == 0 { // still nothing authoritative; upstream fallback
				nextRecords, found = a.upstreamLookup(nextQ, target)
			}
		}
		if !found || len(nextRecords) == 0 {
			// TERMINATION: no further data (RFC: return gathered CNAME chain only)
			break // return accumulated chain; terminal missing data case
		}
		st.current = nextRecords
		if !a.isHeadCNAME(st.current) { // terminal RRset
			// TERMINATION: resolved final RRset (non-CNAME)
			st.chain = append(st.chain, st.current...)
			break
		}
		// LOOP CONTINUE: nextRecords begins with another CNAME; iterate again
	}
	return st.chain, nil
}

// chaseState encapsulates mutable state during alias chasing.
// chaseState captures mutable progress during a single Chase invocation.
// It is deliberately unexported and kept lightweight for stack allocation.
type chaseState struct {
	// query is the original client question (ID reused for follow-up lookups).
	query domain.Question
	// originalType is the RRType requested by the client (CNAME chase targets this).
	originalType domain.RRType
	// chain accumulates ordered CNAME hops and (eventually) the terminal RRset.
	chain []domain.ResourceRecord
	// visited holds lowercase owner names encountered to detect loops.
	visited map[string]struct{}
	// depth counts the number of CNAME hops processed.
	depth int
	// current is the working set of records for the active owner name.
	current []domain.ResourceRecord
	// chased indicates at least one CNAME hop was processed.
	chased bool
}

// newChaseState initializes chase state with capacity sized for small chains.
func newChaseState(q domain.Question, initial []domain.ResourceRecord) chaseState {
	return chaseState{
		query:        q,
		originalType: q.Type,
		chain:        make([]domain.ResourceRecord, 0, len(initial)+4),
		visited:      map[string]struct{}{},
		current:      initial,
	}
}

// currentHead returns the current first record and whether it is a CNAME.
func (st *chaseState) currentHead() (domain.ResourceRecord, bool) {
	if len(st.current) == 0 {
		return domain.ResourceRecord{}, false
	}
	h := st.current[0]
	if h.Type != domain.RRTypeCNAME {
		return h, false
	}
	return h, true
}

// appendTerminal adds the entire current RRset to the chain when terminal.
// appendTerminal removed (was unused after loop refactor eliminating unreachable branch)

// Helper methods on aliasChaser
// shouldChase determines if alias chasing applies to the initial record set.
func (a *aliasChaser) shouldChase(q domain.Question, initial []domain.ResourceRecord) bool {
	return len(initial) > 0 && initial[0].Type == domain.RRTypeCNAME && q.Type != domain.RRTypeCNAME
}

// isHeadCNAME reports whether the first record in the slice is a CNAME.
func (a *aliasChaser) isHeadCNAME(rrs []domain.ResourceRecord) bool {
	return len(rrs) > 0 && rrs[0].Type == domain.RRTypeCNAME
}

// guardDepth increments and validates chain depth against the configured max.
func (a *aliasChaser) guardDepth(st *chaseState, head domain.ResourceRecord) error {
	st.depth++
	if a.maxDepth > 0 && st.depth > a.maxDepth {
		a.logger.Warn(map[string]any{
			"query":           st.query,
			"alias_name":      head.Name,
			"alias_depth":     st.depth,
			"alias_chain_len": len(st.chain) + 1, // +1 for current head about to be appended by caller
		}, "Alias depth exceeded")
		return ErrAliasDepthExceeded
	}
	return nil
}

// guardLoop records the owner name and detects if a loop is present.
func (a *aliasChaser) guardLoop(st *chaseState, head domain.ResourceRecord) error {
	name := strings.ToLower(head.Name)
	if _, ok := st.visited[name]; ok {
		a.logger.Warn(map[string]any{
			"query":           st.query,
			"alias_name":      head.Name,
			"alias_depth":     st.depth,
			"alias_chain_len": len(st.chain) + 1,
		}, "Alias loop detected")
		return ErrAliasLoopDetected
	}
	st.visited[name] = struct{}{}
	return nil
}

// extractTarget obtains the CNAME target from the record's Text field,
// enforcing presence. It no longer appends or enforces a trailing dot;
// callers and downstream components must handle any canonicalization.
func (a *aliasChaser) extractTarget(head domain.ResourceRecord) (string, error) {
	target := strings.TrimSpace(head.Text)
	if target == "" && len(head.Data) > 0 { // fallback decode not feasible without compression context
		a.logger.Debug(map[string]any{"record": head}, "Missing CNAME Text; unable to decode target from Data")
		return "", fmt.Errorf("%w: text missing for %s", ErrAliasTargetInvalid, head.Name)
	}
	if target == "" { // truly empty
		return "", fmt.Errorf("%w: empty for %s", ErrAliasTargetInvalid, head.Name)
	}
	return target, nil
}

// buildNextQuestion creates a synthetic question for the next alias hop.
func (a *aliasChaser) buildNextQuestion(st *chaseState, target string) (domain.Question, error) {
	q, err := domain.NewQuestion(st.query.ID, target, st.originalType, st.query.Class)
	if err != nil {
		return domain.Question{}, fmt.Errorf("%w: %v", ErrAliasQuestionBuild, err)
	}
	return q, nil
}

// authoritativeLookup resolves the question against the authoritative zone cache.
func (a *aliasChaser) authoritativeLookup(q domain.Question) ([]domain.ResourceRecord, bool) {
	if a.zone == nil {
		return nil, false
	}
	return a.zone.FindRecords(q)
}

// upstreamLookup attempts to resolve the question via the configured upstream.
// It returns (nil,false) on error or empty response.
func (a *aliasChaser) upstreamLookup(q domain.Question, target string) ([]domain.ResourceRecord, bool) {
	if a.up == nil {
		return nil, false
	}
	recs, err := a.up.Resolve(context.Background(), q, a.clock.Now())
	if err != nil || len(recs) == 0 {
		if err != nil {
			a.logger.Debug(map[string]any{"error": err, "target": target}, "Upstream lookup during alias chase failed")
		}
		return nil, false
	}
	return recs, true
}

// Interface assertions
var _ AliasResolver = (*aliasChaser)(nil)
var _ AliasResolver = (*noOpAliasResolver)(nil)
