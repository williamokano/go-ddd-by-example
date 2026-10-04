package domain_test

import (
	"errors"
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

func TestNewSeatedSection(t *testing.T) {
	t.Run("capacity is the number of seats in its rows (VEN-7)", func(t *testing.T) {
		rows := []domain.Row{mustRow(t, "A", 10), mustRow(t, "B", 12)}

		section, err := domain.NewSeatedSection(mustCode(t, "ORCH"), "Orchestra", rows)

		if err != nil {
			t.Fatalf("NewSeatedSection() error = %v", err)
		}
		if got, want := section.Capacity(), 22; got != want {
			t.Errorf("Capacity() = %d, want %d", got, want)
		}
		if got, want := section.Kind(), domain.Seated; got != want {
			t.Errorf("Kind() = %v, want %v", got, want)
		}
	})

	t.Run("needs at least one row (VEN-3)", func(t *testing.T) {
		_, err := domain.NewSeatedSection(mustCode(t, "ORCH"), "Orchestra", nil)

		if !errors.Is(err, domain.ErrInvalidSection) {
			t.Errorf("NewSeatedSection() error = %v, want %v", err, domain.ErrInvalidSection)
		}
	})
}
