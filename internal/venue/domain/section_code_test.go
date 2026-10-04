package domain_test

import (
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

func TestNewSectionCode(t *testing.T) {
	t.Run("accepts and normalises valid codes (VEN-2)", func(t *testing.T) {
		tests := []struct{ raw, want string }{
			{"ORCH", "ORCH"},
			{"bal-l", "BAL-L"},
			{" A1 ", "A1"},
		}
		for _, tt := range tests {
			t.Run(tt.raw, func(t *testing.T) {
				code, err := domain.NewSectionCode(tt.raw)

				if err != nil {
					t.Fatalf("NewSectionCode(%q) error = %v", tt.raw, err)
				}
				if got := code.String(); got != tt.want {
					t.Errorf("String() = %q, want %q", got, tt.want)
				}
			})
		}
	})
}
