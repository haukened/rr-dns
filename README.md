[![Static Badge](https://img.shields.io/badge/Arc42-Docs-blue)](docs/arc42.md) 
[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/haukened/rr-dns)](https://go.dev/dl/)
[![GitHub License](https://img.shields.io/github/license/haukened/rr-dns?color=blue)](LICENSE) 
[![GitHub Issues or Pull Requests](https://img.shields.io/github/issues/haukened/rr-dns?color=blue)](https://github.com/haukened/rr-dns/issues)
[![Codecov](https://img.shields.io/codecov/c/github/haukened/rr-dns?color=blue)](https://app.codecov.io/gh/haukened/rr-dns)
[![CodeFactor Grade](https://img.shields.io/codefactor/grade/github/haukened/rr-dns?color=blue)](https://www.codefactor.io/repository/github/haukened/rr-dns)
[![Static Badge](https://img.shields.io/badge/pkg.go.dev-rr--dns-blue)](https://pkg.go.dev/github.com/haukened/rr-dns)


 [![Release CI](https://github.com/haukened/rr-dns/actions/workflows/ci_release.yaml/badge.svg)](https://github.com/haukened/rr-dns/actions/workflows/ci_release.yaml) 
 [![Security Scan](https://github.com/haukened/rr-dns/actions/workflows/ci_security.yaml/badge.svg)](https://github.com/haukened/rr-dns/actions/workflows/ci_security.yaml)

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

## I don't wanna read, i just wanna run it!

> Great, docker is a fast way to do that!

### Environment Variables

| Variable Name | Purpose | Type | Default |
| :-- | :-- | :-- | :-- |
| DNS_CACHE_SIZE | cache entries capacity | Integer, >= 1 | 1000 |
| DNS_DISABLE_CACHE | disable DNS response caching | Boolean | false |
| DNS_ENV | runtime environment | `dev\|prod` | prod |
| DNS_LOG_LEVEL | log verbosity | `debug\|info\|warn\|error` | info |
| DNS_PORT | UDP listening port | Integer, 1-65534 | 8053 [^1] |
| DNS_ZONE_DIR | directory for zone files | String (path) | /zones/ [^2] |
| DNS_SERVERS | upstream DNS servers (ip:port) | List, space or comma-separated [^3] | 1.1.1.1:53, 1.0.0.1:53 |
| DNS_MAX_RECURSION | max in-zone alias chase depth | Integer, >= 1 | 8 |

[^1]: In docker containers, default port is set to 8053 to prevent privileged port use.
[^2]: In docker containers, the default zone directory is changed from `/etc/rr-dns/zones/` to `/zones/` because we use distroless containers `/etc` isn't a guaranteed path, and `/zones/` is pragmatic for mount paths.
[^3]: `DNS_SERVERS` accepts multiple values separated by spaces or commas, for example: `1.1.1.1:53, 1.0.0.1:53`.

### Authoritative and Recursive DNS Modes
rr-dns can operate in two modes:

#### Authoritative DNS Server:

For any zones you define (using standard zone files), rr-dns acts as an authoritative DNS server. This means it will answer queries for those domains directly, using the records you provide.

#### Caching Recursive Resolver:

For domains not covered by your zone files, rr-dns automatically acts as a recursive resolver. It will query upstream DNS servers, cache the results, and return answers to clients.

>Note:
>You are not required to define any zones. If you do not provide zone files, rr-dns will function purely as a recursive caching resolver, forwarding and caching queries for all domains.

This approach allows you to use rr-dns as a flexible DNS solution—either as an authoritative server, a recursive resolver, or both, depending on your configuration.

### Example Compose File

```yaml
version: "3.9"

services:
  dns:
    container_name: rr-dns
    image: ghcr.io/haukened/rr-dns:latest
    environment:
      - DNS_ENV=prod
      - DNS_LOG_LEVEL=info
      - DNS_PORT=8053
      - DNS_ZONE_DIR=/zones
      - DNS_CACHE_SIZE=1000
      - DNS_DISABLE_CACHE=false
      - DNS_MAX_RECURSION=8
      - DNS_SERVERS=1.1.1.1:53,1.0.0.1:53
    volumes:
      - ./zones:/zones:ro
    ports:
      - "8053:8053/udp"  # map host 8053 -> container 8053 (UDP)
      # If you want host port 53, ensure it's free and Docker runs with sufficient privileges:
      # - "53:8053/udp"
    restart: unless-stopped
```

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
  domain/           ← Core types like Question, DNSResponse, ResourceRecord
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
- [x] **UDP Server**: DNS query server implementation
- [x] **Query Resolution Service**: Orchestration of upstream, cache, and zone lookups
- [x] **CNAME Alias Resolution**: RFC 1034 §3.6.2 compliant chain expansion (loop & depth safeguards, partial-chain NOERROR policy, SERVFAIL on loop/depth)
- [X] **Docker Deployment**: Support deploying in docker containers.
- [ ] **Ad/Tracker Blocking**: Blocklist subscription and filtering
- [ ] **Snap Packaging**: Published on snapcraft.io
- [ ] **Apt Packaging**: Apt packages for Debian/Ubuntu/Derivates
- [ ] **DNS over HTTPS**: DoH support for extra privacy.
- [ ] **DNS over TLS**: Secure DNS queries with DoT support
- [ ] **REST API**: Admin endpoints for health checks and metrics
- [ ] **Web Admin UI**: Modern web interface for configuration and monitoring

---

## 🗺️ Roadmap

| Version   | Features                                  | Status       |
|-----------|-------------------------------------------|--------------|
| **v0.1**  | Core DNS Resolution features              | ✅ Complete  |
| **v0.2**  | Docker support                            | ✅ Complete  |
| **v0.3**  | Blocklist subscriptions                   | 🚧 In Progress |
| **v0.4**  | Snap package support                      | 📋 Planned   |
| **v0.5**  | Apt package support                       | 📋 Planned   |
| **v1.0**  | Stable release (no UI)                    | 📋 Planned   |
| **v1.1**  | TLS/DoH support                           | 📋 Planned   |
| **v1.2**  | REST API for config/query/logs            | 📋 Planned   |
| **v1.3**  | Web Admin UI                              | 📋 Planned   |
| **v2.0**  | Stable release with Admin UI              | 📋 Planned   |

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