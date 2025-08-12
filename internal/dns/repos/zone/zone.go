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

	"github.com/haukened/rr-dns/internal/dns/common/rrdata"
	"github.com/haukened/rr-dns/internal/dns/common/utils"
	"github.com/haukened/rr-dns/internal/dns/domain"
)

// LoadZoneDirectory walks the given directory, loading all supported zone files (YAML, JSON, TOML)
// and returning a map of zone roots to their records. This preserves zone organization for cache loading.
// Returns an error if any file fails to parse.
func LoadZoneDirectory(dir string, defaultTTL time.Duration) (map[string][]domain.ResourceRecord, error) {
	zones := make(map[string][]domain.ResourceRecord)

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		zoneRoot, zoneRecords, err := loadZoneFileWithRoot(path, defaultTTL)
		if err != nil {
			return fmt.Errorf("error parsing zone file %s: %w", path, err)
		}
		if zoneRoot != "" && len(zoneRecords) > 0 {
			zones[zoneRoot] = append(zones[zoneRoot], zoneRecords...)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return zones, nil
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

// toStringValues converts a raw koanf-parsed value (string or []any of strings) into a slice of
// non-empty strings, skipping empty or non-string elements. This lets us validate and sanitize
// record values before building ResourceRecords. Invalid types yield an empty slice which the
// caller can treat as no-op (defensive len check) instead of crashing the loader.
func toStringValues(val any) []string {
	switch v := val.(type) {
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return nil
		}
		return []string{s}
	case []any:
		out := make([]string, 0, len(v))
		for _, elem := range v {
			s, ok := elem.(string)
			if !ok {
				continue // skip non-strings silently
			}
			s = strings.TrimSpace(s)
			if s == "" {
				continue // skip empty strings
			}
			out = append(out, s)
		}
		if len(out) == 0 {
			return nil
		}
		return out
	default:
		return nil
	}
}

// buildResourceRecord creates one or more ResourceRecord objects for a given FQDN, RR type,
// and value. The value may be a string or a slice of strings. Returns an error if record creation fails.
func buildResourceRecord(fqdn string, rrType string, values []string, defaultTTL time.Duration) ([]domain.ResourceRecord, error) {
	rType := domain.RRTypeFromString(rrType)
	var records []domain.ResourceRecord
	for _, s := range values {
		if s == "" { // defensive; helper should have stripped empties
			continue
		}
		data, err := rrdata.Encode(rType, s)
		if err != nil {
			return nil, err
		}
		rr, err := domain.NewAuthoritativeResourceRecord(
			fqdn,
			rType,
			domain.RRClass(1),
			uint32(defaultTTL.Seconds()),
			data,
			s, // preserve original text form
		)
		if err != nil {
			return nil, err
		}
		records = append(records, rr)
	}
	return records, nil
}

// loadZoneFileWithRoot loads and parses a single zone file, returning both the zone root and records.
// This is used by LoadZoneDirectory to preserve zone organization.
func loadZoneFileWithRoot(path string, defaultTTL time.Duration) (string, []domain.ResourceRecord, error) {
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
		return "", nil, nil // unsupported file type
	}

	k := koanf.New(".")
	if err := k.Load(file.Provider(path), parser); err != nil {
		return "", nil, fmt.Errorf("failed to load zone file %s: %w", path, err)
	}

	root := k.String("zone_root")
	if root == "" {
		return "", nil, fmt.Errorf("zone file %s missing 'zone_root'", path)
	}

	// Canonicalize the zone root to ensure consistent format with trailing dot
	root = utils.CanonicalDNSName(root)

	var records []domain.ResourceRecord
	for name, raw := range k.Raw() {
		if name == "zone_root" {
			continue
		}
		rawMap, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		fqdn := utils.CanonicalDNSName(expandName(name, root)) // early canonicalization (owner name)
		for rrType, val := range rawMap {
			values := toStringValues(val)
			if len(values) == 0 { // skip silently (empty or invalid elements)
				continue
			}
			recs, err := buildResourceRecord(fqdn, rrType, values, defaultTTL)
			if err != nil {
				return "", nil, fmt.Errorf("invalid record in %s: %w", path, err)
			}
			records = append(records, recs...)
		}
	}
	return root, records, nil
}

// loadZoneFile loads and parses a single zone file at the given path, using the appropriate parser
// for the file extension (YAML, JSON, TOML). Returns a slice of ResourceRecord values or an error.
func loadZoneFile(path string, defaultTTL time.Duration) ([]domain.ResourceRecord, error) {
	_, records, err := loadZoneFileWithRoot(path, defaultTTL)
	return records, err
}
