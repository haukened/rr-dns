package rrdata

import (
	"fmt"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

// Encode encodes a record value based on its type, to its binary representation.
func Encode(rrType domain.RRType, data string) ([]byte, error) {
	switch rrType {
	case domain.RRTypeA: // 1
		return encodeAData(data)
	case domain.RRTypeNS: // 2
		return encodeNSData(data)
	case domain.RRTypeCNAME: // 5
		return encodeCNAMEData(data)
	case domain.RRTypeSOA: // 6
		return encodeSOAData(data)
	case domain.RRTypePTR: // 12
		return encodePTRData(data)
	case domain.RRTypeMX: // 15
		return encodeMXData(data)
	case domain.RRTypeTXT: // 16
		return encodeTXTData(data)
	case domain.RRTypeAAAA: // 28
		return encodeAAAAData(data)
	case domain.RRTypeSRV: // 33
		return encodeSRVData(data)
	case domain.RRTypeNAPTR: // 35
		return encoderNotImplemented(domain.RRTypeNAPTR)
	case domain.RRTypeOPT: // 41
		return encoderNotImplemented(domain.RRTypeOPT)
	case domain.RRTypeDS: // 43
		return encoderNotImplemented(domain.RRTypeDS)
	case domain.RRTypeRRSIG: // 46
		return encoderNotImplemented(domain.RRTypeRRSIG)
	case domain.RRTypeNSEC: // 47
		return encoderNotImplemented(domain.RRTypeNSEC)
	case domain.RRTypeDNSKEY: // 48
		return encoderNotImplemented(domain.RRTypeDNSKEY)
	case domain.RRTypeTLSA: // 52
		return encoderNotImplemented(domain.RRTypeTLSA)
	case domain.RRTypeSVCB: // 64
		return encoderNotImplemented(domain.RRTypeSVCB)
	case domain.RRTypeHTTPS: // 65
		return encoderNotImplemented(domain.RRTypeHTTPS)
	case domain.RRTypeCAA: // 257
		return encodeCAAData(data)
	default:
		return []byte(data), nil
	}
}

// encoderNotImplemented returns an error indicating that the specified DNS record type encoding is not implemented yet.
func encoderNotImplemented(t domain.RRType) ([]byte, error) {
	return nil, fmt.Errorf("%s record encoding not implemented yet", t)
}
