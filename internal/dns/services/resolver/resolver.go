package resolver

type Service struct {
	upstream  UpstreamClient
	transport ServerTransport
}
