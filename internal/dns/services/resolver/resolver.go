package resolver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/haukened/rr-dns/internal/dns/common/clock"
	"github.com/haukened/rr-dns/internal/dns/common/log"
	"github.com/haukened/rr-dns/internal/dns/domain"
)

type Resolver struct {
	blocklist     Blocklist
	clock         clock.Clock
	logger        log.Logger
	upstream      UpstreamClient
	upstreamCache Cache
	zoneCache     ZoneCache
	maxRecursion  int
	aliasResolver AliasResolver
}

type ResolverOptions struct {
	Blocklist     Blocklist
	Clock         clock.Clock
	Logger        log.Logger
	Upstream      UpstreamClient
	UpstreamCache Cache
	ZoneCache     ZoneCache
	MaxRecursion  int
	AliasResolver AliasResolver
}

func NewResolver(opts ResolverOptions) *Resolver {
	return &Resolver{
		blocklist:     opts.Blocklist,
		clock:         opts.Clock,
		logger:        opts.Logger,
		upstream:      opts.Upstream,
		upstreamCache: opts.UpstreamCache,
		zoneCache:     opts.ZoneCache,
		maxRecursion:  opts.MaxRecursion,
		aliasResolver: opts.AliasResolver,
	}
}

func (r *Resolver) HandleQuery(ctx context.Context, query domain.Question, clientAddr net.Addr) (domain.DNSResponse, error) {
	// 1. Check authoritative zone cache first
	records, found, err := r.resolveFromZone(query)
	if found {
		if err != nil {
			// Classify alias errors; fatal ones become SERVFAIL, non-fatal still return partial chain.
			if r.isFatalAliasError(err) {
				r.logger.Error(map[string]any{"error": err, "query": query}, "Fatal alias resolution error")
				return buildResponse(query, domain.SERVFAIL, nil), nil
			}
			// Non-fatal alias errors (e.g. target invalid, question build) return gathered chain with NOERROR.
			r.logger.Warn(map[string]any{"error": err, "query": query}, "Non-fatal alias resolution error; returning partial chain")
		}
		return buildResponse(query, domain.NOERROR, records), nil
	}

	// 2. Check blocklist and fast fail if blocked
	if r.checkBlocklist(query) {
		r.logger.Info(map[string]any{
			"query":     query,
			"client":    clientAddr,
			"timestamp": r.clock.Now(),
		}, "Query blocked by blocklist")
		return buildResponse(query, domain.NXDOMAIN, nil), nil
	}

	// 3. Check upstream cache for cached responses
	if records, found := r.checkUpstreamCache(query); found {
		return buildResponse(query, domain.NOERROR, records), nil
	}

	// 4. If not found, resolve via upstream client
	// if the ctx is cancelled, this will return an error
	// This allows the resolver to respect cancellation requests from the transport layer.
	// It also allows for timeouts to be applied at the transport level.
	records, err = r.resolveUpstream(ctx, query, r.clock.Now())
	if err != nil {
		r.logger.Error(map[string]any{
			"error":     err,
			"query":     query,
			"client":    clientAddr,
			"timestamp": r.clock.Now(),
		}, "Failed to resolve upstream")
		return buildResponse(query, domain.SERVFAIL, nil), nil
	}

	// 5. Store records in upstream cache
	if err := r.cacheUpstreamResponse(records); err != nil {
		r.logger.Error(map[string]any{
			"error":     err,
			"query":     query,
			"client":    clientAddr,
			"timestamp": r.clock.Now(),
		}, "Failed to cache upstream response")
		// Don't return error here - we have a valid response, just couldn't cache it
	}

	// 6. Return response to client
	return buildResponse(query, domain.NOERROR, records), nil
}

func (r *Resolver) resolveFromZone(query domain.Question) ([]domain.ResourceRecord, bool, error) {
	if r.zoneCache == nil {
		return nil, false, nil
	}
	// Lookup exact match in the zone cache
	records, found := r.zoneCache.FindRecords(query)
	if !found || len(records) == 0 {
		return nil, false, nil
	}
	// Always delegate to aliasResolver (if configured); it contains its own fast path
	// via shouldChase to immediately return the input when chasing is not required.
	var err error
	if r.aliasResolver != nil {
		records, err = r.aliasResolver.Chase(query, records)
	}
	return records, true, err
}

// isFatalAliasError determines if an alias expansion error should trigger SERVFAIL.
// Policy: depth exceeded & loop detected considered fatal (operational / config issues).
// Target / question build errors treated non-fatal (return partial chain for transparency).
func (r *Resolver) isFatalAliasError(err error) bool {
	if err == nil {
		return false
	}
	// Use errors.Is to unwrap wrapped sentinel errors.
	if errors.Is(err, ErrAliasDepthExceeded) || errors.Is(err, ErrAliasLoopDetected) {
		return true
	}
	return false
}

func (r *Resolver) checkBlocklist(query domain.Question) bool {
	if r.blocklist == nil {
		return false
	}
	return r.blocklist.IsBlocked(query)
}

func (r *Resolver) checkUpstreamCache(query domain.Question) ([]domain.ResourceRecord, bool) {
	if r.upstreamCache == nil {
		return nil, false
	}
	return r.upstreamCache.Get(query.CacheKey())
}

func (r *Resolver) resolveUpstream(ctx context.Context, query domain.Question, now time.Time) ([]domain.ResourceRecord, error) {
	if r.upstream == nil {
		return nil, fmt.Errorf("no upstream client configured")
	}
	return r.upstream.Resolve(ctx, query, now)
}

func (r *Resolver) cacheUpstreamResponse(records []domain.ResourceRecord) error {
	if r.upstreamCache == nil {
		return nil // No cache configured, not an error
	}
	return r.upstreamCache.Set(records)
}

// buildResponse creates a DNS response with the specified RCode and optional records
func buildResponse(query domain.Question, rcode domain.RCode, records []domain.ResourceRecord) domain.DNSResponse {
	return domain.DNSResponse{
		ID:       query.ID,
		RCode:    rcode,
		Answers:  records,
		Question: query,
		// TODO: Set additional response fields as needed (Authority, Additional sections)
	}
}

// Ensure Resolver implements DNSResponder at compile time
var _ DNSResponder = (*Resolver)(nil)
