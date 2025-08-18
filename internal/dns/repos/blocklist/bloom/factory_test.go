package bloom

import "testing"

func TestFactory_New_Basic(t *testing.T) {
	f := NewFactory()
	bf := f.New(128, 0.01)
	if bf == nil {
		t.Fatalf("expected non-nil bloom filter")
	}

	key := []byte("example.com")
	if bf.MightContain(key) {
		t.Fatalf("unexpected positive before add")
	}
	bf.Add(key)
	if !bf.MightContain(key) {
		t.Fatalf("expected maybe after add")
	}
}

func TestFactory_New_Defaults(t *testing.T) {
	// capacity=0 and invalid fp → internal defaults apply; filter still usable
	f := NewFactory()
	bf := f.New(0, 0)
	key := []byte("default-case.test")
	bf.Add(key)
	if !bf.MightContain(key) {
		t.Fatalf("expected maybe after add with default-sized bloom")
	}
}
