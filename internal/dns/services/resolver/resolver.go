package resolver

type Service struct {
	blocklist     Blocklist
	transport     ServerTransport
	upstream      UpstreamClient
	upstreamCache Cache
	zoneCache     ZoneCache
}
