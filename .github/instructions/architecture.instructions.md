---
applyTo: "*.go"
---

# CLEAN Architecture in Go

This guide defines how to implement CLEAN architecture in Golang for the uDNS project. It is intended to help GitHub Copilot produce consistent, idiomatic, and testable code aligned with the project's goals.

## Principles

CLEAN architecture emphasizes separation of concerns and inward-facing dependencies:

- **Entities (Domain)**: Pure business/data logic
- **Use Cases (Services)**: Application logic
- **Interfaces (Gateways/Repos)**: Abstractions for external systems
- **Infrastructure**: External I/O, implementations of interfaces
- **Entrypoints**: CLI, API, or server setup

## Project Structure

```
/cmd/udnsd             # Main program entrypoint

/internal
  /dns
    /domain            # Core types like DNSQuery, DNSResponse
    /service           # Business logic, e.g. Resolver
    /repo              # Interfaces for data access
    /infra
      /udp             # UDP socket implementation
      /log             # Logging adapter
      /config          # Environment/config loading

/docs                  # Architecture, design, and planning
/pkg                   # Shared libraries (optional)
```

## Rules for CLEAN Code in Go

- Domain objects (`domain/`) must be pure Go structs with no external dependencies.
- Services (`service/`) may only depend on `domain` and `repo` interfaces.
- Repos (`repo/`) define interfaces, not implementations.
- Infra (`infra/`) implements repo interfaces and handles all external systems (e.g., networking, file IO).
- Entry points (`cmd/udnsd`) wire everything together.


## Naming Conventions

- Interface: `ZoneRepository`, `QueryResolver`
- Implementation: `staticZoneRepo`, `resolverService`
- Package names are lowercase, no underscores.

## Copilot Guidance

When generating new code:
- Place domain models in `internal/dns/domain`
- Place resolver logic in `internal/dns/service`
- Define interfaces in `repo`, and implementations in `infra`
- Avoid importing `infra` into `service` or `domain`
- Add unit tests for each `service` method


# Architecture Boundaries for uDNS

This file describes the architectural boundaries enforced by the uDNS project. These constraints are essential for maintaining testability, maintainability, and the separation of concerns defined by the CLEAN architecture.

---

## Project Structure and Layer Responsibilities

```
cmd/
  udnsd/             # Entrypoint, DI setup

internal/
  dns/
    domain/          # Pure domain models and value types
    service/         # Application logic (resolvers, orchestration)
    repo/            # Interfaces to domain-level data access
    infra/
      udp/           # Wire protocol handlers (UDP listener)
      config/        # Environment + runtime configuration
      log/           # Logging implementation
```

---

## Import and Dependency Rules

- `domain` may not import anything.
- `service` may import `domain` and `repo` interfaces.
- `repo` may import `domain`, but not `infra` or `service`.
- `infra` may implement `repo` interfaces, but must not import `service` or `cmd`.
- `cmd/udnsd` is the only layer allowed to wire up dependencies across boundaries.

---

## Responsibilities Per Layer

### Domain (`internal/dns/domain`)
- Pure data types with validation
- No external dependencies or side effects
- Examples: `DNSQuery`, `ResourceRecord`, `RRType`

### Service (`internal/dns/service`)
- Implements business rules and coordination
- Depends only on `domain` and `repo` interfaces
- Must be fully testable without infra

### Repo (`internal/dns/repo`)
- Defines interfaces for data access
- No concrete knowledge of infrastructure
- Example: `ZoneRepository`

### Infra (`internal/dns/infra`)
- Implements side effects and adapters
- Handles UDP, config, file IO, logging
- May import domain types, but not services

### Cmd (`cmd/udnsd`)
- Assembles the application
- Responsible for constructing all layers

---

## What Is Not Allowed

- ❌ Logging in domain types
- ❌ Environment parsing inside services
- ❌ Service methods reaching into config or logger packages
- ❌ Infra importing anything above its layer

---

## Communication Between Layers

- Services are initialized with interfaces from `repo` or `infra`
- All shared data must pass through domain types (never raw maps or side effects)
- Cross-layer interaction should be traceable via dependency injection in `cmd/udnsd`

---

## Testing by Layer

- Domain: fully unit tested and deterministic
- Service: tested with mocks/fakes for `repo` interfaces
- Repo: test implementation logic, but isolate from infra
- Infra: integration tests with real side effects (e.g. UDP, file system)
