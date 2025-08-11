package rrdata

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

// EncodeMXData encodes an MX record string into its binary representation.
func EncodeMXData(data string) ([]byte, error) {
	// data = 10 mail.example.com
	parts := strings.Fields(data)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid MX record format (expected: preference domain): %s", data)
	}
	// pref is a uint16 representing the preference of the mail server
	// It must be between 0 and 65535.
	pref, err := strconv.Atoi(parts[0])
	if err != nil || pref < 0 || pref > 65535 {
		return nil, fmt.Errorf("invalid MX preference: %s", parts[0])
	}
	prefBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(prefBytes, uint16(pref))
	// Encode the domain name of the mail server
	encodedDomain, err := EncodeDomainName(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid MX exchange domain: %s", parts[1])
	}
	return append(prefBytes, encodedDomain...), nil
}
