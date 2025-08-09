package transport

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/haukened/rr-dns/internal/dns/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDNSCodec implements wire.DNSCodec for testing
type MockDNSCodec struct {
	mock.Mock
}

func (m *MockDNSCodec) EncodeQuery(query domain.Question) ([]byte, error) {
	args := m.Called(query)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockDNSCodec) DecodeResponse(data []byte, expectedID uint16, now time.Time) (domain.DNSResponse, error) {
	args := m.Called(data, expectedID, now)
	return args.Get(0).(domain.DNSResponse), args.Error(1)
}

func (m *MockDNSCodec) DecodeQuery(data []byte) (domain.Question, error) {
	args := m.Called(data)
	return args.Get(0).(domain.Question), args.Error(1)
}

func (m *MockDNSCodec) EncodeResponse(resp domain.DNSResponse) ([]byte, error) {
	args := m.Called(resp)
	return args.Get(0).([]byte), args.Error(1)
}

// MockDNSResponder implements resolver.DNSResponder for testing
type MockDNSResponder struct {
	mock.Mock
}

func (m *MockDNSResponder) HandleQuery(ctx context.Context, query domain.Question, clientAddr net.Addr) (domain.DNSResponse, error) {
	args := m.Called(ctx, query, clientAddr)
	return args.Get(0).(domain.DNSResponse), args.Error(1)
}

// MockLogger implements log.Logger for testing
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Info(fields map[string]any, msg string) {
	m.Called(fields, msg)
}

func (m *MockLogger) Error(fields map[string]any, msg string) {
	m.Called(fields, msg)
}

func (m *MockLogger) Debug(fields map[string]any, msg string) {
	m.Called(fields, msg)
}

func (m *MockLogger) Warn(fields map[string]any, msg string) {
	m.Called(fields, msg)
}

func (m *MockLogger) Panic(fields map[string]any, msg string) {
	m.Called(fields, msg)
}

func (m *MockLogger) Fatal(fields map[string]any, msg string) {
	m.Called(fields, msg)
}

// testLogger provides a no-op logger for tests that don't need to verify logging
type testLogger struct{}

func (t *testLogger) Info(map[string]any, string)  {}
func (t *testLogger) Error(map[string]any, string) {}
func (t *testLogger) Debug(map[string]any, string) {}
func (t *testLogger) Warn(map[string]any, string)  {}
func (t *testLogger) Panic(map[string]any, string) {}
func (t *testLogger) Fatal(map[string]any, string) {}

func TestNewUDPTransport(t *testing.T) {
	codec := &MockDNSCodec{}
	logger := &testLogger{}
	addr := "127.0.0.1:5053"

	transport := NewUDPTransport(addr, codec, logger)

	assert.NotNil(t, transport)
	assert.Equal(t, addr, transport.addr)
	assert.Equal(t, codec, transport.codec)
	assert.Equal(t, logger, transport.logger)
	assert.NotNil(t, transport.stopCh)
	assert.False(t, transport.running)
}

func TestUDPTransport_Address(t *testing.T) {
	codec := &MockDNSCodec{}
	logger := &testLogger{}
	addr := "127.0.0.1:5053"

	transport := NewUDPTransport(addr, codec, logger)
	assert.Equal(t, addr, transport.Address())
}

func TestUDPTransport_StartStop(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid address",
			addr:    "127.0.0.1:0", // Let OS choose port
			wantErr: false,
		},
		{
			name:    "invalid address format",
			addr:    "invalid-address",
			wantErr: true,
			errMsg:  "failed to resolve UDP address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codec := &MockDNSCodec{}
			logger := &testLogger{}
			handler := &MockDNSResponder{}

			transport := NewUDPTransport(tt.addr, codec, logger)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			err := transport.Start(ctx, handler)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}

			require.NoError(t, err)
			assert.True(t, transport.running)
			assert.NotNil(t, transport.conn)

			// Test double start fails
			err = transport.Start(ctx, handler)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "already running")

			// Test stop
			err = transport.Stop()
			assert.NoError(t, err)
			assert.False(t, transport.running)
			// Note: conn is not set to nil, just closed

			// Test double stop is safe
			err = transport.Stop()
			assert.NoError(t, err)
		})
	}
}

func TestUDPTransport_QueryHandling(t *testing.T) {
	// Create mocks
	codec := &MockDNSCodec{}
	mockLogger := &MockLogger{}
	handler := &MockDNSResponder{}

	// Setup test data
	testQuery := domain.Question{
		ID:   12345,
		Name: "example.com.",
		Type: 1, // A record
	}

	testResponse := domain.DNSResponse{
		ID:    12345,
		RCode: 0, // NOERROR
		Answers: []domain.ResourceRecord{
			{
				Name:  "example.com.",
				Type:  1, // A record
				Class: 1, // IN
				Data:  []byte("1.2.3.4"),
			},
		},
	}

	queryData := []byte{0x01, 0x02, 0x03}    // Mock DNS query bytes
	responseData := []byte{0x04, 0x05, 0x06} // Mock DNS response bytes

	// Setup codec expectations
	codec.On("DecodeQuery", queryData).Return(testQuery, nil)
	codec.On("EncodeResponse", testResponse).Return(responseData, nil)

	// Setup handler expectations
	handler.On("HandleQuery", mock.AnythingOfType("*context.cancelCtx"), testQuery, mock.AnythingOfType("*net.UDPAddr")).Return(testResponse, nil)

	// Setup logger expectations to be flexible about what gets logged
	mockLogger.On("Info", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Debug", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Warn", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Error", mock.Anything, mock.Anything).Maybe()

	// Create transport and start it
	transport := NewUDPTransport("127.0.0.1:0", codec, mockLogger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(t, err)

	// Get the actual address the transport is bound to
	actualAddr := transport.conn.LocalAddr().(*net.UDPAddr)

	// Create a client connection to send test data
	clientConn, err := net.DialUDP("udp", nil, actualAddr)
	require.NoError(t, err)
	defer func() { require.NoError(t, clientConn.Close()) }()

	// Send test query
	_, err = clientConn.Write(queryData)
	require.NoError(t, err)

	// Read response
	responseBuffer := make([]byte, 512)
	err = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	require.NoError(t, err)
	n, err := clientConn.Read(responseBuffer)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, responseData, responseBuffer[:n])

	// Verify mock expectations
	codec.AssertExpectations(t)
	handler.AssertExpectations(t)

	err = transport.Stop()
	require.NoError(t, err)
}

func TestUDPTransport_CodecDecodeError(t *testing.T) {
	codec := &MockDNSCodec{}
	mockLogger := &MockLogger{}
	handler := &MockDNSResponder{}

	invalidData := []byte{0xFF, 0xFF, 0xFF}

	// Setup codec to return decode error
	codec.On("DecodeQuery", invalidData).Return(domain.Question{}, assert.AnError)

	// Expect warning to be logged
	mockLogger.On("Warn", mock.MatchedBy(func(fields map[string]any) bool {
		return fields["error"] != nil
	}), "Failed to decode DNS query")
	mockLogger.On("Info", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Debug", mock.Anything, mock.Anything).Maybe()

	transport := NewUDPTransport("127.0.0.1:0", codec, mockLogger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(t, err)

	actualAddr := transport.conn.LocalAddr().(*net.UDPAddr)

	clientConn, err := net.DialUDP("udp", nil, actualAddr)
	require.NoError(t, err)
	defer func() { require.NoError(t, clientConn.Close()) }()

	// Send invalid data
	_, err = clientConn.Write(invalidData)
	require.NoError(t, err)

	// Give some time for the packet to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify expectations
	codec.AssertExpectations(t)
	mockLogger.AssertExpectations(t)

	err = transport.Stop()
	require.NoError(t, err)
}

func TestUDPTransport_CodecEncodeError(t *testing.T) {
	codec := &MockDNSCodec{}
	mockLogger := &MockLogger{}
	handler := &MockDNSResponder{}

	testQuery := domain.Question{
		ID:   12345,
		Name: "example.com.",
		Type: 1, // A record
	}

	testResponse := domain.DNSResponse{
		ID:    12345,
		RCode: 0, // NOERROR
	}

	queryData := []byte{0x01, 0x02, 0x03}

	// Setup codec to decode successfully but fail to encode
	codec.On("DecodeQuery", queryData).Return(testQuery, nil)
	codec.On("EncodeResponse", testResponse).Return([]byte{}, assert.AnError)

	// Setup handler
	handler.On("HandleQuery", mock.AnythingOfType("*context.cancelCtx"), testQuery, mock.AnythingOfType("*net.UDPAddr")).Return(testResponse, nil)

	// Expect error to be logged
	mockLogger.On("Error", mock.MatchedBy(func(fields map[string]any) bool {
		return fields["error"] != nil && fields["query_id"] == uint16(12345)
	}), "Failed to encode DNS response")
	mockLogger.On("Info", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Debug", mock.Anything, mock.Anything).Maybe()

	transport := NewUDPTransport("127.0.0.1:0", codec, mockLogger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(t, err)

	actualAddr := transport.conn.LocalAddr().(*net.UDPAddr)

	clientConn, err := net.DialUDP("udp", nil, actualAddr)
	require.NoError(t, err)
	defer func() { require.NoError(t, clientConn.Close()) }()

	// Send test query
	_, err = clientConn.Write(queryData)
	require.NoError(t, err)

	// Give some time for the packet to be processed
	time.Sleep(100 * time.Millisecond)

	// Verify expectations
	codec.AssertExpectations(t)
	handler.AssertExpectations(t)
	mockLogger.AssertExpectations(t)

	err = transport.Stop()
	require.NoError(t, err)
}

func TestUDPTransport_ContextCancellation(t *testing.T) {
	codec := &MockDNSCodec{}
	logger := &testLogger{}
	handler := &MockDNSResponder{}

	transport := NewUDPTransport("127.0.0.1:0", codec, logger)
	ctx, cancel := context.WithCancel(context.Background())

	err := transport.Start(ctx, handler)
	require.NoError(t, err)

	// Wait a moment for the listen loop to start
	time.Sleep(10 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for graceful shutdown
	time.Sleep(100 * time.Millisecond)

	// Transport should still be marked as running since Stop() wasn't called
	transport.mu.RLock()
	running := transport.running
	transport.mu.RUnlock()
	assert.True(t, running)

	// Clean shutdown
	err = transport.Stop()
	assert.NoError(t, err)
}

func TestUDPTransport_ConcurrentRequests(t *testing.T) {
	codec := &MockDNSCodec{}
	logger := &testLogger{}
	handler := &MockDNSResponder{}

	// Setup test data
	testQuery := domain.Question{
		ID:   12345,
		Name: "example.com.",
		Type: 1, // A record
	}

	testResponse := domain.DNSResponse{
		ID:    12345,
		RCode: 0, // NOERROR
	}

	queryData := []byte{0x01, 0x02, 0x03}
	responseData := []byte{0x04, 0x05, 0x06}

	// Setup mocks to handle multiple calls
	codec.On("DecodeQuery", queryData).Return(testQuery, nil).Maybe()
	codec.On("EncodeResponse", testResponse).Return(responseData, nil).Maybe()
	handler.On("HandleQuery", mock.AnythingOfType("*context.cancelCtx"), testQuery, mock.AnythingOfType("*net.UDPAddr")).Return(testResponse, nil).Maybe()

	transport := NewUDPTransport("127.0.0.1:0", codec, logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(t, err)

	actualAddr := transport.conn.LocalAddr().(*net.UDPAddr)

	// Send multiple concurrent requests
	numRequests := 10
	var wg sync.WaitGroup
	wg.Add(numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			defer wg.Done()

			clientConn, err := net.DialUDP("udp", nil, actualAddr)
			if err != nil {
				t.Errorf("Failed to create client connection: %v", err)
				return
			}
			defer func() {
				if err := clientConn.Close(); err != nil {
					t.Logf("clientConn close error: %v", err)
				}
			}()

			_, err = clientConn.Write(queryData)
			if err != nil {
				t.Errorf("Failed to write query: %v", err)
				return
			}

			responseBuffer := make([]byte, 512)
			err = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
			if err != nil {
				t.Errorf("Failed to set read deadline: %v", err)
				return
			}

			n, err := clientConn.Read(responseBuffer)
			if err != nil {
				t.Errorf("Failed to read response: %v", err)
				return
			}

			if !assert.Equal(t, responseData, responseBuffer[:n]) {
				t.Errorf("Response mismatch")
			}
		}()
	}

	wg.Wait()

	err = transport.Stop()
	require.NoError(t, err)
}

func TestUDPTransport_InvalidPortBind(t *testing.T) {
	codec := &MockDNSCodec{}
	logger := &testLogger{}
	handler := &MockDNSResponder{}

	// Try to bind to a port that requires root privileges
	transport := NewUDPTransport("127.0.0.1:53", codec, logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)

	// This should fail unless running as root
	if err != nil {
		assert.Contains(t, err.Error(), "failed to bind UDP socket")
	} else {
		// If it succeeds (running as root), clean up
		err = transport.Stop()
		assert.NoError(t, err)
	}
}

// TestUDPTransport_InterfaceCompliance verifies that UDPTransport implements the ServerTransport interface
func TestUDPTransport_InterfaceCompliance(t *testing.T) {
	codec := &MockDNSCodec{}
	logger := &testLogger{}

	transport := NewUDPTransport("127.0.0.1:0", codec, logger)

	// Verify basic interface methods exist
	assert.NotNil(t, transport.Address)
	assert.NotNil(t, transport.Start)
	assert.NotNil(t, transport.Stop)

	// Test that Address() returns a string
	addr := transport.Address()
	assert.IsType(t, "", addr)
}

// TestUDPTransport_StopWithNilConnection tests Stop() method when connection is nil
func TestUDPTransport_StopWithNilConnection(t *testing.T) {
	codec := &MockDNSCodec{}
	logger := &MockLogger{}

	logger.On("Info", mock.Anything, "DNS transport stopped").Once()

	transport := NewUDPTransport("127.0.0.1:0", codec, logger)

	// Simulate running state with nil connection
	transport.mu.Lock()
	transport.running = true
	transport.conn = nil
	transport.mu.Unlock()

	err := transport.Stop()
	assert.NoError(t, err)
	assert.False(t, transport.running)

	logger.AssertExpectations(t)
}

// TestUDPTransport_WriteToUDPError tests handlePacket when WriteToUDP fails
func TestUDPTransport_WriteToUDPError(t *testing.T) {
	codec := &MockDNSCodec{}
	logger := &testLogger{} // Use simple logger instead of mock to avoid complexity
	handler := &MockDNSResponder{}

	// Setup test data
	testQuery := domain.Question{
		ID:   12345,
		Name: "example.com.",
		Type: 1,
	}

	testResponse := domain.DNSResponse{
		ID:    12345,
		RCode: 0,
	}

	queryData := []byte{0x01, 0x02, 0x03}
	responseData := []byte{0x04, 0x05, 0x06}
	clientAddr := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12345}

	// Setup mocks
	codec.On("DecodeQuery", queryData).Return(testQuery, nil)
	codec.On("EncodeResponse", testResponse).Return(responseData, nil)
	handler.On("HandleQuery", mock.Anything, testQuery, clientAddr).Return(testResponse, nil)

	transport := NewUDPTransport("127.0.0.1:0", codec, logger)

	// Start transport to get a real connection
	ctx := context.Background()
	err := transport.Start(ctx, handler)
	require.NoError(t, err)

	// Close the connection to force WriteToUDP to fail
	require.NoError(t, transport.conn.Close())

	// Call handlePacket directly to test write error path
	transport.handlePacket(ctx, queryData, clientAddr, handler)

	// Clean up - we've already closed the connection, so just stop the transport
	// this should throw
	err = transport.Stop()
	require.Error(t, err)

	codec.AssertExpectations(t)
	handler.AssertExpectations(t)
}

// TestUDPTransport_HandlerError tests handlePacket when DNS handler returns error
func TestUDPTransport_HandlerError(t *testing.T) {
	codec := &MockDNSCodec{}
	logger := &testLogger{} // Use simple logger instead of mock
	handler := &MockDNSResponder{}

	// Setup test data
	testQuery := domain.Question{
		ID:   12345,
		Name: "example.com.",
		Type: 1,
	}

	queryData := []byte{0x01, 0x02, 0x03}
	clientAddr := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12345}

	// Setup mocks
	codec.On("DecodeQuery", queryData).Return(testQuery, nil)
	handler.On("HandleQuery", mock.Anything, testQuery, clientAddr).Return(domain.DNSResponse{}, assert.AnError)

	transport := NewUDPTransport("127.0.0.1:0", codec, logger)

	// Call handlePacket directly to test handler error path
	ctx := context.Background()
	transport.handlePacket(ctx, queryData, clientAddr, handler)

	codec.AssertExpectations(t)
	handler.AssertExpectations(t)
}

// TestUDPTransport_ListenLoopReadError tests listenLoop handling read errors during normal operation
func TestUDPTransport_ListenLoopReadError(t *testing.T) {
	codec := &MockDNSCodec{}
	logger := &testLogger{} // Use simple logger to avoid mock complexity
	handler := &MockDNSResponder{}

	transport := NewUDPTransport("127.0.0.1:0", codec, logger)

	// Start transport to get into running state
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	require.NoError(t, err)

	// Force a read error by closing connection while transport is still marked as running
	require.NoError(t, transport.conn.Close())

	// Give the listen loop a moment to process the read error
	time.Sleep(10 * time.Millisecond)

	// Clean up
	err = transport.Stop()
	require.Error(t, err)
}

// TestUDPTransport_ContextCancellationInListenLoop tests the context cancellation path in listenLoop
func TestUDPTransport_ContextCancellationInListenLoop(t *testing.T) {
	codec := &MockDNSCodec{}
	logger := &testLogger{}
	handler := &MockDNSResponder{}

	transport := NewUDPTransport("127.0.0.1:0", codec, logger)

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())

	err := transport.Start(ctx, handler)
	require.NoError(t, err)

	// Cancel the context to trigger the ctx.Done() case in listenLoop
	cancel()

	// Give the listen loop a moment to process the context cancellation
	time.Sleep(10 * time.Millisecond)

	// Clean up
	err = transport.Stop()
	require.NoError(t, err)
}
