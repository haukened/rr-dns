package rrdata

import (
	"fmt"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

// Encode encodes a record value based on its type, to its binary representation.
func Encode(rrType domain.RRType, data string) ([]byte, error) {
	switch rrType {
	case domain.RRTypeA: // 1
		return EncodeAData(data)
	case domain.RRTypeNS: // 2
		return EncodeNSData(data)
	case domain.RRTypeCNAME: // 5
		return EncodeCNAMEData(data)
	case domain.RRTypeSOA: // 6
		return EncodeSOAData(data)
	case domain.RRTypePTR: // 12
		return EncodePTRData(data)
	case domain.RRTypeMX: // 15
		return EncodeMXData(data)
	case domain.RRTypeTXT: // 16
		return EncodeTXTData(data)
	case domain.RRTypeAAAA: // 28
		return EncodeAAAAData(data)
	case domain.RRTypeSRV: // 33
		return EncodeSRVData(data)
	case domain.RRTypeNAPTR: // 35
		return notimp(domain.RRTypeNAPTR)
	case domain.RRTypeOPT: // 41
		return notAllowedInZone(domain.RRTypeOPT)
	case domain.RRTypeDS: // 43
		return notimp(domain.RRTypeDS)
	case domain.RRTypeRRSIG: // 46
		return notimp(domain.RRTypeRRSIG)
	case domain.RRTypeNSEC: // 47
		return notimp(domain.RRTypeNSEC)
	case domain.RRTypeDNSKEY: // 48
		return notimp(domain.RRTypeDNSKEY)
	case domain.RRTypeTLSA: // 52
		return notimp(domain.RRTypeTLSA)
	case domain.RRTypeSVCB: // 64
		return notimp(domain.RRTypeSVCB)
	case domain.RRTypeHTTPS: // 65
		return notimp(domain.RRTypeHTTPS)
	case domain.RRTypeCAA: // 257
		return EncodeCAAData(data)
	default:
		return []byte(data), nil
	}
}

// notimp returns an error indicating that the specified DNS record type encoding is not implemented yet.
func notimp(t domain.RRType) ([]byte, error) {
	return nil, fmt.Errorf("%s record encoding not implemented yet", t)
}

// notAllowedInZone returns an error indicating that the specified DNS record type is not allowed in zone files.
func notAllowedInZone(t domain.RRType) ([]byte, error) {
	return nil, fmt.Errorf("%s record type not allowed in zone files", t)
}
