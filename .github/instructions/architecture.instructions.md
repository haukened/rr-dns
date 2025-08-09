---
applyTo: "**"
---

# CLEAN Architecture in Go

This guide defines how to implement CLEAN architecture in Golang for the rr-dns project. It is intended to help GitHub Copilot produce consistent, idiomatic, and testable code aligned with the project's goals.

## Principles

CLEAN architecture emphasizes separation of concerns and inward-facing dependencies:

- **Entities (Domain)**: Pure business/data logic
- **Use Cases (Services)**: Application logic
- **Interfaces (Gateways/Repos)**: Abstractions for external systems
- **Infrastructure**: External I/O, implementations of interfaces
- **Entrypoints**: CLI, API, or server setup

## Project Structure

```
/cmd/rr-dnsd            # Main program entrypoint

/internal
  /dns
  /domain            # Core types like Question, DNSResponse
    /service           # Business logic, e.g. Resolver
    /common            # Shared infrastructure (logging, utilities)
    /config            # Configuration management
    /gateways          # External system integrations (transport, upstream, wire)
    /repos             # Data repositories (cache, zones, blocklist)

/docs                  # Architecture, design, and planning
/pkg                   # Shared libraries (optional)
```

## Rules for CLEAN Code in Go

- Domain objects (`domain/`) must be pure Go structs with no external dependencies.
- Services (`service/`) may only depend on `domain` and repository interfaces.
- Common (`common/`) provides shared infrastructure used across layers.
- Config (`config/`) handles environment-based configuration management.
- Gateways (`gateways/`) implement interfaces for external system communication.
- Repos (`repos/`) implement repository patterns for data access.
- Entry points (`cmd/rr-dnsd`) wire everything together.


## Naming Conventions

- Interface: `ZoneRepository`, `QueryResolver`, `ServerTransport`
- Implementation: `staticZoneRepo`, `resolverService`, `udpTransport`
- Package names are lowercase, no underscores.

## Copilot Guidance

When generating new code:
- Place domain models in `internal/dns/domain`
- Place resolver logic in `internal/dns/service`
- Put shared utilities in `common`, configuration in `config`
- Implement external integrations in `gateways`, data access in `repos`
- Avoid importing infrastructure into `service` or `domain`
- Add unit tests for each `service` method


# Architecture Boundaries for rr-dns

This file describes the architectural boundaries enforced by the rr-dns project. These constraints are essential for maintaining testability, maintainability, and the separation of concerns defined by the CLEAN architecture.

---

## Project Structure and Layer Responsibilities

```
cmd/
  rr-dnsd/           # Entrypoint, DI setup

internal/
  dns/
    domain/          # Pure domain models and value types
    service/         # Application logic (resolvers, orchestration)
    common/          # Shared infrastructure (logging, utilities)
    config/          # Configuration management 
    gateways/        # External system integrations (transport, upstream, wire)
    repos/           # Data repositories (cache, zones, blocklist)
```

---

## Import and Dependency Rules

- `domain` may not import anything.
- `service` may import `domain` and repository interfaces from `repos`.
- `common` may import `domain` and standard library only.
- `config` may import `domain` and validation libraries.
- `gateways` may import `domain` and implement external communication interfaces.
- `repos` may import `domain` and implement data access interfaces.
- `cmd/rr-dnsd` is the only layer allowed to wire up dependencies across boundaries.

---

## Responsibilities Per Layer

### Domain (`internal/dns/domain`)
- Pure data types with validation
- No external dependencies or side effects
- Examples: `Question`, `ResourceRecord`, `RRType`

### Service (`internal/dns/service`)
- Implements business rules and coordination
- Depends only on `domain` and repository interfaces
- Must be fully testable without infrastructure dependencies

### Common (`internal/dns/common`)
- Shared infrastructure components (logging, utilities)
- Cross-cutting concerns used by multiple layers
- No business logic or domain knowledge

### Config (`internal/dns/config`)
- Environment-based configuration management
- Validation and type conversion of configuration values
- No runtime state changes

### Gateways (`internal/dns/gateways`)
- External system integrations (network transports, upstream DNS, wire formats)
- Implements interfaces for communicating with external systems
- Protocol-specific implementations

### Repos (`internal/dns/repos`)
- Data access patterns (caching, zone files, blocklists)
- Repository pattern implementations
- Data persistence and retrieval logic

### Cmd (`cmd/rr-dnsd`)
- Assembles the application
- Responsible for constructing all layers

---

## What Is Not Allowed

- ❌ Logging in domain types
- ❌ Environment parsing inside services
- ❌ Service methods reaching into config or logger packages directly
- ❌ Infrastructure components importing service layer
- ❌ Cross-dependencies between gateways and repos
- ❌ Business logic in common, config, gateways, or repos

---

## Communication Between Layers

- Services are initialized with interfaces from `repos` and `gateways`
- All shared data must pass through domain types (never raw maps or side effects)
- Cross-layer interaction should be traceable via dependency injection in `cmd/rr-dnsd`
- Common services (logging) are injected where needed
- Configuration is loaded once and passed to components that need it

---

## Testing by Layer

- Domain: fully unit tested and deterministic
- Service: tested with mocks/fakes for repository and gateway interfaces
- Common: unit tested for utilities, integration tested for shared services
- Config: unit tested for validation and parsing logic
- Gateways: integration tests with real external systems (network, upstream DNS)
- Repos: unit tests for data access logic, integration tests for storage mechanisms
