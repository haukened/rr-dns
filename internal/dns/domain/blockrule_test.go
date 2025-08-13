package domain

import (
	"testing"
	"time"
)

func TestParseBlockRuleKind(t *testing.T) {
	cases := []struct {
		in      string
		want    BlockRuleKind
		wantErr bool
	}{
		{"exact", BlockRuleExact, false},
		{"ExAcT", BlockRuleExact, false},
		{"suffix", BlockRuleSuffix, false},
		{" SUFFIX ", BlockRuleSuffix, false},
		{"", 0, true},
		{"wild", 0, true},
	}

	for _, tc := range cases {
		got, err := ParseBlockRuleKind(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("ParseBlockRuleKind(%q) expected error, got nil", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ParseBlockRuleKind(%q) unexpected error: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("ParseBlockRuleKind(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestNewBlockRule_Valid(t *testing.T) {
	now := time.Now()
	r, err := NewBlockRule("example.com", BlockRuleSuffix, "test-source", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Name != "example.com" {
		t.Errorf("Name = %q, want example.com", r.Name)
	}
	if !r.IsSuffix() {
		t.Errorf("IsSuffix() = false, want true")
	}
	if r.Source != "test-source" {
		t.Errorf("Source = %q, want test-source", r.Source)
	}
	if r.AddedAt.IsZero() {
		t.Errorf("AddedAt should be set")
	}
}

func TestNewBlockRule_Invalid(t *testing.T) {
	now := time.Now()

	_, err := NewBlockRule("", BlockRuleExact, "s", now)
	if err == nil {
		t.Errorf("expected error for empty name")
	}

	_, err = NewBlockRule("example.com", BlockRuleExact, "", now)
	if err == nil {
		t.Errorf("expected error for empty source")
	}

	_, err = NewBlockRule("example.com", BlockRuleExact, "s", time.Time{})
	if err == nil {
		t.Errorf("expected error for zero AddedAt")
	}

	badKind := BlockRuleKind(99)
	_, err = NewBlockRule("example.com", badKind, "s", now)
	if err == nil {
		t.Errorf("expected error for unsupported kind")
	}
}

func TestConvenienceConstructors(t *testing.T) {
	now := time.Now()
	r1, err := NewExactBlockRule("apex.com", "file:A", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !r1.IsExact() {
		t.Errorf("IsExact() = false, want true")
	}

	r2, err := NewSuffixBlockRule("example.com", "file:B", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !r2.IsSuffix() {
		t.Errorf("IsSuffix() = false, want true")
	}
}

func TestBlockRuleKind_String(t *testing.T) {
	cases := []struct {
		kind     BlockRuleKind
		expected string
	}{
		{BlockRuleExact, "exact"},
		{BlockRuleSuffix, "suffix"},
		{BlockRuleKind(42), "BlockRuleKind(42)"},
	}

	for _, tc := range cases {
		got := tc.kind.String()
		if got != tc.expected {
			t.Errorf("BlockRuleKind(%d).String() = %q, want %q", tc.kind, got, tc.expected)
		}
	}
}
