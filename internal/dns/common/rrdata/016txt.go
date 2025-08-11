package rrdata

import (
	"fmt"
	"strings"
)

// encodeTXTData encodes a TXT record string into its binary representation.
func encodeTXTData(data string) ([]byte, error) {
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

// decodeTXTData decodes a TXT record from its binary representation.
func decodeTXTData(b []byte) (string, error) {
	var segments []string
	for i := 0; i < len(b); {
		length := int(b[i])
		i++
		if length == 0 {
			break
		}
		if i+length > len(b) {
			return "", fmt.Errorf("invalid TXT record: segment length exceeds remaining data")
		}
		segments = append(segments, string(b[i:i+length]))
		i += length
	}
	return strings.Join(segments, "; "), nil
}
