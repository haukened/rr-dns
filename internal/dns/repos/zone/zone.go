// Package zone provides functions for loading and parsing DNS zone files in various formats.
// It supports loading zones from YAML, JSON, and TOML files, and converting them into authoritative DNS records.
package zone

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

// LoadZoneDirectory walks the given directory, loading all supported zone files (YAML, JSON, TOML)
// and returning a slice of AuthoritativeRecord pointers. Each file is parsed and its records are appended.
// Returns an error if any file fails to parse.
func LoadZoneDirectory(dir string, defaultTTL time.Duration) ([]*domain.AuthoritativeRecord, error) {
	var records []*domain.AuthoritativeRecord

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		zoneRecords, err := loadZoneFile(path, defaultTTL)
		if err != nil {
			return fmt.Errorf("error parsing zone file %s: %w", path, err)
		}
		records = append(records, zoneRecords...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return records, nil
}

// expandName returns the fully qualified domain name for a label, expanding '@' to the root,
// and appending the root if the label is not already absolute.
func expandName(label, root string) string {
	if label == "@" {
		return root
	}
	if strings.HasSuffix(label, ".") {
		return label
	}
	return label + "." + root
}

// normalize converts a value to a slice of strings. Accepts either a string or a slice of any.
// Used to handle zone file record values that may be single or multiple strings.
func normalize(val any) []string {
	switch v := val.(type) {
	case string:
		return []string{v}
	case []any:
		var out []string
		for _, x := range v {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// buildAuthoritativeRecord creates one or more AuthoritativeRecord objects for a given FQDN, RR type,
// and value. The value may be a string or a slice of strings. Returns an error if record creation fails.
func buildAuthoritativeRecord(fqdn string, rrType string, val any, defaultTTL time.Duration) ([]*domain.AuthoritativeRecord, error) {
	strs := normalize(val)
	var records []*domain.AuthoritativeRecord
	for _, s := range strs {
		ar, err := domain.NewAuthoritativeRecord(
			fqdn,
			domain.RRTypeFromString(rrType),
			domain.RRClass(1),
			uint32(defaultTTL.Seconds()),
			[]byte(s),
		)
		if err != nil {
			return nil, err
		}
		records = append(records, ar)
	}
	return records, nil
}

// loadZoneFile loads and parses a single zone file at the given path, using the appropriate parser
// for the file extension (YAML, JSON, TOML). Returns a slice of AuthoritativeRecord pointers or an error.
func loadZoneFile(path string, defaultTTL time.Duration) ([]*domain.AuthoritativeRecord, error) {
	ext := strings.ToLower(filepath.Ext(path))
	var parser koanf.Parser
	switch ext {
	case ".yaml", ".yml":
		parser = yaml.Parser()
	case ".json":
		parser = json.Parser()
	case ".toml":
		parser = toml.Parser()
	default:
		return nil, nil // unsupported file type
	}

	k := koanf.New(".")
	if err := k.Load(file.Provider(path), parser); err != nil {
		return nil, fmt.Errorf("failed to load zone file %s: %w", path, err)
	}

	root := k.String("zone_root")
	if root == "" {
		return nil, fmt.Errorf("zone file %s missing 'zone_root'", path)
	}

	var records []*domain.AuthoritativeRecord
	for name, raw := range k.Raw() {
		if name == "zone_root" {
			continue
		}
		rawMap, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		fqdn := expandName(name, root)
		for rrType, val := range rawMap {
			recs, err := buildAuthoritativeRecord(fqdn, rrType, val, defaultTTL)
			if err != nil {
				return nil, fmt.Errorf("invalid record in %s: %w", path, err)
			}
			records = append(records, recs...)
		}
	}
	return records, nil
}
