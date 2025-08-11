package rrdata

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

// encodeSOAData encodes an SOA record string into its binary representation.
func encodeSOAData(data string) ([]byte, error) {
	// data = "mname rname serial refresh retry expire minimum"
	parts := strings.Fields(data)
	if len(parts) != 7 {
		return nil, fmt.Errorf("invalid SOA record format (expected 7 fields): %s", data)
	}

	// mname is the primary name server for the zone
	mname, err := encodeDomainName(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid SOA mname: %v", err)
	}

	// rname is the email address of the zone administrator, with '.' replaced by '@'
	// e.g. "hostmaster.example.com" becomes "hostmaster@example.com"
	rname, err := encodeDomainName(parts[1])
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

// decodeSOAData decodes an SOA record from its binary representation.
func decodeSOAData(b []byte) (string, error) {
	if len(b) < 25 {
		return "", fmt.Errorf("invalid SOA data length: %d", len(b))
	}

	// Decode mname
	mname, err := decodeDomainName(b)
	if err != nil {
		return "", fmt.Errorf("invalid SOA mname: %v", err)
	}
	offset := len(mname) + 2 // +2 for root dot and label length

	// Decode rname
	rname, err := decodeDomainName(b[offset:])
	if err != nil {
		return "", fmt.Errorf("invalid SOA rname: %v", err)
	}
	offset += len(rname) + 2

	// Ensure we have at least 20 bytes for the unsigned integers
	if len(b[offset:]) < 20 {
		return "", fmt.Errorf("SOA record missing integer fields")
	}

	// Extract the five unsigned integers
	var u32 [5]uint32
	for i := 0; i < 5; i++ {
		u32[i] = binary.BigEndian.Uint32(b[offset+i*4 : offset+(i+1)*4])
	}

	return fmt.Sprintf("%s %s %d %d %d %d %d", mname, rname, u32[0], u32[1], u32[2], u32[3], u32[4]), nil
}
