package wire

import (
	"encoding/binary"
	"errors"
)

// decodeName decodes a domain name from a DNS message at the specified offset,
// handling label compression as defined in RFC 1035.

// decodeQuestion parses the DNS question section starting at the given offset.
// It returns the domain name, query type, query class, and the updated offset.
func decodeQuestion(data []byte, offset int) (string, uint16, uint16, int, error) {
	name, newOffset, err := decodeName(data, offset)
	if err != nil {
		return "", 0, 0, 0, err
	}
	if newOffset+4 > len(data) {
		return "", 0, 0, 0, errors.New("truncated question fields")
	}
	qtype := binary.BigEndian.Uint16(data[newOffset : newOffset+2])
	qclass := binary.BigEndian.Uint16(data[newOffset+2 : newOffset+4])
	return name, qtype, qclass, newOffset + 4, nil
}
