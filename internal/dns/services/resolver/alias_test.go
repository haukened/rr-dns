package resolver

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/haukened/rr-dns/internal/dns/common/clock"
	"github.com/haukened/rr-dns/internal/dns/domain"
)

// local noop logger (avoid importing test one from other file)
type aliasNoopLogger struct{}

func (n *aliasNoopLogger) Info(map[string]any, string)  {}
func (n *aliasNoopLogger) Error(map[string]any, string) {}
func (n *aliasNoopLogger) Debug(map[string]any, string) {}
func (n *aliasNoopLogger) Warn(map[string]any, string)  {}
func (n *aliasNoopLogger) Panic(map[string]any, string) {}
func (n *aliasNoopLogger) Fatal(map[string]any, string) {}

type fakeCache struct{}

// Keys implements Cache.
func (f *fakeCache) Keys() []string {
	return nil
}

// Set implements Cache.
func (f *fakeCache) Set(record []domain.ResourceRecord) error {
	return nil
}

func (f *fakeCache) Delete(string) {}

func (f *fakeCache) Get(string) ([]domain.ResourceRecord, bool) {
	return nil, false
}
func (f *fakeCache) Put(string, []domain.ResourceRecord) {}

func (f *fakeCache) Len() int { return 0 }

// fake zone cache for alias tests
type fakeZone struct {
	records map[string][]domain.ResourceRecord
}

func (f *fakeZone) FindRecords(q domain.Question) ([]domain.ResourceRecord, bool) {
	if f == nil {
		return nil, false
	}
	recs, ok := f.records[q.CacheKey()]
	return recs, ok
}
func (f *fakeZone) PutZone(string, []domain.ResourceRecord) {}
func (f *fakeZone) RemoveZone(string)                       {}
func (f *fakeZone) Zones() []string                         { return nil }
func (f *fakeZone) Count() int                              { return 0 }

// fake upstream client
type fakeUpstream struct {
	recs []domain.ResourceRecord
	err  error
}

func (f *fakeUpstream) Resolve(ctx context.Context, q domain.Question, now time.Time) ([]domain.ResourceRecord, error) {
	return f.recs, f.err
}

// helper to create authoritative RR quickly (can bypass validation where needed)
func mustAuthRR(name string, t domain.RRType, text string) domain.ResourceRecord {
	rr, _ := domain.NewAuthoritativeResourceRecord(name, t, domain.RRClass(1), 300, nil, text)
	return rr
}

// fabricate record without validation (for invalid text/data scenarios)
func rawRR(name string, t domain.RRType, text string, data []byte) domain.ResourceRecord {
	return domain.ResourceRecord{Name: name, Type: t, Class: domain.RRClass(1), Text: text, Data: data}
}

// build question
func mustQ(name string, t domain.RRType) domain.Question {
	q, _ := domain.NewQuestion(1, name, t, domain.RRClass(1))
	return q
}

func TestAliasChase_FastPathVariants(t *testing.T) {
	ch := NewAliasChaser(nil, nil, nil, &clock.MockClock{CurrentTime: time.Now()}, &aliasNoopLogger{}, 5)
	// 1. empty initial
	q := mustQ("example.com.", domain.RRTypeA)
	recs, err := ch.Chase(q, nil)
	assert.NoError(t, err)
	assert.Len(t, recs, 0)
	// 2. head not CNAME
	aRR := mustAuthRR("example.com.", domain.RRTypeA, "192.0.2.1")
	recs, err = ch.Chase(q, []domain.ResourceRecord{aRR})
	assert.NoError(t, err)
	assert.Equal(t, []domain.ResourceRecord{aRR}, recs)
	// 3. query type is CNAME (should not chase)
	cQ := mustQ("alias.example.", domain.RRTypeCNAME)
	cRR := mustAuthRR("alias.example.", domain.RRTypeCNAME, "target.example.")
	recs, err = ch.Chase(cQ, []domain.ResourceRecord{cRR})
	assert.NoError(t, err)
	assert.Equal(t, []domain.ResourceRecord{cRR}, recs)
}

func zoneKey(name string, t domain.RRType, c domain.RRClass) string {
	q, _ := domain.NewQuestion(0, name, t, c)
	return q.CacheKey()
}

// NOTE: Multi-hop, depth exceed, and loop-detect paths are not currently reachable
// because implementation stops after first hop if next authoritative/upstream
// answer set does not directly produce the original qtype or another CNAME for
// that qtype. Tests asserting those advanced behaviors are deferred until the
// algorithm is extended.
func TestAliasChase_LoopDetected(t *testing.T) {
	zone := &fakeZone{records: map[string][]domain.ResourceRecord{}}
	c1 := mustAuthRR("a.example.", domain.RRTypeCNAME, "b.example.")
	c2 := mustAuthRR("b.example.", domain.RRTypeCNAME, "a.example.")
	zone.records[zoneKey("b.example.", domain.RRTypeCNAME, domain.RRClassIN)] = []domain.ResourceRecord{c2}
	zone.records[zoneKey("a.example.", domain.RRTypeCNAME, domain.RRClassIN)] = []domain.ResourceRecord{c1} // for third hop detection
	ch := NewAliasChaser(zone, nil, nil, &clock.MockClock{CurrentTime: time.Now()}, &aliasNoopLogger{}, 5)
	q := mustQ("a.example.", domain.RRTypeA)
	recs, err := ch.Chase(q, []domain.ResourceRecord{c1})
	assert.ErrorIs(t, err, ErrAliasLoopDetected)
	// Expected chain: first two successful hops + failing head appended
	assert.Equal(t, []domain.ResourceRecord{c1, c2, c1}, recs)
}

// Self-loop: single CNAME whose target is itself (A.example -> A.example). RFC 1034 §3.6.2 loop detection.
func TestAliasChase_SelfLoop(t *testing.T) {
	zone := &fakeZone{records: map[string][]domain.ResourceRecord{}}
	self := mustAuthRR("self.example.", domain.RRTypeCNAME, "self.example.")
	// Store under original query type (A) key so initial FindRecords returns CNAME for A query.
	zone.records[zoneKey("self.example.", domain.RRTypeA, domain.RRClassIN)] = []domain.ResourceRecord{self}
	// Also store under CNAME key for fallback lookup that produces second hop triggering loop detect.
	zone.records[zoneKey("self.example.", domain.RRTypeCNAME, domain.RRClassIN)] = []domain.ResourceRecord{self}
	ch := NewAliasChaser(zone, nil, nil, &clock.MockClock{CurrentTime: time.Now()}, &aliasNoopLogger{}, 5)
	q := mustQ("self.example.", domain.RRTypeA)
	recs, err := ch.Chase(q, []domain.ResourceRecord{self})
	assert.ErrorIs(t, err, ErrAliasLoopDetected)
	// Chain: first hop appended, second iteration detects loop; failing head appended again.
	assert.Equal(t, []domain.ResourceRecord{self, self}, recs)
}

func TestAliasChase_ExtractTargetErrors(t *testing.T) {
	ch := NewAliasChaser(nil, nil, nil, &clock.MockClock{CurrentTime: time.Now()}, &aliasNoopLogger{}, 5)
	q := mustQ("bad.example.", domain.RRTypeA)
	// Text empty, Data present
	r1 := rawRR("bad.example.", domain.RRTypeCNAME, "", []byte{0x01})
	recs, err := ch.Chase(q, []domain.ResourceRecord{r1})
	assert.Error(t, err)
	assert.Equal(t, []domain.ResourceRecord{r1}, recs) // chain collected head then error
	// Text empty, Data empty
	r2 := rawRR("bad2.example.", domain.RRTypeCNAME, "", nil)
	recs, err = ch.Chase(q, []domain.ResourceRecord{r2})
	assert.Error(t, err)
	assert.Equal(t, []domain.ResourceRecord{r2}, recs)
}

func TestAliasChase_UpstreamFallbackVariants(t *testing.T) {
	// zone miss forces upstream
	q := mustQ("alias.example.", domain.RRTypeA)
	cname := mustAuthRR("alias.example.", domain.RRTypeCNAME, "target.example.")
	a := mustAuthRR("target.example.", domain.RRTypeA, "192.0.2.55")
	// success
	upSuccess := &fakeUpstream{recs: []domain.ResourceRecord{a}, err: nil}
	ch := NewAliasChaser(&fakeZone{records: map[string][]domain.ResourceRecord{}}, upSuccess, nil, &clock.MockClock{CurrentTime: time.Now()}, &aliasNoopLogger{}, 5)
	recs, err := ch.Chase(q, []domain.ResourceRecord{cname})
	assert.NoError(t, err)
	assert.Equal(t, []domain.ResourceRecord{cname, a}, recs)
	// upstream error
	upErr := &fakeUpstream{recs: nil, err: errors.New("fail")}
	ch2 := NewAliasChaser(&fakeZone{records: map[string][]domain.ResourceRecord{}}, upErr, nil, &clock.MockClock{CurrentTime: time.Now()}, &aliasNoopLogger{}, 5)
	recs, err = ch2.Chase(q, []domain.ResourceRecord{cname})
	assert.NoError(t, err)
	assert.Equal(t, []domain.ResourceRecord{cname}, recs) // no terminal data
	// upstream empty slice
	upEmpty := &fakeUpstream{recs: []domain.ResourceRecord{}, err: nil}
	ch3 := NewAliasChaser(&fakeZone{records: map[string][]domain.ResourceRecord{}}, upEmpty, nil, &clock.MockClock{CurrentTime: time.Now()}, &aliasNoopLogger{}, 5)
	recs, err = ch3.Chase(q, []domain.ResourceRecord{cname})
	assert.NoError(t, err)
	assert.Equal(t, []domain.ResourceRecord{cname}, recs)
}

func TestAliasChase_TerminalMissingData(t *testing.T) {
	// zone miss then upstream nil -> only chain returned
	cname := mustAuthRR("x.example.", domain.RRTypeCNAME, "z.example.")
	ch := NewAliasChaser(&fakeZone{records: map[string][]domain.ResourceRecord{}}, nil, nil, &clock.MockClock{CurrentTime: time.Now()}, &aliasNoopLogger{}, 5)
	q := mustQ("x.example.", domain.RRTypeA)
	recs, err := ch.Chase(q, []domain.ResourceRecord{cname})
	assert.NoError(t, err)
	assert.Equal(t, []domain.ResourceRecord{cname}, recs)
}

func TestAliasChase_UnlimitedDepthBranch(t *testing.T) {
	// maxDepth=0 means unlimited; ensure guardDepth does not trigger error
	zone := &fakeZone{records: map[string][]domain.ResourceRecord{}}
	c1 := mustAuthRR("a.ud.", domain.RRTypeCNAME, "b.ud.")
	a := mustAuthRR("b.ud.", domain.RRTypeA, "192.0.2.77")
	zone.records[zoneKey("b.ud.", domain.RRTypeA, domain.RRClassIN)] = []domain.ResourceRecord{a}
	ch := NewAliasChaser(zone, nil, nil, &clock.MockClock{CurrentTime: time.Now()}, &aliasNoopLogger{}, 0)
	q := mustQ("a.ud.", domain.RRTypeA)
	recs, err := ch.Chase(q, []domain.ResourceRecord{c1})
	assert.NoError(t, err)
	assert.Equal(t, []domain.ResourceRecord{c1, a}, recs)
}
func TestChaseState_CurrentHead(t *testing.T) {
	// Case 1: Empty current slice
	st := chaseState{current: []domain.ResourceRecord{}}
	rr, isCNAME := st.currentHead()
	assert.False(t, isCNAME)
	assert.Equal(t, domain.ResourceRecord{}, rr)

	// Case 2: Head is not CNAME
	aRR := mustAuthRR("example.com.", domain.RRTypeA, "192.0.2.1")
	st = chaseState{current: []domain.ResourceRecord{aRR}}
	rr, isCNAME = st.currentHead()
	assert.False(t, isCNAME)
	assert.Equal(t, aRR, rr)

	// Case 3: Head is CNAME
	cRR := mustAuthRR("alias.example.", domain.RRTypeCNAME, "target.example.")
	st = chaseState{current: []domain.ResourceRecord{cRR}}
	rr, isCNAME = st.currentHead()
	assert.True(t, isCNAME)
	assert.Equal(t, cRR, rr)

	// Case 4: Multiple records, head is CNAME
	cRR2 := mustAuthRR("another.example.", domain.RRTypeCNAME, "next.example.")
	aRR2 := mustAuthRR("another.example.", domain.RRTypeA, "192.0.2.2")
	st = chaseState{current: []domain.ResourceRecord{cRR2, aRR2}}
	rr, isCNAME = st.currentHead()
	assert.True(t, isCNAME)
	assert.Equal(t, cRR2, rr)

	// Case 5: Multiple records, head is not CNAME
	st = chaseState{current: []domain.ResourceRecord{aRR2, cRR2}}
	rr, isCNAME = st.currentHead()
	assert.False(t, isCNAME)
	assert.Equal(t, aRR2, rr)
}
func TestAliasChaser_guardDepth(t *testing.T) {
	// Setup: aliasNoopLogger avoids side effects
	logger := &aliasNoopLogger{}
	clk := &clock.MockClock{CurrentTime: time.Now()}
	zone := &fakeZone{}
	up := &fakeUpstream{}
	cache := &fakeCache{}

	// Case 1: maxDepth = 0 (unlimited), should never error
	ch := NewAliasChaser(zone, up, cache, clk, logger, 0).(*aliasChaser)
	st := newChaseState(mustQ("a.example.", domain.RRTypeA), nil)
	for i := 0; i < 100; i++ {
		err := ch.guardDepth(&st, mustAuthRR("a.example.", domain.RRTypeCNAME, "b.example."))
		assert.NoError(t, err)
	}

	// Case 2: maxDepth = 3, depth not exceeded
	ch2 := NewAliasChaser(zone, up, cache, clk, logger, 3).(*aliasChaser)
	st2 := newChaseState(mustQ("a.example.", domain.RRTypeA), nil)
	for i := 0; i < 3; i++ {
		err := ch2.guardDepth(&st2, mustAuthRR("a.example.", domain.RRTypeCNAME, "b.example."))
		assert.NoError(t, err)
	}

	// Case 3: maxDepth = 2, depth exceeded
	ch3 := NewAliasChaser(zone, up, cache, clk, logger, 2).(*aliasChaser)
	st3 := newChaseState(mustQ("a.example.", domain.RRTypeA), nil)
	_ = ch3.guardDepth(&st3, mustAuthRR("a.example.", domain.RRTypeCNAME, "b.example."))
	_ = ch3.guardDepth(&st3, mustAuthRR("b.example.", domain.RRTypeCNAME, "c.example."))
	err := ch3.guardDepth(&st3, mustAuthRR("c.example.", domain.RRTypeCNAME, "d.example."))
	assert.ErrorIs(t, err, ErrAliasDepthExceeded)
}
func TestAliasChaser_AuthoritativeLookup(t *testing.T) {
	// Setup: fakeZone implements ZoneCache
	zone := &fakeZone{records: map[string][]domain.ResourceRecord{}}
	aRR := mustAuthRR("example.com.", domain.RRTypeA, "192.0.2.1")
	q := mustQ("example.com.", domain.RRTypeA)
	zone.records[q.CacheKey()] = []domain.ResourceRecord{aRR}

	ch := NewAliasChaser(zone, nil, nil, &clock.MockClock{CurrentTime: time.Now()}, &aliasNoopLogger{}, 5).(*aliasChaser)

	// Case 1: zone is present, record found
	recs, found := ch.authoritativeLookup(q)
	assert.True(t, found)
	assert.Equal(t, []domain.ResourceRecord{aRR}, recs)

	// Case 2: zone is present, record not found
	missQ := mustQ("missing.com.", domain.RRTypeA)
	recs, found = ch.authoritativeLookup(missQ)
	assert.False(t, found)
	assert.Nil(t, recs)

	// Case 3: zone is nil
	chNilZone := NewAliasChaser(nil, nil, nil, &clock.MockClock{CurrentTime: time.Now()}, &aliasNoopLogger{}, 5).(*aliasChaser)
	recs, found = chNilZone.authoritativeLookup(q)
	assert.False(t, found)
	assert.Nil(t, recs)
}

// Question synthesis failure: use an invalid original RRType so NewQuestion inside buildNextQuestion fails
func TestAliasChase_QuestionSynthesisFailure(t *testing.T) {
	// Craft a query with an invalid RRType (bypassing constructor validation)
	invalidType := domain.RRType(9999) // not in IsValid set
	q := domain.Question{ID: 42, Name: "badtype.example.", Type: invalidType, Class: domain.RRClassIN}
	// Initial CNAME record so shouldChase passes (query type != CNAME)
	c1 := mustAuthRR("badtype.example.", domain.RRTypeCNAME, "next.example.")
	ch := NewAliasChaser(nil, nil, nil, &clock.MockClock{CurrentTime: time.Now()}, &aliasNoopLogger{}, 5)
	recs, err := ch.Chase(q, []domain.ResourceRecord{c1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported RRType")
	// Chain should contain only the first (failing) CNAME hop
	assert.Equal(t, []domain.ResourceRecord{c1}, recs)
}

// Multi-hop authoritative success: CNAME -> CNAME -> A (all via authoritative fallback)
func TestAliasChase_MultiHop_AuthoritativeSuccess(t *testing.T) {
	zone := &fakeZone{records: map[string][]domain.ResourceRecord{}}
	// c1: a.mh. -> b.mh.
	c1 := mustAuthRR("a.mh.", domain.RRTypeCNAME, "b.mh.")
	// c2: b.mh. -> c.mh.
	c2 := mustAuthRR("b.mh.", domain.RRTypeCNAME, "c.mh.")
	// terminal A for c.mh.
	aTerm := mustAuthRR("c.mh.", domain.RRTypeA, "192.0.2.200")

	// Populate zone for fallback lookups
	// First hop: when looking for b.mh. type A (miss) then b.mh. type CNAME -> c2
	zone.records[zoneKey("b.mh.", domain.RRTypeCNAME, domain.RRClassIN)] = []domain.ResourceRecord{c2}
	// Second hop: when looking for c.mh. type A -> aTerm
	zone.records[zoneKey("c.mh.", domain.RRTypeA, domain.RRClassIN)] = []domain.ResourceRecord{aTerm}

	ch := NewAliasChaser(zone, nil, nil, &clock.MockClock{CurrentTime: time.Now()}, &aliasNoopLogger{}, 5)
	q := mustQ("a.mh.", domain.RRTypeA)
	recs, err := ch.Chase(q, []domain.ResourceRecord{c1})
	assert.NoError(t, err)
	assert.Equal(t, []domain.ResourceRecord{c1, c2, aTerm}, recs)
}

// Multi-hop termination with missing final data: CNAME -> CNAME then no records (authoritative miss and no upstream)
func TestAliasChase_MultiHop_NoDataTermination(t *testing.T) {
	zone := &fakeZone{records: map[string][]domain.ResourceRecord{}}
	c1 := mustAuthRR("a.nodata.", domain.RRTypeCNAME, "b.nodata.")
	c2 := mustAuthRR("b.nodata.", domain.RRTypeCNAME, "c.nodata.")
	// Only supply c2 (second hop). No records for c.nodata. (A or CNAME) forcing termination.
	zone.records[zoneKey("b.nodata.", domain.RRTypeCNAME, domain.RRClassIN)] = []domain.ResourceRecord{c2}

	ch := NewAliasChaser(zone, nil, nil, &clock.MockClock{CurrentTime: time.Now()}, &aliasNoopLogger{}, 5)
	q := mustQ("a.nodata.", domain.RRTypeA)
	recs, err := ch.Chase(q, []domain.ResourceRecord{c1})
	assert.NoError(t, err)
	// Chain should include both CNAME hops only
	assert.Equal(t, []domain.ResourceRecord{c1, c2}, recs)
}

// Depth exceeded inside full Chase loop (integration); maxDepth=1 with two CNAME hops.
func TestAliasChase_DepthExceededInChase(t *testing.T) {
	zone := &fakeZone{records: map[string][]domain.ResourceRecord{}}
	c1 := mustAuthRR("a.depth.", domain.RRTypeCNAME, "b.depth.")
	c2 := mustAuthRR("b.depth.", domain.RRTypeCNAME, "c.depth.")
	zone.records[zoneKey("b.depth.", domain.RRTypeCNAME, domain.RRClassIN)] = []domain.ResourceRecord{c2}
	ch := NewAliasChaser(zone, nil, nil, &clock.MockClock{CurrentTime: time.Now()}, &aliasNoopLogger{}, 1)
	q := mustQ("a.depth.", domain.RRTypeA)
	recs, err := ch.Chase(q, []domain.ResourceRecord{c1})
	assert.ErrorIs(t, err, ErrAliasDepthExceeded)
	// Chain should contain first hop plus the head where depth exceeded
	assert.Equal(t, []domain.ResourceRecord{c1, c2}, recs)
}

func Test_NoOpChase(t *testing.T) {
	noOpResolver := NewNoOpAliasResolver()
	rr, err := domain.NewAuthoritativeResourceRecord(
		"example.com",
		domain.RRTypeA,
		domain.RRClassIN,
		300,
		[]byte{192, 168, 2, 1},
		"192.168.2.1",
	)
	assert.NoError(t, err)
	foo, err2 := noOpResolver.Chase(mustQ("example.com.", domain.RRTypeA), []domain.ResourceRecord{rr})
	assert.NoError(t, err2)
	assert.Equal(t, rr, foo[0])
}
