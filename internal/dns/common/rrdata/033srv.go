package rrdata

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"github.com/haukened/rr-dns/internal/dns/common/utils"
)

// encodeSRVData encodes an SRV record string into its binary representation.
func encodeSRVData(data string) ([]byte, error) {
	// data = "priority weight port target"
	parts := strings.Fields(data)
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid SRV record format (expected 4 fields): %s", data)
	}

	// priority, weight, and port must be valid unsigned integers
	buf := make([]byte, 6)
	for i := 0; i < 3; i++ {
		val, err := strconv.ParseUint(parts[i], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid SRV field %d: %v", i, err)
		}
		binary.BigEndian.PutUint16(buf[i*2:], uint16(val))
	}

	// target is the domain name of the service
	target, err := encodeDomainName(parts[3])
	if err != nil {
		return nil, fmt.Errorf("invalid SRV target: %v", err)
	}

	// Append the target domain name to the buffer
	return append(buf, target...), nil
}

// decodeSRVData decodes the binary representation of an SRV record into its string format.
func decodeSRVData(data []byte) (string, error) {
	if len(data) < 6 {
		return "", fmt.Errorf("invalid SRV record length: %d", len(data))
	}

	// Read priority, weight, and port
	priority := binary.BigEndian.Uint16(data[0:2])
	weight := binary.BigEndian.Uint16(data[2:4])
	port := binary.BigEndian.Uint16(data[4:6])

	// Read the target domain name
	target, err := decodeDomainName(data[6:])
	if err != nil {
		return "", fmt.Errorf("invalid SRV target: %v", err)
	}

	// re-canonicalize the DNS names.
	decoded := fmt.Sprintf("%d %d %d %s", priority, weight, port, target)
	decoded = utils.CanonicalDNSName(decoded)
	// Format the SRV record string
	return decoded, nil
}
