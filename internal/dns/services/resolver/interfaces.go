package resolver

import (
	"context"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

type DNSCodec interface {
	// Upstream Functions
	// These methods are used to encode and decode DNS messages for communication with upstream servers.
	EncodeQuery(query domain.DNSQuery) ([]byte, error)
	DecodeResponse(data []byte, expectedID uint16) (domain.DNSResponse, error)

	// Authoritative Functions
	// These methods handle encoding and decoding of authoritative records for zone file management.
	DecodeQuery(data []byte) (domain.DNSQuery, error)
	EncodeResponse(resp domain.DNSResponse) ([]byte, error)
}

type UpstreamClient interface {
	Resolve(ctx context.Context, query domain.DNSQuery) (domain.DNSResponse, error)
}
