package resolver

import "github.com/haukened/rr-dns/internal/dns/common/log"

type Resolver struct {
	blocklist     Blocklist
	logger        log.Logger
	transport     ServerTransport
	upstream      UpstreamClient
	upstreamCache Cache
	zoneCache     ZoneCache
}

type ResolverOptions struct {
	Blocklist     Blocklist
	Logger        log.Logger
	Transport     ServerTransport
	Upstream      UpstreamClient
	UpstreamCache Cache
	ZoneCache     ZoneCache
}

func NewResolver(opts ResolverOptions) *Resolver {
	return &Resolver{
		blocklist:     opts.Blocklist,
		logger:        opts.Logger,
		transport:     opts.Transport,
		upstream:      opts.Upstream,
		upstreamCache: opts.UpstreamCache,
		zoneCache:     opts.ZoneCache,
	}
}
