package resolver

import (
	"context"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

type UpstreamClient interface {
	Resolve(ctx context.Context, query domain.DNSQuery) (domain.DNSResponse, error)
}
