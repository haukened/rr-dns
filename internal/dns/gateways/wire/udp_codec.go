// Package wire provides encoding and decoding of DNS messages for UDP transport.
// It handles the DNS wire format as specified in RFC 1035.
package wire

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/haukened/rr-dns/internal/dns/common/log"
	"github.com/haukened/rr-dns/internal/dns/domain"
)

// udpCodec implements the DNSCodec interface for standard DNS over UDP messages.
type udpCodec struct {
	logger log.Logger
}

// NewUDPCodec creates and returns a new instance of udpCodec using the provided logger.
// The logger is used for logging within the codec.
func NewUDPCodec(logger log.Logger) *udpCodec {
	return &udpCodec{
		logger: logger,
	}
}

// EncodeQuery serializes a Question into a binary format suitable for sending via UDP.
func (c *udpCodec) EncodeQuery(query domain.Question) ([]byte, error) {
	var buf bytes.Buffer

	// Header
	_ = binary.Write(&buf, binary.BigEndian, query.ID)       // ID
	_ = binary.Write(&buf, binary.BigEndian, uint16(0x0100)) // Flags: standard query, RD=1
	_ = binary.Write(&buf, binary.BigEndian, uint16(1))      // QDCOUNT
	_ = binary.Write(&buf, binary.BigEndian, uint16(0))      // ANCOUNT
	_ = binary.Write(&buf, binary.BigEndian, uint16(0))      // NSCOUNT
	_ = binary.Write(&buf, binary.BigEndian, uint16(0))      // ARCOUNT

	// Question
	name := strings.TrimSuffix(query.Name, ".") // Remove trailing dot
	labels := strings.Split(name, ".")
	for _, label := range labels {
		if len(label) > 63 {
			return nil, fmt.Errorf("label too long: %s", label)
		}
		if len(label) > 0 { // Skip empty labels
			buf.WriteByte(byte(len(label)))
			buf.WriteString(label)
		}
	}
	buf.WriteByte(0) // End of name
	_ = binary.Write(&buf, binary.BigEndian, uint16(query.Type))
	_ = binary.Write(&buf, binary.BigEndian, uint16(query.Class))

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
func (c *udpCodec) DecodeQuery(data []byte) (domain.Question, error) {
	if len(data) < 12 {
		return domain.Question{}, errors.New("query too short")
	}
	id := binary.BigEndian.Uint16(data[0:2])
	qdCount := binary.BigEndian.Uint16(data[4:6])
	if qdCount != 1 {
		return domain.Question{}, errors.New("expected exactly one question")
	}
	name, qtype, qclass, _, err := decodeQuestion(data, 12)
	if err != nil {
		return domain.Question{}, err
	}
	return domain.Question{
		ID:    id,
		Name:  name,
		Type:  domain.RRType(qtype),
		Class: domain.RRClass(qclass),
	}, nil
}

// EncodeResponse serializes a DNSResponse into a binary format suitable for sending via UDP.
func (c *udpCodec) EncodeResponse(resp domain.DNSResponse) ([]byte, error) {
	var buf bytes.Buffer

	_ = binary.Write(&buf, binary.BigEndian, resp.ID)
	_ = binary.Write(&buf, binary.BigEndian, uint16(0x8180)) // Flags: standard response, RA=1
	_ = binary.Write(&buf, binary.BigEndian, uint16(1))      // QDCOUNT

	// Safely convert slice length to uint16 with bounds check
	answerCount := len(resp.Answers)
	if answerCount > 65535 {
		return nil, fmt.Errorf("too many answer records: %d (max 65535)", answerCount)
	}
	_ = binary.Write(&buf, binary.BigEndian, uint16(answerCount))

	_ = binary.Write(&buf, binary.BigEndian, uint16(0)) // NSCOUNT
	_ = binary.Write(&buf, binary.BigEndian, uint16(0)) // ARCOUNT

	c.logger.Debug(map[string]any{
		"step": "header_written",
		"id":   resp.ID,
		"qd":   1,
		"an":   answerCount,
	}, "Wrote DNS response header")

	// Echo question for response synthesis (stubbed name/type)
	qname, err := encodeDomainName(resp.Answers[0].Name)
	if err != nil {
		return nil, err
	}
	buf.Write(qname)
	_ = binary.Write(&buf, binary.BigEndian, uint16(resp.Answers[0].Type))
	_ = binary.Write(&buf, binary.BigEndian, uint16(resp.Answers[0].Class))
	qnameOffset := 12 // QNAME always starts right after the 12-byte header

	c.logger.Debug(map[string]any{
		"step":  "question_written",
		"name":  resp.Answers[0].Name,
		"type":  resp.Answers[0].Type.String(),
		"class": resp.Answers[0].Class.String(),
	}, "Wrote question section")

	// Answers
	for _, rr := range resp.Answers {
		// Use name compression (pointer to offset where QNAME begins) when the answer name matches the original QNAME.
		// This reduces packet size and avoids duplicate encoding.
		if rr.Name == resp.Answers[0].Name {
			// Use name compression: pointer to the QNAME we just wrote.
			// This reduces packet size and avoids repeating the domain name.
			// Format: 0b11xxxxxx xxxxxxxx (pointer to offset in message)
			buf.Write([]byte{0xC0 | byte(qnameOffset>>8), byte(qnameOffset & 0xFF)})
		} else {
			name, err := encodeDomainName(rr.Name)
			if err != nil {
				return nil, err
			}
			buf.Write(name)
		}
		_ = binary.Write(&buf, binary.BigEndian, uint16(rr.Type))
		_ = binary.Write(&buf, binary.BigEndian, uint16(rr.Class))
		_ = binary.Write(&buf, binary.BigEndian, uint32(rr.TTLRemaining().Seconds()))

		// Safely convert data length to uint16 with bounds check
		dataLen := len(rr.Data)
		if dataLen > 65535 {
			return nil, fmt.Errorf("resource record data too large: %d bytes (max 65535)", dataLen)
		}
		_ = binary.Write(&buf, binary.BigEndian, uint16(dataLen))

		buf.Write(rr.Data)

		c.logger.Debug(map[string]any{
			"step":  "answer_written",
			"name":  rr.Name,
			"type":  rr.Type.String(),
			"class": rr.Class.String(),
			"ttl":   rr.TTLRemaining().Seconds(),
			"dlen":  len(rr.Data),
		}, "Wrote answer record")
	}

	c.logger.Debug(map[string]any{
		"step": "final_packet",
		"size": buf.Len(),
		"raw":  fmt.Sprintf("%x", buf.Bytes()),
	}, "Final encoded DNS response")

	return buf.Bytes(), nil
}

// DecodeResponse parses a raw DNS response from a UDP packet into a DNSResponse,
// validating the response ID and extracting resource records.
func (c *udpCodec) DecodeResponse(data []byte, expectedID uint16, now time.Time) (domain.DNSResponse, error) {
	if len(data) < 12 {
		return domain.DNSResponse{}, errors.New("response too short")
	}
	id := binary.BigEndian.Uint16(data[0:2])
	if id != expectedID {
		return domain.DNSResponse{}, fmt.Errorf("ID mismatch: expected %d, got %d", expectedID, id)
	}

	// Parse flags to extract RCode (lower 4 bits of byte 3)
	flags := binary.BigEndian.Uint16(data[2:4])
	//gosec:disable G115 -- uint16 & 0x000F always results in a uint8 value, so this is safe.
	rcode := domain.RCode(uint8(flags & 0x000F))

	qdCount := binary.BigEndian.Uint16(data[4:6])
	anCount := binary.BigEndian.Uint16(data[6:8])
	nsCount := binary.BigEndian.Uint16(data[8:10])
	arCount := binary.BigEndian.Uint16(data[10:12])

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
		rr, newOffset, err := c.parseResourceRecord(data, offset, now)
		if err != nil {
			return domain.DNSResponse{}, fmt.Errorf("failed to parse answer record %d: %w", i, err)
		}
		answers = append(answers, rr)
		offset = newOffset
	}

	// Parse authority records
	authority := []domain.ResourceRecord{}
	for i := 0; i < int(nsCount); i++ {
		rr, newOffset, err := c.parseResourceRecord(data, offset, now)
		if err != nil {
			return domain.DNSResponse{}, fmt.Errorf("failed to parse authority record %d: %w", i, err)
		}
		authority = append(authority, rr)
		offset = newOffset
	}

	// Parse additional records
	additional := []domain.ResourceRecord{}
	for i := 0; i < int(arCount); i++ {
		rr, newOffset, err := c.parseResourceRecord(data, offset, now)
		if err != nil {
			return domain.DNSResponse{}, fmt.Errorf("failed to parse additional record %d: %w", i, err)
		}
		additional = append(additional, rr)
		offset = newOffset
	}

	return domain.DNSResponse{
		ID:         id,
		RCode:      rcode,
		Answers:    answers,
		Authority:  authority,
		Additional: additional,
	}, nil
}

// parseResourceRecord extracts a single resource record from DNS response data
func (c *udpCodec) parseResourceRecord(data []byte, offset int, now time.Time) (domain.ResourceRecord, int, error) {
	if offset+10 > len(data) {
		return domain.ResourceRecord{}, 0, errors.New("truncated record section")
	}

	name, newOffset, err := decodeName(data, offset)
	if err != nil {
		return domain.ResourceRecord{}, 0, fmt.Errorf("failed to decode record name: %w", err)
	}
	offset = newOffset

	if offset+10 > len(data) {
		return domain.ResourceRecord{}, 0, errors.New("truncated record section after name")
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
		return domain.ResourceRecord{}, 0, errors.New("truncated rdata")
	}
	rdata := make([]byte, rdLen)
	copy(rdata, data[offset:offset+int(rdLen)])
	offset += int(rdLen)

	rrtype := domain.RRType(typ)
	rrclass := domain.RRClass(class)
	rr, err := domain.NewCachedResourceRecord(name, rrtype, rrclass, ttl, rdata, now)
	if err != nil {
		return domain.ResourceRecord{}, 0, fmt.Errorf("invalid resource record: %w", err)
	}

	return rr, offset, nil
}

var _ DNSCodec = &udpCodec{}
