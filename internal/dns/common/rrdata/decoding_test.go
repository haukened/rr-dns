package rrdata

import (
	"fmt"
	"testing"

	"github.com/haukened/rr-dns/internal/dns/domain"
	"github.com/stretchr/testify/require"
)

func TestDecode_SwitchCoverage(t *testing.T) {
	tests := []struct {
		name         string
		rrType       domain.RRType
		wire         []byte
		wantErr      bool
		wantRawEqual bool // for default branch passthrough
	}{
		{"A", domain.RRTypeA, []byte{192, 0, 2, 1}, false, false},
		{"NS", domain.RRTypeNS, []byte{2, 'n', 's', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0}, false, false},
		{"CNAME", domain.RRTypeCNAME, []byte{5, 'a', 'l', 'i', 'a', 's', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0}, false, false},
		{"SOA", domain.RRTypeSOA, []byte{2, 'n', 's', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0, 10, 'h', 'o', 's', 't', 'm', 'a', 's', 't', 'e', 'r', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0, 0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0, 3, 0, 0, 0, 4, 0, 0, 0, 5}, false, false},
		{"PTR", domain.RRTypePTR, []byte{3, 'p', 't', 'r', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0}, false, false},
		{"MX", domain.RRTypeMX, append([]byte{0, 10}, []byte{4, 'm', 'a', 'i', 'l', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0}...), false, false},
		{"TXT", domain.RRTypeTXT, append([]byte{11}, []byte("hello world")...), false, false},
		{"AAAA", domain.RRTypeAAAA, []byte{32, 1, 13, 184, 0, 0, 255, 0, 66, 131, 41, 0, 0, 0, 0, 1}, false, false},
		{"SRV", domain.RRTypeSRV, append([]byte{0, 1, 0, 2, 0, 80}, []byte{6, 't', 'a', 'r', 'g', 'e', 't', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0}...), false, false},
		{"NAPTR not implemented", domain.RRTypeNAPTR, []byte{}, true, false},
		{"OPT not allowed", domain.RRTypeOPT, []byte{}, true, false},
		{"DS not implemented", domain.RRTypeDS, []byte{}, true, false},
		{"RRSIG not implemented", domain.RRTypeRRSIG, []byte{}, true, false},
		{"NSEC not implemented", domain.RRTypeNSEC, []byte{}, true, false},
		{"DNSKEY not implemented", domain.RRTypeDNSKEY, []byte{}, true, false},
		{"TLSA not implemented", domain.RRTypeTLSA, []byte{}, true, false},
		{"SVCB not implemented", domain.RRTypeSVCB, []byte{}, true, false},
		{"HTTPS not implemented", domain.RRTypeHTTPS, []byte{}, true, false},
		{"CAA", domain.RRTypeCAA, append([]byte{0, 5}, append([]byte("issue"), []byte("letsencrypt.org")...)...), false, false},
		{"Default passthrough", domain.RRType(9999), []byte("raw-bytes"), false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Decode(tt.rrType, tt.wire)
			if tt.wantErr {
				require.Error(t, err)
				require.Empty(t, got)
				return
			}
			require.NoError(t, err)
			if tt.wantRawEqual {
				require.Equal(t, tt.wire, got)
			} else {
				require.NotEmpty(t, got)
			}
		})
	}
}

func TestDecoderNotImplemented_ReturnsError(t *testing.T) {
	tests := []struct {
		name   string
		rrType domain.RRType
	}{
		{"NAPTR record", domain.RRTypeNAPTR},
		{"DS record", domain.RRTypeDS},
		{"RRSIG record", domain.RRTypeRRSIG},
		{"NSEC record", domain.RRTypeNSEC},
		{"DNSKEY record", domain.RRTypeDNSKEY},
		{"TLSA record", domain.RRTypeTLSA},
		{"SVCB record", domain.RRTypeSVCB},
		{"HTTPS record", domain.RRTypeHTTPS},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := decoderNotImplemented(tt.rrType)
			require.Empty(t, data, "data should be empty")
			require.Error(t, err, "error should not be nil")
			require.Contains(t, err.Error(), fmt.Sprintf("%s record decoding not implemented yet", tt.rrType))
		})
	}
}
