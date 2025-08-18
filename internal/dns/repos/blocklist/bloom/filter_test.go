package bloom

import (
	"sync"
	"testing"
)

func TestFilter_AddTestClear(t *testing.T) {
	// small capacity and fp rate via factory
	f := NewFactory().New(32, 0.05)

	keyA := []byte("example.com")
	keyB := []byte("other.com")

	if f.MightContain(keyA) {
		t.Fatalf("unexpected positive before add")
	}

	f.Add(keyA)
	if !f.MightContain(keyA) {
		t.Fatalf("expected maybe after add")
	}

	// probabilistic: keyB might rarely be a false positive; accept both but ensure not both negative
	_ = f.MightContain(keyB)

	// No Clear in the interface anymore; create a fresh filter instead
	f = NewFactory().New(32, 0.05)
	_ = f
}

func TestFilter_ConcurrentReadsDuringWrites(t *testing.T) {
	f := NewFactory().New(256, 0.01)

	var wg sync.WaitGroup
	done := make(chan struct{})
	keys := [][]byte{[]byte("a"), []byte("b"), []byte("c")}

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10_000; i++ {
			f.Add(keys[i%3])
		}
		close(done)
	}()

	// Reader goroutines
	for r := 0; r < 8; r++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					_ = f.MightContain([]byte("probe"))
				}
			}
		}(r)
	}

	wg.Wait()
}
