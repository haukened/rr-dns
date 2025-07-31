package domain

import "fmt"

// DNSResponse represents a complete DNS response with answers, authority, and additional sections.
// This follows RFC 1035 §4.1.1 structure for DNS response messages.
type DNSResponse struct {
	ID         uint16
	RCode      RCode
	Answers    []ResourceRecord
	Authority  []ResourceRecord
	Additional []ResourceRecord
}

// NewDNSResponse constructs a DNSResponse and validates its fields.
func NewDNSResponse(id uint16, rcode RCode, answers, authority, additional []ResourceRecord) (DNSResponse, error) {
	resp := DNSResponse{
		ID:         id,
		RCode:      rcode,
		Answers:    answers,
		Authority:  authority,
		Additional: additional,
	}
	if err := resp.Validate(); err != nil {
		return DNSResponse{}, err
	}
	return resp, nil
}

// NewDNSErrorResponse creates a DNSResponse with the specified ID and response code (RCode),
// representing an error response. The Answers, Authority, and Additional sections are set to nil.
func NewDNSErrorResponse(id uint16, rcode RCode) DNSResponse {
	return DNSResponse{
		ID:         id,
		RCode:      rcode,
		Answers:    nil,
		Authority:  nil,
		Additional: nil,
	}
}

// Validate checks whether the DNSResponse fields are structurally valid.
func (resp DNSResponse) Validate() error {
	if !resp.RCode.IsValid() {
		return fmt.Errorf("invalid RCode: %d", resp.RCode)
	}

	// Validate all records in each section
	for i, rr := range resp.Answers {
		if err := rr.Validate(); err != nil {
			return fmt.Errorf("invalid answer record at index %d: %w", i, err)
		}
	}

	for i, rr := range resp.Authority {
		if err := rr.Validate(); err != nil {
			return fmt.Errorf("invalid authority record at index %d: %w", i, err)
		}
	}

	for i, rr := range resp.Additional {
		if err := rr.Validate(); err != nil {
			return fmt.Errorf("invalid additional record at index %d: %w", i, err)
		}
	}

	return nil
}

// IsError returns true if the response indicates an error condition.
func (resp DNSResponse) IsError() bool {
	return resp.RCode != 0 // NOERROR = 0
}

// HasAnswers returns true if the response contains answer records.
func (resp DNSResponse) HasAnswers() bool {
	return len(resp.Answers) > 0
}

// AnswerCount returns the number of answer records in the response.
func (resp DNSResponse) AnswerCount() int {
	return len(resp.Answers)
}

// AuthorityCount returns the number of authority records in the response.
func (resp DNSResponse) AuthorityCount() int {
	return len(resp.Authority)
}

// AdditionalCount returns the number of additional records in the response.
func (resp DNSResponse) AdditionalCount() int {
	return len(resp.Additional)
}
