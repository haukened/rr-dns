package rrdata

import (
	"fmt"
	"strings"
)

// EncodeTXTData encodes a TXT record string into its binary representation.
func EncodeTXTData(data string) ([]byte, error) {
	// Supports multiple strings separated by semicolons for simplicity
	// see RFC 1035 section 3.3.14
	segments := strings.Split(data, ";")
	var encoded []byte
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		if len(segment) > 255 {
			return nil, fmt.Errorf("TXT segment too long: %d bytes", len(segment))
		}
		encoded = append(encoded, byte(len(segment)))
		encoded = append(encoded, []byte(segment)...)
	}
	if len(encoded) == 0 {
		return nil, fmt.Errorf("TXT record must contain at least one segment")
	}
	return encoded, nil
}
