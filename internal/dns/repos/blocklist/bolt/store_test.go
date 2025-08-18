package bolt

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"encoding/binary"

	"github.com/haukened/rr-dns/internal/dns/domain"
	bbolt "go.etcd.io/bbolt"
	bberrors "go.etcd.io/bbolt/errors"
)

func tempDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "bl.db")
}

func TestBoltStore_GetFirstMatch_ExactAndSuffix(t *testing.T) {
	dbPath := tempDB(t)
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = st.Close(); _ = os.Remove(dbPath) })

	// empty DB -> no match
	if _, ok, err := st.GetFirstMatch("a.example.com"); err != nil || ok {
		t.Fatalf("expected empty miss, got ok=%v err=%v", ok, err)
	}

	// Rebuild with a mix of rules
	now := time.Now()
	rules := []domain.BlockRule{
		{Name: "a.example.com", Kind: domain.BlockRuleExact, Source: "t", AddedAt: now},
		{Name: "example.net", Kind: domain.BlockRuleSuffix, Source: "t", AddedAt: now},
	}
	if err := st.RebuildAll(rules, 1, now.Unix()); err != nil {
		t.Fatalf("RebuildAll: %v", err)
	}

	// exact hit
	r, ok, err := st.GetFirstMatch("a.example.com")
	if err != nil || !ok || r.Name != "a.example.com" || r.Kind != domain.BlockRuleExact {
		t.Fatalf("exact unexpected: r=%+v ok=%v err=%v", r, ok, err)
	}

	// suffix hit
	r, ok, err = st.GetFirstMatch("sub.example.net")
	if err != nil || !ok || r.Name != "example.net" || r.Kind != domain.BlockRuleSuffix {
		t.Fatalf("suffix unexpected: r=%+v ok=%v err=%v", r, ok, err)
	}

	// miss
	if _, ok, err = st.GetFirstMatch("nope.tld"); err != nil || ok {
		t.Fatalf("expected miss, got ok=%v err=%v", ok, err)
	}
}

func TestBoltStore_Purge(t *testing.T) {
	dbPath := tempDB(t)
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = st.Close(); _ = os.Remove(dbPath) })

	now := time.Now()
	rules := []domain.BlockRule{{Name: "a.example.com", Kind: domain.BlockRuleExact, Source: "t", AddedAt: now}}
	if err := st.RebuildAll(rules, 1, now.Unix()); err != nil {
		t.Fatalf("RebuildAll: %v", err)
	}
	if _, ok, _ := st.GetFirstMatch("a.example.com"); !ok {
		t.Fatalf("expected hit before purge")
	}
	if err := st.Purge(); err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if _, ok, _ := st.GetFirstMatch("a.example.com"); ok {
		t.Fatalf("expected miss after purge")
	}
}

type fakeBucketCreator struct{ errs map[string]error }

func (f fakeBucketCreator) CreateBucketIfNotExists(name []byte) (*bbolt.Bucket, error) {
	if err := f.errs[string(name)]; err != nil {
		return nil, err
	}
	return nil, nil
}

// Test the error paths for bucket creation by temporarily replacing ensureBucketsFn.
func TestNew_EnsureBucketsErrors(t *testing.T) {
	cases := []struct {
		name string
		fail string
	}{
		{name: "exact bucket fails", fail: string(bucketExact)},
		{name: "suffix bucket fails", fail: string(bucketSuffix)},
		{name: "meta bucket fails", fail: string(bucketMeta)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Swap in a seam that errors on the selected bucket.
			old := ensureBucketsFn
			ensureBucketsFn = func(tx bucketCreator) error {
				fb := fakeBucketCreator{errs: map[string]error{tc.fail: assertErr{}}}
				return ensureBuckets(fb)
			}
			defer func() { ensureBucketsFn = old }()

			// Now create a real DB file, but the Update will invoke our fake and fail.
			dbPath := tempDB(t)
			st, err := New(dbPath)
			if err == nil || st != nil {
				t.Fatalf("expected error from New when %s fails", tc.fail)
			}
			_ = os.Remove(dbPath)
		})
	}
}
func TestDeleteBuckets(t *testing.T) {
	type call struct {
		name string
		err  error
	}
	tests := []struct {
		name    string
		calls   []call
		wantErr bool
	}{
		{
			name: "all buckets deleted successfully",
			calls: []call{
				{name: "a", err: nil},
				{name: "b", err: nil},
			},
			wantErr: false,
		},
		{
			name: "ignore ErrBucketNotFound",
			calls: []call{
				{name: "a", err: bberrors.ErrBucketNotFound},
				{name: "b", err: nil},
			},
			wantErr: false,
		},
		{
			name: "return on first non-ignorable error",
			calls: []call{
				{name: "a", err: assertErr{}},
				{name: "b", err: nil},
			},
			wantErr: true,
		},
		{
			name: "return on second non-ignorable error",
			calls: []call{
				{name: "a", err: nil},
				{name: "b", err: assertErr{}},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			type delCall struct {
				name []byte
			}
			var gotCalls []delCall
			fake := &struct {
				bucketDeleter
			}{
				bucketDeleter: bucketDeleterFunc(func(name []byte) error {
					gotCalls = append(gotCalls, delCall{name})
					for _, c := range tc.calls {
						if string(name) == c.name {
							return c.err
						}
					}
					return nil
				}),
			}
			var names [][]byte
			for _, c := range tc.calls {
				names = append(names, []byte(c.name))
			}
			err := deleteBuckets(fake, names...)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// Ensure New returns an error when the DB file cannot be opened (non-existent parent dir).
func TestNew_OpenError(t *testing.T) {
	base := t.TempDir()
	badPath := filepath.Join(base, "no-such-dir", "bl.db") // parent does not exist
	st, err := New(badPath)
	if err == nil || st != nil {
		t.Fatalf("expected New to fail when parent directory does not exist")
	}
}

// bucketDeleterFunc allows using a func as a bucketDeleter for testing.
type bucketDeleterFunc func(name []byte) error

func (f bucketDeleterFunc) DeleteBucket(name []byte) error {
	return f(name)
}

type assertErr struct{}

func (assertErr) Error() string { return "assert error" }

// Cover decode fallback paths (len(v) < 11) for both exact and suffix rules.
func TestDecodeRuleValueFallbacks(t *testing.T) {
	dbPath := tempDB(t)
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	bs := st.(*boltStore)
	t.Cleanup(func() { _ = st.Close(); _ = os.Remove(dbPath) })

	// Insert minimal value for exact rule.
	if err := bs.db.Update(func(tx *bbolt.Tx) error {
		eb := tx.Bucket(bucketExact)
		return eb.Put([]byte("short.example"), []byte{})
	}); err != nil {
		t.Fatalf("seed exact: %v", err)
	}
	r, ok, err := st.GetFirstMatch("short.example")
	if err != nil || !ok {
		t.Fatalf("expected exact fallback hit: ok=%v err=%v", ok, err)
	}
	if r.Kind != domain.BlockRuleExact || !r.AddedAt.IsZero() || r.Source != "" {
		t.Fatalf("unexpected exact fallback decode: %+v", r)
	}

	// Insert minimal value for suffix rule (reversed key).
	if err := bs.db.Update(func(tx *bbolt.Tx) error {
		sb := tx.Bucket(bucketSuffix)
		key := []byte("gro.elpmaxe") // reverse of example.org
		return sb.Put(key, []byte{})
	}); err != nil {
		t.Fatalf("seed suffix: %v", err)
	}
	r, ok, err = st.GetFirstMatch("a.example.org")
	if err != nil || !ok {
		t.Fatalf("expected suffix fallback hit: ok=%v err=%v", ok, err)
	}
	if r.Kind != domain.BlockRuleSuffix || !r.AddedAt.IsZero() || r.Source != "" {
		t.Fatalf("unexpected suffix fallback decode: %+v", r)
	}
}

func TestPurge_ErrorPaths(t *testing.T) {
	dbPath := tempDB(t)
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = st.Close(); _ = os.Remove(dbPath) })

	// deleteBuckets error
	oldDel := deleteBucketsFn
	deleteBucketsFn = func(_ bucketDeleter, _ ...[]byte) error { return assertErr{} }
	if err := st.Purge(); err == nil {
		t.Fatalf("expected purge to fail on deleteBuckets error")
	}
	deleteBucketsFn = oldDel

	// ensureBuckets error
	oldEns := ensureBucketsFn
	ensureBucketsFn = func(_ bucketCreator) error { return assertErr{} }
	if err := st.Purge(); err == nil {
		t.Fatalf("expected purge to fail on ensureBuckets error")
	}
	ensureBucketsFn = oldEns
}

func TestRebuildAll_ErrorPaths(t *testing.T) {
	dbPath := tempDB(t)
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = st.Close(); _ = os.Remove(dbPath) })

	now := time.Now()
	rules := []domain.BlockRule{{Name: "a.example.com", Kind: domain.BlockRuleExact, Source: "t", AddedAt: now}}

	// deleteBuckets error
	oldDel := deleteBucketsFn
	deleteBucketsFn = func(_ bucketDeleter, _ ...[]byte) error { return assertErr{} }
	if err := st.RebuildAll(rules, 1, now.Unix()); err == nil {
		t.Fatalf("expected rebuild to fail on deleteBuckets error")
	}
	deleteBucketsFn = oldDel

	// ensureBuckets error
	oldEns := ensureBucketsFn
	ensureBucketsFn = func(_ bucketCreator) error { return assertErr{} }
	if err := st.RebuildAll(rules, 1, now.Unix()); err == nil {
		t.Fatalf("expected rebuild to fail on ensureBuckets error")
	}
	ensureBucketsFn = oldEns

	// loadRules error
	oldLoad := loadRulesFn
	loadRulesFn = func(_ *bbolt.Tx, _ []domain.BlockRule) error { return assertErr{} }
	if err := st.RebuildAll(rules, 1, now.Unix()); err == nil {
		t.Fatalf("expected rebuild to fail on loadRules error")
	}
	loadRulesFn = oldLoad

	// writeMeta error
	oldMeta := writeMetaFn
	writeMetaFn = func(_ *bbolt.Tx, _ uint64, _ int64) error { return assertErr{} }
	if err := st.RebuildAll(rules, 1, now.Unix()); err == nil {
		t.Fatalf("expected rebuild to fail on writeMeta error")
	}
	writeMetaFn = oldMeta
}

// Exercise GetFirstMatch when the suffix bucket is missing and when name is empty.
func TestGetFirstMatch_NoSuffixBucket_AndEmptyName(t *testing.T) {
	dbPath := tempDB(t)
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	bs := st.(*boltStore)
	t.Cleanup(func() { _ = st.Close(); _ = os.Remove(dbPath) })

	// Delete the suffix bucket to hit b == nil path.
	if err := bs.db.Update(func(tx *bbolt.Tx) error { return tx.DeleteBucket(bucketSuffix) }); err != nil {
		t.Fatalf("delete suffix bucket: %v", err)
	}
	if _, ok, err := st.GetFirstMatch("anything.example"); err != nil || ok {
		t.Fatalf("expected miss with no suffix bucket, ok=%v err=%v", ok, err)
	}

	// Empty name exercises len(rp)==0 early break.
	if _, ok, err := st.GetFirstMatch(""); err != nil || ok {
		t.Fatalf("expected miss for empty name, ok=%v err=%v", ok, err)
	}
}

// Directly cover decodeRuleValue branches: invalid kind and oversized source length
func TestDecodeRuleValue_InvalidKindAndOversizedSourceLen(t *testing.T) {
	// Build a buffer with invalid kind and an oversized source length.
	v := make([]byte, 11) // header only
	v[0] = 99             // invalid kind
	binary.BigEndian.PutUint64(v[1:9], uint64(time.Unix(123, 0).Unix()))
	binary.BigEndian.PutUint16(v[9:11], 1024) // way larger than available

	r, err := decodeRuleValue("x.example", v, domain.BlockRuleExact)
	if err != nil {
		t.Fatalf("decode err: %v", err)
	}
	if r.Kind != domain.BlockRuleExact { // defaultKind used when invalid
		t.Fatalf("expected default kind, got %v", r.Kind)
	}
	if r.Source != "" { // clamped length results in empty source
		t.Fatalf("expected empty source, got %q", r.Source)
	}
}

// Ensure decodeRuleValue caps oversized uint64 timestamps to MaxInt64 before narrowing to int64.
func TestDecodeRuleValue_TimestampCapToMaxInt64(t *testing.T) {
	// Build a buffer with kind=Exact, timestamp=MaxUint64, and zero-length source.
	v := make([]byte, 11)
	v[0] = byte(domain.BlockRuleExact)
	// Put MaxUint64 into the timestamp field [1:9]
	for i := 1; i < 9; i++ {
		v[i] = 0xFF
	}
	// source length = 0
	binary.BigEndian.PutUint16(v[9:11], 0)

	r, err := decodeRuleValue("cap.example", v, domain.BlockRuleExact)
	if err != nil {
		t.Fatalf("decode err: %v", err)
	}
	// Expect AddedAt clamped to MaxInt64
	maxInt64 := int64(^uint64(0) >> 1)
	if r.AddedAt.Unix() != maxInt64 {
		t.Fatalf("expected AddedAt=%d (MaxInt64), got %d", maxInt64, r.AddedAt.Unix())
	}
	if r.Kind != domain.BlockRuleExact {
		t.Fatalf("expected kind exact, got %v", r.Kind)
	}
	if r.Source != "" {
		t.Fatalf("expected empty source, got %q", r.Source)
	}
}

// When the exact bucket is missing, we should still return a suffix match if present.
func TestGetFirstMatch_NoExactBucket_ButSuffixHit(t *testing.T) {
	dbPath := tempDB(t)
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	bs := st.(*boltStore)
	t.Cleanup(func() { _ = st.Close(); _ = os.Remove(dbPath) })

	now := time.Now()
	rules := []domain.BlockRule{{Name: "example.org", Kind: domain.BlockRuleSuffix, Source: "t", AddedAt: now}}
	if err := st.RebuildAll(rules, 1, now.Unix()); err != nil {
		t.Fatalf("RebuildAll: %v", err)
	}

	// Drop exact bucket to take the b==nil branch for exact lookup.
	if err := bs.db.Update(func(tx *bbolt.Tx) error { return tx.DeleteBucket(bucketExact) }); err != nil {
		t.Fatalf("delete exact bucket: %v", err)
	}

	r, ok, err := st.GetFirstMatch("a.example.org")
	if err != nil || !ok || r.Kind != domain.BlockRuleSuffix || r.Name != "example.org" {
		t.Fatalf("expected suffix hit with no exact bucket: r=%+v ok=%v err=%v", r, ok, err)
	}
}

// Verify readers can continue without disruption while RebuildAll swaps data (RCU-style snapshot semantics),
// and that the active version marker (Stats().Version) reflects the new snapshot after the swap.
func TestRebuildAll_ConcurrentReadsAndVersionMarker(t *testing.T) {
	dbPath := tempDB(t)
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = st.Close(); _ = os.Remove(dbPath) })

	now := time.Now()
	// Initial dataset V1
	v1 := []domain.BlockRule{
		{Name: "a.example.com", Kind: domain.BlockRuleExact, Source: "v1", AddedAt: now},
		{Name: "example.org", Kind: domain.BlockRuleSuffix, Source: "v1", AddedAt: now},
	}
	if err := st.RebuildAll(v1, 1, now.Unix()); err != nil {
		t.Fatalf("RebuildAll v1: %v", err)
	}

	// Prepare dataset V2 with different answers
	v2 := []domain.BlockRule{
		{Name: "b.example.com", Kind: domain.BlockRuleExact, Source: "v2", AddedAt: now.Add(time.Second)},
		{Name: "example.net", Kind: domain.BlockRuleSuffix, Source: "v2", AddedAt: now.Add(time.Second)},
	}

	// Start readers that continuously query while we rebuild.
	done := make(chan struct{})
	var wg sync.WaitGroup
	// Queries that will hit in V1 but not necessarily in V2, and vice versa.
	queries := []string{
		"a.example.com", // exact in V1
		"x.example.org", // suffix in V1
		"b.example.com", // exact in V2
		"y.example.net", // suffix in V2
		"nope.tld",      // always miss
	}
	// Launch several reader goroutines.
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					for _, q := range queries {
						// GetFirstMatch should never error or panic; ok can be true/false depending on timing.
						if _, _, err := st.GetFirstMatch(q); err != nil {
							t.Errorf("GetFirstMatch(%q) err: %v", q, err)
							return
						}
					}
				}
			}
		}()
	}

	// Give readers a moment to start, then swap to V2.
	time.Sleep(25 * time.Millisecond)
	if err := st.RebuildAll(v2, 2, now.Add(time.Second).Unix()); err != nil {
		t.Fatalf("RebuildAll v2: %v", err)
	}
	// Let readers observe post-swap state too.
	time.Sleep(25 * time.Millisecond)
	close(done)
	wg.Wait()

	// Validate version marker reflects latest snapshot.
	stats := st.Stats()
	if stats.Version != 2 {
		t.Fatalf("expected Stats().Version=2, got %d", stats.Version)
	}
	if stats.ExactKeys == 0 || stats.SuffixKeys == 0 {
		t.Fatalf("expected non-zero key counts after swap, got exact=%d suffix=%d", stats.ExactKeys, stats.SuffixKeys)
	}

	// Spot-check a couple of lookups post-swap.
	if r, ok, err := st.GetFirstMatch("b.example.com"); err != nil || !ok || r.Source != "v2" {
		t.Fatalf("post-swap exact miss or wrong source: r=%+v ok=%v err=%v", r, ok, err)
	}
	if r, ok, err := st.GetFirstMatch("z.example.net"); err != nil || !ok || r.Source != "v2" {
		t.Fatalf("post-swap suffix miss or wrong source: r=%+v ok=%v err=%v", r, ok, err)
	}
}

// Ensure unsupported kinds are ignored by loadRules.
func TestRebuildAll_UnsupportedKindIgnored(t *testing.T) {
	dbPath := tempDB(t)
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = st.Close(); _ = os.Remove(dbPath) })

	now := time.Now()
	bad := domain.BlockRule{Name: "bad.example", Kind: 99, Source: "t", AddedAt: now}
	if err := st.RebuildAll([]domain.BlockRule{bad}, 1, now.Unix()); err != nil {
		t.Fatalf("RebuildAll: %v", err)
	}
	if _, ok, err := st.GetFirstMatch("bad.example"); err != nil || ok {
		t.Fatalf("unsupported kind should be ignored, ok=%v err=%v", ok, err)
	}
}

// Smoke-test the small helpers reverseString and reverseBytesInPlace for odd/even lengths.
func TestReverseHelpers(t *testing.T) {
	if got := reverseString("abc"); got != "cba" {
		t.Fatalf("reverseString abc => %s", got)
	}
	if got := reverseString("abcd"); got != "dcba" {
		t.Fatalf("reverseString abcd => %s", got)
	}
	in := []byte("abcde")
	if got := string(reverseBytesInPlace(in)); got != "edcba" {
		t.Fatalf("reverseBytesInPlace abcde => %s", got)
	}
}

// Ensure len(rp)==0 early break in suffix loop is covered (suffix bucket exists).
func TestGetFirstMatch_EmptyNameWithSuffixBucket(t *testing.T) {
	dbPath := tempDB(t)
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = st.Close(); _ = os.Remove(dbPath) })

	// Suffix bucket exists by default after New; call with empty name to hit len(rp)==0.
	if _, ok, err := st.GetFirstMatch(""); err != nil || ok {
		t.Fatalf("expected miss for empty name with suffix bucket present, ok=%v err=%v", ok, err)
	}
}

// writeMeta should error when called within a read-only transaction (View).
func TestWriteMeta_ReadOnlyTxError(t *testing.T) {
	dbPath := tempDB(t)
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = st.Close(); _ = os.Remove(dbPath) })

	bs := st.(*boltStore)
	// Buckets are created by New; call writeMeta inside a View tx to force Put error.
	err = bs.db.View(func(tx *bbolt.Tx) error {
		return writeMeta(tx, 1, time.Now().Unix())
	})
	if err == nil {
		t.Fatalf("expected error from writeMeta in read-only tx")
	}
}

// loadRules should return an error when a blank key is used (both exact and suffix cases).
func TestLoadRules_BlankKeyErrors(t *testing.T) {
	dbPath := tempDB(t)
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = st.Close(); _ = os.Remove(dbPath) })

	bs := st.(*boltStore)
	now := time.Now()

	// Exact with blank name
	err = bs.db.Update(func(tx *bbolt.Tx) error {
		return loadRules(tx, []domain.BlockRule{{Name: "", Kind: domain.BlockRuleExact, AddedAt: now}})
	})
	if err == nil {
		t.Fatalf("expected error for exact rule with blank name")
	}

	// Suffix with blank name (reversed key is blank)
	err = bs.db.Update(func(tx *bbolt.Tx) error {
		return loadRules(tx, []domain.BlockRule{{Name: "", Kind: domain.BlockRuleSuffix, AddedAt: now}})
	})
	if err == nil {
		t.Fatalf("expected error for suffix rule with blank name")
	}
}

// Force decode errors for exact and suffix paths to cover GetFirstMatch error returns.
func TestGetFirstMatch_DecodeErrors(t *testing.T) {
	dbPath := tempDB(t)
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	bs := st.(*boltStore)
	t.Cleanup(func() { _ = st.Close(); _ = os.Remove(dbPath) })

	now := time.Now()
	rules := []domain.BlockRule{
		{Name: "x.example", Kind: domain.BlockRuleExact, Source: "t", AddedAt: now},
		{Name: "y.example", Kind: domain.BlockRuleSuffix, Source: "t", AddedAt: now},
	}
	if err := st.RebuildAll(rules, 1, now.Unix()); err != nil {
		t.Fatalf("RebuildAll: %v", err)
	}

	// Override decode to fail
	old := decodeRuleValueFn
	decodeRuleValueFn = func(name string, v []byte, dk domain.BlockRuleKind) (domain.BlockRule, error) {
		return domain.BlockRule{}, assertErr{}
	}
	defer func() { decodeRuleValueFn = old }()

	// Exact path should propagate error
	if _, _, err := st.GetFirstMatch("x.example"); err == nil {
		t.Fatalf("expected error from exact decode failure")
	}

	// Seed only suffix and test suffix path error
	if err := bs.db.Update(func(tx *bbolt.Tx) error {
		// remove exact bucket content to avoid short-circuiting
		eb := tx.Bucket(bucketExact)
		if eb != nil {
			_ = eb.Delete([]byte("x.example"))
		}
		return nil
	}); err != nil {
		t.Fatalf("prep tx: %v", err)
	}
	if _, _, err := st.GetFirstMatch("a.y.example"); err == nil {
		t.Fatalf("expected error from suffix decode failure")
	}
}

// Ensure writeMeta clamps negative updatedUnix to 0 and Stats() reflects it.
func TestWriteMeta_ClampNegativeUpdatedUnix(t *testing.T) {
	dbPath := tempDB(t)
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = st.Close(); _ = os.Remove(dbPath) })

	now := time.Now()
	rules := []domain.BlockRule{{Name: "neg.example", Kind: domain.BlockRuleExact, Source: "s", AddedAt: now}}
	if err := st.RebuildAll(rules, 1, -5); err != nil { // negative updatedUnix
		t.Fatalf("RebuildAll: %v", err)
	}
	stats := st.Stats()
	if stats.UpdatedUnix != 0 {
		t.Fatalf("expected UpdatedUnix=0 after clamp, got %d", stats.UpdatedUnix)
	}
}

// Force meta.updated to a value > MaxInt64 to exercise Stats() cap when decoding.
func TestStats_UpdatedUnix_CapUint64ToMaxInt64(t *testing.T) {
	dbPath := tempDB(t)
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	bs := st.(*boltStore)
	t.Cleanup(func() { _ = st.Close(); _ = os.Remove(dbPath) })

	// Seed meta buckets with version and an oversized updated value (all 0xFF = MaxUint64)
	if err := bs.db.Update(func(tx *bbolt.Tx) error {
		if err := ensureBuckets(tx); err != nil {
			return err
		}
		mb := tx.Bucket(bucketMeta)
		if err := mb.Put([]byte("version"), func() []byte { b := make([]byte, 8); binary.BigEndian.PutUint64(b, 7); return b }()); err != nil {
			return err
		}
		// updated = MaxUint64
		ub := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
		return mb.Put([]byte("updated"), ub)
	}); err != nil {
		t.Fatalf("seed meta: %v", err)
	}

	s := st.Stats()
	if s.Version != 7 {
		t.Fatalf("expected version 7, got %d", s.Version)
	}
	// Expect cap to MaxInt64
	if s.UpdatedUnix != int64(^uint64(0)>>1) { // math.MaxInt64 without importing math here
		t.Fatalf("expected UpdatedUnix capped to MaxInt64, got %d", s.UpdatedUnix)
	}
}

// Ensure encodeRuleValue clamps negative AddedAt and truncates oversized Source.
func TestEncodeRuleValue_ClampsAndTruncates(t *testing.T) {
	dbPath := tempDB(t)
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = st.Close(); _ = os.Remove(dbPath) })

	// Build a very long source (> 0xFFFF)
	long := make([]byte, 70000)
	for i := range long {
		long[i] = 'x'
	}
	src := string(long)

	// Negative AddedAt should be clamped to 0 when stored.
	rules := []domain.BlockRule{{Name: "clamp.example", Kind: domain.BlockRuleExact, Source: src, AddedAt: time.Unix(-5, 0)}}
	if err := st.RebuildAll(rules, 1, time.Now().Unix()); err != nil {
		t.Fatalf("RebuildAll: %v", err)
	}

	r, ok, err := st.GetFirstMatch("clamp.example")
	if err != nil || !ok {
		t.Fatalf("expected hit: ok=%v err=%v", ok, err)
	}
	if r.AddedAt.Unix() != 0 {
		t.Fatalf("expected AddedAt clamped to 0, got %d", r.AddedAt.Unix())
	}
	if len(r.Source) != 0xFFFF {
		t.Fatalf("expected truncated source length %d, got %d", 0xFFFF, len(r.Source))
	}
	// Verify content equals prefix of original
	if wantPrefix := string(long[:0xFFFF]); r.Source != wantPrefix {
		t.Fatalf("truncated source content mismatch")
	}
}

// TestStats_NegativeKeyN exercises the st.KeyN < 0 branches for both exact and suffix buckets
// by stubbing bucketStatsFn to return negative counts.
func TestStats_NegativeKeyN(t *testing.T) {
	// Save and restore seam
	orig := bucketStatsFn
	t.Cleanup(func() { bucketStatsFn = orig })

	// Stub to return negative KeyN regardless of input bucket
	bucketStatsFn = func(b *bbolt.Bucket) bbolt.BucketStats {
		return bbolt.BucketStats{KeyN: -1}
	}

	dbPath := tempDB(t)
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	t.Cleanup(func() { _ = st.Close(); _ = os.Remove(dbPath) })

	// Ensure meta has version/updated so we also cover that path without caring about values
	bs := st.(*boltStore)
	if err := bs.db.Update(func(tx *bbolt.Tx) error {
		mb := tx.Bucket(bucketMeta)
		if mb == nil {
			return nil
		}
		vbuf := make([]byte, 8)
		ubuf := make([]byte, 8)
		binary.BigEndian.PutUint64(vbuf, 1)
		binary.BigEndian.PutUint64(ubuf, 0)
		if err := mb.Put([]byte("version"), vbuf); err != nil {
			return err
		}
		return mb.Put([]byte("updated"), ubuf)
	}); err != nil {
		t.Fatalf("seed meta: %v", err)
	}

	got := st.Stats()
	if got.ExactKeys != 0 {
		t.Fatalf("ExactKeys: got %d, want 0", got.ExactKeys)
	}
	if got.SuffixKeys != 0 {
		t.Fatalf("SuffixKeys: got %d, want 0", got.SuffixKeys)
	}
}
