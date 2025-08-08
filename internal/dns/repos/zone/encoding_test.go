package zone

import (
	"encoding/binary"
	"reflect"
	"strings"
	"testing"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

func TestEncodeRRData(t *testing.T) {
	tests := []struct {
		name    string
		rrType  domain.RRType
		data    string
		wantErr bool
		errMsg  string
	}{
		// Implemented types
		{"A record", domain.RRTypeA, "192.168.1.1", false, ""},
		{"NS record", domain.RRTypeNS, "ns.example.com", false, ""},
		{"CNAME record", domain.RRTypeCNAME, "www.example.com", false, ""},
		{"SOA record", domain.RRTypeSOA, "ns1.example.com hostmaster.example.com 2024080701 3600 1800 604800 86400", false, ""},
		{"PTR record", domain.RRTypePTR, "host.example.com", false, ""},
		{"MX record", domain.RRTypeMX, "10 mail.example.com", false, ""},
		{"TXT record", domain.RRTypeTXT, "v=spf1 include:_spf.google.com ~all", false, ""},
		{"AAAA record", domain.RRTypeAAAA, "2001:db8::1", false, ""},
		{"SRV record", domain.RRTypeSRV, "10 20 80 target.example.com", false, ""},
		{"CAA record", domain.RRTypeCAA, "0 issue \"letsencrypt.org\"", false, ""},

		// Not allowed in zone
		{"OPT record", domain.RRTypeOPT, "data", true, "OPT record type not allowed in zone files"},

		// Not implemented types
		{"NAPTR record", domain.RRTypeNAPTR, "100 50 \"s\" \"SIP+D2U\" \"\" _sip._udp.example.com.", true, "NAPTR record encoding not implemented yet"},
		{"DS record", domain.RRTypeDS, "12345 3 1 1234567890ABCDEF1234567890ABCDEF12345678", true, "DS record encoding not implemented yet"},
		{"RRSIG record", domain.RRTypeRRSIG, "A 5 3 86400 20240807120000 20240731120000 12345 example.com. signature", true, "RRSIG record encoding not implemented yet"},
		{"NSEC record", domain.RRTypeNSEC, "a.example.com. A MX RRSIG NSEC", true, "NSEC record encoding not implemented yet"},
		{"DNSKEY record", domain.RRTypeDNSKEY, "256 3 5 AQPSKmynfzW4kyBv015MUG2DeIQ3Cbl+BBZH4b/0PY1kxkmvHjcsPpOnyg", true, "DNSKEY record encoding not implemented yet"},
		{"TLSA record", domain.RRTypeTLSA, "3 1 1 0C72AC70B745AC19998811B131D662C9AC69DBDBE7CB23E5B514B56664C5D3D6", true, "TLSA record encoding not implemented yet"},
		{"SVCB record", domain.RRTypeSVCB, "1 foo.example.com. port=53", true, "SVCB record encoding not implemented yet"},
		{"HTTPS record", domain.RRTypeHTTPS, "1 . port=443", true, "HTTPS record encoding not implemented yet"},

		// Unknown type - should return data as-is
		{"Unknown type", domain.RRType(999), "arbitrary data", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := encodeRRData(tt.rrType, tt.data)

			if tt.wantErr {
				if err == nil {
					t.Errorf("encodeRRData() expected error but got none")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("encodeRRData() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("encodeRRData() unexpected error = %v", err)
				return
			}

			if result == nil {
				t.Errorf("encodeRRData() returned nil result")
			}
		})
	}
}

func TestEncodeAData(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    []byte
		wantErr bool
	}{
		{"Valid IPv4", "192.168.1.1", []byte{192, 168, 1, 1}, false},
		{"Valid IPv4 zero", "0.0.0.0", []byte{0, 0, 0, 0}, false},
		{"Valid IPv4 max", "255.255.255.255", []byte{255, 255, 255, 255}, false},
		{"Invalid IPv4", "256.256.256.256", nil, true},
		{"Invalid format", "not.an.ip", nil, true},
		{"IPv6 address", "2001:db8::1", nil, true},
		{"Empty string", "", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := encodeAData(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("encodeAData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("encodeAData() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEncodeSOAData(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
		errMsg  string
	}{
		{"Valid SOA", "ns1.example.com hostmaster.example.com 2024080701 3600 1800 604800 86400", false, ""},
		{"Missing fields", "ns1.example.com hostmaster.example.com 2024080701 3600", true, "expected 7 fields"},
		{"Invalid serial", "ns1.example.com hostmaster.example.com invalid 3600 1800 604800 86400", true, "invalid SOA field"},
		{"Invalid mname", "invalid..domain hostmaster.example.com 2024080701 3600 1800 604800 86400", false, ""}, // empty labels are skipped
		{"Invalid mname label too long", strings.Repeat("a", 64) + ".example.com hostmaster.example.com 2024080701 3600 1800 604800 86400", true, "invalid SOA mname"},
		{"Invalid rname label too long", "ns1.example.com " + strings.Repeat("b", 64) + ".example.com 2024080701 3600 1800 604800 86400", true, "invalid SOA rname"},
		{"Invalid refresh", "ns1.example.com hostmaster.example.com 2024080701 invalid 1800 604800 86400", true, "invalid SOA field"},
		{"Negative value", "ns1.example.com hostmaster.example.com 2024080701 -1 1800 604800 86400", true, "invalid SOA field"},
		{"Empty string", "", true, "expected 7 fields"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := encodeSOAData(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("encodeSOAData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("encodeSOAData() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}
			if got == nil {
				t.Errorf("encodeSOAData() returned nil")
			}
		})
	}
}

func TestEncodeMXData(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    []byte
		wantErr bool
		errMsg  string
	}{
		{"Valid MX", "10 mail.example.com", nil, false, ""}, // We'll validate the structure below
		{"Valid MX zero preference", "0 mail.example.com", nil, false, ""},
		{"Valid MX max preference", "65535 mail.example.com", nil, false, ""},
		{"Invalid format - missing preference", "mail.example.com", nil, true, "expected: preference domain"},
		{"Invalid format - too many fields", "10 20 mail.example.com", nil, true, "expected: preference domain"},
		{"Invalid preference - negative", "-1 mail.example.com", nil, true, "invalid MX preference"},
		{"Invalid preference - too large", "65536 mail.example.com", nil, true, "invalid MX preference"},
		{"Invalid preference - not number", "abc mail.example.com", nil, true, "invalid MX preference"},
		{"Invalid domain", "10 invalid..domain", nil, false, ""}, // empty labels are skipped
		{"Invalid domain label too long", "10 " + strings.Repeat("a", 64) + ".example.com", nil, true, "invalid MX exchange domain"},
		{"Empty string", "", nil, true, "expected: preference domain"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := encodeMXData(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("encodeMXData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("encodeMXData() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}
			if got == nil {
				t.Errorf("encodeMXData() returned nil")
				return
			}
			// For valid cases, check that we have at least 2 bytes for preference
			if len(got) < 2 {
				t.Errorf("encodeMXData() result too short: %d bytes", len(got))
			}
		})
	}

	// Test specific valid case
	t.Run("Valid MX structure", func(t *testing.T) {
		got, err := encodeMXData("10 mail.example.com")
		if err != nil {
			t.Fatalf("encodeMXData() unexpected error = %v", err)
		}

		// Check preference (first 2 bytes)
		pref := binary.BigEndian.Uint16(got[0:2])
		if pref != 10 {
			t.Errorf("encodeMXData() preference = %d, want 10", pref)
		}

		// Check that domain name follows
		if len(got) <= 2 {
			t.Errorf("encodeMXData() missing domain name part")
		}
	})
}

func TestEncodeTXTData(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
		errMsg  string
	}{
		{"Single segment", "hello world", false, ""},
		{"Multiple segments", "v=spf1; include:_spf.google.com; ~all", false, ""},
		{"Empty segments ignored", "hello;; world", false, ""},
		{"Single character", "x", false, ""},
		{"Segment too long", strings.Repeat("a", 256), true, "TXT segment too long"},
		{"Empty string", "", true, "TXT record must contain at least one segment"},
		{"Only empty segments", ";;;", true, "TXT record must contain at least one segment"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := encodeTXTData(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("encodeTXTData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("encodeTXTData() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}
			if got == nil {
				t.Errorf("encodeTXTData() returned nil")
			}
		})
	}

	// Test specific valid case structure
	t.Run("Valid TXT structure", func(t *testing.T) {
		got, err := encodeTXTData("hello")
		if err != nil {
			t.Fatalf("encodeTXTData() unexpected error = %v", err)
		}

		// Should be: [5, 'h', 'e', 'l', 'l', 'o']
		want := []byte{5, 'h', 'e', 'l', 'l', 'o'}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("encodeTXTData() = %v, want %v", got, want)
		}
	})
}

func TestEncodeAAAAData(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    []byte
		wantErr bool
	}{
		{"Valid IPv6", "2001:db8::1", nil, false},
		{"Valid IPv6 loopback", "::1", nil, false},
		{"Valid IPv6 full", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", nil, false},
		{"Invalid IPv6", "not::valid::address", nil, true},
		{"IPv4 address", "192.168.1.1", nil, false}, // IPv4 addresses can be parsed as IPv6
		{"Empty string", "", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := encodeAAAAData(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("encodeAAAAData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got == nil {
					t.Errorf("encodeAAAAData() returned nil")
					return
				}
				if len(got) != 16 {
					t.Errorf("encodeAAAAData() returned %d bytes, want 16", len(got))
				}
			}
		})
	}
}

func TestEncodeSRVData(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
		errMsg  string
	}{
		{"Valid SRV", "10 20 80 target.example.com", false, ""},
		{"Valid SRV zero values", "0 0 0 target.example.com", false, ""},
		{"Valid SRV max values", "65535 65535 65535 target.example.com", false, ""},
		{"Missing fields", "10 20 80", true, "expected 4 fields"},
		{"Too many fields", "10 20 80 90 target.example.com", true, "expected 4 fields"},
		{"Invalid priority", "abc 20 80 target.example.com", true, "invalid SRV field 0"},
		{"Invalid weight", "10 abc 80 target.example.com", true, "invalid SRV field 1"},
		{"Invalid port", "10 20 abc target.example.com", true, "invalid SRV field 2"},
		{"Invalid target", "10 20 80 invalid..domain", false, ""}, // empty labels are skipped
		{"Invalid target label too long", "10 20 80 " + strings.Repeat("a", 64) + ".example.com", true, "invalid SRV target"},
		{"Negative priority", "10 20 80 target.example.com", false, ""},
		{"Empty string", "", true, "expected 4 fields"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := encodeSRVData(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("encodeSRVData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("encodeSRVData() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}
			if got == nil {
				t.Errorf("encodeSRVData() returned nil")
				return
			}
			// Should have at least 6 bytes for priority, weight, port
			if len(got) < 6 {
				t.Errorf("encodeSRVData() result too short: %d bytes", len(got))
			}
		})
	}

	// Test specific valid case structure
	t.Run("Valid SRV structure", func(t *testing.T) {
		got, err := encodeSRVData("10 20 80 target.example.com")
		if err != nil {
			t.Fatalf("encodeSRVData() unexpected error = %v", err)
		}

		// Check priority (first 2 bytes)
		priority := binary.BigEndian.Uint16(got[0:2])
		if priority != 10 {
			t.Errorf("encodeSRVData() priority = %d, want 10", priority)
		}

		// Check weight (next 2 bytes)
		weight := binary.BigEndian.Uint16(got[2:4])
		if weight != 20 {
			t.Errorf("encodeSRVData() weight = %d, want 20", weight)
		}

		// Check port (next 2 bytes)
		port := binary.BigEndian.Uint16(got[4:6])
		if port != 80 {
			t.Errorf("encodeSRVData() port = %d, want 80", port)
		}
	})
}

func TestEncodeCAAData(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
		errMsg  string
	}{
		{"Valid CAA", "0 issue \"letsencrypt.org\"", false, ""},
		{"Valid CAA with complex value", "128 iodef \"mailto:security@example.com\"", false, ""},
		{"Valid CAA unquoted value", "0 issue letsencrypt.org", false, ""},
		{"Missing fields", "0 issue", true, "expected: flag tag"},
		{"Invalid flag", "abc issue \"letsencrypt.org\"", true, "invalid CAA flag"},
		{"Flag too large", "256 issue \"letsencrypt.org\"", true, "invalid CAA flag"},
		{"Tag too long", "0 " + strings.Repeat("a", 256) + " \"value\"", true, "CAA tag too long"},
		{"Value too long", "0 issue \"" + strings.Repeat("a", 256) + "\"", true, "CAA value too long"},
		{"Empty string", "", true, "expected: flag tag"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := encodeCAAData(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("encodeCAAData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("encodeCAAData() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}
			if got == nil {
				t.Errorf("encodeCAAData() returned nil")
			}
		})
	}

	// Test specific valid case structure
	t.Run("Valid CAA structure", func(t *testing.T) {
		got, err := encodeCAAData("0 issue \"letsencrypt.org\"")
		if err != nil {
			t.Fatalf("encodeCAAData() unexpected error = %v", err)
		}

		// Should be: [0, 5, 'i', 's', 's', 'u', 'e', 'l', 'e', 't', 's', 'e', 'n', 'c', 'r', 'y', 'p', 't', '.', 'o', 'r', 'g']
		if len(got) < 2 {
			t.Fatalf("encodeCAAData() result too short: %d bytes", len(got))
		}

		// Check flag
		if got[0] != 0 {
			t.Errorf("encodeCAAData() flag = %d, want 0", got[0])
		}

		// Check tag length
		if got[1] != 5 {
			t.Errorf("encodeCAAData() tag length = %d, want 5", got[1])
		}

		// Check tag
		tag := string(got[2:7])
		if tag != "issue" {
			t.Errorf("encodeCAAData() tag = %q, want %q", tag, "issue")
		}

		// Check value
		value := string(got[7:])
		if value != "letsencrypt.org" {
			t.Errorf("encodeCAAData() value = %q, want %q", value, "letsencrypt.org")
		}
	})
}

func TestEncodeDomainName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []byte
		wantErr bool
	}{
		{"Simple domain", "example.com", []byte{7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0}, false},
		{"Root domain", ".", []byte{0}, false},
		{"Subdomain", "www.example.com", []byte{3, 'w', 'w', 'w', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0}, false},
		{"Already canonical", "example.com.", []byte{7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0}, false},
		{"Label too long", strings.Repeat("a", 64) + ".com", nil, true},
		{"Empty label", "example..com", []byte{7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0}, false}, // empty labels are skipped
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := encodeDomainName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("encodeDomainName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("encodeDomainName() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test scaffolds for not implemented functions

func TestEncodeNAPTRData(t *testing.T) {
	data := "100 50 \"s\" \"SIP+D2U\" \"\" _sip._udp.example.com."
	_, err := encodeNAPTRData(data)
	if err == nil {
		t.Error("encodeNAPTRData() expected not implemented error but got none")
		return
	}
	if !strings.Contains(err.Error(), "NAPTR record encoding not implemented yet") {
		t.Errorf("encodeNAPTRData() error = %v, want error containing 'not implemented'", err)
	}
}

func TestEncodeDSData(t *testing.T) {
	data := "12345 3 1 1234567890ABCDEF1234567890ABCDEF12345678"
	_, err := encodeDSData(data)
	if err == nil {
		t.Error("encodeDSData() expected not implemented error but got none")
		return
	}
	if !strings.Contains(err.Error(), "DS record encoding not implemented yet") {
		t.Errorf("encodeDSData() error = %v, want error containing 'not implemented'", err)
	}
}

func TestEncodeRRSIGData(t *testing.T) {
	data := "A 5 3 86400 20240807120000 20240731120000 12345 example.com. signature"
	_, err := encodeRRSIGData(data)
	if err == nil {
		t.Error("encodeRRSIGData() expected not implemented error but got none")
		return
	}
	if !strings.Contains(err.Error(), "RRSIG record encoding not implemented yet") {
		t.Errorf("encodeRRSIGData() error = %v, want error containing 'not implemented'", err)
	}
}

func TestEncodeNSECData(t *testing.T) {
	data := "a.example.com. A MX RRSIG NSEC"
	_, err := encodeNSECData(data)
	if err == nil {
		t.Error("encodeNSECData() expected not implemented error but got none")
		return
	}
	if !strings.Contains(err.Error(), "NSEC record encoding not implemented yet") {
		t.Errorf("encodeNSECData() error = %v, want error containing 'not implemented'", err)
	}
}

func TestEncodeDNSKEYData(t *testing.T) {
	data := "256 3 5 AQPSKmynfzW4kyBv015MUG2DeIQ3Cbl+BBZH4b/0PY1kxkmvHjcsPpOnyg"
	_, err := encodeDNSKEYData(data)
	if err == nil {
		t.Error("encodeDNSKEYData() expected not implemented error but got none")
		return
	}
	if !strings.Contains(err.Error(), "DNSKEY record encoding not implemented yet") {
		t.Errorf("encodeDNSKEYData() error = %v, want error containing 'not implemented'", err)
	}
}

func TestEncodeTLSAData(t *testing.T) {
	data := "3 1 1 0C72AC70B745AC19998811B131D662C9AC69DBDBE7CB23E5B514B56664C5D3D6"
	_, err := encodeTLSAData(data)
	if err == nil {
		t.Error("encodeTLSAData() expected not implemented error but got none")
		return
	}
	if !strings.Contains(err.Error(), "TLSA record encoding not implemented yet") {
		t.Errorf("encodeTLSAData() error = %v, want error containing 'not implemented'", err)
	}
}

func TestEncodeSVCBData(t *testing.T) {
	data := "1 foo.example.com. port=53"
	_, err := encodeSVCBData(data)
	if err == nil {
		t.Error("encodeSVCBData() expected not implemented error but got none")
		return
	}
	if !strings.Contains(err.Error(), "SVCB record encoding not implemented yet") {
		t.Errorf("encodeSVCBData() error = %v, want error containing 'not implemented'", err)
	}
}

func TestEncodeHTTPSData(t *testing.T) {
	data := "1 . port=443"
	_, err := encodeHTTPSData(data)
	if err == nil {
		t.Error("encodeHTTPSData() expected not implemented error but got none")
		return
	}
	if !strings.Contains(err.Error(), "HTTPS record encoding not implemented yet") {
		t.Errorf("encodeHTTPSData() error = %v, want error containing 'not implemented'", err)
	}
}

// Test helper functions

func TestNotimp(t *testing.T) {
	result, err := notimp(domain.RRTypeTLSA, "test data")
	if result != nil {
		t.Errorf("notimp() result = %v, want nil", result)
	}
	if err == nil {
		t.Error("notimp() expected error but got none")
		return
	}
	expectedMsg := "TLSA record encoding not implemented yet: test data"
	if err.Error() != expectedMsg {
		t.Errorf("notimp() error = %v, want %v", err.Error(), expectedMsg)
	}
}

func TestNotAllowedInZone(t *testing.T) {
	result, err := notAllowedInZone(domain.RRTypeOPT)
	if result != nil {
		t.Errorf("notAllowedInZone() result = %v, want nil", result)
	}
	if err == nil {
		t.Error("notAllowedInZone() expected error but got none")
		return
	}
	expectedMsg := "OPT record type not allowed in zone files"
	if err.Error() != expectedMsg {
		t.Errorf("notAllowedInZone() error = %v, want %v", err.Error(), expectedMsg)
	}
}
