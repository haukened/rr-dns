package upstream

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

// Resolver implements upstream DNS resolution by forwarding queries to external DNS servers.
// It handles the low-level networking concerns of DNS over UDP while maintaining clean
// separation from the service layer business logic.
type Resolver struct {
	servers []string      // List of upstream DNS servers (e.g., "1.1.1.1:53")
	timeout time.Duration // Default timeout for DNS queries
}

// NewResolver creates a new upstream resolver with the specified servers and timeout.
func NewResolver(servers []string, timeout time.Duration) *Resolver {
	if len(servers) == 0 {
		servers = []string{"1.1.1.1:53", "1.0.0.1:53"} // Cloudflare as default
	}
	if timeout == 0 {
		timeout = 5 * time.Second // Default 5-second timeout
	}

	return &Resolver{
		servers: servers,
		timeout: timeout,
	}
}

// Resolve forwards a DNS query to upstream servers and returns the response.
// It tries each configured server in order until one responds successfully.
func (r *Resolver) Resolve(ctx context.Context, query domain.DNSQuery) (domain.DNSResponse, error) {
	deadline, ok := ctx.Deadline()
	if !ok {
		// If no deadline in context, use our default timeout
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.timeout)
		defer cancel()
		deadline = time.Now().Add(r.timeout)
	}

	timeout := time.Until(deadline)
	return r.ResolveWithTimeout(query, timeout)
}

// ResolveWithTimeout forwards a DNS query with a specific timeout.
func (r *Resolver) ResolveWithTimeout(query domain.DNSQuery, timeout time.Duration) (domain.DNSResponse, error) {
	if timeout <= 0 {
		timeout = r.timeout
	}

	var lastErr error

	// Try each upstream server in order
	for _, server := range r.servers {
		response, err := r.queryServer(server, query, timeout)
		if err == nil {
			return response, nil
		}
		lastErr = err
	}

	// If all servers failed, return the last error
	return domain.DNSResponse{}, fmt.Errorf("all upstream servers failed, last error: %w", lastErr)
}

// Health returns the health status of the upstream resolver.
func (r *Resolver) Health() error {
	// Create a simple health check query (A record for a reliable domain)
	healthQuery, err := domain.NewDNSQuery(1, "cloudflare.com.", 1, 1) // A record for cloudflare.com
	if err != nil {
		return fmt.Errorf("failed to create health check query: %w", err)
	}

	// Try to resolve with a short timeout
	_, err = r.ResolveWithTimeout(healthQuery, 2*time.Second)
	if err != nil {
		return fmt.Errorf("upstream resolver health check failed: %w", err)
	}

	return nil
}

// queryServer performs the actual DNS query against a specific server.
func (r *Resolver) queryServer(server string, query domain.DNSQuery, timeout time.Duration) (domain.DNSResponse, error) {
	// Create UDP connection to the DNS server
	conn, err := net.DialTimeout("udp", server, timeout)
	if err != nil {
		return domain.DNSResponse{}, fmt.Errorf("failed to connect to %s: %w", server, err)
	}
	defer conn.Close()

	// Set read/write deadline
	deadline := time.Now().Add(timeout)
	conn.SetDeadline(deadline)

	// Encode the DNS query to wire format
	queryBytes, err := r.encodeDNSQuery(query)
	if err != nil {
		return domain.DNSResponse{}, fmt.Errorf("failed to encode query: %w", err)
	}

	// Send the query
	_, err = conn.Write(queryBytes)
	if err != nil {
		return domain.DNSResponse{}, fmt.Errorf("failed to send query to %s: %w", server, err)
	}

	// Read the response
	buffer := make([]byte, 512) // Standard DNS message size
	n, err := conn.Read(buffer)
	if err != nil {
		return domain.DNSResponse{}, fmt.Errorf("failed to read response from %s: %w", server, err)
	}

	// Decode the DNS response from wire format
	response, err := r.decodeDNSResponse(buffer[:n], query.ID)
	if err != nil {
		return domain.DNSResponse{}, fmt.Errorf("failed to decode response from %s: %w", server, err)
	}

	return response, nil
}

// encodeDNSQuery converts a domain.DNSQuery to DNS wire format bytes.
// This implements the basic DNS message format per RFC 1035.
func (r *Resolver) encodeDNSQuery(query domain.DNSQuery) ([]byte, error) {
	// DNS message format:
	// Header (12 bytes) + Question section + Answer/Authority/Additional sections

	// For now, we'll implement a minimal query encoder
	// In a production system, you might want to use a more robust DNS library

	// DNS Header (12 bytes)
	header := make([]byte, 12)

	// Transaction ID (2 bytes)
	header[0] = byte(query.ID >> 8)
	header[1] = byte(query.ID & 0xFF)

	// Flags (2 bytes) - Standard query with recursion desired
	header[2] = 0x01 // QR=0 (query), Opcode=0 (standard), AA=0, TC=0, RD=1 (recursion desired)
	header[3] = 0x00 // RA=0, Z=0, RCODE=0

	// Question count (2 bytes) - 1 question
	header[4] = 0x00
	header[5] = 0x01

	// Answer/Authority/Additional counts (6 bytes) - all 0 for queries
	for i := 6; i < 12; i++ {
		header[i] = 0x00
	}

	// Question section
	question, err := r.encodeQuestion(query.Name, query.Type, query.Class)
	if err != nil {
		return nil, fmt.Errorf("failed to encode question: %w", err)
	}

	// Combine header and question
	result := append(header, question...)
	return result, nil
}

// encodeQuestion encodes the question section of a DNS query.
func (r *Resolver) encodeQuestion(name string, qtype domain.RRType, qclass domain.RRClass) ([]byte, error) {
	var result []byte

	// Encode domain name using DNS label format
	labels := []string{}
	current := ""
	for _, char := range name {
		if char == '.' {
			if current != "" {
				labels = append(labels, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		labels = append(labels, current)
	}

	// Encode each label
	for _, label := range labels {
		if len(label) > 63 {
			return nil, fmt.Errorf("DNS label too long: %s", label)
		}
		result = append(result, byte(len(label)))
		result = append(result, []byte(label)...)
	}

	// Null terminator for domain name
	result = append(result, 0x00)

	// Question type (2 bytes)
	result = append(result, byte(qtype>>8), byte(qtype&0xFF))

	// Question class (2 bytes)
	result = append(result, byte(qclass>>8), byte(qclass&0xFF))

	return result, nil
}

// decodeDNSResponse converts DNS wire format bytes to a domain.DNSResponse.
func (r *Resolver) decodeDNSResponse(data []byte, expectedID uint16) (domain.DNSResponse, error) {
	if len(data) < 12 {
		return domain.DNSResponse{}, fmt.Errorf("DNS response too short: %d bytes", len(data))
	}

	// Parse header
	id := uint16(data[0])<<8 | uint16(data[1])
	if id != expectedID {
		return domain.DNSResponse{}, fmt.Errorf("response ID mismatch: expected %d, got %d", expectedID, id)
	}

	// Extract response code from flags
	rcode := domain.RCode(data[3] & 0x0F)

	// Parse counts
	qdcount := uint16(data[4])<<8 | uint16(data[5])
	ancount := uint16(data[6])<<8 | uint16(data[7])
	// nscount := uint16(data[8])<<8 | uint16(data[9])  // TODO: parse authority records
	// arcount := uint16(data[10])<<8 | uint16(data[11]) // TODO: parse additional records

	// For now, we'll create a basic response structure
	// In a production system, you'd want to fully parse all sections

	answers := []domain.ResourceRecord{}
	authority := []domain.ResourceRecord{}
	additional := []domain.ResourceRecord{}

	// Parse answer records (simplified - would need full RR parsing in production)
	if ancount > 0 {
		// Skip question section first
		offset := 12
		for i := uint16(0); i < qdcount; i++ {
			// Skip question - find next null terminator plus 4 bytes (type + class)
			for offset < len(data) && data[offset] != 0 {
				labelLen := int(data[offset])
				if labelLen > 63 {
					// Compression pointer - skip 2 bytes
					offset += 2
					break
				}
				offset += labelLen + 1
			}
			if offset < len(data) && data[offset] == 0 {
				offset += 5 // null terminator + type + class
			}
		}

		// Parse answer records (basic implementation)
		for i := uint16(0); i < ancount && offset+10 < len(data); i++ {
			// Skip name (assume compression pointer for simplicity)
			if offset+1 < len(data) && (data[offset]&0xC0) == 0xC0 {
				offset += 2
			} else {
				// Skip uncompressed name
				for offset < len(data) && data[offset] != 0 {
					offset += int(data[offset]) + 1
				}
				offset++ // skip null terminator
			}

			if offset+10 > len(data) {
				break
			}

			// Parse RR fields
			rrtype := domain.RRType(uint16(data[offset])<<8 | uint16(data[offset+1]))
			rrclass := domain.RRClass(uint16(data[offset+2])<<8 | uint16(data[offset+3]))
			ttl := uint32(data[offset+4])<<24 | uint32(data[offset+5])<<16 | uint32(data[offset+6])<<8 | uint32(data[offset+7])
			rdlength := uint16(data[offset+8])<<8 | uint16(data[offset+9])

			offset += 10

			if offset+int(rdlength) > len(data) {
				break
			}

			rdata := make([]byte, rdlength)
			copy(rdata, data[offset:offset+int(rdlength)])
			offset += int(rdlength)

			// Create ResourceRecord
			rr, err := domain.NewResourceRecord("answer.record.", rrtype, rrclass, ttl, rdata)
			if err == nil {
				answers = append(answers, rr)
			}
		}
	}

	// Create and return the DNS response
	return domain.NewDNSResponse(id, rcode, answers, authority, additional)
}
