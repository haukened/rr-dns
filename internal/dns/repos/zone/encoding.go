package zone

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/haukened/rr-dns/internal/dns/common/utils"
	"github.com/haukened/rr-dns/internal/dns/domain"
)

// encodeRRData encodes a record value based on its type, to its binary representation.
func encodeRRData(rrType domain.RRType, data string) ([]byte, error) {
	switch rrType {
	case domain.RRTypeA: // 1
		return encodeAData(data)
	case domain.RRTypeNS: // 2
		return encodeDomainName(data)
	case domain.RRTypeCNAME: // 5
		return encodeDomainName(data)
	case domain.RRTypeSOA: // 6
		return encodeSOAData(data)
	case domain.RRTypePTR: // 12
		return encodeDomainName(data)
	case domain.RRTypeMX: // 15
		return encodeMXData(data)
	case domain.RRTypeTXT: // 16
		return encodeTXTData(data)
	case domain.RRTypeAAAA: // 28
		return encodeAAAAData(data)
	case domain.RRTypeSRV: // 33
		return encodeSRVData(data)
	case domain.RRTypeNAPTR: // 35
		return encodeNAPTRData(data)
	case domain.RRTypeOPT: // 41
		return notAllowedInZone(domain.RRTypeOPT)
	case domain.RRTypeDS: // 43
		return encodeDSData(data)
	case domain.RRTypeRRSIG: // 46
		return encodeRRSIGData(data)
	case domain.RRTypeNSEC: // 47
		return encodeNSECData(data)
	case domain.RRTypeDNSKEY: // 48
		return encodeDNSKEYData(data)
	case domain.RRTypeTLSA: // 52
		return encodeTLSAData(data)
	case domain.RRTypeSVCB: // 64
		return encodeSVCBData(data)
	case domain.RRTypeHTTPS: // 65
		return encodeHTTPSData(data)
	case domain.RRTypeCAA: // 257
		return encodeCAAData(data)
	default:
		return []byte(data), nil
	}
}

// notimp returns an error indicating that encoding for the specified DNS record type is not implemented.
// It takes a domain.RRType and a data string as input, and always returns nil and an error describing
// the unimplemented record type and data.
//
// Parameters:
//
//	t    - The DNS record type for which encoding is not implemented.
//	data - The data associated with the DNS record.
//
// Returns:
//
//	nil and an error indicating the encoding is not implemented.
func notimp(t domain.RRType, data string) ([]byte, error) {
	return nil, fmt.Errorf("%s record encoding not implemented yet: %s", t, data)
}

// notAllowedInZone returns an error indicating that the specified DNS record type is not allowed in zone files.
// It takes a domain.RRType as input and returns nil and a formatted error describing the restriction.
func notAllowedInZone(t domain.RRType) ([]byte, error) {
	return nil, fmt.Errorf("%s record type not allowed in zone files", t)
}

// encodeAData encodes an A record string into its binary representation.
func encodeAData(data string) ([]byte, error) {
	// data = "192.168.0.1"
	ip := net.ParseIP(data).To4()
	if ip == nil {
		return nil, fmt.Errorf("invalid A record IP: %s", data)
	}
	return ip, nil
}

// encodeSOAData encodes an SOA record string into its binary representation.
func encodeSOAData(data string) ([]byte, error) {
	// data = "mname rname serial refresh retry expire minimum"
	parts := strings.Fields(data)
	if len(parts) != 7 {
		return nil, fmt.Errorf("invalid SOA record format (expected 7 fields): %s", data)
	}

	// mname is the primary name server for the zone
	mname, err := encodeDomainName(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid SOA mname: %v", err)
	}

	// rname is the email address of the zone administrator, with '.' replaced by '@'
	// e.g. "hostmaster.example.com" becomes "hostmaster@example.com"
	rname, err := encodeDomainName(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid SOA rname: %v", err)
	}

	// The next five fields are unsigned integers
	// serial, refresh, retry, expire, minimum
	u32 := make([]byte, 20)
	for i := 0; i < 5; i++ {
		val, err := strconv.ParseUint(parts[i+2], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid SOA field %d: %v", i+2, err)
		}
		binary.BigEndian.PutUint32(u32[i*4:], uint32(val))
	}

	// Combine all parts into a single byte slice
	var encoded []byte
	encoded = append(encoded, mname...)
	encoded = append(encoded, rname...)
	encoded = append(encoded, u32...)

	return encoded, nil
}

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

// encodeAAAAData encodes an AAAA record string into its binary representation.
func encodeAAAAData(data string) ([]byte, error) {
	// data = "2001:db8::ff00:42:8329"
	ip := net.ParseIP(data).To16()
	if ip == nil {
		return nil, fmt.Errorf("invalid AAAA record IP: %s", data)
	}
	return ip, nil
}

// encodeSRVData encodes an SRV record string into its binary representation.
func encodeSRVData(data string) ([]byte, error) {
	// data = "priority weight port target"
	parts := strings.Fields(data)
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid SRV record format (expected 4 fields): %s", data)
	}

	// priority, weight, and port must be valid unsigned integers
	buf := make([]byte, 6)
	for i := 0; i < 3; i++ {
		val, err := strconv.ParseUint(parts[i], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid SRV field %d: %v", i, err)
		}
		binary.BigEndian.PutUint16(buf[i*2:], uint16(val))
	}

	// target is the domain name of the service
	target, err := encodeDomainName(parts[3])
	if err != nil {
		return nil, fmt.Errorf("invalid SRV target: %v", err)
	}

	// Append the target domain name to the buffer
	return append(buf, target...), nil
}

// encodeNAPTRData encodes a NAPTR record string into its binary representation.
func encodeNAPTRData(data string) ([]byte, error) {
	return notimp(domain.RRTypeNAPTR, data)
}

// encodeDSData encodes a DS record string into its binary representation.
func encodeDSData(data string) ([]byte, error) {
	return notimp(domain.RRTypeDS, data)
}

// encodeRRSIGData encodes an RRSIG record string into its binary representation.
func encodeRRSIGData(data string) ([]byte, error) {
	return notimp(domain.RRTypeRRSIG, data)
}

// encodeNSECData encodes an NSEC record string into its binary representation.
func encodeNSECData(data string) ([]byte, error) {
	return notimp(domain.RRTypeNSEC, data)
}

// encodeDNSKEYData encodes a DNSKEY record string into its binary representation.
func encodeDNSKEYData(data string) ([]byte, error) {
	return notimp(domain.RRTypeDNSKEY, data)
}

// encodeTLSAData encodes a TLSA record string into its binary representation.
func encodeTLSAData(data string) ([]byte, error) {
	return notimp(domain.RRTypeTLSA, data)
}

// encodeSVCBData encodes a SVCB record string into its binary representation.
func encodeSVCBData(data string) ([]byte, error) {
	return notimp(domain.RRTypeSVCB, data)
}

// encodeHTTPSData encodes an HTTPS record string into its binary representation.
func encodeHTTPSData(data string) ([]byte, error) {
	return notimp(domain.RRTypeHTTPS, data)
}

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

// encodeDomainName encodes a domain name into wire format (length-prefixed labels ending in 0).
func encodeDomainName(name string) ([]byte, error) {
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
