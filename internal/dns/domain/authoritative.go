package domain

import (
	"fmt"
	"time"
)

// AuthoritativeRecord represents a DNS record served from a zone file.
// These records do not expire from memory, and their TTL is preserved for wire responses.
type AuthoritativeRecord struct {
	Name  string
	Type  RRType
	Class RRClass
	TTL   uint32
	Data  []byte
}

// Validate checks whether the AuthoritativeRecord fields are valid.
func (ar AuthoritativeRecord) Validate() error {
	if ar.Name == "" {
		return fmt.Errorf("record name must not be empty")
	}
	if !ar.Type.IsValid() {
		return fmt.Errorf("invalid RRType: %d", ar.Type)
	}
	if !ar.Class.IsValid() {
		return fmt.Errorf("invalid RRClass: %d", ar.Class)
	}
	return nil
}

// NewResourceRecordFromAuthoritative converts an AuthoritativeRecord into a ResourceRecord with expiration.
func NewResourceRecordFromAuthoritative(ar AuthoritativeRecord, now time.Time) ResourceRecord {
	return ResourceRecord{
		Name:      ar.Name,
		Type:      ar.Type,
		Class:     ar.Class,
		ExpiresAt: now.Add(time.Duration(ar.TTL) * time.Second),
		Data:      ar.Data,
	}
}
