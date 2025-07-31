package zonecache

import (
	"errors"
	"sync"
	"time"

	"github.com/haukened/rr-dns/internal/dns/domain"
	"github.com/haukened/rr-dns/internal/dns/services/resolver"
)

var (
	ErrInvalidZoneRoot = errors.New("invalid zone root format")
	ErrZoneNotFound    = errors.New("zone not found")
	ErrEmptyZone       = errors.New("zone contains no records")
)

// ZoneCache is an in-memory implementation of resolver.ZoneCache.
// It provides fast access to authoritative DNS records with concurrent safety.
type ZoneCache struct {
	mu    sync.RWMutex
	zones map[string]map[string]*domain.AuthoritativeRecord
	//    zoneRoot → CacheKey → record
}

// Ensure ZoneCache implements resolver.ZoneCache at compile time
var _ resolver.ZoneCache = (*ZoneCache)(nil)

// New creates a new ZoneCache instance
func New() *ZoneCache {
	return &ZoneCache{
		zones: make(map[string]map[string]*domain.AuthoritativeRecord),
	}
}

// Find returns resource records matching the DNSQuery
func (zc *ZoneCache) Find(query domain.DNSQuery) ([]*domain.ResourceRecord, bool) {
	zc.mu.RLock()
	defer zc.mu.RUnlock()

	fqdn := query.Name

	// Find the zone that contains this FQDN
	var zoneRecords map[string]*domain.AuthoritativeRecord
	var found bool

	// Look for the most specific zone that contains this FQDN
	for zoneRoot, zone := range zc.zones {
		if isInZone(fqdn, zoneRoot) {
			zoneRecords = zone
			found = true
			break
		}
	}

	if !found {
		return nil, false
	}

	key := query.CacheKey()
	record, exists := zoneRecords[key]
	if !exists {
		return nil, false
	}
	rr := domain.NewResourceRecordFromAuthoritative(*record, time.Now())
	return []*domain.ResourceRecord{&rr}, true
}

// ReplaceZone replaces all records for a zone with new records
func (zc *ZoneCache) ReplaceZone(zoneRoot string, records []*domain.AuthoritativeRecord) error {
	if zoneRoot == "" {
		return ErrInvalidZoneRoot
	}

	// Ensure zone root ends with dot
	if zoneRoot[len(zoneRoot)-1] != '.' {
		zoneRoot = zoneRoot + "."
	}

	zc.mu.Lock()
	defer zc.mu.Unlock()

	// Create new zone map
	zoneMap := make(map[string]*domain.AuthoritativeRecord)

	// Group records by CacheKey
	for _, record := range records {
		zoneMap[record.CacheKey()] = record
	}

	// Replace the zone
	zc.zones[zoneRoot] = zoneMap

	return nil
}

// RemoveZone removes all records for a zone
func (zc *ZoneCache) RemoveZone(zoneRoot string) error {
	if zoneRoot == "" {
		return ErrInvalidZoneRoot
	}

	// Ensure zone root ends with dot
	if zoneRoot[len(zoneRoot)-1] != '.' {
		zoneRoot = zoneRoot + "."
	}

	zc.mu.Lock()
	defer zc.mu.Unlock()

	if _, exists := zc.zones[zoneRoot]; !exists {
		return ErrZoneNotFound
	}

	delete(zc.zones, zoneRoot)
	return nil
}

// All returns a snapshot of all zone data
func (zc *ZoneCache) All() map[string][]*domain.AuthoritativeRecord {
	zc.mu.RLock()
	defer zc.mu.RUnlock()

	result := make(map[string][]*domain.AuthoritativeRecord)

	for zoneRoot, zone := range zc.zones {
		var allRecords []*domain.AuthoritativeRecord
		for _, record := range zone {
			allRecords = append(allRecords, record)
		}
		result[zoneRoot] = allRecords
	}

	return result
}

// Zones returns a list of all zone roots currently cached
func (zc *ZoneCache) Zones() []string {
	zc.mu.RLock()
	defer zc.mu.RUnlock()

	zones := make([]string, 0, len(zc.zones))
	for zoneRoot := range zc.zones {
		zones = append(zones, zoneRoot)
	}

	return zones
}

// Count returns the total number of records across all zones
func (zc *ZoneCache) Count() int {
	zc.mu.RLock()
	defer zc.mu.RUnlock()

	count := 0
	for _, zone := range zc.zones {
		count += len(zone)
	}

	return count
}

// isInZone checks if an FQDN belongs to a given zone
func isInZone(fqdn, zoneRoot string) bool {
	// Ensure both end with dots
	if fqdn[len(fqdn)-1] != '.' {
		fqdn = fqdn + "."
	}
	if zoneRoot[len(zoneRoot)-1] != '.' {
		zoneRoot = zoneRoot + "."
	}

	// Exact match (apex record)
	if fqdn == zoneRoot {
		return true
	}

	// Check if FQDN is a subdomain of the zone root
	if len(fqdn) > len(zoneRoot) {
		// Must end with the zone root
		if fqdn[len(fqdn)-len(zoneRoot):] == zoneRoot {
			// Must have a dot before the zone root (proper DNS hierarchy)
			prefixLen := len(fqdn) - len(zoneRoot)
			return prefixLen > 0 && fqdn[prefixLen-1] == '.'
		}
	}

	return false
}
