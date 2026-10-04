package domain_test

import (
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
}
