package transport

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

// BenchmarkUDPTransport_QueryProcessing benchmarks the query processing performance
func BenchmarkUDPTransport_QueryProcessing(b *testing.B) {
	codec := &StubDNSCodec{}
	logger := &testLogger{}
	handler := &StubDNSResponder{}

	queryData := []byte{0x01, 0x02, 0x03}

	transport := NewUDPTransport("127.0.0.1:0", codec, logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	if err != nil {
		b.Fatalf("Failed to start transport: %v", err)
	}
	defer transport.Stop()

	actualAddr := transport.conn.LocalAddr().(*net.UDPAddr)

	clientConn, err := net.DialUDP("udp", nil, actualAddr)
	if err != nil {
		b.Fatalf("Failed to create client connection: %v", err)
	}
	defer clientConn.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := clientConn.Write(queryData)
			if err != nil {
				b.Errorf("Failed to write query: %v", err)
				continue
			}

			responseBuffer := make([]byte, 512)
			clientConn.SetReadDeadline(time.Now().Add(1 * time.Second))
			_, err = clientConn.Read(responseBuffer)
			if err != nil {
				b.Errorf("Failed to read response: %v", err)
			}
		}
	})
}

// BenchmarkUDPTransport_StartStop benchmarks the start/stop performance
func BenchmarkUDPTransport_StartStop(b *testing.B) {
	codec := &StubDNSCodec{}
	logger := &testLogger{}
	handler := &StubDNSResponder{}

	for i := 0; i < b.N; i++ {
		transport := NewUDPTransport("127.0.0.1:0", codec, logger)
		ctx, cancel := context.WithCancel(context.Background())

		err := transport.Start(ctx, handler)
		if err != nil {
			b.Fatalf("Failed to start transport: %v", err)
		}

		err = transport.Stop()
		if err != nil {
			b.Fatalf("Failed to stop transport: %v", err)
		}

		cancel()
	}
}

// BenchmarkUDPTransport_ConcurrentConnections benchmarks multiple concurrent connections
func BenchmarkUDPTransport_ConcurrentConnections(b *testing.B) {
	codec := &StubDNSCodec{}
	logger := &testLogger{}
	handler := &StubDNSResponder{}

	queryData := []byte{0x01, 0x02, 0x03}

	transport := NewUDPTransport("127.0.0.1:0", codec, logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	if err != nil {
		b.Fatalf("Failed to start transport: %v", err)
	}
	defer transport.Stop()

	actualAddr := transport.conn.LocalAddr().(*net.UDPAddr)
	// Use sync.Pool to reuse UDP connections and avoid port exhaustion.
	var connPool = sync.Pool{
		New: func() any {
			conn, err := net.DialUDP("udp", nil, actualAddr)
			if err != nil {
				b.Fatalf("Failed to pre-dial UDP client: %v", err)
			}
			return conn
		},
	}

	// Pre-allocate connections in the pool.
	const preAlloc = 100
	for i := 0; i < preAlloc; i++ {
		conn, err := net.DialUDP("udp", nil, actualAddr)
		if err != nil {
			b.Fatalf("Failed to pre-dial UDP client: %v", err)
		}
		connPool.Put(conn)
	}

	b.ResetTimer()
	b.SetParallelism(10) // 10 concurrent goroutines

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			clientConn := connPool.Get().(*net.UDPConn)

			_, err := clientConn.Write(queryData)
			if err != nil {
				b.Errorf("Failed to write query: %v", err)
				connPool.Put(clientConn)
				continue
			}

			responseBuffer := make([]byte, 512)
			clientConn.SetReadDeadline(time.Now().Add(1 * time.Second))
			_, err = clientConn.Read(responseBuffer)
			if err != nil {
				b.Errorf("Failed to read response: %v", err)
			}

			connPool.Put(clientConn)
		}
	})
}

type StubDNSCodec struct{}

func (s *StubDNSCodec) DecodeQuery([]byte) (domain.DNSQuery, error) {
	return domain.DNSQuery{ID: 12345, Name: "example.com.", Type: 1}, nil
}

func (s *StubDNSCodec) EncodeResponse(resp domain.DNSResponse) ([]byte, error) {
	return []byte{0x04, 0x05, 0x06}, nil
}

func (s *StubDNSCodec) DecodeResponse(_ []byte, _ uint16, _ time.Time) (domain.DNSResponse, error) {
	return domain.DNSResponse{}, nil
}

func (s *StubDNSCodec) EncodeQuery(query domain.DNSQuery) ([]byte, error) {
	return []byte{0x01, 0x02, 0x03}, nil
}

type StubDNSResponder struct{}

func (s *StubDNSResponder) HandleRequest(ctx context.Context, query domain.DNSQuery, client net.Addr) domain.DNSResponse {
	return domain.DNSResponse{ID: query.ID, RCode: 0}
}
