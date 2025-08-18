# DNS Blocklist Repository (v0.3 components)

This folder contains the building blocks for the blocklist repository used by rr-dns. The current implementation provides:

- A Bloom filter wrapper and factory (`bloom/`)
- A persistent Bolt-backed store with first-match semantics (`bolt/`)
- An in-memory LRU decision cache (`lru/`)
- Plain-text and hosts-file parsers that emit `domain.BlockRule` values (`parsers/`)

These parts are designed to be wired together by a higher-level `Repository` (not included here) following CLEAN architecture.

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
- Repository (to be wired elsewhere): `Decide(name)`, `UpdateAll(rules, version, updated)`

## Lookup pipeline: cache → bloom → store

Typical read path for a canonical DNS name:

1) Check `DecisionCache` → return on hit
2) Bloom pre-checks (advisory): test exact FQDN and candidate reversed anchors; skip store entirely if all tests are negative
3) Store authoritative check via `GetFirstMatch(name)` which returns:
     - exact match if present, else
     - first matching suffix anchor found by a reversed-prefix cursor walk
4) Materialize a `domain.BlockDecision` and cache it via `DecisionCache.Put`

Update path (atomic):

- `Store.RebuildAll(allRules, version, updatedUnix)` replaces data in a single write transaction
- Recreate the Bloom (via `BloomFactory`) sized for the dataset
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

if r, ok, _ := st.GetFirstMatch("a.example.com"); ok {
    _ = r // exact
}
if r, ok, _ := st.GetFirstMatch("x.example.org"); ok {
    _ = r // suffix anchor
}
```

## lru/

LRU-backed implementation of `blocklist.DecisionCache`.

- `New(size int) (blocklist.DecisionCache, error)` returns an LRU cache; when `size <= 0`, returns a disabled no-op cache
- Methods: `Get`, `Put`, `Len`, `Purge`
- Backed by `github.com/hashicorp/golang-lru/v2`

Notes:
- No internal stats counters; it’s a simple memoization layer
- Not concurrency-safe by itself; coordinate access at the repository/service layer

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
- `bloom/`, `lru/`, and `parsers/` have focused unit tests covering core behavior and edge cases

## Architecture notes

These components stay within the repos layer and expose minimal interfaces to keep the service layer testable and decoupled. Wiring (BloomFactory + DecisionCache + Store) happens in the composition root (cmd), following the CLEAN architecture boundaries defined in this repo.
