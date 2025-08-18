package bolt

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

func benchMakeExact(n int, suffix string) []domain.BlockRule {
	out := make([]domain.BlockRule, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, domain.BlockRule{
			Name:    fmt.Sprintf("d%04d.%s", i, suffix),
			Kind:    domain.BlockRuleExact,
			Source:  "bench",
			AddedAt: time.Unix(1, 0),
		})
	}
	return out
}

func benchBuildStore(b *testing.B, rules []domain.BlockRule) (closeFn func(), st interface {
	GetFirstMatch(string) (domain.BlockRule, bool, error)
	RebuildAll([]domain.BlockRule, uint64, int64) error
	Close() error
}) {
	b.Helper()
	dir := b.TempDir()
	path := filepath.Join(dir, "bl.db")
	store, err := New(path)
	if err != nil {
		b.Fatalf("bolt.New: %v", err)
	}
	if err := store.RebuildAll(rules, 1, time.Now().Unix()); err != nil {
		b.Fatalf("RebuildAll: %v", err)
	}
	return func() { _ = store.Close() }, store
}

// Exact positive: hits an existing exact key.
func BenchmarkBolt_GetFirstMatch_Exact_Positive(b *testing.B) {
	rules := benchMakeExact(1000, "example.bench")
	closeFn, st := benchBuildStore(b, rules)
	b.Cleanup(closeFn)
	// create a round-robin set of present queries
	queries := make([]string, len(rules))
	for i := range rules {
		queries[i] = rules[i].Name
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = st.GetFirstMatch(queries[i%len(queries)])
	}
}

// Exact negative: query a disjoint name not in the dataset.
func BenchmarkBolt_GetFirstMatch_Exact_Negative(b *testing.B) {
	rules := benchMakeExact(1000, "present.bench")
	closeFn, st := benchBuildStore(b, rules)
	b.Cleanup(closeFn)
	q := "absent.present.bench"

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = st.GetFirstMatch(q)
	}
}

// Suffix positive: anchor present, subdomain matches.
func BenchmarkBolt_GetFirstMatch_Suffix_Positive(b *testing.B) {
	rules := append(benchMakeExact(500, "data.bench"), domain.BlockRule{
		Name:    "example.org",
		Kind:    domain.BlockRuleSuffix,
		Source:  "bench",
		AddedAt: time.Unix(1, 0),
	})
	closeFn, st := benchBuildStore(b, rules)
	b.Cleanup(closeFn)
	q := "sub.example.org"

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = st.GetFirstMatch(q)
	}
}

// Suffix negative: no matching anchor.
func BenchmarkBolt_GetFirstMatch_Suffix_Negative(b *testing.B) {
	rules := append(benchMakeExact(500, "data.bench"), domain.BlockRule{
		Name:    "present.example",
		Kind:    domain.BlockRuleSuffix,
		Source:  "bench",
		AddedAt: time.Unix(1, 0),
	})
	closeFn, st := benchBuildStore(b, rules)
	b.Cleanup(closeFn)
	q := "absent.example.org"

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = st.GetFirstMatch(q)
	}
}
