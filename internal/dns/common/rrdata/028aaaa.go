package rrdata

import (
	"fmt"
	"net"
)

// EncodeAAAAData encodes an AAAA record string into its binary representation.
func EncodeAAAAData(data string) ([]byte, error) {
	// data = "2001:db8::ff00:42:8329"
	ip := net.ParseIP(data)
	if ip == nil || !isIPv6(ip) {
		return nil, fmt.Errorf("invalid AAAA record IP: %s", data)
	}
	return ip.To16(), nil
}
