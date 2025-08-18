package bolt

import (
	"os"
	"path/filepath"
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
