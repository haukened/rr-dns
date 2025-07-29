package upstream

import (
	"github.com/haukened/rr-dns/internal/dns/repo"
)

// Verify that Resolver implements the UpstreamResolver interface
var _ repo.UpstreamResolver = (*Resolver)(nil)
