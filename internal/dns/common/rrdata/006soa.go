package rrdata

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

// EncodeSOAData encodes an SOA record string into its binary representation.
func EncodeSOAData(data string) ([]byte, error) {
	// data = "mname rname serial refresh retry expire minimum"
	parts := strings.Fields(data)
	if len(parts) != 7 {
		return nil, fmt.Errorf("invalid SOA record format (expected 7 fields): %s", data)
	}

	// mname is the primary name server for the zone
	mname, err := EncodeDomainName(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid SOA mname: %v", err)
	}

	// rname is the email address of the zone administrator, with '.' replaced by '@'
	// e.g. "hostmaster.example.com" becomes "hostmaster@example.com"
	rname, err := EncodeDomainName(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid SOA rname: %v", err)
	}

	// The next five fields are unsigned integers
	// serial, refresh, retry, expire, minimum
	u32 := make([]byte, 20)
	for i := 0; i < 5; i++ {
		val, err := strconv.ParseUint(parts[i+2], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid SOA field %d: %v", i+2, err)
		}
		binary.BigEndian.PutUint32(u32[i*4:], uint32(val))
	}

	// Combine all parts into a single byte slice
	var encoded []byte
	encoded = append(encoded, mname...)
	encoded = append(encoded, rname...)
	encoded = append(encoded, u32...)

	return encoded, nil
}
