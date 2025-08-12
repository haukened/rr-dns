package domain

import "fmt"

// Question represents a DNS query section containing a question for resolution.
type Question struct {
	ID    uint16
	Name  string
	Type  RRType
	Class RRClass
}

// NewQuestion constructs a Question and validates its fields.
func NewQuestion(id uint16, name string, rrtype RRType, class RRClass) (Question, error) {
	q := Question{
		ID:    id,
		Name:  name,
		Type:  rrtype,
		Class: class,
	}
	if err := q.Validate(); err != nil {
		return Question{}, err
	}
	return q, nil
}

// Validate checks whether the Question fields are structurally and semantically valid.
func (q Question) Validate() error {
	if q.Name == "" {
		return fmt.Errorf("query name must not be empty")
	}
	if !q.Type.IsValid() {
		return fmt.Errorf("unsupported RRType: %d", q.Type)
	}
	if !q.Class.IsValid() {
		return fmt.Errorf("unsupported RRClass: %d", q.Class)
	}
	return nil
}

// CacheKey returns a cache key string derived from the query's name, type, and class.
func (q Question) CacheKey() string {
	return GenerateCacheKey(q.Name, q.Type, q.Class)
}
