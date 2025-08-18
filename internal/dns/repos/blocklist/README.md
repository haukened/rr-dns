# DNS Blocklist Repository (v0.3)

This folder contains the repository layer for rr-dns’s blocklist, plus its supporting components:

- Bloom filter wrapper and factory (`bloom/`)
- Persistent Bolt-backed store with first-match semantics (`bolt/`)
- In-memory LRU decision cache (`lru/`)
- Plain-text and hosts-file parsers emitting `domain.BlockRule` (`parsers/`)
- A concrete `Repository` implementation (`repo.go`) that wires cache → bloom → store

## Interfaces at a glance

As defined in `interfaces.go`:

- BloomFactory: `New(capacity uint64, fpRate float64) BloomFilter`
- BloomFilter: `Add([]byte)`, `MightContain([]byte) bool`
- DecisionCache: `Get(name)`, `Put(name, decision)`, `Len()`, `Purge()`
- Store:
    - `GetFirstMatch(name) (rule domain.BlockRule, ok bool, err error)`
    - `RebuildAll(rules []domain.BlockRule, version uint64, updatedUnix int64) error`
    - `Purge() error`
    - `Close() error`
- Repository: `Decide(name)`, `UpdateAll(rules, version, updated)`

## Repository behavior: cache → bloom → store

All lookups operate on a canonical DNS name (lowercase, no trailing dots). The read path is:

1) Bloom pre-check (advisory): test exact FQDN and candidate reversed anchors; early-allow if all tests are negative
2) Cache check: return immediately on hit
3) Store authoritative check via `GetFirstMatch(name)` which returns:
     - exact match if present, else
     - first matching suffix anchor found by a reversed-prefix cursor walk
4) Materialize a `domain.BlockDecision` and cache it

Update path (atomic):

- `Store.RebuildAll(allRules, version, updatedUnix)` replaces data in a single write transaction
- Recreate the Bloom (via `BloomFactory`) sized only by supported rules (exact+suffix)
- Purge the decision cache to avoid stale decisions

---

## bloom/

Thread-safe adapter around bits-and-blooms with an internal sizer.

- `factory.go`: `bloom.NewFactory()` returns a `blocklist.BloomFactory`. `New(capacity, fpRate)` computes m/k via standard formulas and builds a filter.
- `filter.go`: `Add` is serialized with a mutex; `MightContain` is read-safe. Interface exposes only `Add` and `MightContain`.
- `sizer.go`: internal helpers for m/k; not exported.

Example:

```go
f := bloom.NewFactory()
bf := f.New(100_000, 0.01)
key := []byte("example.com")
bf.Add(key)
_ = bf.MightContain(key) // probabilistic “maybe”
```

Notes:
- Interface intentionally omits Clear/Swap; rebuild a new filter on updates.

## bolt/

Bolt-backed `blocklist.Store` with first-match semantics and atomic snapshot updates.

Buckets:
- `exact`   → exact FQDN keys
- `suffix`  → reversed domain keys for anchor-style suffix matches (e.g., `example.com` stored as `moc.elpmaxe`)
- `meta`    → versioning metadata (`version`, `updated` as big-endian uint64)

API:
- `New(path) (blocklist.Store, error)` opens/creates the DB and ensures buckets
- `GetFirstMatch(name)` returns exact match first; else walks reversed prefixes most-specific→least and returns first suffix match
- `RebuildAll(rules, version, updatedUnix)` deletes+recreates buckets, loads rules, and writes meta in a single write tx
- `Purge()` clears and recreates buckets
- `Close()` closes the DB

Behavior:
- Suffix search is implemented via a cursor `Seek` on the reversed name and iterative trimming at label boundaries
- Values encode `{kind|addedAt|sourceLen|source}`; decoder tolerates legacy/minimal values
- Writes are atomic; reads are consistent; error paths are covered (including read-only tx and invalid keys)

Example:

```go
st, _ := bolt.New("/tmp/bl.db")
defer st.Close()

rules := []domain.BlockRule{
    domain.MustNewExactBlockRule("a.example.com", "source", time.Now()),
    domain.MustNewSuffixBlockRule("example.org", "source", time.Now()),
}
_ = st.RebuildAll(rules, 1, time.Now().Unix())

if r, ok, _ := st.GetFirstMatch("a.example.com"); ok { _ = r } // exact
if r, ok, _ := st.GetFirstMatch("x.example.org"); ok { _ = r } // suffix
```

## lru/

LRU-backed implementation of `blocklist.DecisionCache`.

- `New(size int) (blocklist.DecisionCache, error)` returns an LRU cache; when `size <= 0`, returns a disabled no-op cache
- Methods: `Get`, `Put`, `Len`, `Purge`
- Backed by `github.com/hashicorp/golang-lru/v2`

Notes:
- No internal stats counters; it’s a simple memoization layer
- Not concurrency-safe by itself; coordinate access at the repository/service layer

## repository/

Concrete `blocklist.Repository` implementation in `repo.go`.

Highlights:
- Canonicalizes input names with `utils.CanonicalDNSName`
- Early-allow when Bloom is definitely negative (tests exact and reversed anchors)
- Reads consult cache before store on maybe-positives
- `UpdateAll` performs a store rebuild, then rebuilds Bloom (using reversed keys for suffix rules), then purges the cache — all under lock during the swap
- On internal errors during reads, policy prefers Allow (not blocked)

End-to-end wiring example:

```go
st, _ := bolt.New("/tmp/bl.db")
cache, _ := lru.New(50_000)
factory := bloom.NewFactory()
repo := blocklist.NewRepository(st, cache, factory, 0.01)

// Load data
_ = repo.UpdateAll(rules, 1, time.Now().Unix())

// Query
dec := repo.Decide("AdS.ExAmPlE.org.")
if dec.IsBlocked() { /* handle */ }
```

## parsers/

Parsers that produce `[]domain.BlockRule` from common list formats. All parsers:

- Trim and normalize via `utils.CanonicalDNSName`
- Validate with a lightweight FQDN check
- Skip empty lines and comments (`#`), support inline comments
- De-duplicate while preserving first-seen order
- Attribute rules to the provided `source` and timestamp with `now`

Plain lists (`plain.go`):
- Each non-empty, non-comment line is a token
- Leading `*.` or `.` indicates a suffix rule; otherwise exact

Hosts files (`hosts.go`):
- Parse `/etc/hosts`-style lines; ignore IP field and extract hostnames
- Only exact rules are emitted; wildcards and names starting with `.` are ignored

Example:

```go
r, _ := parsers.ParsePlainList(strings.NewReader("""
# comment
example.com
*.ads.example.net
"""), "list.txt", logger, time.Now())
// r contains exact(example.com) and suffix(ads.example.net)
```

---

## Testing

This package includes unit tests for all subpackages. To run:

```bash
go test ./internal/dns/repos/blocklist/...
```

Highlights:
- `bolt/` store has 100% coverage of its code paths (including error branches)
- `repo.go` has full unit coverage for read and update flows
- `bloom/`, `lru/`, and `parsers/` have focused tests covering core behavior and edge cases

## Architecture notes

These components stay within the repos layer and expose minimal interfaces to keep the service layer testable and decoupled. Wiring (BloomFactory + DecisionCache + Store) typically happens in the composition root (cmd), following the CLEAN architecture boundaries defined in this repo. The `Repository` here is infrastructure that the composition root can instantiate directly.
