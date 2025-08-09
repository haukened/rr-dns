package wire

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/haukened/rr-dns/internal/dns/common/log"
	"github.com/haukened/rr-dns/internal/dns/domain"
)

func TestUdpCodec_EncodeQuery(t *testing.T) {
	codec := &udpCodec{
		logger: log.NewNoopLogger(),
	}

	tests := []struct {
		name       string
		query      domain.Question
		wantErr    string
		checkBytes func([]byte) bool
	}{
		{
			name: "valid A query",
			query: domain.Question{
				ID:   12345,
				Name: "example.com.",
				Type: 1, // A record
			},
			checkBytes: func(data []byte) bool {
				// Check header
				if len(data) < 12 {
					return false
				}
				// Check ID
				if binary.BigEndian.Uint16(data[0:2]) != 12345 {
					return false
				}
				// Check flags (0x0100 = standard query with RD=1)
				if binary.BigEndian.Uint16(data[2:4]) != 0x0100 {
					return false
				}
				// Check QDCOUNT = 1
				if binary.BigEndian.Uint16(data[4:6]) != 1 {
					return false
				}
				// Check other counts = 0
				if binary.BigEndian.Uint16(data[6:8]) != 0 ||
					binary.BigEndian.Uint16(data[8:10]) != 0 ||
					binary.BigEndian.Uint16(data[10:12]) != 0 {
					return false
				}
				return true
			},
		},
		{
			name: "empty domain name",
			query: domain.Question{
				ID:   1,
				Name: "",
				Type: 1,
			},
			checkBytes: func(data []byte) bool {
				// Should have header + single zero byte for empty name + QTYPE + QCLASS
				// When name is empty, it gets split into [""], so we get 1 byte length + 0 byte string + 0 terminator
				return len(data) >= 12+1+2+2 // At least header + terminator + QTYPE + QCLASS
			},
		},
		{
			name: "long label error",
			query: domain.Question{
				ID:   1,
				Name: "this-is-a-very-long-label-that-exceeds-the-maximum-allowed-length-of-63-characters-for-dns-labels.com.",
				Type: 1,
			},
			wantErr: "label too long",
		},
		{
			name: "single label",
			query: domain.Question{
				ID:   1,
				Name: "localhost.",
				Type: 1,
			},
			checkBytes: func(data []byte) bool {
				// Header + label encoding + QTYPE + QCLASS
				// localhost = 1 + 9 + "localhost" + 1 + 0 + 2 + 2 = 16 bytes after header
				// Actually: 9 + "localhost" + 0 + 2 + 2 = 14 bytes after header
				return len(data) >= 12+1+9+1+2+2 // header + len + "localhost" + terminator + QTYPE + QCLASS
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := codec.EncodeQuery(tt.query)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.checkBytes != nil {
					assert.True(t, tt.checkBytes(result), "encoded bytes validation failed")
				}
			}
		})
	}
}

func TestUdpCodec_DecodeQuery(t *testing.T) {
	codec := &udpCodec{
		logger: log.NewNoopLogger(),
	}

	tests := []struct {
		name     string
		data     []byte
		wantErr  string
		expected domain.Question
	}{
		{
			name: "valid query",
			data: func() []byte {
				// Create a valid DNS query packet for "example.com" A record
				data := make([]byte, 0, 512)

				// Header
				data = binary.BigEndian.AppendUint16(data, 12345)  // ID
				data = binary.BigEndian.AppendUint16(data, 0x0100) // Flags
				data = binary.BigEndian.AppendUint16(data, 1)      // QDCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // ANCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // NSCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // ARCOUNT

				// Question: example.com
				data = append(data, 7) // length of "example"
				data = append(data, []byte("example")...)
				data = append(data, 3) // length of "com"
				data = append(data, []byte("com")...)
				data = append(data, 0)                        // end of name
				data = binary.BigEndian.AppendUint16(data, 1) // QTYPE = A
				data = binary.BigEndian.AppendUint16(data, 1) // QCLASS = IN

				return data
			}(),
			expected: domain.Question{
				ID:   12345,
				Name: "example.com",
				Type: 1, // A record
			},
		},
		{
			name:    "too short",
			data:    []byte{1, 2, 3, 4, 5},
			wantErr: "query too short",
		},
		{
			name: "multiple questions",
			data: func() []byte {
				data := make([]byte, 0, 512)
				data = binary.BigEndian.AppendUint16(data, 1)      // ID
				data = binary.BigEndian.AppendUint16(data, 0x0100) // Flags
				data = binary.BigEndian.AppendUint16(data, 2)      // QDCOUNT = 2
				data = binary.BigEndian.AppendUint16(data, 0)      // ANCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // NSCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // ARCOUNT
				return data
			}(),
			wantErr: "expected exactly one question",
		},
		{
			name: "truncated question",
			data: func() []byte {
				data := make([]byte, 0, 512)
				data = binary.BigEndian.AppendUint16(data, 1)      // ID
				data = binary.BigEndian.AppendUint16(data, 0x0100) // Flags
				data = binary.BigEndian.AppendUint16(data, 1)      // QDCOUNT = 1
				data = binary.BigEndian.AppendUint16(data, 0)      // ANCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // NSCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // ARCOUNT
				// No question data
				return data
			}(),
			wantErr: "offset out of bounds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := codec.DecodeQuery(tt.data)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.ID, result.ID)
				assert.Equal(t, tt.expected.Name, result.Name)
				assert.Equal(t, tt.expected.Type, result.Type)
			}
		})
	}
}

func TestUdpCodec_EncodeResponse(t *testing.T) {
	codec := &udpCodec{
		logger: log.NewNoopLogger(),
	}

	// Create a test resource record
	rr, err := domain.NewAuthoritativeResourceRecord(
		"example.com.",
		1,                    // A record
		1,                    // IN class
		300,                  // TTL
		[]byte{192, 0, 2, 1}, // IP address 192.0.2.1
	)
	assert.NoError(t, err)

	tests := []struct {
		name       string
		response   domain.DNSResponse
		wantErr    string
		checkBytes func([]byte) bool
	}{
		{
			name: "invalid question name label too long",
			response: domain.DNSResponse{
				ID:    1,
				RCode: 0,
				Question: domain.Question{
					Name:  "this-is-a-very-long-label-that-exceeds-the-maximum-allowed-length-of-63-characters-for-dns-labels.com.",
					Type:  domain.RRType(1),
					Class: domain.RRClass(1),
				},
				Answers: nil,
			},
			wantErr: "label too long",
		},
		{
			name: "valid response",
			response: domain.DNSResponse{
				ID:       12345,
				RCode:    0, // NOERROR
				Question: domain.Question{Name: "example.com.", Type: domain.RRType(1), Class: domain.RRClass(1)},
				Answers:  []domain.ResourceRecord{rr},
			},
			checkBytes: func(data []byte) bool {
				if len(data) < 12 {
					return false
				}
				// Check ID
				if binary.BigEndian.Uint16(data[0:2]) != 12345 {
					return false
				}
				// Check flags (0x8180 = response with RA=1)
				if binary.BigEndian.Uint16(data[2:4]) != 0x8180 {
					return false
				}
				// Check QDCOUNT = 1, ANCOUNT = 1
				if binary.BigEndian.Uint16(data[4:6]) != 1 ||
					binary.BigEndian.Uint16(data[6:8]) != 1 {
					return false
				}
				return true
			},
		},
		{
			name: "invalid domain name in second answer",
			response: domain.DNSResponse{
				ID:       1,
				RCode:    0,
				Question: domain.Question{Name: "valid.com.", Type: domain.RRType(1), Class: domain.RRClass(1)},
				Answers: []domain.ResourceRecord{
					{
						Name:  "valid.com.",
						Type:  1,
						Class: 1,
						Data:  []byte{1, 2, 3, 4},
					},
					{
						Name:  "this-is-a-very-long-label-that-exceeds-the-maximum-allowed-length-of-63-characters-for-dns-labels.com.",
						Type:  1,
						Class: 1,
						Data:  []byte{1, 2, 3, 4},
					},
				},
			},
			wantErr: "label too long",
		},
		{
			name: "invalid domain name in answer loop",
			response: domain.DNSResponse{
				ID:       1,
				RCode:    0,
				Question: domain.Question{Name: "valid.com.", Type: domain.RRType(1), Class: domain.RRClass(1)},
				Answers: []domain.ResourceRecord{
					{
						Name:  "valid.com.",
						Type:  1,
						Class: 1,
						Data:  []byte{1, 2, 3, 4},
					},
					{
						Name:  "this-is-a-very-long-label-that-exceeds-the-maximum-allowed-length-of-63-characters-for-dns-labels.com.",
						Type:  1,
						Class: 1,
						Data:  []byte{1, 2, 3, 4},
					},
				},
			},
			wantErr: "label too long",
		},
		{
			name: "too many answer records",
			response: domain.DNSResponse{
				ID:       1,
				RCode:    0,
				Question: domain.Question{Name: "example.com.", Type: domain.RRType(1), Class: domain.RRClass(1)},
				Answers: func() []domain.ResourceRecord {
					// Create more than 65535 answer records to trigger bounds check
					answers := make([]domain.ResourceRecord, 65536)
					for i := range answers {
						answers[i] = domain.ResourceRecord{
							Name:  "example.com.",
							Type:  1,
							Class: 1,
							Data:  []byte{192, 0, 2, 1},
						}
					}
					return answers
				}(),
			},
			wantErr: "too many answer records: 65536 (max 65535)",
		},
		{
			name: "resource record data too large",
			response: domain.DNSResponse{
				ID:       1,
				RCode:    0,
				Question: domain.Question{Name: "example.com.", Type: domain.RRType(1), Class: domain.RRClass(1)},
				Answers: []domain.ResourceRecord{
					{
						Name:  "example.com.",
						Type:  1,
						Class: 1,
						Data: func() []byte {
							// Create data larger than 65535 bytes to trigger bounds check
							data := make([]byte, 65536)
							for i := range data {
								data[i] = byte(i % 256)
							}
							return data
						}(),
					},
				},
			},
			wantErr: "resource record data too large: 65536 bytes (max 65535)",
		},
		{
			name: "multiple answers with different names",
			response: domain.DNSResponse{
				ID:       54321,
				RCode:    0,
				Question: domain.Question{Name: "first.example.com.", Type: domain.RRType(1), Class: domain.RRClass(1)},
				Answers: []domain.ResourceRecord{
					{
						Name:  "first.example.com.",
						Type:  1,
						Class: 1,
						Data:  []byte{192, 0, 2, 1},
					},
					{
						Name:  "second.example.com.",
						Type:  1,
						Class: 1,
						Data:  []byte{192, 0, 2, 2},
					},
					{
						Name:  "third.example.com.",
						Type:  1,
						Class: 1,
						Data:  []byte{192, 0, 2, 3},
					},
				},
			},
			checkBytes: func(data []byte) bool {
				if len(data) < 12 {
					return false
				}
				// Check ID
				if binary.BigEndian.Uint16(data[0:2]) != 54321 {
					return false
				}
				// Check ANCOUNT = 3
				if binary.BigEndian.Uint16(data[6:8]) != 3 {
					return false
				}
				return true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := codec.EncodeResponse(tt.response)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.checkBytes != nil {
					assert.True(t, tt.checkBytes(result), "encoded bytes validation failed")
				}
			}
		})
	}
}

func TestUdpCodec_DecodeResponse(t *testing.T) {
	codec := &udpCodec{
		logger: log.NewNoopLogger(),
	}
	timeFixture := time.Date(2099, 8, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		data       []byte
		expectedID uint16
		wantErr    string
		checkResp  func(domain.DNSResponse) bool
	}{
		{
			name: "valid response",
			data: func() []byte {
				// Create a valid DNS response packet
				data := make([]byte, 0, 512)

				// Header
				data = binary.BigEndian.AppendUint16(data, 12345)  // ID
				data = binary.BigEndian.AppendUint16(data, 0x8180) // Flags: response
				data = binary.BigEndian.AppendUint16(data, 1)      // QDCOUNT
				data = binary.BigEndian.AppendUint16(data, 1)      // ANCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // NSCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // ARCOUNT

				// Question: example.com A IN
				data = append(data, 7) // length of "example"
				data = append(data, []byte("example")...)
				data = append(data, 3) // length of "com"
				data = append(data, []byte("com")...)
				data = append(data, 0)                        // end of name
				data = binary.BigEndian.AppendUint16(data, 1) // QTYPE = A
				data = binary.BigEndian.AppendUint16(data, 1) // QCLASS = IN

				// Answer: example.com A IN 300 192.0.2.1
				data = append(data, 7) // length of "example"
				data = append(data, []byte("example")...)
				data = append(data, 3) // length of "com"
				data = append(data, []byte("com")...)
				data = append(data, 0)                          // end of name
				data = binary.BigEndian.AppendUint16(data, 1)   // TYPE = A
				data = binary.BigEndian.AppendUint16(data, 1)   // CLASS = IN
				data = binary.BigEndian.AppendUint32(data, 300) // TTL
				data = binary.BigEndian.AppendUint16(data, 4)   // RDLENGTH
				data = append(data, 192, 0, 2, 1)               // RDATA: IP address

				return data
			}(),
			expectedID: 12345,
			checkResp: func(resp domain.DNSResponse) bool {
				return resp.ID == 12345 && len(resp.Answers) == 1 &&
					resp.Answers[0].Name == "example.com." &&
					resp.Answers[0].Type == 1
			},
		},
		{
			name:       "too short",
			data:       []byte{1, 2, 3, 4, 5},
			expectedID: 1,
			wantErr:    "response too short",
		},
		{
			name: "ID mismatch",
			data: func() []byte {
				data := make([]byte, 12)
				binary.BigEndian.PutUint16(data[0:2], 999) // Wrong ID
				return data
			}(),
			expectedID: 12345,
			wantErr:    "ID mismatch",
		},
		{
			name: "truncated question name",
			data: func() []byte {
				data := make([]byte, 0, 512)
				data = binary.BigEndian.AppendUint16(data, 12345)  // ID
				data = binary.BigEndian.AppendUint16(data, 0x8180) // Flags
				data = binary.BigEndian.AppendUint16(data, 1)      // QDCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // ANCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // NSCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // ARCOUNT
				// Truncated question - only label length without data
				data = append(data, 10) // label length but no data follows
				return data
			}(),
			expectedID: 12345,
			wantErr:    "truncated question name",
		},
		{
			name: "truncated answer section",
			data: func() []byte {
				data := make([]byte, 0, 512)
				data = binary.BigEndian.AppendUint16(data, 12345)  // ID
				data = binary.BigEndian.AppendUint16(data, 0x8180) // Flags
				data = binary.BigEndian.AppendUint16(data, 0)      // QDCOUNT
				data = binary.BigEndian.AppendUint16(data, 1)      // ANCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // NSCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // ARCOUNT
				// Answer section starts but is truncated
				data = append(data, 0) // empty name
				// Missing TYPE, CLASS, TTL, RDLENGTH, RDATA
				return data
			}(),
			expectedID: 12345,
			wantErr:    "truncated record section",
		},
		{
			name: "failed to decode answer name",
			data: func() []byte {
				data := make([]byte, 0, 512)
				data = binary.BigEndian.AppendUint16(data, 12345)  // ID
				data = binary.BigEndian.AppendUint16(data, 0x8180) // Flags
				data = binary.BigEndian.AppendUint16(data, 0)      // QDCOUNT
				data = binary.BigEndian.AppendUint16(data, 1)      // ANCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // NSCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // ARCOUNT
				// Answer with invalid name that will fail decodeName
				// But first ensure we have enough bytes for the initial 10-byte check
				data = append(data, 0xC0, 0xFF) // compression pointer to invalid offset (way out of bounds)
				// Add 8 more bytes to satisfy the offset+10 check
				for i := 0; i < 8; i++ {
					data = append(data, 0)
				}
				return data
			}(),
			expectedID: 12345,
			wantErr:    "failed to decode record name",
		},
		{
			name: "truncated answer section after name",
			data: func() []byte {
				data := make([]byte, 0, 512)
				data = binary.BigEndian.AppendUint16(data, 12345)  // ID
				data = binary.BigEndian.AppendUint16(data, 0x8180) // Flags
				data = binary.BigEndian.AppendUint16(data, 0)      // QDCOUNT
				data = binary.BigEndian.AppendUint16(data, 1)      // ANCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // NSCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // ARCOUNT
				// Valid name that decodes successfully
				data = append(data, 0) // empty name at offset 12, decodeName returns newOffset=13
				// Add exactly 9 bytes after the name to get total length 22
				// First check: offset(12) + 10 > len(22)? 22 > 22? No, passes.
				// After decodeName: newOffset=13, second check: 13 + 10 > 22? 23 > 22? Yes, fails.
				for i := 0; i < 9; i++ {
					data = append(data, 0)
				}
				return data // Total length: 12 (header) + 1 (name) + 9 (padding) = 22
			}(),
			expectedID: 12345,
			wantErr:    "truncated record section after name",
		},
		{
			name: "truncated rdata",
			data: func() []byte {
				data := make([]byte, 0, 512)
				data = binary.BigEndian.AppendUint16(data, 12345)  // ID
				data = binary.BigEndian.AppendUint16(data, 0x8180) // Flags
				data = binary.BigEndian.AppendUint16(data, 0)      // QDCOUNT
				data = binary.BigEndian.AppendUint16(data, 1)      // ANCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // NSCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // ARCOUNT
				// Answer with valid header but truncated RDATA
				data = append(data, 0)                          // empty name
				data = binary.BigEndian.AppendUint16(data, 1)   // TYPE = A
				data = binary.BigEndian.AppendUint16(data, 1)   // CLASS = IN
				data = binary.BigEndian.AppendUint32(data, 300) // TTL
				data = binary.BigEndian.AppendUint16(data, 4)   // RDLENGTH = 4 (expects 4 bytes)
				// Only provide 2 bytes instead of 4
				data = append(data, 192, 0)
				return data
			}(),
			expectedID: 12345,
			wantErr:    "truncated rdata",
		},
		{
			name: "invalid resource record",
			data: func() []byte {
				data := make([]byte, 0, 512)
				data = binary.BigEndian.AppendUint16(data, 12345)  // ID
				data = binary.BigEndian.AppendUint16(data, 0x8180) // Flags
				data = binary.BigEndian.AppendUint16(data, 0)      // QDCOUNT
				data = binary.BigEndian.AppendUint16(data, 1)      // ANCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // NSCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // ARCOUNT
				// Answer with invalid resource record (e.g., invalid RRType)
				data = append(data, 0)                          // empty name
				data = binary.BigEndian.AppendUint16(data, 999) // Invalid TYPE
				data = binary.BigEndian.AppendUint16(data, 1)   // CLASS = IN
				data = binary.BigEndian.AppendUint32(data, 300) // TTL
				data = binary.BigEndian.AppendUint16(data, 4)   // RDLENGTH
				data = append(data, 192, 0, 2, 1)               // RDATA
				return data
			}(),
			expectedID: 12345,
			wantErr:    "invalid resource record",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := codec.DecodeResponse(tt.data, tt.expectedID, timeFixture)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
				if tt.checkResp != nil {
					assert.True(t, tt.checkResp(result), "response validation failed")
				}
			}
		})
	}
}

func TestUdpCodec_DecodeResponse_AuthorityRecords(t *testing.T) {
	codec := &udpCodec{
		logger: log.NewNoopLogger(),
	}
	timeFixture := time.Unix(1234567890, 0)

	tests := []struct {
		name       string
		data       []byte
		expectedID uint16
		checkResp  func(domain.DNSResponse) bool
		wantErr    string
	}{
		{
			name: "valid response with authority records",
			data: func() []byte {
				data := make([]byte, 0, 200)
				// Header: ID, flags, QDCOUNT=1, ANCOUNT=0, NSCOUNT=1, ARCOUNT=0
				data = binary.BigEndian.AppendUint16(data, 12345)  // ID
				data = binary.BigEndian.AppendUint16(data, 0x8400) // flags: response, authoritative, no error
				data = binary.BigEndian.AppendUint16(data, 1)      // QDCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // ANCOUNT
				data = binary.BigEndian.AppendUint16(data, 1)      // NSCOUNT (1 authority record)
				data = binary.BigEndian.AppendUint16(data, 0)      // ARCOUNT

				// Question: example.com A
				data = append(data, 7) // length of "example"
				data = append(data, []byte("example")...)
				data = append(data, 3) // length of "com"
				data = append(data, []byte("com")...)
				data = append(data, 0)                        // end of name
				data = binary.BigEndian.AppendUint16(data, 1) // QTYPE A
				data = binary.BigEndian.AppendUint16(data, 1) // QCLASS IN

				// Authority record: example.com SOA
				data = append(data, 7) // length of "example"
				data = append(data, []byte("example")...)
				data = append(data, 3) // length of "com"
				data = append(data, []byte("com")...)
				data = append(data, 0)                           // end of name
				data = binary.BigEndian.AppendUint16(data, 6)    // TYPE SOA
				data = binary.BigEndian.AppendUint16(data, 1)    // CLASS IN
				data = binary.BigEndian.AppendUint32(data, 3600) // TTL
				data = binary.BigEndian.AppendUint16(data, 4)    // RDLENGTH 4
				data = append(data, 192, 0, 2, 1)                // dummy SOA data

				return data
			}(),
			expectedID: 12345,
			checkResp: func(r domain.DNSResponse) bool {
				return r.RCode == domain.NOERROR &&
					len(r.Answers) == 0 &&
					len(r.Authority) == 1
			},
		},
		{
			name: "response with multiple authority records",
			data: func() []byte {
				data := make([]byte, 0, 300)
				// Header: ID, flags, QDCOUNT=1, ANCOUNT=0, NSCOUNT=2, ARCOUNT=0
				data = binary.BigEndian.AppendUint16(data, 12345)  // ID
				data = binary.BigEndian.AppendUint16(data, 0x8403) // flags: response, authoritative, NXDOMAIN
				data = binary.BigEndian.AppendUint16(data, 1)      // QDCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // ANCOUNT
				data = binary.BigEndian.AppendUint16(data, 2)      // NSCOUNT (2 authority records)
				data = binary.BigEndian.AppendUint16(data, 0)      // ARCOUNT

				// Question: missing.example.com A
				data = append(data, 7) // length of "missing"
				data = append(data, []byte("missing")...)
				data = append(data, 7) // length of "example"
				data = append(data, []byte("example")...)
				data = append(data, 3) // length of "com"
				data = append(data, []byte("com")...)
				data = append(data, 0)                        // end of name
				data = binary.BigEndian.AppendUint16(data, 1) // QTYPE A
				data = binary.BigEndian.AppendUint16(data, 1) // QCLASS IN

				// First authority record: example.com SOA
				data = append(data, 7) // length of "example"
				data = append(data, []byte("example")...)
				data = append(data, 3) // length of "com"
				data = append(data, []byte("com")...)
				data = append(data, 0)                           // end of name
				data = binary.BigEndian.AppendUint16(data, 6)    // TYPE SOA
				data = binary.BigEndian.AppendUint16(data, 1)    // CLASS IN
				data = binary.BigEndian.AppendUint32(data, 3600) // TTL
				data = binary.BigEndian.AppendUint16(data, 4)    // RDLENGTH 4
				data = append(data, 192, 0, 2, 1)                // dummy SOA data

				// Second authority record: example.com NS
				// Use compression pointer to "example.com" from question section (offset 12+8=20)
				data = append(data, 192, 20)                     // compression pointer to "example.com"
				data = binary.BigEndian.AppendUint16(data, 2)    // TYPE NS
				data = binary.BigEndian.AppendUint16(data, 1)    // CLASS IN
				data = binary.BigEndian.AppendUint32(data, 3600) // TTL
				data = binary.BigEndian.AppendUint16(data, 4)    // RDLENGTH 4
				data = append(data, 192, 0, 2, 2)                // dummy NS data

				return data
			}(),
			expectedID: 12345,
			checkResp: func(r domain.DNSResponse) bool {
				return r.RCode == domain.NXDOMAIN &&
					len(r.Answers) == 0 &&
					len(r.Authority) == 2
			},
		},
		{
			name: "authority record parsing error",
			data: func() []byte {
				data := make([]byte, 0, 100)
				// Header: ID, flags, QDCOUNT=1, ANCOUNT=0, NSCOUNT=1, ARCOUNT=0
				data = binary.BigEndian.AppendUint16(data, 12345)  // ID
				data = binary.BigEndian.AppendUint16(data, 0x8400) // flags: response, authoritative, no error
				data = binary.BigEndian.AppendUint16(data, 1)      // QDCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // ANCOUNT
				data = binary.BigEndian.AppendUint16(data, 1)      // NSCOUNT (1 authority record)
				data = binary.BigEndian.AppendUint16(data, 0)      // ARCOUNT

				// Question: example.com A
				data = append(data, 7) // length of "example"
				data = append(data, []byte("example")...)
				data = append(data, 3) // length of "com"
				data = append(data, []byte("com")...)
				data = append(data, 0)                        // end of name
				data = binary.BigEndian.AppendUint16(data, 1) // QTYPE A
				data = binary.BigEndian.AppendUint16(data, 1) // QCLASS IN

				// Malformed authority record (truncated after name)
				data = append(data, 7) // length of "example"
				data = append(data, []byte("example")...)
				data = append(data, 3) // length of "com"
				data = append(data, []byte("com")...)
				data = append(data, 0) // end of name
				// Missing TYPE, CLASS, TTL, RDLENGTH, RDATA

				return data
			}(),
			expectedID: 12345,
			wantErr:    "truncated record section after name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := codec.DecodeResponse(tt.data, tt.expectedID, timeFixture)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
				if tt.checkResp != nil {
					assert.True(t, tt.checkResp(result), "response validation failed")
				}
			}
		})
	}
}

func TestUdpCodec_DecodeResponse_AdditionalRecords(t *testing.T) {
	codec := &udpCodec{
		logger: log.NewNoopLogger(),
	}
	timeFixture := time.Unix(1234567890, 0)

	tests := []struct {
		name       string
		data       []byte
		expectedID uint16
		checkResp  func(domain.DNSResponse) bool
		wantErr    string
	}{
		{
			name: "valid response with additional records",
			data: func() []byte {
				data := make([]byte, 0, 200)
				// Header: ID, flags, QDCOUNT=1, ANCOUNT=1, NSCOUNT=0, ARCOUNT=1
				data = binary.BigEndian.AppendUint16(data, 12345)  // ID
				data = binary.BigEndian.AppendUint16(data, 0x8400) // flags: response, authoritative, no error
				data = binary.BigEndian.AppendUint16(data, 1)      // QDCOUNT
				data = binary.BigEndian.AppendUint16(data, 1)      // ANCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // NSCOUNT
				data = binary.BigEndian.AppendUint16(data, 1)      // ARCOUNT (1 additional record)

				// Question: example.com MX
				data = append(data, 7) // length of "example"
				data = append(data, []byte("example")...)
				data = append(data, 3) // length of "com"
				data = append(data, []byte("com")...)
				data = append(data, 0)                         // end of name
				data = binary.BigEndian.AppendUint16(data, 15) // QTYPE MX
				data = binary.BigEndian.AppendUint16(data, 1)  // QCLASS IN

				// Answer record: example.com MX
				data = append(data, 7) // length of "example"
				data = append(data, []byte("example")...)
				data = append(data, 3) // length of "com"
				data = append(data, []byte("com")...)
				data = append(data, 0)                           // end of name
				data = binary.BigEndian.AppendUint16(data, 15)   // TYPE MX
				data = binary.BigEndian.AppendUint16(data, 1)    // CLASS IN
				data = binary.BigEndian.AppendUint32(data, 3600) // TTL
				data = binary.BigEndian.AppendUint16(data, 4)    // RDLENGTH 4
				data = append(data, 192, 0, 2, 10)               // dummy MX data

				// Additional record: mail.example.com A
				data = append(data, 4) // length of "mail"
				data = append(data, []byte("mail")...)
				data = append(data, 7) // length of "example"
				data = append(data, []byte("example")...)
				data = append(data, 3) // length of "com"
				data = append(data, []byte("com")...)
				data = append(data, 0)                           // end of name
				data = binary.BigEndian.AppendUint16(data, 1)    // TYPE A
				data = binary.BigEndian.AppendUint16(data, 1)    // CLASS IN
				data = binary.BigEndian.AppendUint32(data, 3600) // TTL
				data = binary.BigEndian.AppendUint16(data, 4)    // RDLENGTH 4
				data = append(data, 192, 0, 2, 100)              // IP address 192.0.2.100

				return data
			}(),
			expectedID: 12345,
			checkResp: func(r domain.DNSResponse) bool {
				return r.RCode == domain.NOERROR &&
					len(r.Answers) == 1 &&
					len(r.Additional) == 1
			},
		},
		{
			name: "response with multiple additional records",
			data: func() []byte {
				data := make([]byte, 0, 300)
				// Header: ID, flags, QDCOUNT=1, ANCOUNT=0, NSCOUNT=1, ARCOUNT=2
				data = binary.BigEndian.AppendUint16(data, 12345)  // ID
				data = binary.BigEndian.AppendUint16(data, 0x8400) // flags: response, authoritative, no error
				data = binary.BigEndian.AppendUint16(data, 1)      // QDCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // ANCOUNT
				data = binary.BigEndian.AppendUint16(data, 1)      // NSCOUNT
				data = binary.BigEndian.AppendUint16(data, 2)      // ARCOUNT (2 additional records)

				// Question: example.com NS
				data = append(data, 7) // length of "example"
				data = append(data, []byte("example")...)
				data = append(data, 3) // length of "com"
				data = append(data, []byte("com")...)
				data = append(data, 0)                        // end of name
				data = binary.BigEndian.AppendUint16(data, 2) // QTYPE NS
				data = binary.BigEndian.AppendUint16(data, 1) // QCLASS IN

				// Authority record: example.com NS ns1.example.com
				data = append(data, 192, 12)                     // compression pointer to "example.com"
				data = binary.BigEndian.AppendUint16(data, 2)    // TYPE NS
				data = binary.BigEndian.AppendUint16(data, 1)    // CLASS IN
				data = binary.BigEndian.AppendUint32(data, 3600) // TTL
				data = binary.BigEndian.AppendUint16(data, 4)    // RDLENGTH 4
				data = append(data, 192, 0, 2, 1)                // dummy NS data

				// First additional record: ns1.example.com A
				data = append(data, 3) // length of "ns1"
				data = append(data, []byte("ns1")...)
				data = append(data, 192, 12)                     // compression pointer to "example.com"
				data = binary.BigEndian.AppendUint16(data, 1)    // TYPE A
				data = binary.BigEndian.AppendUint16(data, 1)    // CLASS IN
				data = binary.BigEndian.AppendUint32(data, 3600) // TTL
				data = binary.BigEndian.AppendUint16(data, 4)    // RDLENGTH 4
				data = append(data, 192, 0, 2, 1)                // IP address 192.0.2.1

				// Second additional record: ns2.example.com A
				data = append(data, 3) // length of "ns2"
				data = append(data, []byte("ns2")...)
				data = append(data, 192, 12)                     // compression pointer to "example.com"
				data = binary.BigEndian.AppendUint16(data, 1)    // TYPE A
				data = binary.BigEndian.AppendUint16(data, 1)    // CLASS IN
				data = binary.BigEndian.AppendUint32(data, 3600) // TTL
				data = binary.BigEndian.AppendUint16(data, 4)    // RDLENGTH 4
				data = append(data, 192, 0, 2, 2)                // IP address 192.0.2.2

				return data
			}(),
			expectedID: 12345,
			checkResp: func(r domain.DNSResponse) bool {
				return r.RCode == domain.NOERROR &&
					len(r.Answers) == 0 &&
					len(r.Authority) == 1 &&
					len(r.Additional) == 2
			},
		},
		{
			name: "additional record parsing error",
			data: func() []byte {
				data := make([]byte, 0, 100)
				// Header: ID, flags, QDCOUNT=1, ANCOUNT=0, NSCOUNT=0, ARCOUNT=1
				data = binary.BigEndian.AppendUint16(data, 12345)  // ID
				data = binary.BigEndian.AppendUint16(data, 0x8400) // flags: response, authoritative, no error
				data = binary.BigEndian.AppendUint16(data, 1)      // QDCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // ANCOUNT
				data = binary.BigEndian.AppendUint16(data, 0)      // NSCOUNT
				data = binary.BigEndian.AppendUint16(data, 1)      // ARCOUNT (1 additional record)

				// Question: example.com A
				data = append(data, 7) // length of "example"
				data = append(data, []byte("example")...)
				data = append(data, 3) // length of "com"
				data = append(data, []byte("com")...)
				data = append(data, 0)                        // end of name
				data = binary.BigEndian.AppendUint16(data, 1) // QTYPE A
				data = binary.BigEndian.AppendUint16(data, 1) // QCLASS IN

				// Malformed additional record (truncated after name)
				data = append(data, 7) // length of "example"
				data = append(data, []byte("example")...)
				data = append(data, 3) // length of "com"
				data = append(data, []byte("com")...)
				data = append(data, 0) // end of name
				// Missing TYPE, CLASS, TTL, RDLENGTH, RDATA

				return data
			}(),
			expectedID: 12345,
			wantErr:    "truncated record section after name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := codec.DecodeResponse(tt.data, tt.expectedID, timeFixture)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
				if tt.checkResp != nil {
					assert.True(t, tt.checkResp(result), "response validation failed")
				}
			}
		})
	}
}

func TestDecodeName(t *testing.T) {
	tests := []struct {
		name       string
		data       []byte
		offset     int
		wantName   string
		wantOffset int
		wantErr    string
	}{
		{
			name: "simple name",
			data: func() []byte {
				data := make([]byte, 0, 100)
				data = append(data, 7) // length of "example"
				data = append(data, []byte("example")...)
				data = append(data, 3) // length of "com"
				data = append(data, []byte("com")...)
				data = append(data, 0) // end of name
				return data
			}(),
			offset:     0,
			wantName:   "example.com",
			wantOffset: 13, // 1 + 7 + 1 + 3 + 1
		},
		{
			name:       "empty name",
			data:       []byte{0},
			offset:     0,
			wantName:   "",
			wantOffset: 1,
		},
		{
			name: "name with compression",
			data: func() []byte {
				data := make([]byte, 0, 100)
				// First occurrence: "example.com"
				data = append(data, 7) // length of "example"
				data = append(data, []byte("example")...)
				data = append(data, 3) // length of "com"
				data = append(data, []byte("com")...)
				data = append(data, 0) // end of name
				// Compressed pointer: "www" + pointer to offset 0
				data = append(data, 3) // length of "www"
				data = append(data, []byte("www")...)
				data = append(data, 0xC0, 0x00) // compression pointer to offset 0
				return data
			}(),
			offset:     13, // Start at "www" label
			wantName:   "www.example.com",
			wantOffset: 19, // 13 + 1 + 3 + 2
		},
		{
			name:    "offset out of bounds",
			data:    []byte{1, 2, 3},
			offset:  10,
			wantErr: "offset out of bounds",
		},
		{
			name:    "label length out of bounds",
			data:    []byte{10, 1, 2, 3}, // label length 10 but only 3 bytes follow
			offset:  0,
			wantErr: "label length out of bounds",
		},
		{
			name:    "compression pointer out of bounds",
			data:    []byte{0xC0}, // compression marker but missing second byte
			offset:  0,
			wantErr: "compression pointer out of bounds",
		},
		{
			name: "compression pointer to invalid data",
			data: func() []byte {
				data := make([]byte, 0, 100)
				// Create invalid data at offset 0: label length that exceeds available data
				data = append(data, 50)                 // label length 50 but we won't provide that much data
				data = append(data, []byte("short")...) // only 5 bytes instead of 50
				// Pad with zeros to ensure we have enough space
				for len(data) < 20 {
					data = append(data, 0)
				}
				// Now at offset 20, create a valid label with compression pointer to invalid data
				data = append(data, 3) // length of "www"
				data = append(data, []byte("www")...)
				data = append(data, 0xC0, 0x00) // compression pointer to offset 0 (invalid label length)
				return data
			}(),
			offset:  20,                           // Start at "www" label which has compression pointer to invalid data
			wantErr: "label length out of bounds", // This is the error from the recursive call
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, offset, err := decodeName(tt.data, tt.offset)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantName, name)
				assert.Equal(t, tt.wantOffset, offset)
			}
		})
	}
}

func TestEncodeDomainName(t *testing.T) {
	tests := []struct {
		name       string
		domain     string
		wantErr    string
		checkBytes func([]byte) bool
	}{
		{
			name:   "simple domain",
			domain: "example.com",
			checkBytes: func(data []byte) bool {
				// Should be: 7 + "example" + 3 + "com" + 0
				expected := []byte{7}
				expected = append(expected, []byte("example")...)
				expected = append(expected, 3)
				expected = append(expected, []byte("com")...)
				expected = append(expected, 0)
				return len(data) == len(expected) && string(data) == string(expected)
			},
		},
		{
			name:   "empty domain",
			domain: "",
			checkBytes: func(data []byte) bool {
				return len(data) == 1 && data[0] == 0
			},
		},
		{
			name:    "label too long",
			domain:  "this-is-a-very-long-label-that-exceeds-the-maximum-allowed-length-of-63-characters-for-dns-labels.com",
			wantErr: "label too long",
		},
		{
			name:   "single label",
			domain: "localhost",
			checkBytes: func(data []byte) bool {
				// Should be: 9 + "localhost" + 0
				return len(data) == 11 && data[0] == 9 && data[10] == 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := encodeDomainName(tt.domain)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.checkBytes != nil {
					assert.True(t, tt.checkBytes(result), "encoded bytes validation failed")
				}
			}
		})
	}
}
func TestNewUDPCodec(t *testing.T) {
	t.Run("returns non-nil codec with provided logger", func(t *testing.T) {
		logger := log.NewNoopLogger()
		codec := NewUDPCodec(logger)
		assert.NotNil(t, codec)
		assert.Equal(t, logger, codec.logger)
	})

	t.Run("returns distinct instances for different loggers", func(t *testing.T) {
		logger1 := log.NewNoopLogger()
		logger2 := log.NewNoopLogger()
		codec1 := NewUDPCodec(logger1)
		codec2 := NewUDPCodec(logger2)
		assert.NotSame(t, codec1, codec2)
		assert.NotSame(t, codec1.logger, codec2.logger)
	})
}
