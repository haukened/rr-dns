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
	transport     ServerTransport
	upstream      UpstreamClient
	upstreamCache Cache
	zoneCache     ZoneCache
}

type ResolverOptions struct {
	Blocklist     Blocklist
	Clock         clock.Clock
	Logger        log.Logger
	Transport     ServerTransport
	Upstream      UpstreamClient
	UpstreamCache Cache
	ZoneCache     ZoneCache
}

func NewResolver(opts ResolverOptions) *Resolver {
	return &Resolver{
		blocklist:     opts.Blocklist,
		clock:         opts.Clock,
		logger:        opts.Logger,
		transport:     opts.Transport,
		upstream:      opts.Upstream,
		upstreamCache: opts.UpstreamCache,
		zoneCache:     opts.ZoneCache,
	}
}

func (r *Resolver) HandleQuery(ctx context.Context, query domain.DNSQuery, clientAddr net.Addr) (domain.DNSResponse, error) {
	// 1. Check authoritative zone cache first
	records, found := r.checkZoneCache(query)
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
	response, err := r.resolveUpstream(query, r.clock.Now())
	if err != nil {
		r.logger.Error(map[string]any{
			"error":     err,
			"query":     query,
			"client":    clientAddr,
			"timestamp": r.clock.Now(),
		}, "Failed to resolve upstream")
		return buildResponse(query, domain.SERVFAIL, nil), nil
	}

	// 5. Store response in upstream cache
	if err := r.cacheUpstreamResponse(response); err != nil {
		r.logger.Error(map[string]any{
			"error":     err,
			"query":     query,
			"client":    clientAddr,
			"timestamp": r.clock.Now(),
		}, "Failed to cache upstream response")
		// Don't return error here - we have a valid response, just couldn't cache it
	}

	// 6. Return response to client
	return response, nil
}

func (r *Resolver) checkZoneCache(query domain.DNSQuery) ([]domain.ResourceRecord, bool) {
	if r.zoneCache == nil {
		return nil, false
	}
	return r.zoneCache.FindRecords(query)
}

func (r *Resolver) checkBlocklist(query domain.DNSQuery) bool {
	if r.blocklist == nil {
		return false
	}
	return r.blocklist.IsBlocked(query)
}

func (r *Resolver) checkUpstreamCache(query domain.DNSQuery) ([]domain.ResourceRecord, bool) {
	if r.upstreamCache == nil {
		return nil, false
	}
	return r.upstreamCache.Get(query.CacheKey())
}

func (r *Resolver) resolveUpstream(query domain.DNSQuery, now time.Time) (domain.DNSResponse, error) {
	if r.upstream == nil {
		return domain.DNSResponse{}, fmt.Errorf("no upstream client configured")
	}
	return r.upstream.Resolve(context.Background(), query, now)
}

func (r *Resolver) cacheUpstreamResponse(response domain.DNSResponse) error {
	if r.upstreamCache == nil {
		return nil // No cache configured, not an error
	}
	return r.upstreamCache.Set(response.Answers)
}

// buildResponse creates a DNS response with the specified RCode and optional records
func buildResponse(query domain.DNSQuery, rcode domain.RCode, records []domain.ResourceRecord) domain.DNSResponse {
	return domain.DNSResponse{
		ID:      query.ID,
		RCode:   rcode,
		Answers: records,
		// TODO: Set additional response fields as needed (Authority, Additional sections)
	}
}
