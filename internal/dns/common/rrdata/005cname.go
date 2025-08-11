package rrdata

// encodeCNAMEData encodes a CNAME record string into its binary representation.
func encodeCNAMEData(data string) ([]byte, error) {
	// data = "cname.example.com"
	return encodeDomainName(data)
}

// decodeCNAMEData decodes the CNAME record data from the given byte slice.
func decodeCNAMEData(b []byte) (string, error) {
	return decodeDomainName(b)
}
