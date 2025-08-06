[![Static Badge](https://img.shields.io/badge/Arc42-Docs-blue)](docs/arc42.md) ![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/haukened/rr-dns) ![GitHub License](https://img.shields.io/github/license/haukened/rr-dns?color=blue) ![GitHub Issues or Pull Requests](https://img.shields.io/github/issues/haukened/rr-dns?color=blue) ![Codecov](https://img.shields.io/codecov/c/github/haukened/rr-dns?color=blue) ![CodeFactor Grade](https://img.shields.io/codefactor/grade/github/haukened/rr-dns?color=blue)

 [![Release CI](https://github.com/haukened/rr-dns/actions/workflows/ci_release.yaml/badge.svg)](https://github.com/haukened/rr-dns/actions/workflows/ci_release.yaml) [![Security Scan](https://github.com/haukened/rr-dns/actions/workflows/ci_security.yaml/badge.svg)](https://github.com/haukened/rr-dns/actions/workflows/ci_security.yaml)

# RR-DNS
A small, lightning fast, local DNS caching resolver with Ad-Blocking. Written in Go. RR is a [double entendre](https://en.wikipedia.org/wiki/Double_entendre) for "Rapid Resolver" (what it does) and "Resource Record" (the core object of DNS servers).

**RR-DNS** (Rapid Resolver DNS) is a lightweight, fast, and modern DNS server written in Go. It is designed to be simple to operate, easy to extend, and highly testable — ideal for home networks, containers, embedded environments, and security-conscious setups.

> **Repo**: https://github.com/haukened/rr-dns

---

## 🚀 Purpose

RR-DNS exists to provide a minimal but robust DNS server that:

- Responds quickly and correctly to DNS queries
- Is small and efficient enough to run anywhere
- Follows strict architectural principles (CLEAN, SOLID)
- Can be easily extended to include features like blocklists, query logging, or a web admin interface

We are not trying to be BIND or Unbound. This is DNS done right — but simple.

---

## 🌐 Why Another DNS Server?

The DNS ecosystem is full of powerful resolvers — from BIND to Unbound, dnsmasq to CoreDNS — each built for different environments and complexity levels. But many of them:

- Try to solve *all* DNS use cases (authoritative, recursive, DHCP integration, plugin systems)
- Come with large configuration surfaces or legacy constraints
- Aren't optimized for simplicity, testability, or container-native workflows

We built **RR-DNS** because we wanted something different:
- A resolver focused on **local caching, speed, and blocking**
- An implementation that is **cleanly architected**, **easy to reason about**, and **fun to work on**
- A system that fits naturally into **modern deployment environments** like Docker, k8s sidecars, or embedded devices
- A Go-based project where features like **ad-blocking**, **web admin**, and **logging** can evolve modularly over time

In short: **RR-DNS fills the gap** between full-stack DNS suites and toy resolvers — with a maintainable, developer-friendly design.
---

## 🛠️ Architecture

RR-DNS is built with **CLEAN architecture** at its core:

```
cmd/rrdnsd            ← CLI entrypoint
docs/                 ← Project documentation (Arc42, design notes)
internal/
  dns/
    domain/           ← Core types like DNSQuery, DNSResponse, ResourceRecord
    infra/
      config/         ← Config loading via env or CLI
      log/            ← Structured logging with zap
      wire/           ← DNS wire format codec (RFC 1035)
      upstream/       ← Upstream DNS resolver with caching
      dnscache/       ← In-memory DNS response caching
      zone/           ← Static zone file loading (JSON/YAML/TOML)
      blocklist/      ← Ad/tracker blocking infrastructure
    repo/             ← Repository layer for data access
    service/          ← Query resolution, orchestration logic
pkg/                  ← Shared library code (if needed)
```

### Guiding Principles

- [x] **CLEAN architecture**: clear boundaries between domain, service, and infra
- [x] **SOLID principles**: small interfaces, testable logic, dependency inversion
- [x] **Testability first**: domain and service layers are fully unit-testable with 100% coverage
- [x] **Go idioms**, not Go monoliths: no unnecessary abstractions, only meaningful ones
- [x] **RFC compliance**: Full DNS wire format implementation per RFC 1035

For a detailed architectural breakdown, see the [Arc42 documentation](docs/arc42.md).

---

## 📦 Current Features

RR-DNS currently supports:

- [x] **DNS Wire Format Codec**: Complete RFC 1035 implementation with compression support
- [x] **Upstream DNS Resolution**: Configurable upstream resolvers (Google, Cloudflare, custom)
- [x] **Response Caching**: In-memory DNS response caching with TTL management
- [x] **Static Zone Support**: Load zones from JSON/YAML/TOML files
- [x] **Structured Logging**: Production-ready logging with zap (dev/prod modes)
- [x] **Configuration Management**: Environment variables and CLI argument support
- [x] **Comprehensive Testing**: 100% test coverage on core infrastructure
- [x] **Error Handling**: Robust error handling for malformed packets and edge cases
- [ ] **UDP Server**: DNS query server implementation
- [ ] **Query Resolution Service**: Orchestration of upstream, cache, and zone lookups
- [ ] **Ad/Tracker Blocking**: Blocklist subscription and filtering

---

## 🗺️ Roadmap

| Phase       | Features                                       | Status |
|-------------|------------------------------------------------|---------|
| **v0.1**    | Complete DNS server: wire codec, UDP server, upstream resolution, caching, zone support | 🚧 In Progress |
| **v0.2**    | Blocklist subscriptions (ad/tracker blocking) | 📋 Planned |
| **v0.3**    | Docker support, metrics, health checks        | 📋 Planned |
| **v0.4**    | Web admin UI, config reloading                | 📋 Planned |
| **v1.0**    | Snap/apt packages, TLS/DoH support            | 📋 Planned |
| **v1.1**    | TLS/DoH support                               | 📋 Planned |

### Infrastructure Completed ✅
- DNS wire format encoding/decoding (100% RFC 1035 compliant)
- Upstream DNS resolver with connection pooling
- In-memory DNS response caching with TTL management  
- Static zone file loading (JSON/YAML/TOML)
- Structured logging with development and production modes
- Configuration management with environment variables
- Comprehensive test coverage on all infrastructure components

### Remaining for v0.1 🚧
- UDP DNS server implementation
- Query resolution service orchestration
- Integration testing and end-to-end validation

---

## � Testing & Quality

RR-DNS prioritizes code quality and reliability:

- **100% Test Coverage**: All infrastructure components have comprehensive unit tests
- **Error Path Testing**: Extensive testing of edge cases and malformed data handling
- **RFC Compliance Testing**: DNS wire format validated against RFC 1035 specifications
- **Structured Testing**: Table-driven tests with clear scenarios and validations
- **Continuous Integration**: Automated testing on all commits

```bash
# Run tests with coverage
go test -cover ./...

# Generate detailed coverage report
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

---

## 🤝 Contributing

We welcome contributions! Please:
- Follow Go formatting conventions
- Respect CLEAN boundaries — infra never calls domain, tests should focus on services and domain first
- Add unit tests for all logic (aim for 100% coverage)
- Log using structured logs: `log.Info(map[string]any{"queryID": id, "name": name}, "Received query")`
- Test error paths thoroughly, including edge cases and malformed data

### Development Setup

```bash
# Clone the repository
git clone https://github.com/haukened/rr-dns.git
cd rr-dns

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Format code
go fmt ./...
```

---

## 📜 License

- [MIT](LICENSE)

---

## 🙋‍♀️ Maintainers

- [@haukened](https://github.com/haukened)

---

## 🌱 Inspiration

We’re inspired by the spirit of projects like 
- [dnsmasq](http://www.thekelleys.org.uk/dnsmasq/doc.html)
- [CoreDNS](https://coredns.io/)
- [Pi-hole](https://pi-hole.net/) 
- [Technitium](https://technitium.com/dns/)

but RR-DNS is built from the ground up with modern, maintainable Go in mind.