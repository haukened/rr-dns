package transport

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/haukened/rr-dns/internal/dns/common/log"
	"github.com/haukened/rr-dns/internal/dns/gateways/wire"
	"github.com/haukened/rr-dns/internal/dns/services/resolver"
)

// UDPTransport implements ServerTransport for standard DNS over UDP (RFC 1035).
// It handles UDP socket management, packet reception/transmission, and wire format
// conversion while delegating DNS logic to the service layer.
type UDPTransport struct {
	addr   string
	conn   *net.UDPConn
	codec  wire.DNSCodec
	logger log.Logger

	// Synchronization for graceful shutdown
	mu      sync.RWMutex
	running bool
	stopCh  chan struct{}
}

// NewUDPTransport creates a new UDP transport instance.
func NewUDPTransport(addr string, codec wire.DNSCodec, logger log.Logger) *UDPTransport {
	return &UDPTransport{
		addr:   addr,
		codec:  codec,
		logger: logger,
		stopCh: make(chan struct{}),
	}
}

// Start begins listening for UDP DNS queries on the configured address.
// It binds to the UDP socket and starts the packet handling loop.
func (t *UDPTransport) Start(ctx context.Context, handler resolver.DNSResponder) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		return fmt.Errorf("UDP transport already running")
	}

	// Parse and bind to UDP address
	udpAddr, err := net.ResolveUDPAddr("udp", t.addr)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address %s: %w", t.addr, err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("failed to bind UDP socket on %s: %w", t.addr, err)
	}

	t.conn = conn
	t.running = true

	t.logger.Info(map[string]any{
		"transport": "udp",
		"address":   t.addr,
	}, "DNS transport started")

	// Start the packet handling loop
	go t.listenLoop(ctx, handler)

	return nil
}

// Stop gracefully shuts down the UDP transport.
func (t *UDPTransport) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return nil
	}

	// Signal stop and close connection
	close(t.stopCh)

	var closeErr error
	if t.conn != nil {
		closeErr = t.conn.Close()
		if closeErr != nil {
			t.logger.Warn(map[string]any{
				"error": closeErr.Error(),
			}, "Error closing UDP connection")
		}
	}

	t.running = false

	t.logger.Info(map[string]any{
		"transport": "udp",
		"address":   t.addr,
	}, "DNS transport stopped")

	return closeErr
}

// Address returns the network address the transport is bound to.
func (t *UDPTransport) Address() string {
	return t.addr
}

// listenLoop continuously listens for UDP packets and handles them.
func (t *UDPTransport) listenLoop(ctx context.Context, handler resolver.DNSResponder) {
	buffer := make([]byte, 512) // Standard DNS UDP packet size limit

	for {
		select {
		case <-ctx.Done():
			t.logger.Debug(nil, "UDP transport stopping due to context cancellation")
			return
		case <-t.stopCh:
			t.logger.Debug(nil, "UDP transport stopping due to stop signal")
			return
		default:
			// Read incoming packet
			n, clientAddr, err := t.conn.ReadFromUDP(buffer)
			if err != nil {
				// Check if we're shutting down
				t.mu.RLock()
				running := t.running
				t.mu.RUnlock()

				if !running {
					return // Normal shutdown
				}

				t.logger.Warn(map[string]any{
					"error": err.Error(),
				}, "Failed to read UDP packet")
				continue
			}

			packet := make([]byte, n)
			copy(packet, buffer[:n])
			go t.handlePacket(ctx, packet, clientAddr, handler)
		}
	}
}

// handlePacket processes a single UDP DNS packet.
func (t *UDPTransport) handlePacket(ctx context.Context, data []byte, clientAddr *net.UDPAddr, handler resolver.DNSResponder) {
	// Debug log raw incoming data
	t.logger.Debug(map[string]any{
		"client": clientAddr.String(),
		"size":   len(data),
		"raw":    fmt.Sprintf("%x", data),
	}, "Received raw DNS query data")

	// Decode wire format to domain object
	query, err := t.codec.DecodeQuery(data)
	if err != nil {
		t.logger.Warn(map[string]any{
			"client": clientAddr.String(),
			"error":  err.Error(),
			"size":   len(data),
		}, "Failed to decode DNS query")
		return
	}

	t.logger.Debug(map[string]any{
		"client":   clientAddr.String(),
		"query_id": query.ID,
		"name":     query.Name,
		"type":     query.Type,
	}, "Received DNS query")

	// Pass domain object to service layer
	response, err := handler.HandleQuery(ctx, query, clientAddr)
	if err != nil {
		t.logger.Error(map[string]any{
			"client":   clientAddr.String(),
			"query_id": query.ID,
			"error":    err.Error(),
		}, "Failed to handle DNS query")
		return
	}

	// Encode domain object back to wire format
	responseData, err := t.codec.EncodeResponse(response)
	if err != nil {
		t.logger.Error(map[string]any{
			"client":   clientAddr.String(),
			"query_id": query.ID,
			"error":    err.Error(),
		}, "Failed to encode DNS response")
		return
	}

	// Debug log raw outgoing data
	t.logger.Debug(map[string]any{
		"client":   clientAddr.String(),
		"query_id": response.ID,
		"size":     len(responseData),
		"raw":      fmt.Sprintf("%x", responseData),
	}, "Encoded DNS response data")

	// Send response back to client
	_, err = t.conn.WriteToUDP(responseData, clientAddr)
	if err != nil {
		t.logger.Error(map[string]any{
			"client":   clientAddr.String(),
			"query_id": response.ID,
			"error":    err.Error(),
		}, "Failed to send DNS response")
		return
	}

	t.logger.Debug(map[string]any{
		"client":   clientAddr.String(),
		"query_id": response.ID,
		"rcode":    response.RCode,
		"answers":  len(response.Answers),
		"size":     len(responseData),
	}, "Sent DNS response")
}
