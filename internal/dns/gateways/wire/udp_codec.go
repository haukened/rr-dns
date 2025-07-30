// Package wire provides encoding and decoding of DNS messages for UDP transport.
// It handles the DNS wire format as specified in RFC 1035.
package wire

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

// udpCodec implements the DNSCodec interface for standard DNS over UDP messages.
type udpCodec struct{}

// EncodeQuery serializes a DNSQuery into a binary format suitable for sending via UDP.
func (c *udpCodec) EncodeQuery(query domain.DNSQuery) ([]byte, error) {
	var buf bytes.Buffer

	// Header
	_ = binary.Write(&buf, binary.BigEndian, query.ID)       // ID
	_ = binary.Write(&buf, binary.BigEndian, uint16(0x0100)) // Flags: standard query, RD=1
	_ = binary.Write(&buf, binary.BigEndian, uint16(1))      // QDCOUNT
	_ = binary.Write(&buf, binary.BigEndian, uint16(0))      // ANCOUNT
	_ = binary.Write(&buf, binary.BigEndian, uint16(0))      // NSCOUNT
	_ = binary.Write(&buf, binary.BigEndian, uint16(0))      // ARCOUNT

	// Question
	labels := strings.Split(query.Name, ".")
	for _, label := range labels {
		if len(label) > 63 {
			return nil, fmt.Errorf("label too long: %s", label)
		}
		buf.WriteByte(byte(len(label)))
		buf.WriteString(label)
	}
	buf.WriteByte(0) // End of name
	_ = binary.Write(&buf, binary.BigEndian, uint16(query.Type))
	_ = binary.Write(&buf, binary.BigEndian, uint16(1)) // QCLASS=IN

	return buf.Bytes(), nil
}

// decodeName decodes a domain name from a DNS message at the specified offset,
// handling label compression as defined in RFC 1035.
func decodeName(data []byte, offset int) (string, int, error) {
	var labels []string
	for {
		if offset >= len(data) {
			return "", 0, errors.New("offset out of bounds")
		}
		length := int(data[offset])
		if length == 0 {
			offset++
			break
		}
		if length&0xC0 == 0xC0 {
			if offset+1 >= len(data) {
				return "", 0, errors.New("compression pointer out of bounds")
			}
			ptr := int(binary.BigEndian.Uint16(data[offset:offset+2]) & 0x3FFF)
			suffix, _, err := decodeName(data, ptr)
			if err != nil {
				return "", 0, err
			}
			labels = append(labels, suffix)
			offset += 2
			break
		}
		offset++
		if offset+length > len(data) {
			return "", 0, errors.New("label length out of bounds")
		}
		labels = append(labels, string(data[offset:offset+length]))
		offset += length
	}
	return strings.Join(labels, "."), offset, nil
}

// encodeDomainName encodes a domain name into DNS wire format without compression.
func encodeDomainName(name string) ([]byte, error) {
	var buf bytes.Buffer
	if name == "" {
		buf.WriteByte(0)
		return buf.Bytes(), nil
	}
	labels := strings.Split(name, ".")
	for _, label := range labels {
		if len(label) > 63 {
			return nil, fmt.Errorf("label too long: %s", label)
		}
		buf.WriteByte(byte(len(label)))
		buf.WriteString(label)
	}
	buf.WriteByte(0)
	return buf.Bytes(), nil
}

// DecodeQuery parses a DNS query message from data.
func (c *udpCodec) DecodeQuery(data []byte) (domain.DNSQuery, error) {
	if len(data) < 12 {
		return domain.DNSQuery{}, errors.New("query too short")
	}
	id := binary.BigEndian.Uint16(data[0:2])
	qdCount := binary.BigEndian.Uint16(data[4:6])
	if qdCount != 1 {
		return domain.DNSQuery{}, errors.New("expected exactly one question")
	}
	name, qtype, _, _, err := decodeQuestion(data, 12)
	if err != nil {
		return domain.DNSQuery{}, err
	}
	return domain.DNSQuery{
		ID:   id,
		Name: name,
		Type: domain.RRType(qtype),
	}, nil
}

// EncodeResponse serializes a DNSResponse into a binary format suitable for sending via UDP.
func (c *udpCodec) EncodeResponse(resp domain.DNSResponse) ([]byte, error) {
	var buf bytes.Buffer

	_ = binary.Write(&buf, binary.BigEndian, resp.ID)
	_ = binary.Write(&buf, binary.BigEndian, uint16(0x8180)) // Flags: standard response, RA=1
	_ = binary.Write(&buf, binary.BigEndian, uint16(1))      // QDCOUNT
	_ = binary.Write(&buf, binary.BigEndian, uint16(len(resp.Answers)))
	_ = binary.Write(&buf, binary.BigEndian, uint16(0)) // NSCOUNT
	_ = binary.Write(&buf, binary.BigEndian, uint16(0)) // ARCOUNT

	// Echo question for response synthesis (stubbed name/type)
	qname, err := encodeDomainName(resp.Answers[0].Name)
	if err != nil {
		return nil, err
	}
	buf.Write(qname)
	_ = binary.Write(&buf, binary.BigEndian, uint16(resp.Answers[0].Type))
	_ = binary.Write(&buf, binary.BigEndian, uint16(resp.Answers[0].Class))

	// Answers
	for _, rr := range resp.Answers {
		name, err := encodeDomainName(rr.Name)
		if err != nil {
			return nil, err
		}
		buf.Write(name)
		_ = binary.Write(&buf, binary.BigEndian, uint16(rr.Type))
		_ = binary.Write(&buf, binary.BigEndian, uint16(rr.Class))
		_ = binary.Write(&buf, binary.BigEndian, uint32(rr.TTLRemaining().Seconds()))
		_ = binary.Write(&buf, binary.BigEndian, uint16(len(rr.Data)))
		buf.Write(rr.Data)
	}
	return buf.Bytes(), nil
}

// DecodeResponse parses a raw DNS response from a UDP packet into a DNSResponse,
// validating the response ID and extracting resource records.
func (c *udpCodec) DecodeResponse(data []byte, expectedID uint16) (domain.DNSResponse, error) {
	if len(data) < 12 {
		return domain.DNSResponse{}, errors.New("response too short")
	}
	id := binary.BigEndian.Uint16(data[0:2])
	if id != expectedID {
		return domain.DNSResponse{}, fmt.Errorf("ID mismatch: expected %d, got %d", expectedID, id)
	}

	qdCount := binary.BigEndian.Uint16(data[4:6])
	anCount := binary.BigEndian.Uint16(data[6:8])

	offset := 12
	// Skip questions
	for i := 0; i < int(qdCount); i++ {
		for {
			if offset >= len(data) {
				return domain.DNSResponse{}, errors.New("truncated question name")
			}
			l := int(data[offset])
			offset++
			if l == 0 {
				break
			}
			offset += l
		}
		offset += 4 // QTYPE + QCLASS
	}

	// Parse answers
	answers := []domain.ResourceRecord{}
	for i := 0; i < int(anCount); i++ {
		if offset+10 > len(data) {
			return domain.DNSResponse{}, errors.New("truncated answer section")
		}

		name, newOffset, err := decodeName(data, offset)
		if err != nil {
			return domain.DNSResponse{}, fmt.Errorf("failed to decode answer name: %w", err)
		}
		offset = newOffset

		if offset+10 > len(data) {
			return domain.DNSResponse{}, errors.New("truncated answer section after name")
		}

		typ := binary.BigEndian.Uint16(data[offset : offset+2])
		offset += 2
		class := binary.BigEndian.Uint16(data[offset : offset+2])
		offset += 2
		ttl := binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4
		rdLen := binary.BigEndian.Uint16(data[offset : offset+2])
		offset += 2

		if offset+int(rdLen) > len(data) {
			return domain.DNSResponse{}, errors.New("truncated rdata")
		}
		rdata := make([]byte, rdLen)
		copy(rdata, data[offset:offset+int(rdLen)])
		offset += int(rdLen)

		rrtype := domain.RRType(typ)
		rrclass := domain.RRClass(class)
		rr, err := domain.NewResourceRecord(name, rrtype, rrclass, ttl, rdata)
		if err != nil {
			return domain.DNSResponse{}, fmt.Errorf("invalid resource record: %w", err)
		}
		answers = append(answers, rr)
	}

	return domain.DNSResponse{
		ID:      id,
		Answers: answers,
	}, nil
}

var UDP DNSCodec = &udpCodec{}
