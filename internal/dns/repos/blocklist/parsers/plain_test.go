package parsers

import (
	"bytes"
	"testing"
	"time"

	"github.com/haukened/rr-dns/internal/dns/common/log"
)

func TestParsePlainList_Basics(t *testing.T) {
	input := `
# comment at top
Example.COM   
example.com.#inline comment

	sub.Example.com.
# explicit suffix markers
*.wild.example.com
.root.example.org
# another comment
\t\n
example.com   # duplicate
`

	now := time.Unix(1723550000, 0)
	got, err := ParsePlainList(bytes.NewBufferString(input), "test-source", log.NewNoopLogger(), now)
	if err != nil {
		t.Fatalf("ParsePlainList returned error: %v", err)
	}

	if len(got) != 4 {
		t.Fatalf("expected 4 rules, got %d: %#v", len(got), got)
	}

	// Expect order: example.com (exact), sub.example.com (exact), wild.example.com (suffix), root.example.org (suffix)
	if got[0].Name != "example.com" || got[0].IsSuffix() {
		t.Fatalf("rule[0] unexpected: %+v", got[0])
	}
	if got[1].Name != "sub.example.com" || got[1].IsSuffix() {
		t.Fatalf("rule[1] unexpected: %+v", got[1])
	}
	if got[2].Name != "wild.example.com" || !got[2].IsSuffix() {
		t.Fatalf("rule[2] unexpected: %+v", got[2])
	}
	if got[3].Name != "root.example.org" || !got[3].IsSuffix() {
		t.Fatalf("rule[3] unexpected: %+v", got[3])
	}

	for i, r := range got {
		if r.Source != "test-source" {
			t.Fatalf("rule[%d].Source = %q, want %q", i, r.Source, "test-source")
		}
		if !r.AddedAt.Equal(now) {
			t.Fatalf("rule[%d].AddedAt = %v, want %v", i, r.AddedAt, now)
		}
	}
}

func TestParsePlainList_EmptyAndCommentsOnly(t *testing.T) {
	input := "\n# only comments\n   # another\n\n"
	now := time.Now()
	got, err := ParsePlainList(bytes.NewBufferString(input), "s", log.NewNoopLogger(), now)
	if err != nil {
		t.Fatalf("ParsePlainList returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 rules, got %d", len(got))
	}
}

func TestParsePlainList_ConstructRuleErrorsAreSkipped(t *testing.T) {
	input := "example.com\n*.sub.example.com\n"

	t.Run("empty source", func(t *testing.T) {
		now := time.Unix(1723550000, 0)
		got, err := ParsePlainList(bytes.NewBufferString(input), "", log.NewNoopLogger(), now)
		if err != nil {
			t.Fatalf("ParsePlainList returned error: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected 0 rules due to constructor error, got %d: %#v", len(got), got)
		}
	})

	t.Run("zero time", func(t *testing.T) {
		zero := time.Time{}
		got, err := ParsePlainList(bytes.NewBufferString(input), "src", log.NewNoopLogger(), zero)
		if err != nil {
			t.Fatalf("ParsePlainList returned error: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected 0 rules due to constructor error, got %d: %#v", len(got), got)
		}
	})
}

func TestParsePlainList_ScannerError(t *testing.T) {
	// Create a line longer than bufio.Scanner's default max token size (~64K)
	big := bytes.Repeat([]byte{'a'}, 70000)
	input := string(big) // no newline, single oversized token

	got, err := ParsePlainList(bytes.NewBufferString(input), "src", log.NewNoopLogger(), time.Now())
	if err == nil {
		t.Fatalf("expected error from scanner, got nil")
	}
	if got != nil {
		t.Fatalf("expected nil result on error, got len=%d", len(got))
	}
}
