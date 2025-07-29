package domain

type DNSCodec interface {
	// Upstream Functions
	// These methods are used to encode and decode DNS messages for communication with upstream servers.
	EncodeQuery(query DNSQuery) ([]byte, error)
	DecodeResponse(data []byte, expectedID uint16) (DNSResponse, error)

	// Authoritative Functions
	// These methods handle encoding and decoding of authoritative records for zone file management.
	DecodeQuery(data []byte) (DNSQuery, error)
	EncodeResponse(resp DNSResponse) ([]byte, error)
}
