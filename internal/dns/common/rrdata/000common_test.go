package rrdata

import (
	"net"
	"strings"
	"testing"
)

func TestEncodeDomainName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []byte
		wantErr bool
	}{
		{
			name:    "simple domain",
			input:   "Foo.Example.com.",
			want:    []byte{3, 'f', 'o', 'o', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0},
			wantErr: false,
		},
		{
			name:    "single label",
			input:   "LOCALHOST.",
			want:    []byte{9, 'l', 'o', 'c', 'a', 'l', 'h', 'o', 's', 't', 0},
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   " ",
			want:    []byte{0},
			wantErr: false,
		},
		{
			name:    "label too long",
			input:   strings.Repeat("A", 64) + ".COM.",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "trailing dot omitted",
			input:   "Foo.Example.com",
			want:    []byte{3, 'f', 'o', 'o', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0},
			wantErr: false,
		},
		{
			name:    "multiple consecutive dots",
			input:   "foo..example.com.",
			want:    []byte{3, 'f', 'o', 'o', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := encodeDomainName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("EncodeDomainName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !equalBytes(got, tt.want) {
				t.Errorf("EncodeDomainName() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestIsIPv4(t *testing.T) {
	tests := []struct {
		name string
		ip   net.IP
		want bool
	}{
		{
			name: "valid IPv4",
			ip:   net.ParseIP("192.168.1.1"),
			want: true,
		},
		{
			name: "valid IPv6",
			ip:   net.ParseIP("2001:db8::1"),
			want: false,
		},
		{
			name: "nil IP",
			ip:   nil,
			want: false,
		},
		{
			name: "empty IP",
			ip:   net.IP{},
			want: false,
		},
		{
			name: "IPv4-mapped IPv6",
			ip:   net.ParseIP("::ffff:192.168.1.1"),
			want: true,
		},
		{
			name: "invalid IP string",
			ip:   net.ParseIP("not.an.ip"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isIPv4(tt.ip)
			if got != tt.want {
				t.Errorf("isIPv4(%v) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}
func TestIsIPv6(t *testing.T) {
	tests := []struct {
		name string
		ip   net.IP
		want bool
	}{
		{
			name: "valid IPv6",
			ip:   net.ParseIP("2001:db8::1"),
			want: true,
		},
		{
			name: "valid IPv4",
			ip:   net.ParseIP("192.168.1.1"),
			want: false,
		},
		{
			name: "IPv4-mapped IPv6",
			ip:   net.ParseIP("::ffff:192.168.1.1"),
			want: false,
		},
		{
			name: "nil IP",
			ip:   nil,
			want: false,
		},
		{
			name: "empty IP",
			ip:   net.IP{},
			want: false,
		},
		{
			name: "invalid IP string",
			ip:   net.ParseIP("not.an.ip"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isIPv6(tt.ip)
			if got != tt.want {
				t.Errorf("isIPv6(%v) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}
func TestDecodeDomainName(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    string
		wantErr bool
	}{
		{
			name:    "simple domain",
			input:   []byte{3, 'f', 'o', 'o', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0},
			want:    "foo.example.com",
			wantErr: false,
		},
		{
			name:    "single label",
			input:   []byte{9, 'l', 'o', 'c', 'a', 'l', 'h', 'o', 's', 't', 0},
			want:    "localhost",
			wantErr: false,
		},
		{
			name:    "empty input",
			input:   []byte{0},
			want:    "",
			wantErr: false,
		},
		{
			name:    "invalid encoding (label length exceeds input)",
			input:   []byte{4, 'a', 'b', 0},
			want:    "",
			wantErr: true,
		},
		{
			name:    "multiple labels",
			input:   []byte{2, 'a', 'b', 2, 'c', 'd', 0},
			want:    "ab.cd",
			wantErr: false,
		},
		{
			name:    "trailing zero only",
			input:   []byte{0},
			want:    "",
			wantErr: false,
		},
		{
			name:    "label length zero in middle",
			input:   []byte{3, 'f', 'o', 'o', 0, 3, 'b', 'a', 'r', 0},
			want:    "foo",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeDomainName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeDomainName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("decodeDomainName() = %q, want %q", got, tt.want)
			}
		})
	}
}
func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
