package domain

import "testing"

func TestBlockDecision_IsBlocked(t *testing.T) {
	d1 := BlockDecision{Blocked: true}
	if !d1.IsBlocked() {
		t.Fatalf("expected true")
	}
	d2 := BlockDecision{Blocked: false}
	if d2.IsBlocked() {
		t.Fatalf("expected false")
	}
}

func TestEmptyDecision(t *testing.T) {
	d := EmptyDecision()
	if d.Blocked {
		t.Fatalf("empty decision should not be blocked")
	}
	if d.MatchedRule != "" || d.Source != "" {
		t.Fatalf("empty decision should have empty fields")
	}
}
