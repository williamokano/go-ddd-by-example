package domain_test

import (
	"errors"
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

const validUUID = "0192f5e0-7c1a-7b3e-9d2a-3f4b5c6d7e8f"

func TestIDs(t *testing.T) {
	type id interface {
		String() string
		IsZero() bool
	}
	parsers := map[string]func(string) (id, error){
		"ShowID":     func(s string) (id, error) { return domain.ParseShowID(s) },
		"VenueID":    func(s string) (id, error) { return domain.ParseVenueID(s) },
		"PromoterID": func(s string) (id, error) { return domain.ParsePromoterID(s) },
	}
	for name, parse := range parsers {
		t.Run(name+" round-trips", func(t *testing.T) {
			got, err := parse(validUUID)
			if err != nil || got.String() != validUUID || got.IsZero() {
				t.Errorf("parse = %v, %v", got, err)
			}
		})
		for _, bad := range []string{"", "nope", "00000000-0000-0000-0000-000000000000"} {
			t.Run(name+" rejects "+bad, func(t *testing.T) {
				if _, err := parse(bad); !errors.Is(err, domain.ErrInvalidID) {
					t.Errorf("error = %v, want %v", err, domain.ErrInvalidID)
				}
			})
		}
	}
}
