package domain_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

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

	t.Run("rejects garbage", func(t *testing.T) {
		_, err := domain.ParseVenueID("not-a-uuid")

		if !errors.Is(err, domain.ErrInvalidVenueID) {
			t.Errorf("ParseVenueID() error = %v, want %v", err, domain.ErrInvalidVenueID)
		}
	})

	t.Run("rejects the nil UUID", func(t *testing.T) {
		_, err := domain.ParseVenueID("00000000-0000-0000-0000-000000000000")

		if !errors.Is(err, domain.ErrInvalidVenueID) {
			t.Errorf("ParseVenueID() error = %v, want %v", err, domain.ErrInvalidVenueID)
		}
	})
}

func TestNewVenueID(t *testing.T) {
	u := uuid.MustParse("0192f5e0-7c1a-7b3e-9d2a-3f4b5c6d7e8f")

	id := domain.NewVenueID(u)

	if got := id.String(); got != u.String() {
		t.Errorf("String() = %q, want %q", got, u.String())
	}
}
