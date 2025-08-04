package zone

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const testYAML = `
zone_root: example.com
www:
  A: "1.2.3.4"
`

const testInvalidYAML = `
zone_root: example.com
www:
mail:
		Foo: "bar"`

const testJSON = `{
	"zone_root": "example.org",
	"api": {
	  "A": "5.6.7.8"
	}
}
`

const testTOML = `zone_root = "example.net"
[web]
A = "1.2.3.4"
`

func TestLoadZoneDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "zone.yaml")
	jsonFile := filepath.Join(tmpDir, "zone.json")
	tomlFile := filepath.Join(tmpDir, "zone.toml")

	if err := os.WriteFile(yamlFile, []byte(testYAML), 0644); err != nil {
		t.Fatalf("failed to write YAML file: %v", err)
	}
	if err := os.WriteFile(jsonFile, []byte(testJSON), 0644); err != nil {
		t.Fatalf("failed to write JSON file: %v", err)
	}
	if err := os.WriteFile(tomlFile, []byte(testTOML), 0644); err != nil {
		t.Fatalf("failed to write TOML file: %v", err)
	}

	records, err := LoadZoneDirectory(tmpDir, 60*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 3 {
		t.Errorf("expected 3 records, got %d", len(records))
	}
}

func TestLoadZoneDrirectory_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	records, err := LoadZoneDirectory(tmpDir, 60*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

func TestLoadZoneDirectory_InvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	invalidFile := filepath.Join(tmpDir, "invalid.txt")
	if err := os.WriteFile(invalidFile, []byte("invalid content"), 0644); err != nil {
		t.Fatalf("failed to write invalid file: %v", err)
	}

	records, err := LoadZoneDirectory(tmpDir, 60*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if records != nil {
		t.Errorf("expected nil records for unsupported extension, got %v", records)
	}
}

func TestLoadZoneDirectory_MalformedFile(t *testing.T) {
	tmpDir := t.TempDir()
	malformedFile := filepath.Join(tmpDir, "malformed.yaml")
	if err := os.WriteFile(malformedFile, []byte(testInvalidYAML), 0644); err != nil {
		t.Fatalf("failed to write malformed file: %v", err)
	}

	records, err := LoadZoneDirectory(tmpDir, 60*time.Second)
	if err == nil {
		t.Errorf("expected error for malformed file, got nil")
	}
	if records != nil {
		t.Errorf("expected nil records for malformed file, got %v", records)
	}
}

func TestLoadZoneFile_YAML(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "testzone.yaml")
	/*
		* We will load multiple A records for the same FQDN
		* to ensure that the parser can handle multiple values correctly.
		* per RFC 1035, Secion 3.4.1 "A RDATA format":

		> Hosts that have multiple Internet addresses will have multiple A records.
		> The RDATA section of an A line in a master file is an Internet address expressed as four
		> decimal numbers separated by dots without any imbedded spaces (e.g., "10.2.0.52" or "192.0.5.6").

		* Therefore, we expect 3 records to be created.
	*/

	content := `
zone_root: example.com
www:
  A: 
    - "1.2.3.4"
    - "5.6.7.8"
mail:
  MX: "mail.example.com"
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	records, err := loadZoneFile(tmpFile, 300*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 3 {
		t.Errorf("expected 3 records, got %d", len(records))
	}
	names := map[string]bool{}
	types := map[string]bool{}
	for _, r := range records {
		names[r.Name] = true
		types[r.Type.String()] = true
	}
	if !names["www.example.com."] || !names["mail.example.com."] {
		t.Errorf("unexpected record names: %v", names)
	}
	if !types["A"] || !types["MX"] {
		t.Errorf("unexpected record types: %v", types)
	}
	aCount := 0
	for _, r := range records {
		if r.Name == "www.example.com." && r.Type.String() == "A" {
			aCount++
		}
	}
	if aCount != 2 {
		t.Errorf("expected 2 A records for www, got %d", aCount)
	}
}

func TestLoadZoneFile_JSON(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "testzone.json")
	content := `{
	"zone_root": "example.org",
	"api": {
	"A": "5.6.7.8"
	}
}`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	records, err := loadZoneFile(tmpFile, 120*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}
	r := records[0]
	if r.Name != "api.example.org." {
		t.Errorf("unexpected name: %s", r.Name)
	}
	if r.Type.String() != "A" {
		t.Errorf("unexpected type: %s", r.Type.String())
	}
	if string(r.Data) != "5.6.7.8" {
		t.Errorf("unexpected data: %s", string(r.Data))
	}
	if r.TTL() != 120 {
		t.Errorf("unexpected TTL: %d", r.TTL())
	}
}

func TestLoadZoneFile_TOML(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "testzone.toml")
	content := `zone_root = "example.net"

[web]
A = "9.8.7.6"
[mx]
MX = "mx.example.net"
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	records, err := loadZoneFile(tmpFile, 180*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}
	names := map[string]bool{}
	types := map[string]bool{}
	for _, r := range records {
		names[r.Name] = true
		types[r.Type.String()] = true
	}
	if !names["web.example.net."] || !names["mx.example.net."] {
		t.Errorf("unexpected record names: %v", names)
	}
	if !types["A"] || !types["MX"] {
		t.Errorf("unexpected record types: %v", types)
	}
	for _, r := range records {
		if r.TTL() != 180 {
			t.Errorf("unexpected TTL: %d", r.TTL())
		}
	}
}

func TestLoadZoneFile_UnsupportedExtension(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "testzone.txt")
	if err := os.WriteFile(tmpFile, []byte("irrelevant"), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	records, err := loadZoneFile(tmpFile, 60*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if records != nil {
		t.Errorf("expected nil records for unsupported extension, got %v", records)
	}
}

func TestLoadZoneFile_MissingZoneRoot(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "testzone.yaml")
	content := `
www:
  A: "1.2.3.4"
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := loadZoneFile(tmpFile, 60*time.Second)
	if err == nil || !strings.Contains(err.Error(), "missing 'zone_root'") {
		t.Errorf("expected missing zone_root error, got: %v", err)
	}
}

func TestLoadZoneFile_InvalidFile(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "testzone.yaml")
	content := `:invalid_yaml`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := loadZoneFile(tmpFile, 60*time.Second)
	if err == nil {
		t.Errorf("expected error for invalid file, got nil")
	}
}

func TestLoadZoneFile_EmptyRawMap(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "testzone.yaml")
	content := `
zone_root: example.com
www:
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	records, err := loadZoneFile(tmpFile, 60*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

func TestLoadZoneFile_BadResourceRecord(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "testzone.yaml")
	content := `
zone_root: example.com
www:
  INVALID: "1.2.3.4"`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	records, err := loadZoneFile(tmpFile, 60*time.Second)
	if err == nil {
		t.Errorf("expected error for bad resource record, got nil")
	}
	if records != nil {
		t.Errorf("expected nil records for bad resource record, got %v", records)
	}
}

func TestBuildResourceRecord(t *testing.T) {
	fqdn := "foo.example.com."
	rrType := "A"
	val := "1.2.3.4"
	defaultTTL := 60 * time.Second
	records, err := buildResourceRecord(fqdn, rrType, val, defaultTTL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	ar := records[0]
	if ar.Name != fqdn {
		t.Errorf("Name = %v, want %v", ar.Name, fqdn)
	}
	if ar.Type != 1 {
		t.Errorf("Type = %v, want 1", ar.Type)
	}
	if ar.Class != 1 {
		t.Errorf("Class = %v, want 1", ar.Class)
	}
	if ar.TTL() != 60 {
		t.Errorf("TTL = %v, want 60", ar.TTL())
	}
	if string(ar.Data) != val {
		t.Errorf("Data = %v, want %v", ar.Data, val)
	}
}

func TestBuildResourceRecord_InvalidType(t *testing.T) {
	fqdn := "foo.example.com."
	rrType := "INVALID"
	val := "1.2.3.4"
	defaultTTL := 60 * time.Second
	_, err := buildResourceRecord(fqdn, rrType, val, defaultTTL)
	if err == nil {
		t.Errorf("expected error for invalid RRType, got nil")
	}
}

func TestBuildResourceRecord_Multi(t *testing.T) {
	fqdn := "foo.example.com."
	rrType := "A"
	val := []any{"1.2.3.4", "5.6.7.8"}
	defaultTTL := 60 * time.Second
	records, err := buildResourceRecord(fqdn, rrType, val, defaultTTL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if string(records[0].Data) != "1.2.3.4" || string(records[1].Data) != "5.6.7.8" {
		t.Errorf("unexpected Data: %v, %v", records[0].Data, records[1].Data)
	}
}

func TestExpandName(t *testing.T) {
	cases := []struct {
		label string
		root  string
		want  string
	}{
		{"@", "example.com.", "example.com."},
		{"foo", "example.com.", "foo.example.com."},
		{"bar.", "example.com.", "bar."},
	}
	for _, tc := range cases {
		got := expandName(tc.label, tc.root)
		if got != tc.want {
			t.Errorf("expandName(%q, %q) = %q, want %q", tc.label, tc.root, got, tc.want)
		}
	}
}

func TestNormalize(t *testing.T) {
	cases := []struct {
		input any
		want  []string
	}{
		{"foo", []string{"foo"}},
		{[]any{"bar", "baz"}, []string{"bar", "baz"}},
		{123, nil},
		{[]any{123, "x"}, []string{"x"}},
	}
	for _, tc := range cases {
		got := normalize(tc.input)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("normalize(%v) = %v, want %v", tc.input, got, tc.want)
		}
	}
}
