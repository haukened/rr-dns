package transport

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/haukened/rr-dns/internal/dns/common/log"
)

func TestNewTransport(t *testing.T) {
	logger := log.NewNoopLogger()
	codec := &MockDNSCodec{}

	tests := []struct {
		name          string
		transportType TransportType
		addr          string
		wantErr       bool
		errContains   string
	}{
		{
			name:          "UDP transport success",
			transportType: TransportUDP,
			addr:          "127.0.0.1:0",
			wantErr:       false,
		},
		{
			name:          "DoH transport not implemented",
			transportType: TransportDoH,
			addr:          "127.0.0.1:443",
			wantErr:       true,
			errContains:   "DNS over HTTPS transport not yet implemented",
		},
		{
			name:          "DoT transport not implemented",
			transportType: TransportDoT,
			addr:          "127.0.0.1:853",
			wantErr:       true,
			errContains:   "DNS over TLS transport not yet implemented",
		},
		{
			name:          "DoQ transport not implemented",
			transportType: TransportDoQ,
			addr:          "127.0.0.1:853",
			wantErr:       true,
			errContains:   "DNS over QUIC transport not yet implemented",
		},
		{
			name:          "unsupported transport type",
			transportType: TransportType("unknown"),
			addr:          "127.0.0.1:53",
			wantErr:       true,
			errContains:   "unsupported transport type: unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport, err := NewTransport(tt.transportType, tt.addr, codec, logger)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, transport)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, transport)
				assert.Equal(t, tt.addr, transport.Address())
			}
		})
	}
}

func TestGetSupportedTransports(t *testing.T) {
	supported := GetSupportedTransports()

	assert.NotEmpty(t, supported)
	assert.Contains(t, supported, TransportUDP)

	// Verify it returns a new slice each time (not a shared reference)
	supported1 := GetSupportedTransports()
	supported2 := GetSupportedTransports()

	// Modify one slice
	if len(supported1) > 0 {
		supported1[0] = TransportType("modified")
	}

	// Other slice should be unchanged
	assert.NotEqual(t, supported1[0], supported2[0])
}

func TestIsTransportSupported(t *testing.T) {
	tests := []struct {
		name          string
		transportType TransportType
		expected      bool
	}{
		{
			name:          "UDP is supported",
			transportType: TransportUDP,
			expected:      true,
		},
		{
			name:          "DoH is not supported yet",
			transportType: TransportDoH,
			expected:      false,
		},
		{
			name:          "DoT is not supported yet",
			transportType: TransportDoT,
			expected:      false,
		},
		{
			name:          "DoQ is not supported yet",
			transportType: TransportDoQ,
			expected:      false,
		},
		{
			name:          "unknown transport is not supported",
			transportType: TransportType("unknown"),
			expected:      false,
		},
		{
			name:          "empty transport type is not supported",
			transportType: TransportType(""),
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTransportSupported(tt.transportType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransportConstants(t *testing.T) {
	// Verify transport type constants are defined correctly
	assert.Equal(t, TransportType("udp"), TransportUDP)
	assert.Equal(t, TransportType("doh"), TransportDoH)
	assert.Equal(t, TransportType("dot"), TransportDoT)
	assert.Equal(t, TransportType("doq"), TransportDoQ)
}

func TestServerTransportInterface(t *testing.T) {
	// Verify UDPTransport implements ServerTransport interface
	logger := log.NewNoopLogger()
	codec := &MockDNSCodec{}

	transport := NewUDPTransport("127.0.0.1:0", codec, logger)

	// Type assertion to verify interface implementation
	var _ ServerTransport = transport

	// Verify interface methods exist and return correct types
	require.NotNil(t, transport.Start)
	require.NotNil(t, transport.Stop)
	require.NotNil(t, transport.Address)

	addr := transport.Address()
	assert.IsType(t, "", addr)
}
