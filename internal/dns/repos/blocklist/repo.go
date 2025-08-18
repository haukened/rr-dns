package blocklist

import (
	"strings"
	"sync"

	"github.com/haukened/rr-dns/internal/dns/common/utils"
	"github.com/haukened/rr-dns/internal/dns/domain"
)

// repository implements the Repository interface by composing a Store,
// a Bloom filter (via factory), and a DecisionCache. It applies a cache → bloom → store pipeline
// on reads and performs atomic snapshot updates on writes.
type repository struct {
	mu      sync.RWMutex
	store   Store
	cache   DecisionCache
	bloom   BloomFilter
	factory BloomFactory
	fpRate  float64
}

// NewRepository constructs a Repository.
// fpRate is the target false-positive rate for the Bloom filter when rebuilding.
func NewRepository(store Store, cache DecisionCache, factory BloomFactory, fpRate float64) Repository {
	return &repository{store: store, cache: cache, factory: factory, fpRate: fpRate}
}

// Decide returns a BlockDecision for the provided domain name.
// Policy: on internal errors, prefer Allow (not blocked).
func (r *repository) Decide(name string) domain.BlockDecision {
	cn := utils.CanonicalDNSName(name)
	// 1) checkBloom: early-allow if definitively negative
	if !r.checkBloom(cn) {
		return domain.EmptyDecision()
	}
	// 2) checkCache
	if d, ok := r.checkCache(cn); ok {
		return d
	}
	// 3) checkStore
	dec := r.checkStore(cn)
	// 4) updateCache
	r.updateCache(cn, dec)
	return dec
}

// UpdateAll performs an atomic snapshot update across store, bloom, and cache.
func (r *repository) UpdateAll(rules []domain.BlockRule, version uint64, updatedUnix int64) error {
	// 1) Rebuild the persistent store first.
	if err := r.store.RebuildAll(rules, version, updatedUnix); err != nil {
		return err
	}

	// 2) Build a fresh Bloom filter sized for the dataset.
	// Count supported rules.
	var n uint64
	for _, ru := range rules {
		if ru.Kind == domain.BlockRuleExact || ru.Kind == domain.BlockRuleSuffix {
			n++
		}
	}
	bf := r.factory.New(n, r.fpRate)
	for _, ru := range rules {
		switch ru.Kind {
		case domain.BlockRuleExact:
			bf.Add([]byte(ru.Name))
		case domain.BlockRuleSuffix:
			bf.Add([]byte(reverseString(ru.Name)))
		default:
			// ignore
		}
	}

	// 3) Swap bloom and purge decision cache under lock.
	r.mu.Lock()
	r.bloom = bf
	r.cache.Purge()
	r.mu.Unlock()
	return nil
}

// reverseString reverses the string bytes. Must match the store's reversal logic
// used for suffix anchors to keep Bloom keys aligned with Bolt keys.
func reverseString(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

// checkBloom returns true if we should consult the store (maybe-positive),
// or false if we can early-allow (definitely negative). If no bloom is loaded,
// returns true to allow authoritative checking.
func (r *repository) checkBloom(cn string) bool {
	r.mu.RLock()
	bf := r.bloom
	r.mu.RUnlock()
	if bf == nil {
		return true
	}
	if bf.MightContain([]byte(cn)) {
		return true
	}
	// test reversed anchors for suffix candidates, most-specific → apex
	a := cn
	for {
		rev := reverseString(a)
		if bf.MightContain([]byte(rev)) {
			return true
		}
		if i := strings.IndexByte(a, '.'); i >= 0 {
			a = a[i+1:]
			if a == "" {
				break
			}
		} else {
			break
		}
	}
	return false
}

// checkCache returns a cached decision when present.
func (r *repository) checkCache(cn string) (domain.BlockDecision, bool) {
	r.mu.RLock()
	d, ok := r.cache.Get(cn)
	r.mu.RUnlock()
	return d, ok
}

// checkStore consults the authoritative store and materializes a decision.
// On any error or miss, returns Allow (EmptyDecision).
func (r *repository) checkStore(cn string) domain.BlockDecision {
	rule, ok, err := r.store.GetFirstMatch(cn)
	if err == nil && ok {
		return domain.BlockDecision{Blocked: true, MatchedRule: rule.Name, Source: rule.Source, Kind: rule.Kind}
	}
	return domain.EmptyDecision()
}

// updateCache writes the final decision.
func (r *repository) updateCache(cn string, dec domain.BlockDecision) {
	r.mu.Lock()
	r.cache.Put(cn, dec)
	r.mu.Unlock()
}
