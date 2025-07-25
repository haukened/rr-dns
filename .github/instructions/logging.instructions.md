---
applyTo: "**"
---

# Logging Instructions for uDNS

This guide defines how logging is implemented and should be used throughout the uDNS codebase.

## Overview

uDNS uses a singleton-style logger based on [`zap`](https://github.com/uber-go/zap) for structured, leveled logging. The logger is initialized once during startup and accessed globally via convenience functions.

## Configuration

The global logger is configured at runtime via:

```go
log.Configure(env, logLevel)
```

- `env` is either `"dev"` or `"prod"`:
  - `"dev"` enables human-readable logs with colored levels
  - `"prod"` enables structured JSON logs
- `logLevel` accepts one of: `"debug"`, `"info"`, `"warn"`, `"error"`, `"panic"`, `"fatal"`

Default: `prod` environment with `info` level logging.

## Usage

Instead of using `fmt.Println` or `log.Printf`, always use the structured logger:

```go
log.Info(map[string]any{
  "zone": "example.com",
  "count": 4,
}, "Loaded zone records")
```

Available levels:

```go
log.Debug(fields, msg)
log.Info(fields, msg)
log.Warn(fields, msg)
log.Error(fields, msg)
log.Panic(fields, msg) // Triggers panic
log.Fatal(fields, msg) // Exits the process
```

Always use the `fields` map for structured data, and a short descriptive message as the second argument.

## Guidelines

- Do not include log output in the domain layer.
- Service and infra layers may log operational info, warnings, or errors.
- Do not use string interpolation in messages — prefer structured fields.
- Avoid logging secrets, tokens, or raw query payloads.
- Use consistent field names: `zone`, `query`, `rcode`, `duration_ms`, etc.

---

### How much to log

Log enough information to explain what the system is doing and why — but avoid flooding the logs with redundant or trivial messages. A good rule of thumb: log what you'd want to know to debug production without access to breakpoints.

---

### When to log

- When a user-triggered or network-triggered action occurs
- When external dependencies (e.g. DNS clients, file reads) are involved
- When something unexpected or invalid is encountered
- At service startup/shutdown or reload events

---

### What to log at each level

- `Debug`: internal decisions, parsed input, skipped lookups, retries — used in dev/debugging
- `Info`: successful resolutions, config state, normal operational milestones
- `Warn`: non-fatal issues like fallback to default, malformed query components, retries
- `Error`: system or DNS failures, invalid input, configuration rejection
- `Panic`: corrupted internal state, unreachable conditions (in dev only)
- `Fatal`: unrecoverable failure causing process exit (e.g. cannot bind port)

---

### Actions that must always be logged

- DNS query receipt and response (at least `Info`)
- DNS resolution failures or fallbacks (`Warn` or `Error`)
- Zone loading success/failure
- Server startup and shutdown (including config)
- Any panic or fatal that would crash the service

## Testing and Overrides

For tests:

- Use `log.SetLogger(...)` to inject a test or no-op logger.
- `noopLogger` and `testLogger` are provided for silencing or inspecting logs.
- Fatal and Panic can be safely stubbed in `testLogger` to avoid exit or panic in tests.

```go
log.SetLogger(&testLogger{})
log.Info(nil, "test message")
```