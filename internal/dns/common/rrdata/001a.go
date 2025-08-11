package rrdata

import (
	"fmt"
	"net"
)

// EncodeAData encodes an A record string into its binary representation.
func EncodeAData(data string) ([]byte, error) {
	// data = "192.168.0.1"
	ip := net.ParseIP(data)
	if ip == nil || !isIPv4(ip) {
		return nil, fmt.Errorf("invalid A record IP: %s", data)
	}
	return ip.To4(), nil
}
