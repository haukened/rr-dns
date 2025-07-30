package transport

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/haukened/rr-dns/internal/dns/common/log"
	"github.com/haukened/rr-dns/internal/dns/services/resolver"
)

// UDPTransport implements ServerTransport for standard DNS over UDP (RFC 1035).
// It handles UDP socket management, packet reception/transmission, and wire format
// conversion while delegating DNS logic to the service layer.
type UDPTransport struct {
	addr  string
	conn  *net.UDPConn
	codec resolver.DNSCodec

	// Synchronization for graceful shutdown
	mu      sync.RWMutex
	running bool
	stopCh  chan struct{}
}

// NewUDPTransport creates a new UDP transport instance.
func NewUDPTransport(addr string, codec resolver.DNSCodec) *UDPTransport {
	return &UDPTransport{
		addr:   addr,
		codec:  codec,
		stopCh: make(chan struct{}),
	}
}

// Start begins listening for UDP DNS queries on the configured address.
// It binds to the UDP socket and starts the packet handling loop.
func (t *UDPTransport) Start(ctx context.Context, handler RequestHandler) error {
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

	log.Info(map[string]any{
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

	if t.conn != nil {
		if err := t.conn.Close(); err != nil {
			log.Warn(map[string]any{
				"error": err.Error(),
			}, "Error closing UDP connection")
		}
	}

	t.running = false

	log.Info(map[string]any{
		"transport": "udp",
		"address":   t.addr,
	}, "DNS transport stopped")

	return nil
}

// Address returns the network address the transport is bound to.
func (t *UDPTransport) Address() string {
	return t.addr
}

// listenLoop continuously listens for UDP packets and handles them.
func (t *UDPTransport) listenLoop(ctx context.Context, handler RequestHandler) {
	buffer := make([]byte, 512) // Standard DNS UDP packet size limit

	for {
		select {
		case <-ctx.Done():
			log.Debug(nil, "UDP transport stopping due to context cancellation")
			return
		case <-t.stopCh:
			log.Debug(nil, "UDP transport stopping due to stop signal")
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

				log.Warn(map[string]any{
					"error": err.Error(),
				}, "Failed to read UDP packet")
				continue
			}

			// Handle packet in separate goroutine to avoid blocking the listen loop
			go t.handlePacket(ctx, buffer[:n], clientAddr, handler)
		}
	}
}

// handlePacket processes a single UDP DNS packet.
func (t *UDPTransport) handlePacket(ctx context.Context, data []byte, clientAddr *net.UDPAddr, handler RequestHandler) {
	// Decode wire format to domain object
	query, err := t.codec.DecodeQuery(data)
	if err != nil {
		log.Warn(map[string]any{
			"client": clientAddr.String(),
			"error":  err.Error(),
			"size":   len(data),
		}, "Failed to decode DNS query")
		return
	}

	log.Debug(map[string]any{
		"client":   clientAddr.String(),
		"query_id": query.ID,
		"name":     query.Name,
		"type":     query.Type,
	}, "Received DNS query")

	// Pass domain object to service layer
	response := handler.HandleRequest(ctx, query, clientAddr)

	// Encode domain object back to wire format
	responseData, err := t.codec.EncodeResponse(response)
	if err != nil {
		log.Error(map[string]any{
			"client":   clientAddr.String(),
			"query_id": query.ID,
			"error":    err.Error(),
		}, "Failed to encode DNS response")
		return
	}

	// Send response back to client
	_, err = t.conn.WriteToUDP(responseData, clientAddr)
	if err != nil {
		log.Error(map[string]any{
			"client":   clientAddr.String(),
			"query_id": response.ID,
			"error":    err.Error(),
		}, "Failed to send DNS response")
		return
	}

	log.Debug(map[string]any{
		"client":   clientAddr.String(),
		"query_id": response.ID,
		"rcode":    response.RCode,
		"answers":  len(response.Answers),
		"size":     len(responseData),
	}, "Sent DNS response")
}
