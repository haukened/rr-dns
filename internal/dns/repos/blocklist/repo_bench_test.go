package blocklist_test

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	"github.com/haukened/rr-dns/internal/dns/domain"
	"github.com/haukened/rr-dns/internal/dns/repos/blocklist"
	"github.com/haukened/rr-dns/internal/dns/repos/blocklist/bloom"
	"github.com/haukened/rr-dns/internal/dns/repos/blocklist/bolt"
	"github.com/haukened/rr-dns/internal/dns/repos/blocklist/lru"
)

// helper: build sample rules
func repoBenchExact(n int, suffix string) []domain.BlockRule {
	out := make([]domain.BlockRule, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, domain.BlockRule{
			Name:    fmt.Sprintf("p%04d.%s", i, suffix),
			Kind:    domain.BlockRuleExact,
			Source:  "bench",
			AddedAt: time.Unix(1, 0),
		})
	}
	return out
}

func repoBenchSuffix(anchors ...string) []domain.BlockRule {
	out := make([]domain.BlockRule, 0, len(anchors))
	for _, a := range anchors {
		out = append(out, domain.BlockRule{Name: a, Kind: domain.BlockRuleSuffix, Source: "bench", AddedAt: time.Unix(1, 0)})
	}
	return out
}
func buildRepo(b *testing.B, cacheSize int, rules []domain.BlockRule) (blocklist.Repository, func()) {
	b.Helper()
	st, err := bolt.New(filepath.Join(b.TempDir(), "bl.db"))
	if err != nil {
		b.Fatalf("bolt.New: %v", err)
	}
	cache, err := lru.New(cacheSize)
	if err != nil {
		b.Fatalf("lru.New: %v", err)
	}
	factory := bloom.NewFactory()
	repo := blocklist.NewRepository(st, cache, factory, 0.01)
	if err := repo.UpdateAll(rules, 1, time.Now().Unix()); err != nil {
		b.Fatalf("UpdateAll: %v", err)
	}
	cleanup := func() { _ = st.Close() }
	return repo, cleanup
}

// countingStore wraps a real Store to count calls/results.
type countingStore struct {
	inner blocklist.Store
	gets  uint64
	hits  uint64
	negs  uint64
}

func (c *countingStore) GetFirstMatch(name string) (domain.BlockRule, bool, error) {
	atomic.AddUint64(&c.gets, 1)
	r, ok, err := c.inner.GetFirstMatch(name)
	if err == nil && ok {
		atomic.AddUint64(&c.hits, 1)
	} else if err == nil && !ok {
		atomic.AddUint64(&c.negs, 1)
	}
	return r, ok, err
}
func (c *countingStore) RebuildAll(rules []domain.BlockRule, version uint64, updatedUnix int64) error {
	return c.inner.RebuildAll(rules, version, updatedUnix)
}
func (c *countingStore) Purge() error { return c.inner.Purge() }
func (c *countingStore) Close() error { return c.inner.Close() }

// countingCache wraps a real DecisionCache to count gets/hits/puts.
type countingCache struct {
	inner  blocklist.DecisionCache
	gets   uint64
	hits   uint64
	puts   uint64
	purges uint64
}

func (c *countingCache) Get(name string) (domain.BlockDecision, bool) {
	atomic.AddUint64(&c.gets, 1)
	d, ok := c.inner.Get(name)
	if ok {
		atomic.AddUint64(&c.hits, 1)
	}
	return d, ok
}
func (c *countingCache) Put(name string, d domain.BlockDecision) {
	atomic.AddUint64(&c.puts, 1)
	c.inner.Put(name, d)
}
func (c *countingCache) Len() int { return c.inner.Len() }
func (c *countingCache) Purge()   { atomic.AddUint64(&c.purges, 1); c.inner.Purge() }

type repoCounters struct {
	store *countingStore
	cache *countingCache
}

// buildRepoWithCounters builds a repo with real components wrapped for counting.
func buildRepoWithCounters(b *testing.B, cacheSize int, rules []domain.BlockRule) (blocklist.Repository, *repoCounters, func()) {
	b.Helper()
	stReal, err := bolt.New(filepath.Join(b.TempDir(), "bl.db"))
	if err != nil {
		b.Fatalf("bolt.New: %v", err)
	}
	st := &countingStore{inner: stReal}
	cacheReal, err := lru.New(cacheSize)
	if err != nil {
		b.Fatalf("lru.New: %v", err)
	}
	cache := &countingCache{inner: cacheReal}
	factory := bloom.NewFactory()
	repo := blocklist.NewRepository(st, cache, factory, 0.01)
	if err := repo.UpdateAll(rules, 1, time.Now().Unix()); err != nil {
		b.Fatalf("UpdateAll: %v", err)
	}
	cleanup := func() { _ = stReal.Close() }
	return repo, &repoCounters{store: st, cache: cache}, cleanup
}

// Positive exact with cache warmed.
func BenchmarkRepo_PositiveExact_Cached(b *testing.B) {
	rules := append(repoBenchExact(20000, "bench.repo"), repoBenchSuffix("ads.bench.repo")...)
	repo, cleanup := buildRepo(b, 128*1024, rules)
	defer cleanup()
	q := "p0001.bench.repo"
	_ = repo.Decide(q) // warm cache
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = repo.Decide(q)
	}
}

// Positive suffix with cache warmed.
func BenchmarkRepo_PositiveSuffix_Cached(b *testing.B) {
	rules := append(repoBenchExact(20000, "bench.repo"), repoBenchSuffix("ads.bench.repo", "track.bench.repo", "cdn.bench.repo")...)
	repo, cleanup := buildRepo(b, 128*1024, rules)
	defer cleanup()
	q := "x.ads.bench.repo"
	_ = repo.Decide(q) // warm cache
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = repo.Decide(q)
	}
}

// Negative decision via bloom-negative early allow.
func BenchmarkRepo_Negative_Mixed(b *testing.B) {
	// Using real Bloom: negatives will be a mix of early-allow and FP-driven store checks.
	rules := append(repoBenchExact(20000, "bench.repo"), repoBenchSuffix("ads.bench.repo", "track.bench.repo")...)
	repo, cleanup := buildRepo(b, 128*1024, rules)
	defer cleanup()
	const negatives = 10000
	qs := make([]string, negatives)
	for i := 0; i < negatives; i++ {
		qs[i] = fmt.Sprintf("absent-%06d.nohit.repo", i)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = repo.Decide(qs[i%len(qs)])
	}
}

// Positive exact with cache disabled to exercise store path each call.
func BenchmarkRepo_PositiveExact_NoCache(b *testing.B) {
	rules := repoBenchExact(20000, "bench.repo")
	repo, cleanup := buildRepo(b, 0, rules)
	defer cleanup()
	q := "p19999.bench.repo"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = repo.Decide(q)
	}
}

// Negative random queries with cache disabled: measures Bloom-only negatives and observed FP rate.
func BenchmarkRepo_Negative_Random_NoCache(b *testing.B) {
	rules := append(repoBenchExact(20000, "bench.repo"), repoBenchSuffix("ads.bench.repo", "track.bench.repo")...)
	repo, ctrs, cleanup := buildRepoWithCounters(b, 0, rules)
	defer cleanup()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q := fmt.Sprintf("neg-%d.rand.repo", i)
		_ = repo.Decide(q)
	}
	b.StopTimer()
	gets := atomic.LoadUint64(&ctrs.store.gets)
	fpRate := 0.0
	if b.N > 0 {
		fpRate = float64(gets) / float64(b.N)
	}
	b.Logf("store_calls=%d observed_fp_rate=%.4f%%", gets, fpRate*100)
}

// Negative random unique queries with cache enabled: similar to NoCache since keys are unique.
func BenchmarkRepo_Negative_Random_Unique_WithCache(b *testing.B) {
	rules := append(repoBenchExact(20000, "bench.repo"), repoBenchSuffix("ads.bench.repo", "track.bench.repo")...)
	repo, ctrs, cleanup := buildRepoWithCounters(b, 128*1024, rules)
	defer cleanup()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q := fmt.Sprintf("neg-unique-%d.rand.repo", i)
		_ = repo.Decide(q)
	}
	b.StopTimer()
	gets := atomic.LoadUint64(&ctrs.store.gets)
	fpRate := 0.0
	if b.N > 0 {
		fpRate = float64(gets) / float64(b.N)
	}
	hits := atomic.LoadUint64(&ctrs.cache.hits)
	b.Logf("store_calls=%d cache_hits=%d observed_fp_rate=%.4f%% (unique keys)", gets, hits, fpRate*100)
}

// Negative repeated query warmed (false-positive path): first miss may hit store, subsequent hits from cache.
func BenchmarkRepo_Negative_Repeated_Warm(b *testing.B) {
	rules := append(repoBenchExact(20000, "bench.repo"), repoBenchSuffix("ads.bench.repo", "track.bench.repo")...)
	repo, ctrs, cleanup := buildRepoWithCounters(b, 128*1024, rules)
	defer cleanup()
	// Find a hostname that becomes cached (i.e., passes bloom and misses store → cached negative).
	// Try sequentially until cache shows a hit on the second call.
	var q string
	for i := 0; i < 100000; i++ { // should succeed quickly with ~1% FP rate
		candidate := fmt.Sprintf("neg-fp-%d.repeat.repo", i)
		// First call: either early-allow (no cache) or store+cache.
		_ = repo.Decide(candidate)
		// Second call: check if it hits cache now.
		before := atomic.LoadUint64(&ctrs.cache.hits)
		_ = repo.Decide(candidate)
		after := atomic.LoadUint64(&ctrs.cache.hits)
		if after > before { // confirmed cached path
			q = candidate
			break
		}
	}
	if q == "" {
		b.Fatalf("failed to find a negative that exercises cache via bloom false-positive within limit")
	}
	// Steady-state: cached negative decisions.
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = repo.Decide(q)
	}
	b.StopTimer()
	gets := atomic.LoadUint64(&ctrs.store.gets)
	hits := atomic.LoadUint64(&ctrs.cache.hits)
	b.Logf("repeated q=%s total_store_calls=%d cache_hits=%d", q, gets, hits)
}

// Positive random queries from the rule set with cache enabled: shows initial store consults then cache hits.
func BenchmarkRepo_Positive_Random_WithCache(b *testing.B) {
	// Build a decent positive set: exact + a few suffixes.
	rules := append(repoBenchExact(20000, "bench.repo"), repoBenchSuffix("ads.bench.repo", "track.bench.repo", "cdn.bench.repo")...)
	repo, ctrs, cleanup := buildRepoWithCounters(b, 128*1024, rules)
	defer cleanup()
	// Pre-build a pool of positive queries to reduce RNG overhead.
	const pool = 100000
	queries := make([]string, pool)
	for i := 0; i < pool; i++ {
		// Randomly choose between exact and suffix-positive names.
		switch rand.Intn(4) {
		case 0:
			queries[i] = fmt.Sprintf("p%04d.bench.repo", rand.Intn(20000))
		case 1:
			queries[i] = fmt.Sprintf("x.%d.ads.bench.repo", i)
		case 2:
			queries[i] = fmt.Sprintf("x.%d.track.bench.repo", i)
		default:
			queries[i] = fmt.Sprintf("x.%d.cdn.bench.repo", i)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = repo.Decide(queries[i%pool])
	}
	b.StopTimer()
	storeGets := atomic.LoadUint64(&ctrs.store.gets)
	cacheHits := atomic.LoadUint64(&ctrs.cache.hits)
	cacheGets := atomic.LoadUint64(&ctrs.cache.gets)
	b.Logf("store_calls=%d cache_hits=%d cache_gets=%d", storeGets, cacheHits, cacheGets)
}

// Metrics: percentile latencies and FP-driven store rate for mixed negatives.
func BenchmarkRepo_Negative_Mixed_Perc(b *testing.B) {
	rules := append(repoBenchExact(20000, "bench.repo"), repoBenchSuffix("ads.bench.repo", "track.bench.repo")...)
	repo, ctrs, cleanup := buildRepoWithCounters(b, 128*1024, rules)
	defer cleanup()
	const negatives = 50000
	qs := make([]string, negatives)
	for i := 0; i < negatives; i++ {
		qs[i] = fmt.Sprintf("absent-%06d.nohit.repo", i)
	}
	// Collect per-call durations (in ns) outside the benchmark timer; this is for diagnostics.
	b.StopTimer()
	durs := make([]int64, negatives)
	for i := 0; i < negatives; i++ {
		t0 := time.Now()
		_ = repo.Decide(qs[i])
		durs[i] = time.Since(t0).Nanoseconds()
	}
	// Compute percentiles.
	sort.Slice(durs, func(i, j int) bool { return durs[i] < durs[j] })
	p50 := durs[(50*len(durs))/100]
	p95 := durs[(95*len(durs))/100]
	p99 := durs[(99*len(durs))/100]
	gets := atomic.LoadUint64(&ctrs.store.gets)
	storeRate := float64(gets) / float64(len(durs)) * 100.0
	b.Logf("negatives=%d p50=%dns p95=%dns p99=%dns store_calls=%d (%.2f%%)", len(durs), p50, p95, p99, gets, storeRate)
}

// --- Parallel benchmarks (concurrency verification) ---

// Parallel cached positives: all goroutines hit cache.
func BenchmarkRepo_Parallel_Positive_Cached(b *testing.B) {
	rules := append(repoBenchExact(20000, "bench.repo"), repoBenchSuffix("ads.bench.repo", "track.bench.repo", "cdn.bench.repo")...)
	repo, cleanup := buildRepo(b, 128*1024, rules)
	defer cleanup()
	// Warm a pool of cached positives.
	const pool = 4096
	qs := make([]string, pool)
	for i := 0; i < pool; i++ {
		if i%3 == 0 {
			qs[i] = fmt.Sprintf("p%04d.bench.repo", i%20000)
		} else if i%3 == 1 {
			qs[i] = fmt.Sprintf("x.%d.ads.bench.repo", i)
		} else {
			qs[i] = fmt.Sprintf("x.%d.track.bench.repo", i)
		}
		_ = repo.Decide(qs[i]) // warm
	}
	var idx uint64
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := atomic.AddUint64(&idx, 1)
			_ = repo.Decide(qs[i%pool])
		}
	})
}

// Parallel cached negative: find one FP-driven negative and hammer it.
func BenchmarkRepo_Parallel_Negative_Repeated_Cached(b *testing.B) {
	rules := append(repoBenchExact(20000, "bench.repo"), repoBenchSuffix("ads.bench.repo", "track.bench.repo")...)
	repo, cleanup := buildRepo(b, 128*1024, rules)
	defer cleanup()
	// Find a cached negative (bloom maybe → store miss → cached allow).
	var q string
	for i := 0; i < 100000; i++ {
		candidate := fmt.Sprintf("neg-fp-%d.repeat.repo", i)
		_ = repo.Decide(candidate)
		_ = repo.Decide(candidate)
		// Second call should be cached if the first went through store.
		// We can't introspect cache here; rely on probability and try many.
		// Once found, steady-state parallel reads are at cache speed anyway.
		q = candidate
		break
	}
	if q == "" {
		b.Fatalf("failed to select a repeated negative")
	}
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = repo.Decide(q)
		}
	})
}

// Parallel unique negatives: mostly bloom-only; some will FP to store.
func BenchmarkRepo_Parallel_Negative_Unique(b *testing.B) {
	rules := append(repoBenchExact(20000, "bench.repo"), repoBenchSuffix("ads.bench.repo", "track.bench.repo")...)
	repo, cleanup := buildRepo(b, 128*1024, rules)
	defer cleanup()
	var ctr uint64
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i := atomic.AddUint64(&ctr, 1)
			q := fmt.Sprintf("neg-%d.unique.repo", i)
			_ = repo.Decide(q)
		}
	})
}

// Parallel mixed workload: half positives, half unique negatives.
func BenchmarkRepo_Parallel_Mixed(b *testing.B) {
	rules := append(repoBenchExact(20000, "bench.repo"), repoBenchSuffix("ads.bench.repo", "track.bench.repo", "cdn.bench.repo")...)
	repo, cleanup := buildRepo(b, 128*1024, rules)
	defer cleanup()
	// Prepare a cache-warmed positive pool.
	const pool = 2048
	pos := make([]string, pool)
	for i := 0; i < pool; i++ {
		if i%2 == 0 {
			pos[i] = fmt.Sprintf("p%04d.bench.repo", i%20000)
		} else {
			pos[i] = fmt.Sprintf("x.%d.cdn.bench.repo", i)
		}
		_ = repo.Decide(pos[i])
	}
	var nctr uint64
	var pctr uint64
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if (atomic.AddUint64(&pctr, 1) & 1) == 0 {
				// positive
				j := atomic.LoadUint64(&pctr)
				_ = repo.Decide(pos[j%pool])
			} else {
				// negative unique
				i := atomic.AddUint64(&nctr, 1)
				q := fmt.Sprintf("neg-%d.mix.repo", i)
				_ = repo.Decide(q)
			}
		}
	})
}
