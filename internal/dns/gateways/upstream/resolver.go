package upstream

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/haukened/rr-dns/internal/dns/domain"
	"github.com/haukened/rr-dns/internal/dns/gateways/wire"
	"github.com/haukened/rr-dns/internal/dns/services/resolver"
)

// Error message constants for consistent error handling
const (
	errNoServersProvided = "no upstream DNS servers provided"
	errCodecRequired     = "DNS codec is required"
	errServerFailed      = "server %s: %w"
	errAllServersFailed  = "all %d upstream servers failed"
	errQueryTimeout      = "query timeout after %v"
	errFailedToConnect   = "failed to connect: %w"
	errEncodeFailed      = "encode failed: %w"
	errWriteFailed       = "write failed: %w"
	errReadFailed        = "read failed: %w"
)

// Resolver implements upstream DNS resolution by forwarding queries to external DNS servers.
// It handles the low-level networking concerns of DNS over UDP while maintaining clean
// separation from the service layer business logic.
type Resolver struct {
	servers  []string      // List of upstream DNS servers (e.g., "1.1.1.1:53")
	timeout  time.Duration // Default timeout for DNS queries
	codec    wire.DNSCodec // Codec for encoding/decoding DNS messages
	parallel bool          // Whether to resolve queries in parallel
	dial     DialFunc      // Dial function to create network connections
}

// DialFunc defines a function type for establishing a network connection.
// It takes a context for cancellation, the network type (e.g., "tcp", "udp"),
// and the address to connect to, returning a net.Conn and an error if any occurs.
type DialFunc func(ctx context.Context, network, address string) (net.Conn, error)

// Options defines configuration parameters for the upstream DNS resolver.
// It includes the list of DNS servers to query, request timeout duration,
// DNS codec for encoding/decoding messages, whether to perform parallel queries,
// and a custom dial function for network connections.
type Options struct {
	// required parameters
	Servers  []string
	Timeout  time.Duration
	Parallel bool
	// options to inject for testing purposes
	Codec wire.DNSCodec
	Dial  DialFunc
}

// NewResolver creates a new upstream resolver with the specified options.
// Returns an error if the server list is empty or the codec is not provided.
// Sets default timeout to 5 seconds and default dial function if not provided.
func NewResolver(opts Options) (*Resolver, error) {
	if len(opts.Servers) == 0 {
		return nil, fmt.Errorf(errNoServersProvided)
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 5 * time.Second
	}
	if opts.Codec == nil {
		return nil, fmt.Errorf(errCodecRequired)
	}
	if opts.Dial == nil {
		opts.Dial = (&net.Dialer{}).DialContext
	}
	return &Resolver{
		servers:  opts.Servers,
		timeout:  opts.Timeout,
		codec:    opts.Codec,
		parallel: opts.Parallel,
		dial:     opts.Dial,
	}, nil
}

// ensureContextDeadline ensures the context has a deadline, adding the resolver's default timeout if needed.
// Returns the context (potentially with added timeout) and a cancel function if one was created.
func (r *Resolver) ensureContextDeadline(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); !ok {
		return context.WithTimeout(ctx, r.timeout)
	}
	return ctx, nil
}

// setTimeout sets the timeout duration for DNS queries.
// just for testing purposes, not part of the public API.
func (r *Resolver) setTimeout(d time.Duration) {
	if d > 0 {
		r.timeout = d
	}
}

// Resolve forwards a DNS query to upstream servers and returns the response.
// It tries either parallel or serial resolution depending on the Resolver's parallel flag.
// The method respects the deadline set in the context or applies the default timeout.
func (r *Resolver) Resolve(ctx context.Context, query domain.DNSQuery, now time.Time) (domain.DNSResponse, error) {
	ctx, cancel := r.ensureContextDeadline(ctx)
	if cancel != nil {
		defer cancel()
	}

	if r.parallel {
		return r.resolveWithContext(ctx, query, now)
	}
	return r.resolveSerialWithContext(ctx, query, now)
}

// resolveSerialWithContext attempts to query each server in order until one responds successfully.
func (r *Resolver) resolveSerialWithContext(ctx context.Context, query domain.DNSQuery, now time.Time) (domain.DNSResponse, error) {
	var lastErr error
	for _, server := range r.servers {
		resp, err := r.queryServerWithContext(ctx, server, query, now)
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}
	return domain.DNSResponse{}, fmt.Errorf(errAllServersFailed+": %w", len(r.servers), lastErr)
}

// resolveWithContext forwards a DNS query using parallel server attempts for better performance.
func (r *Resolver) resolveWithContext(ctx context.Context, query domain.DNSQuery, now time.Time) (domain.DNSResponse, error) {
	// Channel to receive the first successful response
	responseChan := make(chan domain.DNSResponse, 1)
	errorChan := make(chan error, len(r.servers))

	// Launch goroutines for each server
	for _, server := range r.servers {
		go func(srv string) {
			response, err := r.queryServerWithContext(ctx, srv, query, now)
			if err != nil {
				errorChan <- fmt.Errorf(errServerFailed, srv, err)
				return
			}

			// Try to send response (non-blocking)
			select {
			case responseChan <- response:
				// Response sent successfully
			default:
				// Another goroutine already sent a response
			}
		}(server)
	}

	// Wait for first success or all failures
	var errors []error
	for i := 0; i < len(r.servers); i++ {
		select {
		case response := <-responseChan:
			return response, nil
		case err := <-errorChan:
			errors = append(errors, err)
		case <-ctx.Done():
			return domain.DNSResponse{}, fmt.Errorf(errQueryTimeout, r.timeout)
		}
	}

	// All servers failed
	return domain.DNSResponse{}, fmt.Errorf(errAllServersFailed+": %v", len(r.servers), errors)
}

// queryServerWithContext performs DNS query with context cancellation support.
func (r *Resolver) queryServerWithContext(ctx context.Context, server string, query domain.DNSQuery, now time.Time) (domain.DNSResponse, error) {
	// Create UDP connection
	conn, err := r.dial(ctx, "udp", server)
	if err != nil {
		return domain.DNSResponse{}, fmt.Errorf(errFailedToConnect, err)
	}
	defer conn.Close()

	// Set deadline from context
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}

	// Encode and send query
	queryBytes, err := r.codec.EncodeQuery(query)
	if err != nil {
		return domain.DNSResponse{}, fmt.Errorf(errEncodeFailed, err)
	}

	// Use goroutine for write/read to enable context cancellation
	type result struct {
		response domain.DNSResponse
		err      error
	}

	resultChan := make(chan result, 1)

	go func() {
		// Send query
		_, err := conn.Write(queryBytes)
		if err != nil {
			resultChan <- result{err: fmt.Errorf(errWriteFailed, err)}
			return
		}

		// Read response
		buffer := make([]byte, 512)
		n, err := conn.Read(buffer)
		if err != nil {
			resultChan <- result{err: fmt.Errorf(errReadFailed, err)}
			return
		}

		// Decode response
		response, err := r.codec.DecodeResponse(buffer[:n], query.ID, now)
		resultChan <- result{response: response, err: err}
	}()

	// Wait for result or context cancellation
	select {
	case res := <-resultChan:
		return res.response, res.err
	case <-ctx.Done():
		return domain.DNSResponse{}, ctx.Err()
	}
}

var _ resolver.UpstreamClient = (*Resolver)(nil)
