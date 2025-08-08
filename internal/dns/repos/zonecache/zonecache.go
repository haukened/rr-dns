package zonecache

import (
	"sync"

	"github.com/haukened/rr-dns/internal/dns/common/utils"
	"github.com/haukened/rr-dns/internal/dns/domain"
	"github.com/haukened/rr-dns/internal/dns/services/resolver"
)

// ZoneCache is an in-memory implementation of resolver.ZoneCache.
// It provides fast access to authoritative DNS records with concurrent safety and value-based storage.
type ZoneCache struct {
	mu    sync.RWMutex
	zones map[string]map[string][]domain.ResourceRecord
	//    zoneRoot → CacheKey → records (value-based)
}

// New creates a new ZoneCache instance
func New() *ZoneCache {
	return &ZoneCache{
		zones: make(map[string]map[string][]domain.ResourceRecord),
	}
}

// FindRecords returns authoritative records matching the Question.
// Zero allocations - returns slice directly from cache.
func (zc *ZoneCache) FindRecords(query domain.Question) ([]domain.ResourceRecord, bool) {
	zc.mu.RLock()
	defer zc.mu.RUnlock()

	fqdn := utils.CanonicalDNSName(query.Name)
	zone := utils.GetApexDomain(fqdn)

	zoneRecords, found := zc.zones[zone]
	if !found {
		return nil, false
	}

	key := query.CacheKey()
	records, exists := zoneRecords[key]
	if !exists {
		return nil, false
	}

	return records, true // ✅ Zero allocations - return slice directly
}

// PutZone replaces all records for a zone with new records
func (zc *ZoneCache) PutZone(zoneRoot string, records []domain.ResourceRecord) {
	zoneRoot = utils.CanonicalDNSName(zoneRoot)

	zc.mu.Lock()
	defer zc.mu.Unlock()

	// Create new zone map
	zoneMap := make(map[string][]domain.ResourceRecord)

	// Group records by CacheKey
	for _, record := range records {
		key := record.CacheKey()
		zoneMap[key] = append(zoneMap[key], record)
	}

	// Replace the zone
	zc.zones[zoneRoot] = zoneMap
}

// RemoveZone removes all records for a zone
func (zc *ZoneCache) RemoveZone(zoneRoot string) {
	zoneRoot = utils.CanonicalDNSName(zoneRoot)

	zc.mu.Lock()
	defer zc.mu.Unlock()

	delete(zc.zones, zoneRoot)
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

// Ensure ZoneCache implements resolver.ZoneCache at compile time
var _ resolver.ZoneCache = (*ZoneCache)(nil)
