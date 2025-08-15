package bolt

import (
	"bytes"
	"encoding/binary"
	"time"

	bbolt "go.etcd.io/bbolt"

	"github.com/haukened/rr-dns/internal/dns/repos/blocklist"
)

var (
	bucketExact  = []byte("exact")
	bucketSuffix = []byte("suffix")
	bucketMeta   = []byte("meta")
)

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
	if err := db.Update(func(tx *bbolt.Tx) error {
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
	}); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &boltStore{db: db}, nil
}

func (s *boltStore) Close() error { return s.db.Close() }

func (s *boltStore) ExistsExact(name string) (bool, error) {
	var present bool
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketExact)
		if b == nil {
			present = false
			return nil
		}
		v := b.Get([]byte(name))
		present = v != nil
		return nil
	})
	return present, err
}

// VisitSuffixes walks existing suffix anchors for the provided reversed
// name prefix, from most-specific to least (trimming at dot boundaries),
// and invokes visit for each match. If visit returns false, iteration stops.
func (s *boltStore) VisitSuffixes(revPrefix []byte, visit func(key []byte) bool) error {
	return s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketSuffix)
		if b == nil {
			return nil
		}
		c := b.Cursor()
		rp := make([]byte, len(revPrefix))
		copy(rp, revPrefix)
		for {
			if len(rp) == 0 {
				break
			}
			// Prefix scan via cursor: seek to rp and iterate matches with rp as prefix.
			for k, _ := c.Seek(rp); k != nil && bytes.HasPrefix(k, rp); k, _ = c.Next() {
				kk := make([]byte, len(k))
				copy(kk, k)
				if !visit(kk) {
					return nil
				}
				// Only need the first match for this rp (most-specific to least handling is done by rp trimming).
				break
			}
			idx := bytes.LastIndexByte(rp, '.')
			if idx < 0 {
				break
			}
			rp = rp[:idx]
		}
		return nil
	})
}

func (s *boltStore) Stats() blocklist.StoreStats {
	st := blocklist.StoreStats{}
	_ = s.db.View(func(tx *bbolt.Tx) error {
		if b := tx.Bucket(bucketExact); b != nil {
			st.ExactCount = uint64(b.Stats().KeyN)
		}
		if b := tx.Bucket(bucketSuffix); b != nil {
			st.SuffixCount = uint64(b.Stats().KeyN)
		}
		if b := tx.Bucket(bucketMeta); b != nil {
			if v := b.Get([]byte("version")); len(v) == 8 {
				st.Version = binary.BigEndian.Uint64(v)
			}
			if v := b.Get([]byte("updated")); len(v) == 8 {
				st.UpdatedUnix = int64(binary.BigEndian.Uint64(v))
			}
		}
		return nil
	})
	return st
}

// Below are helper methods to populate data (used in tests and future updaters).

func (s *boltStore) putExact(name string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketExact)
		return b.Put([]byte(name), []byte{1})
	})
}

func (s *boltStore) putSuffix(reversed string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketSuffix)
		return b.Put([]byte(reversed), []byte{1})
	})
}

func (s *boltStore) setMeta(version uint64, updatedUnix int64) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketMeta)
		vbuf := make([]byte, 8)
		ubuf := make([]byte, 8)
		binary.BigEndian.PutUint64(vbuf, version)
		binary.BigEndian.PutUint64(ubuf, uint64(updatedUnix))
		if err := b.Put([]byte("version"), vbuf); err != nil {
			return err
		}
		return b.Put([]byte("updated"), ubuf)
	})
}
