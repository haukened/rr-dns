package bloom

import (
	"math"

	"github.com/haukened/rr-dns/internal/dns/repos/blocklist"
)

// sizer implements blocklist.BloomSizer using standard formulas:
//
//	m = - (n * ln p) / (ln 2)^2
//	k = (m / n) * ln 2
//
// Results are clamped to at least 1.
// Pure math; no external dependencies beyond stdlib.
type sizer struct{}

// NewSizer returns a BloomSizer implementation.
func NewSizer() blocklist.BloomSizer { return sizer{} }

func (s sizer) Size(n uint64, p float64) (uint64, uint8) {
	if n == 0 {
		n = 1
	}
	if !(p > 0 && p < 1) {
		p = 0.01 // default 1% if invalid
	}
	ln2 := math.Ln2
	m := uint64(math.Ceil(-float64(n) * math.Log(p) / (ln2 * ln2)))
	if m == 0 {
		m = 1
	}
	k := uint8(math.Max(1, math.Round((float64(m)/float64(n))*ln2)))
	return m, k
}
