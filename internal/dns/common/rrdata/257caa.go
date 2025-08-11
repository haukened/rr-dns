package rrdata

import (
	"fmt"
	"strconv"
	"strings"
)

// encodeCAAData encodes a CAA record string into its binary representation.
func encodeCAAData(data string) ([]byte, error) {
	// data = "0 issue \"letsencrypt.org\""
	parts := strings.Fields(data)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid CAA record format (expected: flag tag \"value\"): %s", data)
	}

	// Parse flag
	flag, err := strconv.ParseUint(parts[0], 10, 8)
	if err != nil {
		return nil, fmt.Errorf("invalid CAA flag: %v", err)
	}

	// Tag is the second field
	tag := parts[1]
	if len(tag) > 255 {
		return nil, fmt.Errorf("CAA tag too long")
	}

	// Value is everything after the tag — join and remove surrounding quotes
	rawValue := strings.Join(parts[2:], " ")
	value := strings.Trim(rawValue, "\"")
	if len(value) > 255 {
		return nil, fmt.Errorf("CAA value too long")
	}

	// Encode: 1 byte flag + 1 byte tag length + tag + value
	encoded := []byte{byte(flag), byte(len(tag))}
	encoded = append(encoded, []byte(tag)...)
	encoded = append(encoded, []byte(value)...)

	return encoded, nil
}

// decodeCAAData decodes the binary representation of a CAA record into its string format.
func decodeCAAData(data []byte) (string, error) {
	if len(data) < 2 {
		return "", fmt.Errorf("invalid CAA record length: %d", len(data))
	}

	// Read flag and tag length
	flag := data[0]
	tagLen := data[1]

	// Read tag
	if len(data) < int(2+tagLen) {
		return "", fmt.Errorf("invalid CAA tag length: %d", tagLen)
	}
	tag := string(data[2 : 2+tagLen])

	// CAA note: Do NOT canonicalize the value portion.
	// The CAA value is opaque: for issue/issuewild it’s a CA domain (often without trailing dot),
	// for iodef (and others) it can be a mailto: or https: URI. Adding a trailing dot or other
	// domain canonicalization would corrupt non-domain values. We only parse flag/tag and pass
	// the value through unchanged (minus surrounding quotes).
	value := string(data[2+tagLen:])

	return fmt.Sprintf("%d %s \"%s\"", flag, tag, value), nil
}
