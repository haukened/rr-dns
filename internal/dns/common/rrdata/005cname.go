package rrdata

// EncodeCNAMEData encodes a CNAME record string into its binary representation.
func EncodeCNAMEData(data string) ([]byte, error) {
	// data = "cname.example.com"
	return EncodeDomainName(data)
}
