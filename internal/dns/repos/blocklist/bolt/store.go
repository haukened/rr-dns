package bolt

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"time"

	bbolt "go.etcd.io/bbolt"
	bberrors "go.etcd.io/bbolt/errors"

	"github.com/haukened/rr-dns/internal/dns/domain"
	"github.com/haukened/rr-dns/internal/dns/repos/blocklist"
)

var (
	bucketExact  = []byte("exact")
	bucketSuffix = []byte("suffix")
	bucketMeta   = []byte("meta")
)

// bucketCreator is the minimal contract needed for creating buckets.
// It matches the method on *bbolt.Tx so we can pass it directly, and also
// allows tests to provide a fake to simulate error paths.
type bucketCreator interface {
	CreateBucketIfNotExists(name []byte) (*bbolt.Bucket, error)
}

// bucketDeleter is the minimal contract needed for deleting buckets.
type bucketDeleter interface {
	DeleteBucket(name []byte) error
}

// boltStore implements blocklist.Store using bbolt.
type boltStore struct {
	db *bbolt.DB
}

// New opens (or creates) a Bolt database at path and ensures buckets exist.
func New(path string) (blocklist.Store, error) {
	db, err := bbolt.Open(path, 0o600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}
	if err := db.Update(func(tx *bbolt.Tx) error { return ensureBucketsFn(tx) }); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &boltStore{db: db}, nil
}

func (s *boltStore) Close() error { return s.db.Close() }

// GetFirstMatch returns exact match first; otherwise walks suffix anchors from
// most- to least-specific and returns the first match.
func (s *boltStore) GetFirstMatch(name string) (out domain.BlockRule, ok bool, err error) {
	err = s.db.View(func(tx *bbolt.Tx) error {
		// 1) Exact match
		if b := tx.Bucket(bucketExact); b != nil {
			if v := b.Get([]byte(name)); v != nil {
				rule, err := decodeRuleValueFn(name, v, domain.BlockRuleExact)
				if err != nil {
					return err
				}
				out = rule
				return nil
			}
		}
		// 2) Suffix walk using reversed prefix trimming
		b := tx.Bucket(bucketSuffix)
		if b == nil {
			return nil
		}
		c := b.Cursor()
		// Build reversed bytes directly from the query with a single allocation.
		rp := reverseToBytesFromString(name)
		for len(rp) > 0 {
			k, v := c.Seek(rp)
			if k != nil && bytes.HasPrefix(k, rp) {
				// first hit at this specificity
				// Reverse key bytes directly to construct anchor without intermediate string conversions.
				anchor := string(reverseBytesToNew(k))
				rule, err := decodeRuleValueFn(anchor, v, domain.BlockRuleSuffix)
				if err != nil {
					return err
				}
				out = rule
				return nil
			}
			idx := bytes.LastIndexByte(rp, '.')
			if idx < 0 {
				break
			}
			rp = rp[:idx]
		}
		return nil
	})
	if err != nil {
		return domain.BlockRule{}, false, err
	}
	if out.Name == "" {
		return domain.BlockRule{}, false, nil
	}
	return out, true, nil
}

// RebuildAll replaces all data atomically in a single write transaction.
func (s *boltStore) RebuildAll(rules []domain.BlockRule, version uint64, updatedUnix int64) (err error) {
	return s.db.Update(func(tx *bbolt.Tx) error {
		// Drop buckets if present, then (re)create them via ensureBucketsFn.
		if err := deleteBucketsFn(tx, bucketExact, bucketSuffix, bucketMeta); err != nil {
			return err
		}
		if err := ensureBucketsFn(tx); err != nil {
			return err
		}
		if err := loadRulesFn(tx, rules); err != nil {
			return err
		}
		return writeMetaFn(tx, version, updatedUnix)
	})
}

// Purge clears all buckets.
func (s *boltStore) Purge() error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		if err := deleteBucketsFn(tx, bucketExact, bucketSuffix, bucketMeta); err != nil {
			return err
		}
		return ensureBucketsFn(tx)
	})
}

// Stats returns cheap counts and metadata from the store.
func (s *boltStore) Stats() blocklist.StoreStats {
	var out blocklist.StoreStats
	_ = s.db.View(func(tx *bbolt.Tx) error {
		if mb := tx.Bucket(bucketMeta); mb != nil {
			if v := mb.Get([]byte("version")); len(v) == 8 {
				out.Version = binary.BigEndian.Uint64(v)
			}
			if v := mb.Get([]byte("updated")); len(v) == 8 {
				u := binary.BigEndian.Uint64(v)
				if u > math.MaxInt64 {
					out.UpdatedUnix = math.MaxInt64
				} else {
					out.UpdatedUnix = int64(u)
				}
			}
		}
		if eb := tx.Bucket(bucketExact); eb != nil {
			st := bucketStatsFn(eb)
			if st.KeyN < 0 {
				out.ExactKeys = 0
			} else {
				out.ExactKeys = uint64(st.KeyN)
			}
		}
		if sb := tx.Bucket(bucketSuffix); sb != nil {
			st := bucketStatsFn(sb)
			if st.KeyN < 0 {
				out.SuffixKeys = 0
			} else {
				out.SuffixKeys = uint64(st.KeyN)
			}
		}
		return nil
	})
	return out
}

// Helpers for encoding/decoding rule values.
// value format: [kind:1][addedAt:8be][sourceLen:2be][source bytes]
func encodeRuleValue(r domain.BlockRule) []byte {
	// Cap source length to 65535 to fit into uint16 field
	src := []byte(r.Source)
	if len(src) > 0xFFFF {
		src = src[:0xFFFF]
	}
	buf := make([]byte, 1+8+2+len(src))
	buf[0] = byte(r.Kind)
	// Store non-negative Unix seconds; clamp negatives to 0
	ts := r.AddedAt.Unix()
	if ts < 0 {
		ts = 0
	}
	// #nosec G115 -- ts is clamped to >= 0 above; safe narrowing to uint64 seconds
	binary.BigEndian.PutUint64(buf[1:9], uint64(ts))
	// #nosec G115 -- src length is truncated to <= 0xFFFF above; safe narrowing to uint16
	binary.BigEndian.PutUint16(buf[9:11], uint16(len(src)))
	copy(buf[11:], src)
	return buf
}

func decodeRuleValue(name string, v []byte, defaultKind domain.BlockRuleKind) (domain.BlockRule, error) {
	var r domain.BlockRule
	r.Name = name
	if len(v) < 11 {
		// tolerate legacy minimal values; fill defaults
		r.Kind = defaultKind
		r.AddedAt = time.Time{}
		r.Source = ""
		return r, nil
	}
	r.Kind = domain.BlockRuleKind(v[0])
	u := binary.BigEndian.Uint64(v[1:9])
	if u > math.MaxInt64 {
		u = uint64(math.MaxInt64)
	}
	// #nosec G115 -- u is capped to MaxInt64 above; safe narrowing to int64
	ts := int64(u)
	r.AddedAt = time.Unix(ts, 0)
	sl := int(binary.BigEndian.Uint16(v[9:11]))
	if 11+sl > len(v) {
		sl = len(v) - 11
	}
	r.Source = string(v[11 : 11+sl])
	if r.Kind != domain.BlockRuleExact && r.Kind != domain.BlockRuleSuffix {
		r.Kind = defaultKind
	}
	return r, nil
}

// decodeRuleValueFn allows tests to override decoding to simulate errors.
var decodeRuleValueFn = decodeRuleValue

// reverseString reverses s by bytes. Domain names are ASCII, so byte-wise reverse is sufficient here.
func reverseString(s string) string {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		out[i] = s[len(s)-1-i]
	}
	return string(out)
}

// reverseToBytesFromString returns a new []byte containing the bytes of s in reverse order.
func reverseToBytesFromString(s string) []byte {
	out := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		out[i] = s[len(s)-1-i]
	}
	return out
}

// reverseBytesToNew returns a new []byte containing b reversed.
func reverseBytesToNew(b []byte) []byte {
	out := make([]byte, len(b))
	for i := 0; i < len(b); i++ {
		out[i] = b[len(b)-1-i]
	}
	return out
}

// ensureBuckets creates all required buckets. Kept as a var for test seams.
var ensureBucketsFn = ensureBuckets

func ensureBuckets(tx bucketCreator) error {
	if _, err := tx.CreateBucketIfNotExists(bucketExact); err != nil {
		return err
	}
	if _, err := tx.CreateBucketIfNotExists(bucketSuffix); err != nil {
		return err
	}
	if _, err := tx.CreateBucketIfNotExists(bucketMeta); err != nil {
		return err
	}
	return nil
}

// deleteBuckets removes the provided buckets, ignoring ErrBucketNotFound.
// Exposed via var for test seams if needed.
var deleteBucketsFn = deleteBuckets

func deleteBuckets(tx bucketDeleter, names ...[]byte) error {
	for _, n := range names {
		if err := tx.DeleteBucket(n); err != nil {
			if errors.Is(err, bberrors.ErrBucketNotFound) {
				continue
			}
			return err
		}
	}
	return nil
}

// loadRules writes all rules into the newly created buckets. Seamed for tests.
var loadRulesFn = loadRules

func loadRules(tx *bbolt.Tx, rules []domain.BlockRule) error {
	eb := tx.Bucket(bucketExact)
	sb := tx.Bucket(bucketSuffix)
	for _, r := range rules {
		val := encodeRuleValue(r)
		switch r.Kind {
		case domain.BlockRuleExact:
			if err := eb.Put([]byte(r.Name), val); err != nil {
				return err
			}
		case domain.BlockRuleSuffix:
			key := reverseToBytesFromString(r.Name)
			if err := sb.Put(key, val); err != nil {
				return err
			}
		default:
			// ignore unsupported kinds
		}
	}
	return nil
}

// writeMeta persists version and updated timestamp. Seamed for tests.
var writeMetaFn = writeMeta

func writeMeta(tx *bbolt.Tx, version uint64, updatedUnix int64) error {
	mb := tx.Bucket(bucketMeta)
	vbuf := make([]byte, 8)
	ubuf := make([]byte, 8)
	binary.BigEndian.PutUint64(vbuf, version)
	// Store non-negative Unix seconds; clamp negatives to 0
	if updatedUnix < 0 {
		binary.BigEndian.PutUint64(ubuf, 0)
	} else {
		binary.BigEndian.PutUint64(ubuf, uint64(updatedUnix))
	}
	if err := mb.Put([]byte("version"), vbuf); err != nil {
		return err
	}
	return mb.Put([]byte("updated"), ubuf)
}

// reverseBytesInPlace keeps the old API used in tests; historically it returned a new slice.
func reverseBytesInPlace(b []byte) []byte { return reverseBytesToNew(b) }

// bucketStatsFn allows tests to stub bucket statistics.
var bucketStatsFn = func(b *bbolt.Bucket) bbolt.BucketStats { return b.Stats() }
