---
applyTo: "**/*.go"
---

# Testing Guidelines for rr-dns

This guide defines how to structure and write tests for the rr-dns project. It supports Copilot and contributors in producing high-quality, maintainable, and properly isolated tests.

---

## Testing Philosophy

- Test behavior, not implementation details.
- Keep domain and service layers fully unit tested.
- Infrastructure tests should verify integration (e.g., UDP packet flow).
- Prefer table-driven tests for coverage and clarity.
- Keep test setup explicit and small.

---

## Layer-Specific Testing

### Domain (`internal/dns/domain`)

- All domain types (e.g. `DNSQuery`, `ResourceRecord`) must have unit tests.
- Focus on validation, equality, construction logic.
- Tests should be deterministic, pure, and have no dependencies.

### Services (`internal/dns/service`)

- Services (e.g. `resolverService`) must be fully tested using mocks for repository and gateway interfaces.
- Use Go interfaces to inject mocks or fakes.
- Test edge cases: NXDOMAIN, multiple answers, empty result.

### Common (`internal/dns/common`)

- Shared utilities and infrastructure components should have unit tests.
- Test logging configuration, utility functions, and cross-cutting concerns.
- Keep tests isolated from external dependencies.

### Config (`internal/dns/config`)

- Test configuration parsing, validation, and default value handling.
- Test environment variable loading and type conversion.
- Verify error handling for invalid configurations.

### Gateways (`internal/dns/gateways`)

- Test external system integrations with both unit and integration tests.
- Use mocks for unit testing of transport, upstream, and wire format logic.
- Integration tests should verify real network communication (with skip flags).

### Repos (`internal/dns/repos`)

- Repository implementations (e.g. `zoneRepo`, `cacheRepo`) should be covered with focused tests.
- Test data access patterns, caching behavior, and persistence logic.
- Validate logic like record filtering, matching, and TTL handling.

### Mocking

rr-dns uses `github.com/stretchr/testify/mock` for mocking interfaces in tests.

To create a mock:
```go
type MockZoneRepo struct {
    mock.Mock
}

func (m *MockZoneRepo) FindRecords(name string, qtype RRType) ([]ResourceRecord, error) {
    args := m.Called(name, qtype)
    return args.Get(0).([]ResourceRecord), args.Error(1)
}
```

To use in a test:
```go
repo := new(MockZoneRepo)
repo.On("FindRecords", "example.com.", A).Return([]ResourceRecord{...}, nil)

resolver := resolverService{zoneRepo: repo}
resp := resolver.Resolve(query)
repo.AssertExpectations(t)
```

---

## Naming & Placement

- Test files are named `*_test.go`.
- Benchmark files are named `*_bench_test.go`.
- Test files are placed in the same package as the code they test.
- Test functions are named `TestXxx`.
- Use descriptive subtests with `t.Run`.

---

## Example Pattern

```go
func TestResolver_Resolve(t *testing.T) {
  tests := []struct {
    name     string
    query    DNSQuery
    wantCode RCODE
  }{
    {"valid A record", queryA, NOERROR},
    {"nonexistent name", queryNX, NXDOMAIN},
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      got := resolver.Resolve(tt.query)
      if got.RCode != tt.wantCode {
        t.Errorf("got %v, want %v", got.RCode, tt.wantCode)
      }
    })
  }
}
```

---

## Copilot Guidance

When writing tests:
- Use table-driven tests for all services.
- Keep test cases short and focused.
- Use mocks for repository and gateway interfaces in `service/` tests.
- Avoid depending on real infrastructure in `domain`, `service`, or `common` tests.
- Test configuration parsing and validation in `config/` tests.
- Use integration tests for `gateways/` components with real external systems.
- Test data access patterns and caching behavior in `repos/` tests.
- Name your test inputs and assert outcomes explicitly.
- Use `testify/mock` to mock interfaces like `ZoneRepository` and `ServerTransport` in unit tests.
