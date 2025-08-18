package blocklist

import (
	"errors"
	"reflect"
	"testing"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

func TestReverseString(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"a", "a"},
		{"ab", "ba"},
		{"abc", "cba"},
		{"domain.com", "moc.niamod"},
		{"example.", ".elpmaxe"},
		{"sub.domain.com", "moc.niamod.bus"},
		{"12345", "54321"},
		{"a.b.c", "c.b.a"},
		{"你好", "好你"},
	}

	for _, tt := range tests {
		got := reverseString(tt.in)
		if got != tt.want {
			t.Errorf("reverseString(%q) = %q; want %q", tt.in, got, tt.want)
		}
	}
}

// --- fakes ---

type fakeStore struct {
	getRule      domain.BlockRule
	getOK        bool
	getErr       error
	getCalls     int
	rebuildRules []domain.BlockRule
	rebuildVer   uint64
	rebuildUpd   int64
	rebuildCalls int
	rebuildErr   error
}

func (s *fakeStore) GetFirstMatch(name string) (domain.BlockRule, bool, error) {
	s.getCalls++
	return s.getRule, s.getOK, s.getErr
}

func (s *fakeStore) RebuildAll(rules []domain.BlockRule, version uint64, updatedUnix int64) error {
	s.rebuildCalls++
	s.rebuildRules = append([]domain.BlockRule(nil), rules...)
	s.rebuildVer = version
	s.rebuildUpd = updatedUnix
	return s.rebuildErr
}

func (s *fakeStore) Purge() error      { return nil }
func (s *fakeStore) Close() error      { return nil }
func (s *fakeStore) Stats() StoreStats { return StoreStats{} }

type fakeCache struct {
	m          map[string]domain.BlockDecision
	getCalls   int
	putCalls   int
	purgeCalls int
	lastPutKey string
	lastPutVal domain.BlockDecision
}

func newFakeCache() *fakeCache { return &fakeCache{m: make(map[string]domain.BlockDecision)} }

func (c *fakeCache) Get(name string) (domain.BlockDecision, bool) {
	c.getCalls++
	v, ok := c.m[name]
	return v, ok
}

func (c *fakeCache) Put(name string, d domain.BlockDecision) {
	c.putCalls++
	c.lastPutKey = name
	c.lastPutVal = d
	c.m[name] = d
}

func (c *fakeCache) Len() int { return len(c.m) }
func (c *fakeCache) Purge()   { c.purgeCalls++; c.m = make(map[string]domain.BlockDecision) }
func (c *fakeCache) Stats() CacheStats {
	return CacheStats{Capacity: 0, Size: len(c.m)}
}

type fakeBloom struct {
	contains map[string]bool
	added    []string
}

func newFakeBloom() *fakeBloom { return &fakeBloom{contains: make(map[string]bool)} }

func (b *fakeBloom) Add(key []byte) { b.added = append(b.added, string(key)) }

func (b *fakeBloom) MightContain(key []byte) bool { return b.contains[string(key)] }

type fakeFactory struct {
	newCap   uint64
	newFp    float64
	newCalls int
	ret      *fakeBloom
}

func (f *fakeFactory) New(capacity uint64, fpRate float64) BloomFilter {
	f.newCalls++
	f.newCap = capacity
	f.newFp = fpRate
	if f.ret == nil {
		f.ret = newFakeBloom()
	}
	return f.ret
}

// --- tests ---

func TestDecide_BloomNegativeEarlyAllow(t *testing.T) {
	st := &fakeStore{}
	ca := newFakeCache()
	bf := newFakeBloom() // empty: everything negative
	repo := &repository{store: st, cache: ca, bloom: bf}

	dec := repo.Decide("sub.Domain.com.") // mixed case with dot to ensure canonicalization still early-allows
	if dec.Blocked {
		t.Fatalf("want allow, got blocked decision: %+v", dec)
	}
	if st.getCalls != 0 {
		t.Fatalf("store should not be consulted when bloom is negative; got %d calls", st.getCalls)
	}
	if ca.getCalls != 1 {
		t.Fatalf("cache should be checked first even on bloom-negative; gets=%d", ca.getCalls)
	}
	if ca.putCalls != 0 {
		t.Fatalf("cache should not be updated on early allow; puts=%d", ca.putCalls)
	}
}

func TestDecide_CacheHitShortCircuit(t *testing.T) {
	st := &fakeStore{}
	ca := newFakeCache()
	ca.m["example.com"] = domain.BlockDecision{Blocked: true, MatchedRule: "example.com", Kind: domain.BlockRuleExact}
	bf := newFakeBloom()
	bf.contains["example.com"] = true // bloom positive so we'd proceed if not cached

	repo := &repository{store: st, cache: ca, bloom: bf}
	dec := repo.Decide("ExAmPlE.CoM.") // canonicalizes to example.com

	if !dec.Blocked || dec.MatchedRule != "example.com" || dec.Kind != domain.BlockRuleExact {
		t.Fatalf("unexpected decision: %+v", dec)
	}
	if st.getCalls != 0 {
		t.Fatalf("store should not be consulted on cache hit; got %d calls", st.getCalls)
	}
	if ca.putCalls != 0 {
		t.Fatalf("cache should not be updated on cache hit; puts=%d", ca.putCalls)
	}
}

func TestDecide_StoreHit_CachesAndReturns(t *testing.T) {
	st := &fakeStore{getRule: domain.BlockRule{Name: "example.com", Kind: domain.BlockRuleExact, Source: "s"}, getOK: true}
	ca := newFakeCache()
	bf := newFakeBloom()
	bf.contains["example.com"] = true
	repo := &repository{store: st, cache: ca, bloom: bf}

	dec := repo.Decide("example.com")
	if !dec.Blocked || dec.MatchedRule != "example.com" || dec.Kind != domain.BlockRuleExact || dec.Source != "s" {
		t.Fatalf("unexpected decision from store hit: %+v", dec)
	}
	if ca.putCalls != 1 || ca.lastPutKey != "example.com" || !reflect.DeepEqual(ca.lastPutVal, dec) {
		t.Fatalf("cache not updated correctly; calls=%d key=%q val=%+v", ca.putCalls, ca.lastPutKey, ca.lastPutVal)
	}
}

func TestDecide_StoreError_AllowsAndCaches(t *testing.T) {
	st := &fakeStore{getErr: errors.New("boom")}
	ca := newFakeCache()
	bf := newFakeBloom()
	// Force bloom positive via suffix anchor to exercise loop path
	bf.contains[reverseString("domain.com")] = true
	repo := &repository{store: st, cache: ca, bloom: bf}

	dec := repo.Decide("sub.domain.com")
	if dec.Blocked {
		t.Fatalf("want allow on store error; got: %+v", dec)
	}
	if ca.putCalls != 1 || ca.lastPutKey != "sub.domain.com" || !reflect.DeepEqual(ca.lastPutVal, dec) {
		t.Fatalf("cache should receive allow decision; calls=%d key=%q val=%+v", ca.putCalls, ca.lastPutKey, ca.lastPutVal)
	}
}

func TestDecide_NilBloom_ConsultsStore(t *testing.T) {
	st := &fakeStore{getRule: domain.BlockRule{Name: "ads.example", Kind: domain.BlockRuleSuffix}, getOK: true}
	ca := newFakeCache()
	repo := &repository{store: st, cache: ca, bloom: nil}

	dec := repo.Decide("sub.ads.example")
	if !dec.Blocked || dec.Kind != domain.BlockRuleSuffix || dec.MatchedRule != "ads.example" {
		t.Fatalf("unexpected decision with nil bloom: %+v", dec)
	}
	if st.getCalls != 1 {
		t.Fatalf("expected store to be consulted once; got %d", st.getCalls)
	}
}

func TestUpdateAll_ErrorFromStore(t *testing.T) {
	st := &fakeStore{rebuildErr: errors.New("fail")}
	ca := newFakeCache()
	fac := &fakeFactory{ret: newFakeBloom()}
	// pre-set a bloom to ensure it doesn't change on error
	oldBloom := newFakeBloom()
	repo := &repository{store: st, cache: ca, factory: fac, fpRate: 0.01, bloom: oldBloom}

	rules := []domain.BlockRule{{Name: "example.com", Kind: domain.BlockRuleExact}}
	err := repo.UpdateAll(rules, 42, 12345)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if fac.newCalls != 0 {
		t.Fatalf("bloom factory should not be called on store error; got %d", fac.newCalls)
	}
	if repo.bloom != oldBloom {
		t.Fatalf("bloom should not be swapped on error")
	}
	if ca.purgeCalls != 0 {
		t.Fatalf("cache should not be purged on error")
	}
}

func TestUpdateAll_Success_BuildsBloomAndPurgesCache(t *testing.T) {
	st := &fakeStore{}
	ca := newFakeCache()
	fac := &fakeFactory{ret: newFakeBloom()}
	repo := &repository{store: st, cache: ca, factory: fac, fpRate: 0.02}

	rules := []domain.BlockRule{
		{Name: "example.com", Kind: domain.BlockRuleExact, Source: "s1"},
		{Name: "ads.example.com", Kind: domain.BlockRuleSuffix, Source: "s2"},
		{Name: "ignored.example", Kind: 255, Source: "x"}, // unsupported kind should be ignored in bloom build
	}

	if err := repo.UpdateAll(rules, 7, 999); err != nil {
		t.Fatalf("UpdateAll error: %v", err)
	}

	if st.rebuildCalls != 1 || st.rebuildVer != 7 || st.rebuildUpd != 999 || len(st.rebuildRules) != len(rules) {
		t.Fatalf("store rebuild not called as expected: calls=%d ver=%d upd=%d rules=%d", st.rebuildCalls, st.rebuildVer, st.rebuildUpd, len(st.rebuildRules))
	}

	if fac.newCalls != 1 || fac.newCap != 2 || fac.newFp != 0.02 {
		t.Fatalf("factory not called as expected: calls=%d cap=%d fp=%v", fac.newCalls, fac.newCap, fac.newFp)
	}

	fb := fac.ret
	// exact key added as-is; suffix key added reversed
	wantAdded := map[string]bool{
		"example.com":                    true,
		reverseString("ads.example.com"): true,
		"ignored.example":                false,
	}
	got := make(map[string]bool)
	for _, k := range fb.added {
		got[k] = true
	}
	for k, want := range wantAdded {
		if want && !got[k] {
			t.Fatalf("expected bloom to contain key %q", k)
		}
		if !want && got[k] {
			t.Fatalf("did not expect bloom to contain key %q", k)
		}
	}

	if repo.bloom != BloomFilter(fb) {
		t.Fatalf("bloom not swapped on repo")
	}
	if ca.purgeCalls != 1 {
		t.Fatalf("cache not purged; calls=%d", ca.purgeCalls)
	}
}

func TestCheckBloom_SuffixAnchorPath(t *testing.T) {
	bf := newFakeBloom()
	// only suffix anchor present (domain.com), not full name
	bf.contains[reverseString("domain.com")] = true
	repo := &repository{bloom: bf}

	if !repo.checkBloom("sub.domain.com") {
		t.Fatalf("expected maybe-positive due to suffix anchor")
	}
	// Negative path when no anchors found
	bf2 := newFakeBloom()
	repo2 := &repository{bloom: bf2}
	if repo2.checkBloom("no.hit.example") {
		t.Fatalf("expected negative when bloom has no anchors")
	}
	// Nil bloom returns true to consult store
	repo3 := &repository{bloom: nil}
	if !repo3.checkBloom("anything") {
		t.Fatalf("nil bloom should return true to consult store")
	}
}

func TestCheckBloom_EmptyLabelBreak(t *testing.T) {
	bf := newFakeBloom() // negative
	repo := &repository{bloom: bf}
	if repo.checkBloom(".") { // a becomes "" after first iteration (i==0), hits a=="" break path
		t.Fatalf("expected negative for '.' with empty bloom")
	}
}

func TestNewRepository_SetsFields(t *testing.T) {
	st := &fakeStore{}
	ca := newFakeCache()
	fac := &fakeFactory{}
	repoIface := NewRepository(st, ca, fac, 0.123)
	r, ok := repoIface.(*repository)
	if !ok {
		t.Fatalf("expected *repository concrete type")
	}
	if r.store != st || r.cache != ca || r.factory != fac || r.fpRate != 0.123 {
		t.Fatalf("repository fields not set correctly: %+v", r)
	}
	if r.bloom != nil {
		t.Fatalf("new repository should start with nil bloom")
	}
}
func TestCacheStats_ReturnsCacheStats(t *testing.T) {
	st := &fakeStore{}
	ca := newFakeCache()
	ca.m["foo.com"] = domain.BlockDecision{Blocked: true}
	repo := &repository{store: st, cache: ca}

	stats := repo.CacheStats()
	if stats.Size != 1 {
		t.Fatalf("expected cache size 1, got %d", stats.Size)
	}
}

func TestCacheStats_NilCacheReturnsZeroStats(t *testing.T) {
	repo := &repository{cache: nil}
	stats := repo.CacheStats()
	if stats.Size != 0 || stats.Capacity != 0 {
		t.Fatalf("expected zero stats for nil cache, got %+v", stats)
	}
}
func TestStoreStats_ReturnsStoreStats(t *testing.T) {
	st := &fakeStore{}
	repo := &repository{store: st}

	stats := repo.StoreStats()
	if stats != (StoreStats{}) {
		t.Fatalf("expected zero stats from fakeStore, got %+v", stats)
	}
}

func TestStoreStats_NilStoreReturnsZeroStats(t *testing.T) {
	repo := &repository{store: nil}
	stats := repo.StoreStats()
	if stats.Version != 0 || stats.UpdatedUnix != 0 {
		t.Fatalf("expected zero stats for nil store, got %+v", stats)
	}
}
func TestIsASCII(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"", true},
		{"abc", true},
		{"ABC123", true},
		{"domain.com", true},
		{"with space", true},
		{"with-dash_underscore", true},
		{"\x7F", true},  // DEL is still ASCII
		{"\x80", false}, // first non-ASCII byte
		{"abc\x80", false},
		{"你好", false},
		{"abc你好", false},
		{"😀", false},
		{"abc😀", false},
		{"\xff", false},
	}

	for _, tt := range tests {
		got := isASCII(tt.in)
		if got != tt.want {
			t.Errorf("isASCII(%q) = %v; want %v", tt.in, got, tt.want)
		}
	}
}
func TestRepository_checkBloom_ExactMatch(t *testing.T) {
	bf := newFakeBloom()
	bf.contains["foo.com"] = true
	repo := &repository{bloom: bf}

	if !repo.checkBloom("foo.com") {
		t.Fatalf("expected true for exact match in bloom")
	}
}

func TestRepository_checkBloom_SuffixASCII(t *testing.T) {
	bf := newFakeBloom()
	// Suffix anchor for "bar.com" (reversed)
	bf.contains[reverseString("bar.com")] = true
	repo := &repository{bloom: bf}

	if !repo.checkBloom("sub.bar.com") {
		t.Fatalf("expected true for suffix anchor in bloom (ASCII)")
	}
}

func TestRepository_checkBloom_SuffixUnicode(t *testing.T) {
	bf := newFakeBloom()
	// Suffix anchor for "例子.公司" (reversed)
	bf.contains[reverseString("例子.公司")] = true
	repo := &repository{bloom: bf}

	if !repo.checkBloom("子域.例子.公司") {
		t.Fatalf("expected true for suffix anchor in bloom (Unicode)")
	}
}

// Cover the Unicode fallback loop breaks and the final return false.
func TestRepository_checkBloom_UnicodeBreaksAndReturnFalse(t *testing.T) {
	t.Run("break when a becomes empty after dot", func(t *testing.T) {
		bf := newFakeBloom() // negative bloom
		repo := &repository{bloom: bf}
		// Non-ASCII with trailing dot to force a=="" after trimming at '.'
		if repo.checkBloom("你好.") { // isASCII=false -> Unicode path
			t.Fatalf("expected false for Unicode name with trailing dot and empty bloom")
		}
	})
	t.Run("break when no dot present", func(t *testing.T) {
		bf := newFakeBloom() // negative bloom
		repo := &repository{bloom: bf}
		// Non-ASCII single-label name -> strings.IndexByte returns -1 -> else break
		if repo.checkBloom("例子") { // "example" in Chinese
			t.Fatalf("expected false for single-label Unicode name with empty bloom")
		}
	})
}
