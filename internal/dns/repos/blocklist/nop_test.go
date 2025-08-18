package blocklist

import (
	"testing"

	"github.com/haukened/rr-dns/internal/dns/domain"
)

func TestNoopBlocklist_IsBlocked(t *testing.T) {
	blocklist := &NoopBlocklist{}

	tests := []struct {
		name     string
		question domain.Question
		want     bool
	}{
		{
			name:     "returns false for any question",
			question: domain.Question{Name: "example.com.", Type: 1, Class: 1},
			want:     false,
		},
		{
			name:     "returns false for empty question",
			question: domain.Question{},
			want:     false,
		},
		{
			name:     "returns false for another domain",
			question: domain.Question{Name: "blocked.com.", Type: 28, Class: 1},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := blocklist.IsBlocked(tt.question)
			if got != tt.want {
				t.Errorf("IsBlocked() = %v, want %v", got, tt.want)
			}
		})
	}
}
