package domain_test

import (
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

func TestParseVenueID(t *testing.T) {
	t.Run("accepts a valid UUID and round-trips it", func(t *testing.T) {
		raw := "0192f5e0-7c1a-7b3e-9d2a-3f4b5c6d7e8f"

		id, err := domain.ParseVenueID(raw)

		if err != nil {
			t.Fatalf("ParseVenueID(%q) error = %v", raw, err)
		}
		if got := id.String(); got != raw {
			t.Errorf("String() = %q, want %q", got, raw)
		}
	})
}
