package zone

import (
	"fmt"
	"testing"

	"github.com/haukened/rr-dns/internal/dns/domain"
	"github.com/stretchr/testify/require"
)

func TestNotAllowedInZone_ReturnsErrorAndNil(t *testing.T) {
	tests := []struct {
		name   string
		rrType domain.RRType
	}{
		{"OPT record", domain.RRTypeOPT},
		{"Unknown record", domain.RRType(9999)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := notAllowedInZone(tt.rrType)
			require.Nil(t, data, "data should be nil")
			require.Error(t, err, "error should not be nil")
			require.Contains(t, err.Error(), fmt.Sprintf("%s record type not allowed in zone files", tt.rrType))
		})
	}
}
func TestNotimp_ReturnsErrorAndNil(t *testing.T) {
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
			data, err := notimp(tt.rrType)
			require.Nil(t, data, "data should be nil")
			require.Error(t, err, "error should not be nil")
			require.Contains(t, err.Error(), fmt.Sprintf("%s record encoding not implemented yet", tt.rrType))
		})
	}
}

// TestEncodeRRData_SwitchCoverage ensures each case in encodeRRData's switch is executed at least once.
func TestEncodeRRData_SwitchCoverage(t *testing.T) {
	tests := []struct {
		name         string
		rrType       domain.RRType
		data         string
		wantErr      bool
		wantRawEqual bool // for default branch passthrough
	}{
		{"A", domain.RRTypeA, "192.0.2.1", false, false},
		{"NS", domain.RRTypeNS, "ns.example.com", false, false},
		{"CNAME", domain.RRTypeCNAME, "alias.example.com", false, false},
		{"SOA", domain.RRTypeSOA, "ns.example.com hostmaster.example.com 2024010101 7200 3600 1209600 3600", false, false},
		{"PTR", domain.RRTypePTR, "ptr.example.com", false, false},
		{"MX", domain.RRTypeMX, "10 mail.example.com", false, false},
		{"TXT", domain.RRTypeTXT, "hello world", false, false},
		{"AAAA", domain.RRTypeAAAA, "2001:db8::1", false, false},
		{"SRV", domain.RRTypeSRV, "1 2 80 target.example.com", false, false},
		{"NAPTR not implemented", domain.RRTypeNAPTR, "ignored", true, false},
		{"OPT not allowed", domain.RRTypeOPT, "ignored", true, false},
		{"DS not implemented", domain.RRTypeDS, "ignored", true, false},
		{"RRSIG not implemented", domain.RRTypeRRSIG, "ignored", true, false},
		{"NSEC not implemented", domain.RRTypeNSEC, "ignored", true, false},
		{"DNSKEY not implemented", domain.RRTypeDNSKEY, "ignored", true, false},
		{"TLSA not implemented", domain.RRTypeTLSA, "ignored", true, false},
		{"SVCB not implemented", domain.RRTypeSVCB, "ignored", true, false},
		{"HTTPS not implemented", domain.RRTypeHTTPS, "ignored", true, false},
		{"CAA", domain.RRTypeCAA, "0 issue \"letsencrypt.org\"", false, false},
		{"Default passthrough", domain.RRType(9999), "raw-bytes", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := encodeRRData(tt.rrType, tt.data)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			if tt.wantRawEqual {
				require.Equal(t, []byte(tt.data), got)
			} else {
				require.NotEmpty(t, got)
			}
		})
	}
}
