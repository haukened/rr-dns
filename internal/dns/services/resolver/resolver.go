package resolver

import (
	"context"
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
}

type ResolverOptions struct {
	Blocklist     Blocklist
	Clock         clock.Clock
	Logger        log.Logger
	Upstream      UpstreamClient
	UpstreamCache Cache
	ZoneCache     ZoneCache
	MaxRecursion  int
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
	}
}

func (r *Resolver) HandleQuery(ctx context.Context, query domain.Question, clientAddr net.Addr) (domain.DNSResponse, error) {
	// 1. Check authoritative zone cache first
	records, found := r.resolveFromZone(query)
	if found {
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
	records, err := r.resolveUpstream(ctx, query, r.clock.Now())
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

func (r *Resolver) resolveFromZone(query domain.Question) ([]domain.ResourceRecord, bool) {
	if r.zoneCache == nil {
		return nil, false
	}
	// Lookup exact match in the zone cache
	records, found := r.zoneCache.FindRecords(query)
	if !found || len(records) == 0 {
		return nil, false
	}
	// TODO: If the first record is a CNAME and the query type isn't CNAME,
	// implement in-zone CNAME chasing up to r.maxRecursion.
	return records, true
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
