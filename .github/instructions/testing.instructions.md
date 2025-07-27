---
applyTo: "*_test.go"
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

- Services (e.g. `resolverService`) must be fully tested using mocks for repo interfaces.
- Use Go interfaces to inject mocks or fakes.
- Test edge cases: NXDOMAIN, multiple answers, empty result.

### Repos (`internal/dns/repo`)

- Interfaces don’t require tests.
- Implementations (e.g. `staticZoneRepo`) should be covered with focused tests.
- Validate logic like record filtering and matching.

### Infra (`internal/dns/infra`)

- Infra components (e.g. `udpServer`) should be tested using integration tests.
- Simulate actual UDP traffic to verify packet receipt and response.

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

resolver := resolverService{repo: repo}
resp := resolver.Resolve(query)
repo.AssertExpectations(t)
```

---

## Naming & Placement

- Test files are named `*_test.go`.
- Unit test files go next to implementation.
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
- Use mocks for interfaces in `service/`.
- Avoid depending on real infrastructure in `domain` or `service` tests.
- Name your test inputs and assert outcomes explicitly.
- Use `testify/mock` to mock interfaces like `ZoneRepository` in unit tests.
