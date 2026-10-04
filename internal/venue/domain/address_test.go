package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

func TestNewAddress(t *testing.T) {
	t.Run("valid input gives trimmed getters (VEN-1)", func(t *testing.T) {
		addr, err := domain.NewAddress("  Rua da Alegria 12 ", " Lisboa ", "PT")

		if err != nil {
			t.Fatalf("NewAddress() error = %v", err)
		}
		if got, want := addr.Street(), "Rua da Alegria 12"; got != want {
			t.Errorf("Street() = %q, want %q", got, want)
		}
		if got, want := addr.City(), "Lisboa"; got != want {
			t.Errorf("City() = %q, want %q", got, want)
		}
		if got, want := addr.Country(), "PT"; got != want {
			t.Errorf("Country() = %q, want %q", got, want)
		}
	})

	t.Run("rejects invalid parts and names the field (VEN-1)", func(t *testing.T) {
		tests := []struct {
			name                  string
			street, city, country string
			wantField             string
		}{
			{"blank street", "  ", "Lisboa", "PT", "street"},
			{"blank city", "Rua da Alegria 12", "", "PT", "city"},
			{"country too short", "Rua da Alegria 12", "Lisboa", "P", "country"},
			{"country too long", "Rua da Alegria 12", "Lisboa", "PRT", "country"},
			{"country not letters", "Rua da Alegria 12", "Lisboa", "P1", "country"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := domain.NewAddress(tt.street, tt.city, tt.country)

				if !errors.Is(err, domain.ErrInvalidAddress) {
					t.Fatalf("NewAddress() error = %v, want %v", err, domain.ErrInvalidAddress)
				}
				if !strings.Contains(err.Error(), tt.wantField) {
					t.Errorf("error %q does not name the field %q", err, tt.wantField)
				}
			})
		}
	})

	t.Run("normalises the country code to upper case", func(t *testing.T) {
		addr, err := domain.NewAddress("Rua da Alegria 12", "Lisboa", "pt")

		if err != nil {
			t.Fatalf("NewAddress() error = %v", err)
		}
		if got, want := addr.Country(), "PT"; got != want {
			t.Errorf("Country() = %q, want %q", got, want)
		}
	})

	t.Run("addresses built from the same input are equal", func(t *testing.T) {
		a, _ := domain.NewAddress("Rua da Alegria 12", "Lisboa", "PT")
		b, _ := domain.NewAddress(" Rua da Alegria 12", "Lisboa ", "pt")

		if a != b {
			t.Errorf("%+v != %+v, want value equality", a, b)
		}
	})
}
