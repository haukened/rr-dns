package rrdata

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

// encodeMXData encodes an MX record string into its binary representation.
func encodeMXData(data string) ([]byte, error) {
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
	encodedDomain, err := encodeDomainName(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid MX exchange domain: %s", parts[1])
	}
	return append(prefBytes, encodedDomain...), nil
}

// decodeMXData decodes MX (Mail Exchange) record data from the given byte slice.
func decodeMXData(b []byte) (string, error) {
	if len(b) < 2 {
		return "", fmt.Errorf("invalid MX data length")
	}
	pref := binary.BigEndian.Uint16(b[:2])
	// Decode the domain name
	domain, err := decodeDomainName(b[2:])
	if err != nil {
		return "", fmt.Errorf("invalid MX exchange domain: %v", err)
	}
	return fmt.Sprintf("%d %s", pref, domain), nil
}
