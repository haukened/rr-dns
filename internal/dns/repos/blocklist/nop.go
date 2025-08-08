package blocklist

import (
	"github.com/haukened/rr-dns/internal/dns/domain"
	"github.com/haukened/rr-dns/internal/dns/services/resolver"
)

type NoopBlocklist struct{}

func (n *NoopBlocklist) IsBlocked(q domain.Question) bool {
	// Noop implementation, always returns false
	return false
}

var _ resolver.Blocklist = (*NoopBlocklist)(nil)
