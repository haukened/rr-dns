package transport

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/haukened/rr-dns/internal/dns/domain"
	"github.com/stretchr/testify/mock"
)

// BenchmarkUDPTransport_QueryProcessing benchmarks the query processing performance
func BenchmarkUDPTransport_QueryProcessing(b *testing.B) {
	codec := &MockDNSCodec{}
	logger := &testLogger{}
	handler := &MockDNSResponder{}

	testQuery := domain.DNSQuery{
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

	queryData := []byte{0x01, 0x02, 0x03}
	responseData := []byte{0x04, 0x05, 0x06}

	codec.On("DecodeQuery", queryData).Return(testQuery, nil)
	codec.On("EncodeResponse", testResponse).Return(responseData, nil)
	handler.On("HandleRequest", mock.AnythingOfType("*context.cancelCtx"), testQuery, mock.AnythingOfType("*net.UDPAddr")).Return(testResponse)

	transport := NewUDPTransport("127.0.0.1:0", codec, logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	if err != nil {
		b.Fatalf("Failed to start transport: %v", err)
	}
	defer transport.Stop()

	actualAddr := transport.conn.LocalAddr().(*net.UDPAddr)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			clientConn, err := net.DialUDP("udp", nil, actualAddr)
			if err != nil {
				b.Errorf("Failed to create client connection: %v", err)
				continue
			}

			_, err = clientConn.Write(queryData)
			if err != nil {
				b.Errorf("Failed to write query: %v", err)
				clientConn.Close()
				continue
			}

			responseBuffer := make([]byte, 512)
			clientConn.SetReadDeadline(time.Now().Add(1 * time.Second))
			_, err = clientConn.Read(responseBuffer)
			if err != nil {
				b.Errorf("Failed to read response: %v", err)
			}

			clientConn.Close()
		}
	})
}

// BenchmarkUDPTransport_StartStop benchmarks the start/stop performance
func BenchmarkUDPTransport_StartStop(b *testing.B) {
	codec := &MockDNSCodec{}
	logger := &testLogger{}
	handler := &MockDNSResponder{}

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
	codec := &MockDNSCodec{}
	logger := &testLogger{}
	handler := &MockDNSResponder{}

	testQuery := domain.DNSQuery{
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

	codec.On("DecodeQuery", queryData).Return(testQuery, nil)
	codec.On("EncodeResponse", testResponse).Return(responseData, nil)
	handler.On("HandleRequest", mock.AnythingOfType("*context.cancelCtx"), testQuery, mock.AnythingOfType("*net.UDPAddr")).Return(testResponse)

	transport := NewUDPTransport("127.0.0.1:0", codec, logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := transport.Start(ctx, handler)
	if err != nil {
		b.Fatalf("Failed to start transport: %v", err)
	}
	defer transport.Stop()

	actualAddr := transport.conn.LocalAddr().(*net.UDPAddr)

	b.ResetTimer()
	b.SetParallelism(10) // 10 concurrent goroutines

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			clientConn, err := net.DialUDP("udp", nil, actualAddr)
			if err != nil {
				b.Errorf("Failed to create client connection: %v", err)
				continue
			}
			defer clientConn.Close()

			_, err = clientConn.Write(queryData)
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
