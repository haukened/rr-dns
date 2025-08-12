package rrdata

import (
	"fmt"
	"net"
)

// encodeAData encodes an A record string into its binary representation.
func encodeAData(data string) ([]byte, error) {
	// data = "192.168.0.1"
	ip := net.ParseIP(data)
	if ip == nil || !isIPv4(ip) {
		return nil, fmt.Errorf("invalid A record IP: %s", data)
	}
	return ip.To4(), nil
}

// decodeAData decodes a byte slice representing an IPv4 address from a DNS A record.
// It returns the string representation of the IP address if the input is valid,
// or an error if the input length is incorrect or the IP address is invalid.
func decodeAData(b []byte) (string, error) {
	ip := net.IP(b)
	if ip == nil || !isIPv4(ip) {
		return "", fmt.Errorf("invalid A record IP: %v", b)
	}
	return ip.To4().String(), nil
}
