package rrdata

// encodePTRData encodes a PTR record string into its binary representation.
func encodePTRData(data string) ([]byte, error) {
	// data = "ptr.example.com"
	return encodeDomainName(data)
}

// decodePTRData decodes a PTR (Pointer) record's RDATA from the given byte slice.
// It returns the domain name as a string and an error if decoding fails.
func decodePTRData(b []byte) (string, error) {
	return decodeDomainName(b)
}
