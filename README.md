# uDNS
A small, lightning fast, local DNS caching resolver with Ad-Blocking. Written in Go.

# uDNS

**uDNS** (micro DNS) is a lightweight, fast, and modern DNS server written in Go. It is designed to be simple to operate, easy to extend, and highly testable — ideal for home networks, containers, embedded environments, and security-conscious setups.

> **Repo**: https://github.com/haukened/uDNS

---

## 🚀 Purpose

uDNS exists to provide a minimal but robust DNS server that:

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

We built **uDNS** because we wanted something different:
- A resolver focused on **local caching, speed, and blocking**
- An implementation that is **cleanly architected**, **easy to reason about**, and **fun to work on**
- A system that fits naturally into **modern deployment environments** like Docker, k8s sidecars, or embedded devices
- A Go-based project where features like **ad-blocking**, **web admin**, and **logging** can evolve modularly over time

In short: **uDNS fills the gap** between full-stack DNS suites and toy resolvers — with a maintainable, developer-friendly design.
---

## 🛠️ Architecture

uDNS is built with **CLEAN architecture** at its core:

```
cmd/udnsd            ← CLI entrypoint
docs/                ← Project documentation (Arc42, design notes)
internal/
  dns/
    domain/          ← Core types like DNSQuery, DNSResponse, ResourceRecord
    infra/
      config/        ← Config loading via env or CLI
      log/           ← Pluggable logging backend
      udp/           ← UDP socket server
    repo/            ← In-memory/static zone storage, later dynamic options
    service/         ← Query resolution, orchestration logic
pkg/                 ← Shared library code (if needed)
```

### Guiding Principles

- [x] **CLEAN architecture**: clear boundaries between domain, service, and infra
- [x] **SOLID principles**: small interfaces, testable logic, dependency inversion
- [x] **Testability first**: domain and service layers are fully unit-testable
- [x] **Go idioms**, not Go monoliths: no unnecessary abstractions, only meaningful ones

For a detailed architectural breakdown, see the [Arc42 documentation](docs/arc42.md).

---

## 📦 MVP Features

The first version of `udnsd` will support:

- [x] Responding to A/AAAA queries over UDP
- [x] Serving static zones from in-memory definitions
- [x] Returning correct response codes (NOERROR, NXDOMAIN, etc.)
- [ ] Logging queries to stdout
- [ ] Docker support

---

## 🗺️ Roadmap

| Phase       | Features                                       |
|-------------|------------------------------------------------|
| - [ ] v0.1     | A/AAAA query support, static zones, UDP server|
| - [ ] v0.2     | Logging, metrics, Dockerfile                  |
| - [ ] v0.3     | Blocklist subscriptions (ad/tracker blocking) |
| - [ ] v0.4     | Web admin UI, config reloading                |
| - [ ] v1.0     | Snap/apt packages, TLS/DoH support            |

---

## 🤝 Contributing

We welcome contributions! Please:
- Follow Go formatting conventions
- Respect CLEAN boundaries — infra never calls domain, tests should focus on services and domain first
- Add unit tests for all logic
- Log using structured logs: `logger.Info({ queryID, name }, "Received query")`

---

## 📜 License

- [GNU GPL v3](LICENSE)

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

but uDNS is built from the ground up with modern, maintainable Go in mind.