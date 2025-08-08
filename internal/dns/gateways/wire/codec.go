package wire

import (
	"time"

	"github.com/haukened/rr-dns/internal/dns/common/log"
	"github.com/haukened/rr-dns/internal/dns/domain"
)

type DNSCodec interface {
	// Upstream Functions
	// These methods are used to encode and decode DNS messages for communication with upstream servers.
	EncodeQuery(query domain.Question) ([]byte, error)
	DecodeResponse(data []byte, expectedID uint16, now time.Time) (domain.DNSResponse, error)

	// Authoritative Functions
	// These methods handle encoding and decoding of authoritative records for zone file management.
	DecodeQuery(data []byte) (domain.Question, error)
	EncodeResponse(resp domain.DNSResponse, logger log.Logger) ([]byte, error)
}
