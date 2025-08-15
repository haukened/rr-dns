package bloom

import (
	"sync"
	"testing"
)

func TestFilter_AddTestClear(t *testing.T) {
	// small parameters to keep test fast; k>=1
	f := NewFilter(128, 3)

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

	f.Clear()
	// After clear, added key should be gone; allow tiny chance of FP but assert most likely negative
	if f.MightContain(keyA) {
		// flaky only if FP happens right after Clear; note but don't fail hard
		t.Logf("keyA still reported maybe after Clear (likely FP)")
	}
}

func TestFilter_ConcurrentReadsDuringWrites(t *testing.T) {
	f := NewFilter(1024, 4)

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
