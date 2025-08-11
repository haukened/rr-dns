package rrdata

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

// EncodeSRVData encodes an SRV record string into its binary representation.
func EncodeSRVData(data string) ([]byte, error) {
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
	target, err := EncodeDomainName(parts[3])
	if err != nil {
		return nil, fmt.Errorf("invalid SRV target: %v", err)
	}

	// Append the target domain name to the buffer
	return append(buf, target...), nil
}
