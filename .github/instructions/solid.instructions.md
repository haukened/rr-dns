---
applyTo: "*.go"
---

# SOLID Principles in Go

This guide describes how to apply the SOLID principles in the rr-dns project using idiomatic Go. It supports Copilot and developers in writing modular, clean, and testable code.

---

## S — Single Responsibility Principle (SRP)

A Go struct or package should have only one reason to change.

✅ Good:
- `resolverService` resolves queries — nothing else.
- `staticZoneRepo` just looks up records from memory.

Avoid combining unrelated responsibilities, even if the logic is short.

---

## O — Open/Closed Principle (OCP)

Code should be open for extension, but closed for modification.

✅ Strategy in Go:
- Define interfaces in `repo/` and `service/`.
- Implement them in `infra/`.

This allows new implementations (e.g. file-backed zones) without changing resolver logic.

---

## L — Liskov Substitution Principle (LSP)

Any type that implements an interface should behave correctly when used via that interface.

✅ Tips:
- Keep interfaces small.
- Don’t surprise the consumer. A `ZoneRepository` that returns nil or malformed records breaks expectations.

---

## I — Interface Segregation Principle (ISP)

Clients should not be forced to depend on things they don’t use.

✅ In Go:
- Avoid giant interfaces. Favor narrow interfaces like:
  ```go
  type ZoneRepository interface {
      FindRecords(name string, qtype RRType) ([]ResourceRecord, error)
  }
  ```

---

## D — Dependency Inversion Principle (DIP)

High-level modules (like services) should not depend on low-level modules (like UDP). Both should depend on abstractions.

✅ rr-dns:
- The `resolverService` depends on `ZoneRepository` interface.
- `udpServer` depends on `QueryResolver`.

Concrete dependencies are injected from `cmd/rr-dnsd`.

---

## Summary for Copilot

When generating Go code:
- Place logic in packages that follow SRP.
- Define narrow interfaces in `repo/`.
- Implement those interfaces in `infra/`.
- Ensure interface contracts are respected.
- Never have domain or service depend on `infra`.
