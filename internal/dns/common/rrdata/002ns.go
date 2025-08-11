package rrdata

// EncodeNSData encodes an NS record string into its binary representation.
func EncodeNSData(data string) ([]byte, error) {
	// data = "ns.example.com"
	return EncodeDomainName(data)
}
