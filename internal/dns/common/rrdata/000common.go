package rrdata

import (
	"fmt"
	"net"
	"strings"

	"github.com/haukened/rr-dns/internal/dns/common/utils"
)

// encodeDomainName encodes a domain name into wire format (length-prefixed labels ending in 0).
// used in multiple record types
func EncodeDomainName(name string) ([]byte, error) {
	// name = foo.example.com.
	name = utils.CanonicalDNSName(name)
	labels := strings.Split(name, ".")
	var encoded []byte
	for _, label := range labels {
		if len(label) == 0 {
			continue
		}
		if len(label) > 63 {
			return nil, fmt.Errorf("label too long: %s", label)
		}
		encoded = append(encoded, byte(len(label)))
		encoded = append(encoded, label...)
	}
	encoded = append(encoded, 0) // null terminator
	return encoded, nil
}

// isIPv4 checks whether the provided net.IP address is an IPv4 address.
// It returns true if the IP is not nil and can be converted to IPv4 format.
func isIPv4(ip net.IP) bool {
	return ip != nil && ip.To4() != nil
}

// isIPv6 checks whether the provided net.IP is a valid IPv6 address.
// It returns true if the IP is not nil, has a valid 16-byte representation,
// and does not have a valid 4-byte IPv4 representation.
func isIPv6(ip net.IP) bool {
	return ip != nil && ip.To16() != nil && ip.To4() == nil
}
