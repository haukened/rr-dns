package wire

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeQuestion(t *testing.T) {
	tests := []struct {
		name       string
		data       []byte
		offset     int
		wantName   string
		wantType   uint16
		wantClass  uint16
		wantOffset int
		wantErr    string
	}{
		{
			name: "valid question",
			data: func() []byte {
				data := make([]byte, 0, 100)
				// Name: example.com
				data = append(data, 7) // length of "example"
				data = append(data, []byte("example")...)
				data = append(data, 3) // length of "com"
				data = append(data, []byte("com")...)
				data = append(data, 0) // end of name
				// QTYPE and QCLASS
				data = binary.BigEndian.AppendUint16(data, 1) // A record
				data = binary.BigEndian.AppendUint16(data, 1) // IN class
				return data
			}(),
			offset:     0,
			wantName:   "example.com",
			wantType:   1,
			wantClass:  1,
			wantOffset: 17, // 1+7+1+3+1+2+2
		},
		{
			name: "question with compression",
			data: func() []byte {
				data := make([]byte, 0, 100)
				// First name: example.com
				data = append(data, 7) // length of "example"
				data = append(data, []byte("example")...)
				data = append(data, 3) // length of "com"
				data = append(data, []byte("com")...)
				data = append(data, 0) // end of name
				// Padding to create offset
				data = append(data, 0, 0, 0, 0)
				// Second name with compression: www + pointer to offset 0
				data = append(data, 3) // length of "www"
				data = append(data, []byte("www")...)
				data = append(data, 0xC0, 0x00) // compression pointer to offset 0
				// QTYPE and QCLASS
				data = binary.BigEndian.AppendUint16(data, 28) // AAAA record
				data = binary.BigEndian.AppendUint16(data, 1)  // IN class
				return data
			}(),
			offset:     17, // Start at the "www" label
			wantName:   "www.example.com",
			wantType:   28,
			wantClass:  1,
			wantOffset: 27, // 17 + 1 + 3 + 2 + 2 + 2
		},
		{
			name: "empty name",
			data: func() []byte {
				data := make([]byte, 0, 10)
				data = append(data, 0)                        // empty name
				data = binary.BigEndian.AppendUint16(data, 1) // A record
				data = binary.BigEndian.AppendUint16(data, 1) // IN class
				return data
			}(),
			offset:     0,
			wantName:   "",
			wantType:   1,
			wantClass:  1,
			wantOffset: 5, // 1 + 2 + 2
		},
		{
			name: "truncated question fields",
			data: func() []byte {
				data := make([]byte, 0, 10)
				data = append(data, 0)                        // empty name
				data = binary.BigEndian.AppendUint16(data, 1) // A record
				// Missing QCLASS
				return data
			}(),
			offset:  0,
			wantErr: "truncated question fields",
		},
		{
			name: "invalid name encoding",
			data: func() []byte {
				data := make([]byte, 0, 10)
				data = append(data, 10) // label length 10 but no data follows
				return data
			}(),
			offset:  0,
			wantErr: "label length out of bounds",
		},
		{
			name: "question at non-zero offset",
			data: func() []byte {
				data := make([]byte, 0, 100)
				// Some header bytes
				data = append(data, 0, 0, 0, 0, 0)
				// Name at offset 5: test.example
				data = append(data, 4) // length of "test"
				data = append(data, []byte("test")...)
				data = append(data, 7) // length of "example"
				data = append(data, []byte("example")...)
				data = append(data, 0) // end of name
				// QTYPE and QCLASS
				data = binary.BigEndian.AppendUint16(data, 15) // MX record
				data = binary.BigEndian.AppendUint16(data, 1)  // IN class
				return data
			}(),
			offset:     5,
			wantName:   "test.example",
			wantType:   15,
			wantClass:  1,
			wantOffset: 23, // 5 + 1 + 4 + 1 + 7 + 1 + 2 + 2
		},
		{
			name: "compression pointer recursion",
			data: func() []byte {
				data := make([]byte, 0, 100)
				// First name: example.com
				data = append(data, 7) // length of "example"
				data = append(data, []byte("example")...)
				data = append(data, 3) // length of "com"
				data = append(data, []byte("com")...)
				data = append(data, 0) // end of name
				// Second name: subdomain.example.com using compression
				data = append(data, 9) // length of "subdomain"
				data = append(data, []byte("subdomain")...)
				data = append(data, 0xC0, 0x00) // pointer to "example.com" at offset 0
				// QTYPE and QCLASS for second name
				data = binary.BigEndian.AppendUint16(data, 1) // A record
				data = binary.BigEndian.AppendUint16(data, 1) // IN class
				return data
			}(),
			offset:     13, // Start at "subdomain" label
			wantName:   "subdomain.example.com",
			wantType:   1,
			wantClass:  1,
			wantOffset: 29, // 13 + 1 + 9 + 2 + 2 + 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, qtype, qclass, offset, err := decodeQuestion(tt.data, tt.offset)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantName, name)
				assert.Equal(t, tt.wantType, qtype)
				assert.Equal(t, tt.wantClass, qclass)
				assert.Equal(t, tt.wantOffset, offset)
			}
		})
	}
}

func TestDecodeQuestion_EdgeCases(t *testing.T) {
	t.Run("question at end of data", func(t *testing.T) {
		data := func() []byte {
			data := make([]byte, 0, 10)
			data = append(data, 0)                        // empty name
			data = binary.BigEndian.AppendUint16(data, 1) // A record
			data = binary.BigEndian.AppendUint16(data, 1) // IN class
			return data
		}()

		name, qtype, qclass, offset, err := decodeQuestion(data, 0)
		assert.NoError(t, err)
		assert.Equal(t, "", name)
		assert.Equal(t, uint16(1), qtype)
		assert.Equal(t, uint16(1), qclass)
		assert.Equal(t, len(data), offset)
	})

	t.Run("complex compression scenario", func(t *testing.T) {
		// Test a more complex compression scenario with multiple levels
		data := func() []byte {
			data := make([]byte, 0, 100)
			// Name 1: com (at offset 0)
			data = append(data, 3) // length of "com"
			data = append(data, []byte("com")...)
			data = append(data, 0) // end of name

			// Name 2: example.com (at offset 5)
			data = append(data, 7) // length of "example"
			data = append(data, []byte("example")...)
			data = append(data, 0xC0, 0x00) // pointer to "com" at offset 0

			// Name 3: www.example.com (using pointer to name 2)
			data = append(data, 3) // length of "www"
			data = append(data, []byte("www")...)
			data = append(data, 0xC0, 0x05) // pointer to "example.com" at offset 5

			// QTYPE and QCLASS for name 3
			data = binary.BigEndian.AppendUint16(data, 1) // A record
			data = binary.BigEndian.AppendUint16(data, 1) // IN class

			return data
		}()

		name, qtype, qclass, offset, err := decodeQuestion(data, 15) // Start at "www" label
		assert.NoError(t, err)
		assert.Equal(t, "www.example.com", name)
		assert.Equal(t, uint16(1), qtype)
		assert.Equal(t, uint16(1), qclass)
		assert.Equal(t, 25, offset) // 15 + 1 + 3 + 2 + 2 + 2
	})

	t.Run("maximum label length", func(t *testing.T) {
		// Test with maximum allowed label length (63 characters)
		data := func() []byte {
			data := make([]byte, 0, 100)
			// Create a 63-character label
			longLabel := make([]byte, 63)
			for i := range longLabel {
				longLabel[i] = 'a'
			}
			data = append(data, 63) // length
			data = append(data, longLabel...)
			data = append(data, 0)                        // end of name
			data = binary.BigEndian.AppendUint16(data, 1) // A record
			data = binary.BigEndian.AppendUint16(data, 1) // IN class
			return data
		}()

		name, qtype, qclass, offset, err := decodeQuestion(data, 0)
		assert.NoError(t, err)
		expectedName := string(make([]byte, 63))
		for i := 0; i < 63; i++ {
			expectedName = expectedName[:i] + "a" + expectedName[i+1:]
		}
		assert.Equal(t, expectedName, name)
		assert.Equal(t, uint16(1), qtype)
		assert.Equal(t, uint16(1), qclass)
		assert.Equal(t, 69, offset) // 1 + 63 + 1 + 2 + 2
	})
}
