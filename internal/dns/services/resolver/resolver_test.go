package resolver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/haukened/rr-dns/internal/dns/common/clock"
	"github.com/haukened/rr-dns/internal/dns/domain"
)

// aliasTestLogger is a minimal logger for alias classification tests.
type aliasTestLogger struct{}

func (l *aliasTestLogger) Info(map[string]any, string)  {}
func (l *aliasTestLogger) Error(map[string]any, string) {}
func (l *aliasTestLogger) Debug(map[string]any, string) {}
func (l *aliasTestLogger) Warn(map[string]any, string)  {}
func (l *aliasTestLogger) Panic(map[string]any, string) {}
func (l *aliasTestLogger) Fatal(map[string]any, string) {}

// stubAliasResolver allows injecting predetermined records + error.
type stubAliasResolver struct {
	recs []domain.ResourceRecord
	err  error
}

func (s *stubAliasResolver) Chase(domain.Question, []domain.ResourceRecord) ([]domain.ResourceRecord, error) {
	return s.recs, s.err
}

// ensure interfaces compile (alias resolver stub only; zone uses existing stubZoneCache from benchmarks)
var _ AliasResolver = (*stubAliasResolver)(nil)

func newTestCNAME(t *testing.T, name, target string) domain.ResourceRecord {
	rr, err := domain.NewAuthoritativeResourceRecord(name, domain.RRTypeCNAME, domain.RRClassIN, 300, nil, target)
	assert.NoError(t, err)
	return rr
}

func TestResolver_HandleQuery_AliasFatalErrorServfail(t *testing.T) {
	cname := newTestCNAME(t, "example.com.", "target.example.")
	zone := &stubZoneCache{records: []domain.ResourceRecord{cname}, found: true}
	ar := &stubAliasResolver{recs: []domain.ResourceRecord{cname}, err: ErrAliasDepthExceeded}
	clk := &clock.MockClock{CurrentTime: time.Now()}
	r := NewResolver(ResolverOptions{ZoneCache: zone, AliasResolver: ar, Clock: clk, Logger: &aliasTestLogger{}})
	q, err := domain.NewQuestion(1, "example.com.", domain.RRTypeA, domain.RRClassIN)
	assert.NoError(t, err)
	resp, hErr := r.HandleQuery(context.Background(), q, nil)
	assert.NoError(t, hErr)
	assert.Equal(t, domain.SERVFAIL, resp.RCode)
	assert.Len(t, resp.Answers, 0, "fatal alias error should suppress answers")
}

func TestResolver_HandleQuery_AliasNonFatalErrorPartialChain(t *testing.T) {
	cname := newTestCNAME(t, "example.com.", "target.example.")
	zone := &stubZoneCache{records: []domain.ResourceRecord{cname}, found: true}
	nfErr := fmt.Errorf("%w: empty target", ErrAliasTargetInvalid)
	ar := &stubAliasResolver{recs: []domain.ResourceRecord{cname}, err: nfErr}
	clk := &clock.MockClock{CurrentTime: time.Now()}
	r := NewResolver(ResolverOptions{ZoneCache: zone, AliasResolver: ar, Clock: clk, Logger: &aliasTestLogger{}})
	q, err := domain.NewQuestion(2, "example.com.", domain.RRTypeA, domain.RRClassIN)
	assert.NoError(t, err)
	resp, hErr := r.HandleQuery(context.Background(), q, nil)
	assert.NoError(t, hErr)
	assert.Equal(t, domain.NOERROR, resp.RCode)
	assert.Equal(t, []domain.ResourceRecord{cname}, resp.Answers)
}

// Mock implementations for testing
type MockBlocklist struct {
	mock.Mock
}

func (m *MockBlocklist) IsBlocked(q domain.Question) bool {
	args := m.Called(q)
	return args.Bool(0)
}

// noopLogger is a test logger that discards all messages
type noopLogger struct{}

func (n *noopLogger) Info(map[string]any, string)  {}
func (n *noopLogger) Error(map[string]any, string) {}
func (n *noopLogger) Debug(map[string]any, string) {}
func (n *noopLogger) Warn(map[string]any, string)  {}
func (n *noopLogger) Panic(map[string]any, string) {}
func (n *noopLogger) Fatal(map[string]any, string) {}

type MockCache struct {
	mock.Mock
}

func (m *MockCache) Set(record []domain.ResourceRecord) error {
	args := m.Called(record)
	return args.Error(0)
}

func (m *MockCache) Get(key string) ([]domain.ResourceRecord, bool) {
	args := m.Called(key)
	return args.Get(0).([]domain.ResourceRecord), args.Bool(1)
}

func (m *MockCache) Delete(key string) {
	m.Called(key)
}

func (m *MockCache) Len() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockCache) Keys() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

type MockUpstreamClient struct {
	mock.Mock
}

func (m *MockUpstreamClient) Resolve(ctx context.Context, query domain.Question, now time.Time) ([]domain.ResourceRecord, error) {
	args := m.Called(ctx, query, now)
	return args.Get(0).([]domain.ResourceRecord), args.Error(1)
}

type MockZoneCache struct {
	mock.Mock
}

func (m *MockZoneCache) FindRecords(query domain.Question) ([]domain.ResourceRecord, bool) {
	args := m.Called(query)
	return args.Get(0).([]domain.ResourceRecord), args.Bool(1)
}

func (m *MockZoneCache) PutZone(zoneRoot string, records []domain.ResourceRecord) {
	m.Called(zoneRoot, records)
}

func (m *MockZoneCache) RemoveZone(zoneRoot string) {
	m.Called(zoneRoot)
}

func (m *MockZoneCache) Zones() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockZoneCache) Count() int {
	args := m.Called()
	return args.Int(0)
}

// Test helpers
func createTestQuery(name string, qtype domain.RRType) domain.Question {
	query, _ := domain.NewQuestion(1, name, qtype, domain.RRClass(1)) // IN class
	return query
}

func createTestRecord(name string, rtype domain.RRType, data []byte, text string) domain.ResourceRecord {
	record, _ := domain.NewCachedResourceRecord(name, rtype, domain.RRClass(1), 300, data, text, time.Now())
	return record
}

func TestResolver_HandleQuery_AuthoritativeZone(t *testing.T) {
	tests := []struct {
		name          string
		query         domain.Question
		zoneRecords   []domain.ResourceRecord
		zoneFound     bool
		expectedRCode domain.RCode
		expectedCount int
	}{
		{
			name:          "authoritative A record found",
			query:         createTestQuery("example.com.", domain.RRType(1)), // A record
			zoneRecords:   []domain.ResourceRecord{createTestRecord("example.com.", domain.RRType(1), []byte{192, 0, 2, 1}, "192.0.2.1")},
			zoneFound:     true,
			expectedRCode: domain.NOERROR,
			expectedCount: 1,
		},
		{
			name:          "authoritative AAAA record found",
			query:         createTestQuery("example.com.", domain.RRType(28)), // AAAA record
			zoneRecords:   []domain.ResourceRecord{createTestRecord("example.com.", domain.RRType(28), make([]byte, 16), "2001:db8::1")},
			zoneFound:     true,
			expectedRCode: domain.NOERROR,
			expectedCount: 1,
		},
		{
			name:  "multiple authoritative records",
			query: createTestQuery("example.com.", domain.RRType(1)), // A record
			zoneRecords: []domain.ResourceRecord{
				createTestRecord("example.com.", domain.RRType(1), []byte{192, 0, 2, 1}, "192.0.2.1"),
				createTestRecord("example.com.", domain.RRType(1), []byte{192, 0, 2, 2}, "192.0.2.2"),
			},
			zoneFound:     true,
			expectedRCode: domain.NOERROR,
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockZoneCache := &MockZoneCache{}
			mockBlocklist := &MockBlocklist{}
			mockUpstreamCache := &MockCache{}
			mockUpstream := &MockUpstreamClient{}
			mockClock := &clock.MockClock{
				CurrentTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			}
			mockLogger := &noopLogger{}

			// Configure zone cache expectations
			mockZoneCache.On("FindRecords", tt.query).Return(tt.zoneRecords, tt.zoneFound)

			// Create resolver
			resolver := NewResolver(ResolverOptions{
				Blocklist:     mockBlocklist,
				Clock:         mockClock,
				Logger:        mockLogger,
				Upstream:      mockUpstream,
				UpstreamCache: mockUpstreamCache,
				ZoneCache:     mockZoneCache,
			})

			// Execute
			ctx := context.Background()
			clientAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
			response, err := resolver.HandleQuery(ctx, tt.query, clientAddr)

			// Verify
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedRCode, response.RCode)
			assert.Equal(t, tt.expectedCount, len(response.Answers))
			assert.Equal(t, tt.query.ID, response.ID)

			// Verify mocks
			mockZoneCache.AssertExpectations(t)
		})
	}
}

func TestResolver_HandleQuery_Blocklist(t *testing.T) {
	tests := []struct {
		name          string
		query         domain.Question
		isBlocked     bool
		expectedRCode domain.RCode
	}{
		{
			name:          "query blocked",
			query:         createTestQuery("malware.com.", domain.RRType(1)),
			isBlocked:     true,
			expectedRCode: domain.NXDOMAIN,
		},
		{
			name:          "query not blocked",
			query:         createTestQuery("google.com.", domain.RRType(1)),
			isBlocked:     false,
			expectedRCode: domain.SERVFAIL, // Will fail upstream since we don't mock it
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockZoneCache := &MockZoneCache{}
			mockBlocklist := &MockBlocklist{}
			mockUpstreamCache := &MockCache{}
			mockUpstream := &MockUpstreamClient{}
			mockClock := &clock.MockClock{
				CurrentTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			}
			mockLogger := &noopLogger{}

			// Configure expectations
			mockZoneCache.On("FindRecords", tt.query).Return([]domain.ResourceRecord{}, false)
			mockBlocklist.On("IsBlocked", tt.query).Return(tt.isBlocked)

			if !tt.isBlocked {
				// If not blocked, will check upstream cache and then upstream
				mockUpstreamCache.On("Get", tt.query.CacheKey()).Return([]domain.ResourceRecord{}, false)
				mockUpstream.On("Resolve", mock.Anything, tt.query, mock.Anything).Return([]domain.ResourceRecord(nil), errors.New("upstream error"))
			}

			// Create resolver
			resolver := NewResolver(ResolverOptions{
				Blocklist:     mockBlocklist,
				Clock:         mockClock,
				Logger:        mockLogger,
				Upstream:      mockUpstream,
				UpstreamCache: mockUpstreamCache,
				ZoneCache:     mockZoneCache,
			})

			// Execute
			ctx := context.Background()
			clientAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
			response, err := resolver.HandleQuery(ctx, tt.query, clientAddr)

			// Verify
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedRCode, response.RCode)
			assert.Equal(t, tt.query.ID, response.ID)

			// Verify mocks
			mockZoneCache.AssertExpectations(t)
			mockBlocklist.AssertExpectations(t)
			if !tt.isBlocked {
				mockUpstreamCache.AssertExpectations(t)
				mockUpstream.AssertExpectations(t)
			}
		})
	}
}

func TestResolver_HandleQuery_UpstreamCache(t *testing.T) {
	tests := []struct {
		name          string
		query         domain.Question
		cachedRecords []domain.ResourceRecord
		cacheHit      bool
		expectedRCode domain.RCode
		expectedCount int
	}{
		{
			name:          "upstream cache hit",
			query:         createTestQuery("cached.com.", domain.RRType(1)), // A record
			cachedRecords: []domain.ResourceRecord{createTestRecord("cached.com.", domain.RRType(1), []byte{192, 0, 2, 1}, "192.0.2.1")},
			cacheHit:      true,
			expectedRCode: domain.NOERROR,
			expectedCount: 1,
		},
		{
			name:          "upstream cache miss",
			query:         createTestQuery("uncached.com.", domain.RRType(1)), // A record
			cachedRecords: []domain.ResourceRecord{},
			cacheHit:      false,
			expectedRCode: domain.SERVFAIL, // Will fail upstream since we don't mock successful resolution
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockZoneCache := &MockZoneCache{}
			mockBlocklist := &MockBlocklist{}
			mockUpstreamCache := &MockCache{}
			mockUpstream := &MockUpstreamClient{}
			mockClock := &clock.MockClock{
				CurrentTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			}
			mockLogger := &noopLogger{}

			// Configure expectations
			mockZoneCache.On("FindRecords", tt.query).Return([]domain.ResourceRecord{}, false)
			mockBlocklist.On("IsBlocked", tt.query).Return(false)
			mockUpstreamCache.On("Get", tt.query.CacheKey()).Return(tt.cachedRecords, tt.cacheHit)

			if !tt.cacheHit {
				// Cache miss, will go to upstream
				mockUpstream.On("Resolve", mock.Anything, tt.query, mock.Anything).Return([]domain.ResourceRecord(nil), errors.New("upstream error"))
			}

			// Create resolver
			resolver := NewResolver(ResolverOptions{
				Blocklist:     mockBlocklist,
				Clock:         mockClock,
				Logger:        mockLogger,
				Upstream:      mockUpstream,
				UpstreamCache: mockUpstreamCache,
				ZoneCache:     mockZoneCache,
			})

			// Execute
			ctx := context.Background()
			clientAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
			response, err := resolver.HandleQuery(ctx, tt.query, clientAddr)

			// Verify
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedRCode, response.RCode)
			assert.Equal(t, tt.expectedCount, len(response.Answers))
			assert.Equal(t, tt.query.ID, response.ID)

			// Verify mocks
			mockZoneCache.AssertExpectations(t)
			mockBlocklist.AssertExpectations(t)
			mockUpstreamCache.AssertExpectations(t)
			if !tt.cacheHit {
				mockUpstream.AssertExpectations(t)
			}
		})
	}
}

func TestResolver_HandleQuery_UpstreamResolution(t *testing.T) {
	tests := []struct {
		name            string
		query           domain.Question
		upstreamRecords []domain.ResourceRecord
		upstreamErr     error
		cacheSetErr     error
		expectedRCode   domain.RCode
		expectedCount   int
		shouldCallCache bool
	}{
		{
			name:            "successful upstream resolution with caching",
			query:           createTestQuery("upstream.com.", domain.RRType(1)), // A record
			upstreamRecords: []domain.ResourceRecord{createTestRecord("upstream.com.", domain.RRType(1), []byte{192, 0, 2, 1}, "192.0.2.1")},
			upstreamErr:     nil,
			cacheSetErr:     nil,
			expectedRCode:   domain.NOERROR,
			expectedCount:   1,
			shouldCallCache: true,
		},
		{
			name:            "upstream resolution failure",
			query:           createTestQuery("failed.com.", domain.RRType(1)), // A record
			upstreamRecords: nil,
			upstreamErr:     errors.New("network timeout"),
			cacheSetErr:     nil,
			expectedRCode:   domain.SERVFAIL,
			expectedCount:   0,
			shouldCallCache: false,
		},
		{
			name:            "successful upstream resolution with cache error",
			query:           createTestQuery("cache-fail.com.", domain.RRType(1)), // A record
			upstreamRecords: []domain.ResourceRecord{createTestRecord("cache-fail.com.", domain.RRType(1), []byte{192, 0, 2, 1}, "192.0.2.1")},
			upstreamErr:     nil,
			cacheSetErr:     errors.New("cache full"),
			expectedRCode:   domain.NOERROR,
			expectedCount:   1,
			shouldCallCache: true,
		},
		{
			name:            "successful upstream resolution with nil cache",
			query:           createTestQuery("nil-cache.com.", domain.RRType(1)), // A record
			upstreamRecords: []domain.ResourceRecord{createTestRecord("nil-cache.com.", domain.RRType(1), []byte{192, 0, 2, 1}, "192.0.2.1")},
			upstreamErr:     nil,
			cacheSetErr:     nil,
			expectedRCode:   domain.NOERROR,
			expectedCount:   1,
			shouldCallCache: false, // nil cache, so no Get() or Set() calls
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockZoneCache := &MockZoneCache{}
			mockBlocklist := &MockBlocklist{}
			var mockUpstreamCache Cache
			if tt.shouldCallCache {
				mockUpstreamCache = &MockCache{}
			} else {
				mockUpstreamCache = nil // Test nil cache path
			}
			mockUpstream := &MockUpstreamClient{}
			mockClock := &clock.MockClock{
				CurrentTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			}
			mockLogger := &noopLogger{}

			// Configure expectations
			mockZoneCache.On("FindRecords", tt.query).Return([]domain.ResourceRecord{}, false)
			mockBlocklist.On("IsBlocked", tt.query).Return(false)

			if mockUpstreamCache != nil {
				mockUpstreamCache.(*MockCache).On("Get", tt.query.CacheKey()).Return([]domain.ResourceRecord{}, false)
			}

			mockUpstream.On("Resolve", mock.Anything, tt.query, mock.Anything).Return(tt.upstreamRecords, tt.upstreamErr)

			if tt.shouldCallCache && mockUpstreamCache != nil && tt.upstreamErr == nil {
				mockUpstreamCache.(*MockCache).On("Set", tt.upstreamRecords).Return(tt.cacheSetErr)
			}

			// Create resolver
			resolver := NewResolver(ResolverOptions{
				Blocklist:     mockBlocklist,
				Clock:         mockClock,
				Logger:        mockLogger,
				Upstream:      mockUpstream,
				UpstreamCache: mockUpstreamCache,
				ZoneCache:     mockZoneCache,
			})

			// Execute
			ctx := context.Background()
			clientAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
			response, err := resolver.HandleQuery(ctx, tt.query, clientAddr)

			// Verify
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedRCode, response.RCode)
			assert.Equal(t, tt.expectedCount, len(response.Answers))
			assert.Equal(t, tt.query.ID, response.ID)

			// Verify mocks
			mockZoneCache.AssertExpectations(t)
			mockBlocklist.AssertExpectations(t)
			if mockUpstreamCache != nil {
				mockUpstreamCache.(*MockCache).AssertExpectations(t)
			}
			mockUpstream.AssertExpectations(t)
		})
	}
}

func TestResolver_HandleQuery_ContextCancellation(t *testing.T) {
	// Setup mocks
	mockZoneCache := &MockZoneCache{}
	mockBlocklist := &MockBlocklist{}
	mockUpstreamCache := &MockCache{}
	mockUpstream := &MockUpstreamClient{}
	mockClock := &clock.MockClock{
		CurrentTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	mockLogger := &noopLogger{}

	query := createTestQuery("timeout.com.", domain.RRType(1)) // A record

	// Configure expectations
	mockZoneCache.On("FindRecords", query).Return([]domain.ResourceRecord{}, false)
	mockBlocklist.On("IsBlocked", query).Return(false)
	mockUpstreamCache.On("Get", query.CacheKey()).Return([]domain.ResourceRecord{}, false)
	mockUpstream.On("Resolve", mock.Anything, query, mock.Anything).Return([]domain.ResourceRecord(nil), context.Canceled)

	// Create resolver
	resolver := NewResolver(ResolverOptions{
		Blocklist:     mockBlocklist,
		Clock:         mockClock,
		Logger:        mockLogger,
		Upstream:      mockUpstream,
		UpstreamCache: mockUpstreamCache,
		ZoneCache:     mockZoneCache,
	})

	// Execute with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	clientAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
	response, err := resolver.HandleQuery(ctx, query, clientAddr)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, domain.SERVFAIL, response.RCode)
	assert.Equal(t, 0, len(response.Answers))
	assert.Equal(t, query.ID, response.ID)

	// Verify mocks
	mockZoneCache.AssertExpectations(t)
	mockBlocklist.AssertExpectations(t)
	mockUpstreamCache.AssertExpectations(t)
	mockUpstream.AssertExpectations(t)
}

func TestResolver_HandleQuery_NilDependencies(t *testing.T) {
	tests := []struct {
		name          string
		blocklist     Blocklist
		upstreamCache Cache
		zoneCache     ZoneCache
		upstream      UpstreamClient
		expectedRCode domain.RCode
	}{
		{
			name:          "nil zone cache",
			blocklist:     &MockBlocklist{},
			upstreamCache: &MockCache{},
			zoneCache:     nil,
			upstream:      &MockUpstreamClient{},
			expectedRCode: domain.SERVFAIL, // Will fail at upstream
		},
		{
			name:          "nil blocklist",
			blocklist:     nil,
			upstreamCache: &MockCache{},
			zoneCache:     &MockZoneCache{},
			upstream:      &MockUpstreamClient{},
			expectedRCode: domain.SERVFAIL, // Will fail at upstream
		},
		{
			name:          "nil upstream cache",
			blocklist:     &MockBlocklist{},
			upstreamCache: nil,
			zoneCache:     &MockZoneCache{},
			upstream:      &MockUpstreamClient{},
			expectedRCode: domain.SERVFAIL, // Will fail at upstream
		},
		{
			name:          "nil upstream client",
			blocklist:     &MockBlocklist{},
			upstreamCache: &MockCache{},
			zoneCache:     &MockZoneCache{},
			upstream:      nil,
			expectedRCode: domain.SERVFAIL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := createTestQuery("test.com.", domain.RRType(1)) // A record
			mockClock := &clock.MockClock{
				CurrentTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			}
			mockLogger := &noopLogger{}

			// Configure non-nil mocks
			if zc, ok := tt.zoneCache.(*MockZoneCache); ok {
				zc.On("FindRecords", query).Return([]domain.ResourceRecord{}, false)
			}
			if bl, ok := tt.blocklist.(*MockBlocklist); ok {
				bl.On("IsBlocked", query).Return(false)
			}
			if uc, ok := tt.upstreamCache.(*MockCache); ok {
				uc.On("Get", query.CacheKey()).Return([]domain.ResourceRecord{}, false)
			}
			if up, ok := tt.upstream.(*MockUpstreamClient); ok {
				up.On("Resolve", mock.Anything, query, mock.Anything).Return([]domain.ResourceRecord(nil), errors.New("upstream error"))
			}

			// Create resolver
			resolver := NewResolver(ResolverOptions{
				Blocklist:     tt.blocklist,
				Clock:         mockClock,
				Logger:        mockLogger,
				Upstream:      tt.upstream,
				UpstreamCache: tt.upstreamCache,
				ZoneCache:     tt.zoneCache,
			})

			// Execute
			ctx := context.Background()
			clientAddr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
			response, err := resolver.HandleQuery(ctx, query, clientAddr)

			// Verify
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedRCode, response.RCode)
			assert.Equal(t, query.ID, response.ID)

			// Verify mocks (only non-nil ones)
			if zc, ok := tt.zoneCache.(*MockZoneCache); ok {
				zc.AssertExpectations(t)
			}
			if bl, ok := tt.blocklist.(*MockBlocklist); ok {
				bl.AssertExpectations(t)
			}
			if uc, ok := tt.upstreamCache.(*MockCache); ok {
				uc.AssertExpectations(t)
			}
			if up, ok := tt.upstream.(*MockUpstreamClient); ok {
				up.AssertExpectations(t)
			}
		})
	}
}

func TestNewResolver(t *testing.T) {
	mockBlocklist := &MockBlocklist{}
	mockCache := &MockCache{}
	mockUpstream := &MockUpstreamClient{}
	mockZoneCache := &MockZoneCache{}
	mockClock := &clock.MockClock{
		CurrentTime: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	mockLogger := &noopLogger{}
	alias := NewNoOpAliasResolver()

	opts := ResolverOptions{
		Blocklist:     mockBlocklist,
		Clock:         mockClock,
		Logger:        mockLogger,
		Upstream:      mockUpstream,
		UpstreamCache: mockCache,
		ZoneCache:     mockZoneCache,
		AliasResolver: alias,
	}

	resolver := NewResolver(opts)

	assert.NotNil(t, resolver)
	assert.Equal(t, mockBlocklist, resolver.blocklist)
	assert.Equal(t, mockClock, resolver.clock)
	assert.Equal(t, mockLogger, resolver.logger)
	assert.Equal(t, mockUpstream, resolver.upstream)
	assert.Equal(t, mockCache, resolver.upstreamCache)
	assert.Equal(t, mockZoneCache, resolver.zoneCache)
	assert.Equal(t, alias, resolver.aliasResolver)
}

func TestBuildResponse(t *testing.T) {
	query := createTestQuery("test.com.", domain.RRType(1)) // A record
	records := []domain.ResourceRecord{
		createTestRecord("test.com.", domain.RRType(1), []byte{192, 0, 2, 1}, "192.0.2.1"),
	}

	tests := []struct {
		name          string
		query         domain.Question
		rcode         domain.RCode
		records       []domain.ResourceRecord
		expectedID    uint16
		expectedRCode domain.RCode
		expectedCount int
	}{
		{
			name:          "successful response with records",
			query:         query,
			rcode:         domain.NOERROR,
			records:       records,
			expectedID:    query.ID,
			expectedRCode: domain.NOERROR,
			expectedCount: 1,
		},
		{
			name:          "error response without records",
			query:         query,
			rcode:         domain.NXDOMAIN,
			records:       nil,
			expectedID:    query.ID,
			expectedRCode: domain.NXDOMAIN,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := buildResponse(tt.query, tt.rcode, tt.records)

			assert.Equal(t, tt.expectedID, response.ID)
			assert.Equal(t, tt.expectedRCode, response.RCode)
			assert.Equal(t, tt.expectedCount, len(response.Answers))
		})
	}
}

func TestResolver_CacheUpstreamResponse(t *testing.T) {
	tests := []struct {
		name          string
		upstreamCache Cache
		records       []domain.ResourceRecord
		expectError   bool
		expectCall    bool
	}{
		{
			name:          "successful caching",
			upstreamCache: &MockCache{},
			records:       []domain.ResourceRecord{createTestRecord("test.com.", domain.RRType(1), []byte{192, 0, 2, 1}, "192.0.2.1")},
			expectError:   false,
			expectCall:    true,
		},
		{
			name:          "cache error",
			upstreamCache: &MockCache{},
			records:       []domain.ResourceRecord{createTestRecord("test.com.", domain.RRType(1), []byte{192, 0, 2, 1}, "192.0.2.1")},
			expectError:   true,
			expectCall:    true,
		},
		{
			name:          "nil cache - no error",
			upstreamCache: nil,
			records:       []domain.ResourceRecord{createTestRecord("test.com.", domain.RRType(1), []byte{192, 0, 2, 1}, "192.0.2.1")},
			expectError:   false,
			expectCall:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &Resolver{
				upstreamCache: tt.upstreamCache,
			}

			if tt.expectCall && tt.upstreamCache != nil {
				mockCache := tt.upstreamCache.(*MockCache)
				if tt.expectError {
					mockCache.On("Set", tt.records).Return(errors.New("cache error"))
				} else {
					mockCache.On("Set", tt.records).Return(nil)
				}
			}

			err := resolver.cacheUpstreamResponse(tt.records)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.expectCall && tt.upstreamCache != nil {
				tt.upstreamCache.(*MockCache).AssertExpectations(t)
			}
		})
	}
}
func TestResolver_isFatalAliasError(t *testing.T) {
	resolver := &Resolver{}

	tests := []struct {
		name      string
		err       error
		wantFatal bool
	}{
		{
			name:      "nil error is not fatal",
			err:       nil,
			wantFatal: false,
		},
		{
			name:      "ErrAliasDepthExceeded is fatal",
			err:       ErrAliasDepthExceeded,
			wantFatal: true,
		},
		{
			name:      "ErrAliasLoopDetected is fatal",
			err:       ErrAliasLoopDetected,
			wantFatal: true,
		},
		{
			name:      "wrapped ErrAliasDepthExceeded is fatal",
			err:       fmt.Errorf("wrapped: %w", ErrAliasDepthExceeded),
			wantFatal: true,
		},
		{
			name:      "wrapped ErrAliasLoopDetected is fatal",
			err:       fmt.Errorf("wrapped: %w", ErrAliasLoopDetected),
			wantFatal: true,
		},
		{
			name:      "other error is not fatal",
			err:       errors.New("some other error"),
			wantFatal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.isFatalAliasError(tt.err)
			assert.Equal(t, tt.wantFatal, got)
		})
	}
}
