package rrdata

import (
	"fmt"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

// Decode decodes a record value based on its type, from its binary representation.
func Decode(rrType domain.RRType, data []byte) (string, error) {
	switch rrType {
	case domain.RRTypeA: // 1
		return decodeAData(data)
	case domain.RRTypeNS: // 2
		return decodeNSData(data)
	case domain.RRTypeCNAME: // 5
		return decodeCNAMEData(data)
	case domain.RRTypeSOA: // 6
		return decodeSOAData(data)
	case domain.RRTypePTR: // 12
		return decodePTRData(data)
	case domain.RRTypeMX: // 15
		return decodeMXData(data)
	case domain.RRTypeTXT: // 16
		return decodeTXTData(data)
	case domain.RRTypeAAAA: // 28
		return decodeAAAAData(data)
	case domain.RRTypeSRV: // 33
		return decodeSRVData(data)
	case domain.RRTypeNAPTR: // 35
		return decoderNotImplemented(domain.RRTypeNAPTR)
	case domain.RRTypeOPT: // 41
		return decoderNotImplemented(domain.RRTypeOPT)
	case domain.RRTypeDS: // 43
		return decoderNotImplemented(domain.RRTypeDS)
	case domain.RRTypeRRSIG: // 46
		return decoderNotImplemented(domain.RRTypeRRSIG)
	case domain.RRTypeNSEC: // 47
		return decoderNotImplemented(domain.RRTypeNSEC)
	case domain.RRTypeDNSKEY: // 48
		return decoderNotImplemented(domain.RRTypeDNSKEY)
	case domain.RRTypeTLSA: // 52
		return decoderNotImplemented(domain.RRTypeTLSA)
	case domain.RRTypeSVCB: // 64
		return decoderNotImplemented(domain.RRTypeSVCB)
	case domain.RRTypeHTTPS: // 65
		return decoderNotImplemented(domain.RRTypeHTTPS)
	case domain.RRTypeCAA: // 257
		return decodeCAAData(data)
	default:
		return string(data), nil
	}
}

func decoderNotImplemented(t domain.RRType) (string, error) {
	return "", fmt.Errorf("%s record decoding not implemented yet", t)
}
