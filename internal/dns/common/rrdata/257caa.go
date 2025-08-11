package rrdata

import (
	"fmt"
	"strconv"
	"strings"
)

// EncodeCAAData encodes a CAA record string into its binary representation.
func EncodeCAAData(data string) ([]byte, error) {
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
