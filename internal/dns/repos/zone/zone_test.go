package zone

import (
	"bytes"
	"net"
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

	zones, err := LoadZoneDirectory(tmpDir, 60*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(zones) != 3 {
		t.Errorf("expected 3 zones, got %d", len(zones))
	}

	// Check that we have the expected zones (without assuming canonicalization)
	expectedZones := []string{"example.com", "example.org", "example.net"}
	for _, expected := range expectedZones {
		found := false
		for zoneRoot := range zones {
			if strings.TrimSuffix(zoneRoot, ".") == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected zone %s not found in zones: %v", expected, zones)
		}
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
	if len(records) != 0 {
		t.Errorf("expected empty map for unsupported extension, got %v", records)
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
  MX: ["10 mail.example.com."]
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
	// Check binary encoded data
	expected := net.ParseIP("5.6.7.8").To4()
	if !bytes.Equal(r.Data, expected) {
		t.Errorf("unexpected data: got %v, want %v", r.Data, expected)
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
[mail]
MX = ["10 mail.example.com."]
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
	if !names["web.example.net."] || !names["mail.example.net."] {
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
	if !bytes.Equal(ar.Data, net.ParseIP(val).To4()) {
		t.Errorf("data does not equal bytes for IP %s", val)
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
	if !bytes.Equal(records[0].Data, net.ParseIP("1.2.3.4").To4()) || !bytes.Equal(records[1].Data, net.ParseIP("5.6.7.8").To4()) {
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

// Additional tests for missing coverage

func TestBuildResourceRecord_EncodeError(t *testing.T) {
	fqdn := "foo.example.com."
	rrType := "A"
	val := "invalid.ip.address"
	defaultTTL := 60 * time.Second
	_, err := buildResourceRecord(fqdn, rrType, val, defaultTTL)
	if err == nil {
		t.Errorf("expected error for invalid A record data, got nil")
	}
}

func TestBuildResourceRecord_InvalidRRType(t *testing.T) {
	fqdn := "foo.example.com."
	rrType := "UNKNOWN"
	val := "1.2.3.4"
	defaultTTL := 60 * time.Second
	_, err := buildResourceRecord(fqdn, rrType, val, defaultTTL)
	if err == nil {
		t.Errorf("expected error for unknown RRType, got nil")
	}
}

func TestBuildResourceRecord_ValidationError(t *testing.T) {
	// Empty FQDN should cause validation error
	fqdn := ""
	rrType := "A"
	val := "1.2.3.4"
	defaultTTL := 60 * time.Second
	_, err := buildResourceRecord(fqdn, rrType, val, defaultTTL)
	if err == nil {
		t.Errorf("expected validation error for empty FQDN, got nil")
	}
}

func TestLoadZoneDirectory_WalkError(t *testing.T) {
	// Test with non-existent directory
	nonExistentDir := "/non/existent/directory"
	_, err := LoadZoneDirectory(nonExistentDir, 60*time.Second)
	if err == nil {
		t.Errorf("expected error for non-existent directory, got nil")
	}
}

func TestLoadZoneDirectory_FileError(t *testing.T) {
	// Create a temporary directory with a file that has zone parsing errors
	tmpDir := t.TempDir()

	// Create a file with valid format but encoding errors
	yamlFile := filepath.Join(tmpDir, "bad.yaml")
	content := `
zone_root: example.com
www:
  A: "invalid.ip.address.format"
`
	if err := os.WriteFile(yamlFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := LoadZoneDirectory(tmpDir, 60*time.Second)
	if err == nil {
		t.Errorf("expected error for file with encoding errors, got nil")
	}
}

func TestLoadZoneFileWithRoot_FileReadError(t *testing.T) {
	// Test with non-existent file
	_, _, err := loadZoneFileWithRoot("/non/existent/file.yaml", 60*time.Second)
	if err == nil {
		t.Errorf("expected error for non-existent file, got nil")
	}
}

func TestLoadZoneFile_NonExistentFile(t *testing.T) {
	_, err := loadZoneFile("/non/existent/file.yaml", 60*time.Second)
	if err == nil {
		t.Errorf("expected error for non-existent file, got nil")
	}
}

func TestNormalize_EmptySlice(t *testing.T) {
	// Test normalize with empty slice - should return nil
	got := normalize([]any{})
	if got != nil {
		t.Errorf("normalize([]any{}) = %v, want nil", got)
	}
}

func TestNormalize_NonStringSlice(t *testing.T) {
	// Test normalize with slice containing only non-strings
	got := normalize([]any{123, 456, true})
	if got != nil {
		t.Errorf("normalize([]any{123, 456, true}) = %v, want nil", got)
	}
}
