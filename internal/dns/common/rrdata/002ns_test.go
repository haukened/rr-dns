package rrdata

import "testing"

func TestEncodeNSData(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []byte
		wantErr bool
	}{
		{
			name:    "valid domain",
			input:   "ns.example.com",
			want:    []byte{2, 'n', 's', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0},
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			want:    []byte{0},
			wantErr: false,
		},
		{
			name:    "single label",
			input:   "localhost",
			want:    []byte{9, 'l', 'o', 'c', 'a', 'l', 'h', 'o', 's', 't', 0},
			wantErr: false,
		},
		{
			name:    "trailing dot",
			input:   "ns.example.com.",
			want:    []byte{2, 'n', 's', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EncodeNSData(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("EncodeNSData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !equalBytes(got, tt.want) {
				t.Errorf("EncodeNSData() = %v, want %v", got, tt.want)
			}
		})
	}
}
