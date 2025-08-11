package rrdata

// EncodePTRData encodes a PTR record string into its binary representation.
func EncodePTRData(data string) ([]byte, error) {
	// data = "ptr.example.com"
	return EncodeDomainName(data)
}
