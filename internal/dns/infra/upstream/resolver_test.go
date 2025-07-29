package upstream

import (
	"context"
	"testing"
	"time"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

func TestNewResolver(t *testing.T) {
	tests := []struct {
		name            string
		servers         []string
		timeout         time.Duration
		expectedLen     int
		expectedFirst   string
		expectedTimeout time.Duration
	}{
		{
			name:            "default servers and timeout",
			servers:         nil,
			timeout:         0,
			expectedLen:     2,
			expectedFirst:   "1.1.1.1:53",
			expectedTimeout: 5 * time.Second,
		},
		{
			name:            "custom servers and timeout",
			servers:         []string{"8.8.8.8:53", "8.8.4.4:53"},
			timeout:         3 * time.Second,
			expectedLen:     2,
			expectedFirst:   "8.8.8.8:53",
			expectedTimeout: 3 * time.Second,
		},
		{
			name:            "empty servers with custom timeout",
			servers:         []string{},
			timeout:         10 * time.Second,
			expectedLen:     2,
			expectedFirst:   "1.1.1.1:53",
			expectedTimeout: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewResolver(tt.servers, tt.timeout)

			if len(resolver.servers) != tt.expectedLen {
				t.Errorf("expected %d servers, got %d", tt.expectedLen, len(resolver.servers))
			}

			if len(resolver.servers) > 0 && resolver.servers[0] != tt.expectedFirst {
				t.Errorf("expected first server %q, got %q", tt.expectedFirst, resolver.servers[0])
			}

			if resolver.timeout != tt.expectedTimeout {
				t.Errorf("expected timeout %v, got %v", tt.expectedTimeout, resolver.timeout)
			}
		})
	}
}

func TestResolver_encodeDNSQuery(t *testing.T) {
	resolver := NewResolver([]string{"8.8.8.8:53"}, 5*time.Second)

	query, err := domain.NewDNSQuery(12345, "example.com.", 1, 1)
	if err != nil {
		t.Fatalf("failed to create test query: %v", err)
	}

	encoded, err := resolver.encodeDNSQuery(query)
	if err != nil {
		t.Fatalf("failed to encode query: %v", err)
	}

	// Verify basic structure
	if len(encoded) < 12 {
		t.Errorf("encoded query too short: %d bytes", len(encoded))
	}

	// Verify transaction ID
	id := uint16(encoded[0])<<8 | uint16(encoded[1])
	if id != 12345 {
		t.Errorf("expected ID 12345, got %d", id)
	}

	// Verify flags (recursion desired)
	if encoded[2] != 0x01 {
		t.Errorf("expected RD flag set, got 0x%02x", encoded[2])
	}

	// Verify question count
	qdcount := uint16(encoded[4])<<8 | uint16(encoded[5])
	if qdcount != 1 {
		t.Errorf("expected 1 question, got %d", qdcount)
	}
}

func TestResolver_encodeQuestion(t *testing.T) {
	resolver := NewResolver([]string{"8.8.8.8:53"}, 5*time.Second)

	tests := []struct {
		name     string
		domain   string
		qtype    domain.RRType
		qclass   domain.RRClass
		minBytes int
	}{
		{
			name:     "simple domain",
			domain:   "example.com.",
			qtype:    1,  // A
			qclass:   1,  // IN
			minBytes: 16, // 7+7+1+2+2 = domain labels + null + type + class
		},
		{
			name:     "subdomain",
			domain:   "www.example.com.",
			qtype:    28, // AAAA
			qclass:   1,  // IN
			minBytes: 20, // 3+7+3+1+2+2 = domain labels + null + type + class
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := resolver.encodeQuestion(tt.domain, tt.qtype, tt.qclass)
			if err != nil {
				t.Fatalf("failed to encode question: %v", err)
			}

			if len(encoded) < tt.minBytes {
				t.Errorf("encoded question too short: expected at least %d bytes, got %d", tt.minBytes, len(encoded))
			}

			// Verify null terminator exists
			found := false
			for i := 0; i < len(encoded)-4; i++ {
				if encoded[i] == 0x00 {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("null terminator not found in encoded domain name")
			}
		})
	}
}

func TestResolver_encodeQuestion_InvalidDomain(t *testing.T) {
	resolver := NewResolver([]string{"8.8.8.8:53"}, 5*time.Second)

	// Test with a label that's too long (>63 characters)
	longLabel := ""
	for i := 0; i < 64; i++ {
		longLabel += "a"
	}
	invalidDomain := longLabel + ".example.com."

	_, err := resolver.encodeQuestion(invalidDomain, 1, 1)
	if err == nil {
		t.Errorf("expected error for domain with label >63 characters, got nil")
	}
}

func TestResolver_decodeDNSResponse(t *testing.T) {
	resolver := NewResolver([]string{"8.8.8.8:53"}, 5*time.Second)

	// Create a minimal valid DNS response (NXDOMAIN)
	response := []byte{
		// Header (12 bytes)
		0x30, 0x39, // ID: 12345
		0x81, 0x83, // Flags: response, NXDOMAIN
		0x00, 0x01, // QDCOUNT: 1
		0x00, 0x00, // ANCOUNT: 0
		0x00, 0x00, // NSCOUNT: 0
		0x00, 0x00, // ARCOUNT: 0
		// Question section (minimal)
		0x07, 'e', 'x', 'a', 'm', 'p', 'l', 'e',
		0x03, 'c', 'o', 'm',
		0x00,       // null terminator
		0x00, 0x01, // type A
		0x00, 0x01, // class IN
	}

	decoded, err := resolver.decodeDNSResponse(response, 12345)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if decoded.ID != 12345 {
		t.Errorf("expected ID 12345, got %d", decoded.ID)
	}

	if decoded.RCode != 3 { // NXDOMAIN
		t.Errorf("expected RCODE 3 (NXDOMAIN), got %d", decoded.RCode)
	}

	if len(decoded.Answers) != 0 {
		t.Errorf("expected 0 answers, got %d", len(decoded.Answers))
	}
}

func TestResolver_decodeDNSResponse_Errors(t *testing.T) {
	resolver := NewResolver([]string{"8.8.8.8:53"}, 5*time.Second)

	tests := []struct {
		name       string
		data       []byte
		expectedID uint16
		expectErr  bool
	}{
		{
			name:       "too short",
			data:       []byte{0x30, 0x39}, // Only 2 bytes
			expectedID: 12345,
			expectErr:  true,
		},
		{
			name: "ID mismatch",
			data: []byte{
				0x99, 0x99, // Wrong ID
				0x81, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			expectedID: 12345,
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := resolver.decodeDNSResponse(tt.data, tt.expectedID)
			if tt.expectErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestResolver_ResolveWithTimeout(t *testing.T) {
	// Note: This test requires network connectivity and may be slow
	// In a real project, you'd want to mock the network calls
	t.Skip("Integration test - requires network connectivity")

	resolver := NewResolver([]string{"1.1.1.1:53"}, 5*time.Second)

	query, err := domain.NewDNSQuery(1, "cloudflare.com.", 1, 1) // A record
	if err != nil {
		t.Fatalf("failed to create test query: %v", err)
	}

	response, err := resolver.ResolveWithTimeout(query, 10*time.Second)
	if err != nil {
		t.Fatalf("failed to resolve query: %v", err)
	}

	if response.ID != 1 {
		t.Errorf("expected response ID 1, got %d", response.ID)
	}

	if response.RCode != 0 { // NOERROR
		t.Errorf("expected RCODE 0 (NOERROR), got %d", response.RCode)
	}
}

func TestResolver_Health(t *testing.T) {
	// Note: This test requires network connectivity
	t.Skip("Integration test - requires network connectivity")

	resolver := NewResolver([]string{"1.1.1.1:53"}, 5*time.Second)

	err := resolver.Health()
	if err != nil {
		t.Errorf("health check failed: %v", err)
	}
}

func TestResolver_Resolve_WithContext(t *testing.T) {
	resolver := NewResolver([]string{"invalid.server:53"}, 1*time.Second)

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	query, err := domain.NewDNSQuery(1, "example.com.", 1, 1)
	if err != nil {
		t.Fatalf("failed to create test query: %v", err)
	}

	_, err = resolver.Resolve(ctx, query)
	if err == nil {
		t.Errorf("expected error with cancelled context, got nil")
	}
}

func TestResolver_Resolve_AllServersFail(t *testing.T) {
	// Use invalid servers to force failure
	resolver := NewResolver([]string{"192.0.2.1:53", "192.0.2.2:53"}, 1*time.Second)

	query, err := domain.NewDNSQuery(1, "example.com.", 1, 1)
	if err != nil {
		t.Fatalf("failed to create test query: %v", err)
	}

	_, err = resolver.ResolveWithTimeout(query, 2*time.Second)
	if err == nil {
		t.Errorf("expected error when all servers fail, got nil")
	}

	// Verify error message mentions all servers failed
	if err != nil && len(err.Error()) == 0 {
		t.Errorf("error message should not be empty")
	}
}
