package upstream

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

// MockCodec implements domain.DNSCodec for testing
type MockCodec struct {
	mock.Mock
}

func (m *MockCodec) EncodeQuery(query domain.Question) ([]byte, error) {
	args := m.Called(query)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockCodec) DecodeResponse(data []byte, queryID uint16, now time.Time) (domain.DNSResponse, error) {
	args := m.Called(data, queryID, now)
	return args.Get(0).(domain.DNSResponse), args.Error(1)
}

func (m *MockCodec) DecodeQuery(data []byte) (domain.Question, error) {
	args := m.Called(data)
	return args.Get(0).(domain.Question), args.Error(1)
}

func (m *MockCodec) EncodeResponse(resp domain.DNSResponse) ([]byte, error) {
	args := m.Called(resp)
	return args.Get(0).([]byte), args.Error(1)
}

// MockConn implements net.Conn for testing
type MockConn struct {
	mock.Mock
	readData         []byte
	writeData        []byte
	setDeadlineError error // if non-nil, SetDeadline will return this error
}

func (m *MockConn) Read(b []byte) (n int, err error) {
	args := m.Called(b)
	if m.readData != nil {
		copy(b, m.readData)
		return len(m.readData), args.Error(1)
	}
	return args.Int(0), args.Error(1)
}

func (m *MockConn) Write(b []byte) (n int, err error) {
	args := m.Called(b)
	m.writeData = make([]byte, len(b))
	copy(m.writeData, b)
	return args.Int(0), args.Error(1)
}

func (m *MockConn) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockConn) LocalAddr() net.Addr  { return nil }
func (m *MockConn) RemoteAddr() net.Addr { return nil }
func (m *MockConn) SetDeadline(t time.Time) error {
	if m.setDeadlineError != nil {
		return m.setDeadlineError
	}
	return nil
}
func (m *MockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *MockConn) SetWriteDeadline(t time.Time) error { return nil }

// Helper functions for creating test data
func createTestQuery() domain.Question {
	return domain.Question{
		ID:    12345,
		Name:  "example.com.",
		Type:  1, // A record
		Class: 1, // IN class
	}
}

func createTestResponse() domain.DNSResponse {
	rr, _ := domain.NewAuthoritativeResourceRecord(
		"example.com.",
		1,   // A record
		1,   // IN class
		300, // 5 minutes TTL
		[]byte("1.2.3.4"),
	)
	return domain.DNSResponse{
		ID:      12345,
		RCode:   0, // NOERROR
		Answers: []domain.ResourceRecord{rr},
	}
}

func createTimeFixture() time.Time {
	return time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
}

func TestNewResolver(t *testing.T) {
	tests := []struct {
		name    string
		opts    Options
		wantErr string
	}{
		{
			name: "valid options",
			opts: Options{
				Servers: []string{"1.1.1.1:53"},
				Timeout: 5 * time.Second,
				Codec:   &MockCodec{},
			},
			wantErr: "",
		},
		{
			name: "no servers provided",
			opts: Options{
				Codec: &MockCodec{},
			},
			wantErr: errNoServersProvided,
		},
		{
			name: "no codec provided",
			opts: Options{
				Servers: []string{"1.1.1.1:53"},
			},
			wantErr: errCodecRequired,
		},
		{
			name: "default timeout applied",
			opts: Options{
				Servers: []string{"1.1.1.1:53"},
				Timeout: 0, // Should get default 5s
				Codec:   &MockCodec{},
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver, err := NewResolver(tt.opts)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, resolver)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resolver)

				// Verify defaults are applied
				if tt.opts.Timeout <= 0 {
					assert.Equal(t, 5*time.Second, resolver.timeout)
				} else {
					assert.Equal(t, tt.opts.Timeout, resolver.timeout)
				}

				if tt.opts.Dial == nil {
					assert.NotNil(t, resolver.dial)
				}
			}
		})
	}
}

func TestResolver_ensureContextDeadline(t *testing.T) {
	codec := &MockCodec{}
	resolver, err := NewResolver(Options{
		Servers: []string{"1.1.1.1:53"},
		Timeout: 2 * time.Second,
		Codec:   codec,
	})
	assert.NoError(t, err)

	t.Run("context without deadline", func(t *testing.T) {
		ctx := context.Background()
		resultCtx, cancel := resolver.ensureContextDeadline(ctx)

		assert.NotNil(t, cancel, "cancel function should be provided when timeout is added")
		_, hasDeadline := resultCtx.Deadline()
		assert.True(t, hasDeadline, "context should have deadline")
		cancel()
	})

	t.Run("context with existing deadline", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		resultCtx, cancelFunc := resolver.ensureContextDeadline(ctx)

		assert.Nil(t, cancelFunc, "cancel function should be nil when deadline already exists")
		assert.Equal(t, ctx, resultCtx, "context should be unchanged")
	})
}

func TestResolver_Resolve_Serial(t *testing.T) {
	tf := createTimeFixture()
	query := createTestQuery()
	response := createTestResponse()
	queryBytes := []byte("query")
	responseBytes := []byte("response")

	tests := []struct {
		name       string
		servers    []string
		setupMocks func(*MockCodec, *MockConn)
		wantErr    string
		wantResp   domain.DNSResponse
	}{
		{
			name:    "successful query first server",
			servers: []string{"1.1.1.1:53"},
			setupMocks: func(codec *MockCodec, conn *MockConn) {
				codec.On("EncodeQuery", query).Return(queryBytes, nil)
				codec.On("DecodeResponse", responseBytes, query.ID, tf).Return(response, nil)
				conn.On("Write", queryBytes).Return(len(queryBytes), nil)
				conn.On("Read", mock.AnythingOfType("[]uint8")).Return(len(responseBytes), nil)
				conn.On("Close").Return(nil)
				conn.readData = responseBytes
			},
			wantResp: response,
		},
		{
			name:    "encode error",
			servers: []string{"1.1.1.1:53"},
			setupMocks: func(codec *MockCodec, conn *MockConn) {
				codec.On("EncodeQuery", query).Return([]byte(nil), errors.New("encode failed"))
				conn.On("Close").Return(nil)
			},
			wantErr: "encode failed",
		},
		{
			name:    "connection error",
			servers: []string{"1.1.1.1:53"},
			setupMocks: func(codec *MockCodec, conn *MockConn) {
				// Dial will fail, so codec won't be called
			},
			wantErr: "failed to connect",
		},
		{
			name:    "write error",
			servers: []string{"1.1.1.1:53"},
			setupMocks: func(codec *MockCodec, conn *MockConn) {
				codec.On("EncodeQuery", query).Return(queryBytes, nil)
				conn.On("Write", queryBytes).Return(0, errors.New("write failed"))
				conn.On("Close").Return(nil)
			},
			wantErr: "write failed",
		},
		{
			name:    "read error",
			servers: []string{"1.1.1.1:53"},
			setupMocks: func(codec *MockCodec, conn *MockConn) {
				codec.On("EncodeQuery", query).Return(queryBytes, nil)
				conn.On("Write", queryBytes).Return(len(queryBytes), nil)
				conn.On("Read", mock.AnythingOfType("[]uint8")).Return(0, errors.New("read failed"))
				conn.On("Close").Return(nil)
			},
			wantErr: "read failed",
		},
		{
			name:    "multiple servers - first fails, second succeeds",
			servers: []string{"1.1.1.1:53", "8.8.8.8:53"},
			setupMocks: func(codec *MockCodec, conn *MockConn) {
				// First server call will fail at dial level (handled by test dial func)
				// Second server call succeeds
				codec.On("EncodeQuery", query).Return(queryBytes, nil)
				codec.On("DecodeResponse", responseBytes, query.ID, tf).Return(response, nil)
				conn.On("Write", queryBytes).Return(len(queryBytes), nil)
				conn.On("Read", mock.AnythingOfType("[]uint8")).Return(len(responseBytes), nil)
				conn.On("Close").Return(nil)
				conn.readData = responseBytes
			},
			wantResp: response,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codec := &MockCodec{}
			conn := &MockConn{}

			tt.setupMocks(codec, conn)

			callCount := 0
			dial := func(ctx context.Context, network, address string) (net.Conn, error) {
				callCount++
				if tt.name == "connection error" {
					return nil, errors.New("connection refused")
				}
				if tt.name == "multiple servers - first fails, second succeeds" && callCount == 1 {
					return nil, errors.New("first server failed")
				}
				return conn, nil
			}

			resolver, err := NewResolver(Options{
				Servers:  tt.servers,
				Timeout:  time.Second,
				Parallel: false, // Serial mode
				Codec:    codec,
				Dial:     dial,
			})
			assert.NoError(t, err)

			ctx := context.Background()
			resp, err := resolver.Resolve(ctx, query, tf)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResp, resp)
			}

			codec.AssertExpectations(t)
			conn.AssertExpectations(t)
		})
	}
}

func TestResolver_Resolve_Parallel(t *testing.T) {
	tf := createTimeFixture()
	query := createTestQuery()
	response := createTestResponse()
	queryBytes := []byte("query")
	responseBytes := []byte("response")

	tests := []struct {
		name         string
		servers      []string
		setupMocks   func(*MockCodec, *MockConn)
		dialBehavior func(address string) error // nil means success
		wantErr      string
		wantResp     domain.DNSResponse
	}{
		{
			name:    "parallel success from first responding server",
			servers: []string{"1.1.1.1:53", "8.8.8.8:53"},
			setupMocks: func(codec *MockCodec, conn *MockConn) {
				codec.On("EncodeQuery", query).Return(queryBytes, nil)
				codec.On("DecodeResponse", responseBytes, query.ID, tf).Return(response, nil)
				conn.On("Write", queryBytes).Return(len(queryBytes), nil)
				conn.On("Read", mock.AnythingOfType("[]uint8")).Return(len(responseBytes), nil)
				conn.On("Close").Return(nil)
				conn.readData = responseBytes
			},
			dialBehavior: func(address string) error {
				return nil // All connections succeed
			},
			wantResp: response,
		},
		{
			name:    "parallel all servers fail",
			servers: []string{"1.1.1.1:53", "8.8.8.8:53"},
			setupMocks: func(codec *MockCodec, conn *MockConn) {
				// No codec calls expected since dial fails
			},
			dialBehavior: func(address string) error {
				return errors.New("connection failed")
			},
			wantErr: "all 2 upstream servers failed",
		},
		{
			name:    "parallel context timeout during wait",
			servers: []string{"1.1.1.1:53", "8.8.8.8:53"},
			setupMocks: func(codec *MockCodec, conn *MockConn) {
				// Set up mocks for successful connection but slow response
				codec.On("EncodeQuery", query).Return(queryBytes, nil)
				conn.On("Write", queryBytes).Return(len(queryBytes), nil)
				// Simulate slow read that will be interrupted by context timeout
				conn.On("Read", mock.AnythingOfType("[]uint8")).Run(func(args mock.Arguments) {
					// Simulate a slow operation that takes longer than the context timeout
					time.Sleep(50 * time.Millisecond)
				}).Return(0, errors.New("read timeout"))
				conn.On("Close").Return(nil)
			},
			dialBehavior: func(address string) error {
				return nil // Connections succeed but responses are slow
			},
			wantErr: "query timeout after",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codec := &MockCodec{}
			conn := &MockConn{}

			tt.setupMocks(codec, conn)

			dial := func(ctx context.Context, network, address string) (net.Conn, error) {
				if err := tt.dialBehavior(address); err != nil {
					return nil, err
				}
				return conn, nil
			}

			resolver, err := NewResolver(Options{
				Servers:  tt.servers,
				Timeout:  time.Second,
				Parallel: true, // Parallel mode
				Codec:    codec,
				Dial:     dial,
			})
			assert.NoError(t, err)

			ctx := context.Background()
			// Use a short timeout for the context timeout test case
			if tt.name == "parallel context timeout during wait" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
				defer cancel()
			}

			resp, err := resolver.Resolve(ctx, query, tf)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResp, resp)
			}

			codec.AssertExpectations(t)
			conn.AssertExpectations(t)
		})
	}
}

func TestResolver_Resolve_ContextCancellation(t *testing.T) {
	tf := createTimeFixture()
	query := createTestQuery()
	queryBytes := []byte("query")

	codec := &MockCodec{}
	conn := &MockConn{}

	codec.On("EncodeQuery", query).Return(queryBytes, nil)
	conn.On("Write", queryBytes).Return(len(queryBytes), nil)
	conn.On("Close").Return(nil)
	// Simulate slow read that will be cancelled
	conn.On("Read", mock.AnythingOfType("[]uint8")).Return(0, errors.New("read timeout"))

	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		return conn, nil
	}

	resolver, err := NewResolver(Options{
		Servers:  []string{"1.1.1.1:53"},
		Timeout:  time.Second,
		Parallel: false,
		Codec:    codec,
		Dial:     dial,
	})
	assert.NoError(t, err)

	// Create context that cancels quickly
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err = resolver.Resolve(ctx, query, tf)
	assert.Error(t, err)

	codec.AssertExpectations(t)
	conn.AssertExpectations(t)
}

func TestResolver_setTimeout(t *testing.T) {
	codec := &MockCodec{}
	resolver, err := NewResolver(Options{
		Servers: []string{"1.1.1.1:53"},
		Timeout: time.Second,
		Codec:   codec,
	})
	assert.NoError(t, err)

	// Test setting valid timeout
	resolver.setTimeout(2 * time.Second)
	assert.Equal(t, 2*time.Second, resolver.timeout)

	// Test setting invalid timeout (should be ignored)
	originalTimeout := resolver.timeout
	resolver.setTimeout(0)
	assert.Equal(t, originalTimeout, resolver.timeout)

	resolver.setTimeout(-time.Second)
	assert.Equal(t, originalTimeout, resolver.timeout)
}

func TestResolver_queryServerWithContext(t *testing.T) {
	query := createTestQuery()
	response := createTestResponse()
	queryBytes := []byte("query")
	responseBytes := []byte("response")

	tests := []struct {
		name       string
		setupMocks func(*MockCodec, *MockConn)
		wantErr    string
		wantResp   domain.DNSResponse
	}{
		{
			name: "successful query",
			setupMocks: func(codec *MockCodec, conn *MockConn) {
				codec.On("EncodeQuery", query).Return(queryBytes, nil)
				codec.On("DecodeResponse", responseBytes, query.ID, mock.AnythingOfType("time.Time")).Return(response, nil)
				conn.On("Write", queryBytes).Return(len(queryBytes), nil)
				conn.On("Read", mock.AnythingOfType("[]uint8")).Return(len(responseBytes), nil)
				conn.On("Close").Return(nil)
				conn.readData = responseBytes
			},
			wantResp: response,
		},
		{
			name: "decode error",
			setupMocks: func(codec *MockCodec, conn *MockConn) {
				codec.On("EncodeQuery", query).Return(queryBytes, nil)
				codec.On("DecodeResponse", responseBytes, query.ID, mock.AnythingOfType("time.Time")).Return(domain.DNSResponse{}, errors.New("decode failed"))
				conn.On("Write", queryBytes).Return(len(queryBytes), nil)
				conn.On("Read", mock.AnythingOfType("[]uint8")).Return(len(responseBytes), nil)
				conn.On("Close").Return(nil)
				conn.readData = responseBytes
			},
			wantErr: "decode failed",
		},
	}

	for _, tt := range tests {
		tf := createTimeFixture()
		t.Run(tt.name, func(t *testing.T) {
			codec := &MockCodec{}
			conn := &MockConn{}

			tt.setupMocks(codec, conn)

			dial := func(ctx context.Context, network, address string) (net.Conn, error) {
				return conn, nil
			}

			resolver, err := NewResolver(Options{
				Servers: []string{"1.1.1.1:53"},
				Timeout: time.Second,
				Codec:   codec,
				Dial:    dial,
			})
			assert.NoError(t, err)

			ctx := context.Background()
			resp, err := resolver.queryServerWithContext(ctx, "1.1.1.1:53", query, tf)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResp, resp)
			}

			codec.AssertExpectations(t)
			conn.AssertExpectations(t)
		})
	}
}

func TestResolver_queryServerWithContext_SetDeadlineError(t *testing.T) {
	query := createTestQuery()
	tf := createTimeFixture()

	codec := &MockCodec{}
	conn := &MockConn{
		setDeadlineError: errors.New("set deadline failed"),
	}

	conn.On("Close").Return(nil)

	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		return conn, nil
	}

	resolver, err := NewResolver(Options{
		Servers: []string{"1.1.1.1:53"},
		Timeout: time.Second,
		Codec:   codec,
		Dial:    dial,
	})
	assert.NoError(t, err)

	// Create context with deadline to trigger SetDeadline call
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, err = resolver.queryServerWithContext(ctx, "1.1.1.1:53", query, tf)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to set connection deadline")
	assert.Contains(t, err.Error(), "set deadline failed")

	conn.AssertExpectations(t)
}
