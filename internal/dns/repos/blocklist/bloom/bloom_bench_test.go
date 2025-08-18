package bloom

import (
	"fmt"
	"testing"
)

func benchMakeDomains(n int, suffix string) [][]byte {
	out := make([][]byte, n)
	for i := 0; i < n; i++ {
		out[i] = []byte(fmt.Sprintf("d%03d.%s", i, suffix))
	}
	return out
}

// Benchmark positive and negative lookups on a Bloom filter populated with 1,000 domains.
func BenchmarkBloom_Positive(b *testing.B) {
	const n = 1000
	f := NewFactory()
	bf := f.New(n, 0.01)
	keys := benchMakeDomains(n, "bench.test")
	for _, k := range keys {
		bf.Add(k)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bf.MightContain(keys[i%len(keys)])
	}
}

func BenchmarkBloom_Negative(b *testing.B) {
	const n = 1000
	f := NewFactory()
	bf := f.New(n, 0.01)
	// populate with a separate set
	present := benchMakeDomains(n, "present.test")
	for _, k := range present {
		bf.Add(k)
	}
	// query a disjoint set
	absent := benchMakeDomains(n, "absent.test")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bf.MightContain(absent[i%len(absent)])
	}
}

// BenchmarkBloom_FalsePositiveRate measures the observed false positive rate
// by querying a disjoint set and reporting fp_count and fp_percent metrics.
func BenchmarkBloom_FalsePositiveRate(b *testing.B) {
	const n = 1000
	const p = 0.01
	const trials = 100_000

	f := NewFactory()
	bf := f.New(n, p)
	present := benchMakeDomains(n, "present.fpr")
	for _, k := range present {
		bf.Add(k)
	}
	absent := benchMakeDomains(trials, "absent.fpr")

	b.ReportAllocs()
	b.ResetTimer()
	fp := 0
	for i := 0; i < trials; i++ {
		if bf.MightContain(absent[i]) {
			fp++
		}
	}
	b.StopTimer()
	rate := float64(fp) / float64(trials)
	b.ReportMetric(float64(fp), "fp_count")
	b.ReportMetric(rate*100, "fp_percent")
}
