package parsers

import (
	"bytes"
	"testing"
	"time"

	"github.com/haukened/rr-dns/internal/dns/common/log"
)

func TestParseHostsFile_Basic(t *testing.T) {
	input := `
# comment
127.0.0.1 localhost
::1 localhost ip6-localhost ip6-loopback
0.0.0.0 example.com example.org # inline comment
# wildcard-like entries should be ignored
0.0.0.0 *.bad.example.com .also.bad.example.com
192.168.1.1 sub.Example.com
1.2.3.4 . .
255.255.255.255 broadcast
`
	now := time.Unix(1723551000, 0)
	got, err := ParseHostsFile(bytes.NewBufferString(input), "hosts-src", log.NewNoopLogger(), now)
	if err != nil {
		t.Fatalf("ParseHostsFile returned error: %v", err)
	}

	// Expected: example.com, example.org, sub.example.com (localhost tokens ignored; wildcard/leading-dot skipped)
	if len(got) != 3 {
		t.Fatalf("expected 3 rules, got %d: %#v", len(got), got)
	}
	if got[0].Name != "example.com" || !got[0].IsExact() {
		t.Fatalf("rule[0] unexpected: %+v", got[0])
	}
	if got[1].Name != "example.org" || !got[1].IsExact() {
		t.Fatalf("rule[1] unexpected: %+v", got[1])
	}
	if got[2].Name != "sub.example.com" || !got[2].IsExact() {
		t.Fatalf("rule[2] unexpected: %+v", got[2])
	}
	for i, r := range got {
		if r.Source != "hosts-src" || !r.AddedAt.Equal(now) {
			t.Fatalf("rule[%d] meta unexpected: %+v", i, r)
		}
	}
}

func TestParseHostsFile_DuplicatesAndScannerError(t *testing.T) {
	// Duplicates across lines and same line
	input := "0.0.0.0 dup.example.com dup.example.com\n0.0.0.0 dup.example.com\n"
	got, err := ParseHostsFile(bytes.NewBufferString(input), "s", log.NewNoopLogger(), time.Now())
	if err != nil {
		t.Fatalf("ParseHostsFile returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 rule after dedupe, got %d", len(got))
	}

	// Oversized line to trigger scanner error
	big := bytes.Repeat([]byte{'a'}, 70000)
	_, err = ParseHostsFile(bytes.NewBuffer(big), "s", log.NewNoopLogger(), time.Now())
	if err == nil {
		t.Fatalf("expected scanner error, got nil")
	}
}

func TestParseHostsFile_NoHostnames_And_ConstructorErrors(t *testing.T) {
	// No hostnames after IP should be skipped (hosts_no_hostnames)
	input := "192.0.2.1\n0.0.0.0 example.com\n"

	// With valid metadata, only the second line should produce a rule
	got, err := ParseHostsFile(bytes.NewBufferString(input), "src", log.NewNoopLogger(), time.Now())
	if err != nil {
		t.Fatalf("ParseHostsFile returned error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "example.com" {
		t.Fatalf("expected one rule for example.com, got %#v", got)
	}

	// Empty source triggers constructor error skip
	got, err = ParseHostsFile(bytes.NewBufferString("0.0.0.0 example.com\n"), "", log.NewNoopLogger(), time.Unix(1723552000, 0))
	if err != nil {
		t.Fatalf("ParseHostsFile returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 rules due to constructor error (empty source), got %d", len(got))
	}

	// Zero time triggers constructor error skip
	got, err = ParseHostsFile(bytes.NewBufferString("0.0.0.0 example.com\n"), "src", log.NewNoopLogger(), time.Time{})
	if err != nil {
		t.Fatalf("ParseHostsFile returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 rules due to constructor error (zero time), got %d", len(got))
	}
}
