package rrdata

import (
	"fmt"
	"net"
)

// encodeAAAAData encodes an AAAA record string into its binary representation.
func encodeAAAAData(data string) ([]byte, error) {
	// data = "2001:db8::ff00:42:8329"
	ip := net.ParseIP(data)
	if ip == nil || !isIPv6(ip) {
		return nil, fmt.Errorf("invalid AAAA record IP: %s", data)
	}
	return ip.To16(), nil
}

// decodeAAAAData decodes an AAAA record from its binary representation.
func decodeAAAAData(data []byte) (string, error) {
	if len(data) != net.IPv6len {
		return "", fmt.Errorf("invalid AAAA record length: %d", len(data))
	}
	ip := net.IP(data)
	if ip == nil || !isIPv6(ip) {
		return "", fmt.Errorf("invalid AAAA record IP: %v", data)
	}
	return ip.To16().String(), nil
}
