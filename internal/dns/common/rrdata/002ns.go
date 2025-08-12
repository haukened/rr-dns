package rrdata

// encodeNSData encodes an NS record string into its binary representation.
func encodeNSData(data string) ([]byte, error) {
	// data = "ns.example.com"
	return encodeDomainName(data)
}

// decodeNSData decodes a byte slice representing an NS (Name Server) record's RDATA
func decodeNSData(b []byte) (string, error) {
	return decodeDomainName(b)
}
