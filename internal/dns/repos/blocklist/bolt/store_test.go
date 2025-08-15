package bolt

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func tempDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "bl.db")
}

func TestBoltStore_ExactAndSuffix(t *testing.T) {
	dbPath := tempDB(t)
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	bs := st.(*boltStore)
	t.Cleanup(func() { _ = st.Close(); _ = os.Remove(dbPath) })

	// empty
	present, err := st.ExistsExact("a.example.com")
	if err != nil || present {
		t.Fatalf("empty exists: present=%v err=%v", present, err)
	}

	// add exact and suffix anchors
	if err := bs.putExact("a.example.com"); err != nil {
		t.Fatalf("putExact: %v", err)
	}
	if err := bs.putSuffix("com.example"); err != nil { // reversed label order used by VisitSuffixes
		t.Fatalf("putSuffix: %v", err)
	}

	present, err = st.ExistsExact("a.example.com")
	if err != nil || !present {
		t.Fatalf("exists after put: present=%v err=%v", present, err)
	}

	// visit suffixes from most-specific to least; we stored only the apex anchor
	var visited [][]byte
	err = st.VisitSuffixes([]byte("com.example.foo.bar.baz"), func(key []byte) bool { // reversed label order, stop at first hit
		cp := make([]byte, len(key))
		copy(cp, key)
		visited = append(visited, cp)
		return false
	})
	if err != nil {
		t.Fatalf("VisitSuffixes: %v", err)
	}
	if len(visited) != 1 || string(visited[0]) != "com.example" {
		t.Fatalf("unexpected visited: %q", visited)
	}
}

func TestBoltStore_VisitStopEarly(t *testing.T) {
	dbPath := tempDB(t)
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	bs := st.(*boltStore)
	t.Cleanup(func() { _ = st.Close(); _ = os.Remove(dbPath) })

	_ = bs.putSuffix("com.example")
	_ = bs.putSuffix("com.example.foo")

	var count int
	err = st.VisitSuffixes([]byte("com.example.foo.bar.baz"), func(key []byte) bool {
		count++
		return false // stop at first hit
	})
	if err != nil {
		t.Fatalf("VisitSuffixes: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected early stop at 1, got %d", count)
	}
}

func TestBoltStore_StatsAndClose(t *testing.T) {
	dbPath := tempDB(t)
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	bs := st.(*boltStore)
	t.Cleanup(func() { _ = st.Close(); _ = os.Remove(dbPath) })

	_ = bs.putExact("a.example.com")
	_ = bs.putExact("b.example.com")
	_ = bs.putSuffix("com.example")
	_ = bs.setMeta(3, time.Now().Unix())

	stats := st.Stats()
	if stats.ExactCount != 2 || stats.SuffixCount != 1 || stats.Version != 3 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}
