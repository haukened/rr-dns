package zone

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const benchmarkYAML = `
zone_root: example.com
www:
  A: 
    - "192.0.2.1"
    - "192.0.2.2"
    - "192.0.2.3"
mail:
  A: "192.0.2.10"
  MX: "mail.example.com"
ftp:
  A: "192.0.2.20"
  CNAME: "files.example.com"
api:
  A: "192.0.2.30"
admin:
  A: "192.0.2.40"
blog:
  A: "192.0.2.50"
shop:
  A: "192.0.2.60"
support:
  A: "192.0.2.70"
docs:
  A: "192.0.2.80"
cdn:
  A: "192.0.2.90"
`

const benchmarkJSON = `{
  "zone_root": "example.org",
  "www": {
    "A": ["203.0.113.1", "203.0.113.2", "203.0.113.3"]
  },
  "mail": {
    "A": "203.0.113.10",
    "MX": "mail.example.org"
  },
  "ftp": {
    "A": "203.0.113.20"
  },
  "api": {
    "A": "203.0.113.30"
  },
  "admin": {
    "A": "203.0.113.40"
  },
  "blog": {
    "A": "203.0.113.50"
  },
  "shop": {
    "A": "203.0.113.60"
  },
  "support": {
    "A": "203.0.113.70"
  },
  "docs": {
    "A": "203.0.113.80"
  },
  "cdn": {
    "A": "203.0.113.90"
  }
}`

const benchmarkTOML = `zone_root = "example.net"

[www]
A = ["198.51.100.1", "198.51.100.2", "198.51.100.3"]

[mail]
A = "198.51.100.10"
MX = "mail.example.net"

[ftp]
A = "198.51.100.20"

[api]
A = "198.51.100.30"

[admin]
A = "198.51.100.40"

[blog]
A = "198.51.100.50"

[shop]
A = "198.51.100.60"

[support]
A = "198.51.100.70"

[docs]
A = "198.51.100.80"

[cdn]
A = "198.51.100.90"
`

func BenchmarkLoadZoneFile_YAML(b *testing.B) {
	tmpFile := filepath.Join(b.TempDir(), "benchmark.yaml")
	if err := os.WriteFile(tmpFile, []byte(benchmarkYAML), 0644); err != nil {
		b.Fatalf("failed to write temp file: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := loadZoneFile(tmpFile, 300*time.Second)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkLoadZoneFile_JSON(b *testing.B) {
	tmpFile := filepath.Join(b.TempDir(), "benchmark.json")
	if err := os.WriteFile(tmpFile, []byte(benchmarkJSON), 0644); err != nil {
		b.Fatalf("failed to write temp file: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := loadZoneFile(tmpFile, 300*time.Second)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkLoadZoneFile_TOML(b *testing.B) {
	tmpFile := filepath.Join(b.TempDir(), "benchmark.toml")
	if err := os.WriteFile(tmpFile, []byte(benchmarkTOML), 0644); err != nil {
		b.Fatalf("failed to write temp file: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := loadZoneFile(tmpFile, 300*time.Second)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkLoadZoneDirectory(b *testing.B) {
	tmpDir := b.TempDir()

	// Create multiple zone files
	files := map[string]string{
		"zone1.yaml": benchmarkYAML,
		"zone2.json": benchmarkJSON,
		"zone3.toml": benchmarkTOML,
	}

	for filename, content := range files {
		filePath := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			b.Fatalf("failed to write %s: %v", filename, err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := LoadZoneDirectory(tmpDir, 300*time.Second)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkBuildResourceRecord_Single(b *testing.B) {
	fqdn := "www.example.com."
	rrType := "A"
	val := "192.0.2.1"
	defaultTTL := 300 * time.Second

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := buildResourceRecord(fqdn, rrType, []string{val}, defaultTTL)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkBuildResourceRecord_Multiple(b *testing.B) {
	fqdn := "www.example.com."
	rrType := "A"
	raw := []any{"192.0.2.1", "192.0.2.2", "192.0.2.3", "192.0.2.4", "192.0.2.5"}
	values := toStringValues(raw)
	defaultTTL := 300 * time.Second

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := buildResourceRecord(fqdn, rrType, values, defaultTTL)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkExpandName(b *testing.B) {
	label := "www"
	root := "example.com"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = expandName(label, root)
	}
}
