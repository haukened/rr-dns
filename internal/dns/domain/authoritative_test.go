package domain

import (
	"testing"
	"time"
)

func TestAuthoritativeRecord_Validate(t *testing.T) {
	cases := []struct {
		name    string
		ar      AuthoritativeRecord
		wantErr bool
	}{
		{
			name: "valid record",
			ar: AuthoritativeRecord{
				Name:  "example.com.",
				Type:  1, // A
				Class: 1, // IN
				TTL:   60,
				Data:  []byte{1, 2, 3, 4},
			},
			wantErr: false,
		},
		{
			name: "empty name",
			ar: AuthoritativeRecord{
				Name:  "",
				Type:  1,
				Class: 1,
				TTL:   60,
				Data:  []byte{1, 2, 3, 4},
			},
			wantErr: true,
		},
		{
			name: "invalid type",
			ar: AuthoritativeRecord{
				Name:  "example.com.",
				Type:  9999,
				Class: 1,
				TTL:   60,
				Data:  []byte{1, 2, 3, 4},
			},
			wantErr: true,
		},
		{
			name: "invalid class",
			ar: AuthoritativeRecord{
				Name:  "example.com.",
				Type:  1,
				Class: 9999,
				TTL:   60,
				Data:  []byte{1, 2, 3, 4},
			},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.ar.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestNewResourceRecordFromAuthoritative(t *testing.T) {
	ar := AuthoritativeRecord{
		Name:  "example.com.",
		Type:  1,
		Class: 1,
		TTL:   120,
		Data:  []byte{192, 168, 1, 1},
	}
	now := time.Now()
	rr := NewResourceRecordFromAuthoritative(ar, now)
	if rr.Name != ar.Name {
		t.Errorf("Name mismatch: got %v, want %v", rr.Name, ar.Name)
	}
	if rr.Type != ar.Type {
		t.Errorf("Type mismatch: got %v, want %v", rr.Type, ar.Type)
	}
	if rr.Class != ar.Class {
		t.Errorf("Class mismatch: got %v, want %v", rr.Class, ar.Class)
	}
	if rr.Data == nil || len(rr.Data) != len(ar.Data) {
		t.Errorf("Data mismatch: got %v, want %v", rr.Data, ar.Data)
	}
	ttl := rr.ExpiresAt.Sub(now)
	if ttl < 119*time.Second || ttl > 121*time.Second {
		t.Errorf("TTL mismatch: got %v, want ~120s", ttl)
	}
}
func TestNewAuthoritativeRecord(t *testing.T) {
	cases := []struct {
		name    string
		input   AuthoritativeRecord
		wantErr bool
	}{
		{
			name: "valid record",
			input: AuthoritativeRecord{
				Name:  "example.com.",
				Type:  1,
				Class: 1,
				TTL:   60,
				Data:  []byte{1, 2, 3, 4},
			},
			wantErr: false,
		},
		{
			name: "empty name",
			input: AuthoritativeRecord{
				Name:  "",
				Type:  1,
				Class: 1,
				TTL:   60,
				Data:  []byte{1, 2, 3, 4},
			},
			wantErr: true,
		},
		{
			name: "invalid type",
			input: AuthoritativeRecord{
				Name:  "example.com.",
				Type:  9999,
				Class: 1,
				TTL:   60,
				Data:  []byte{1, 2, 3, 4},
			},
			wantErr: true,
		},
		{
			name: "invalid class",
			input: AuthoritativeRecord{
				Name:  "example.com.",
				Type:  1,
				Class: 9999,
				TTL:   60,
				Data:  []byte{1, 2, 3, 4},
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ar, err := NewAuthoritativeRecord(tc.input.Name, tc.input.Type, tc.input.Class, tc.input.TTL, tc.input.Data)
			if (err != nil) != tc.wantErr {
				t.Errorf("NewAuthoritativeRecord() error = %v, wantErr %v", err, tc.wantErr)
			}
			if err == nil && ar != nil {
				if ar.Name != tc.input.Name {
					t.Errorf("Name mismatch: got %v, want %v", ar.Name, tc.input.Name)
				}
				if ar.Type != tc.input.Type {
					t.Errorf("Type mismatch: got %v, want %v", ar.Type, tc.input.Type)
				}
				if ar.Class != tc.input.Class {
					t.Errorf("Class mismatch: got %v, want %v", ar.Class, tc.input.Class)
				}
				if ar.TTL != tc.input.TTL {
					t.Errorf("TTL mismatch: got %v, want %v", ar.TTL, tc.input.TTL)
				}
				if len(ar.Data) != len(tc.input.Data) {
					t.Errorf("Data length mismatch: got %v, want %v", len(ar.Data), len(tc.input.Data))
				}
			}
		})
	}
}

func TestAuthoritativeRecord_CacheKey(t *testing.T) {
	cases := []struct {
		name string
		ar   AuthoritativeRecord
		want string
	}{
		{
			name: "standard A record",
			ar: AuthoritativeRecord{
				Name:  "example.com.",
				Type:  1,
				Class: 1,
			},
			want: "example.com.|example.com|A|IN",
		},
		{
			name: "AAAA record with different class",
			ar: AuthoritativeRecord{
				Name:  "host.example.com.",
				Type:  28,
				Class: 3,
			},
			want: "example.com.|host.example.com|AAAA|CH",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.ar.CacheKey()
			if got != tc.want {
				t.Errorf("CacheKey() = %v, want %v", got, tc.want)
			}
		})
	}
}
