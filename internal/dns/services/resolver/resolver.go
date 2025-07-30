package resolver

type Service struct {
	transport     ServerTransport
	upstream      UpstreamClient
	upstreamCache Cache
	zoneCache     ZoneCache
}
