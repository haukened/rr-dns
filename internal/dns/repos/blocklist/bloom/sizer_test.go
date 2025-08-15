package bloom

import "testing"

func TestSizer_CommonCases(t *testing.T) {
	s := NewSizer()

	// n=1, p=1% → m≈10, k≈7
	m, k := s.Size(1, 0.01)
	if m < 10 || k != 7 {
		t.Fatalf("n=1,p=0.01: got m=%d k=%d; want m>=10 k=7", m, k)
	}

	// n=1e6, p=1% → m≈9.585e6 bits, k≈7
	m, k = s.Size(1_000_000, 0.01)
	if m < 9_500_000 || m > 9_700_000 { // loose bounds around expectation
		t.Fatalf("n=1e6,p=0.01: unexpected m=%d (expected around 9.6e6)", m)
	}
	if k != 7 {
		t.Fatalf("n=1e6,p=0.01: k=%d; want 7", k)
	}

	// p=0.5 → k rounds to 1 (very small number of hashes)
	m, k = s.Size(10_000, 0.5)
	if k != 1 {
		t.Fatalf("p=0.5: k=%d; want 1", k)
	}
	if m == 0 {
		t.Fatalf("p=0.5: m should be >=1")
	}
}

func TestSizer_ClampingAndDefaults(t *testing.T) {
	s := NewSizer()

	// n=0 → treated as 1; invalid p (<=0) defaults to 0.01
	m, k := s.Size(0, 0)
	if m == 0 || k == 0 {
		t.Fatalf("n=0,p=0: expected m>=1 and k>=1; got m=%d k=%d", m, k)
	}

	// p>=1 → defaults to 0.01
	m2, k2 := s.Size(100, 1.0)
	if m2 == 0 || k2 == 0 {
		t.Fatalf("p>=1 default: expected m>=1 and k>=1; got m=%d k=%d", m2, k2)
	}
}
