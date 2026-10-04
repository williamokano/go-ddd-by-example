package domain_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

func mustCode(t *testing.T, raw string) domain.SectionCode {
	t.Helper()
	code, err := domain.NewSectionCode(raw)
	if err != nil {
		t.Fatalf("NewSectionCode(%q) error = %v", raw, err)
	}
	return code
}

func mustRow(t *testing.T, label string, seats int) domain.Row {
	t.Helper()
	row, err := domain.NewRow(label, seats)
	if err != nil {
		t.Fatalf("NewRow(%q, %d) error = %v", label, seats, err)
	}
	return row
}

func mustAddress(t *testing.T) domain.Address {
	t.Helper()
	addr, err := domain.NewAddress("Rua Portas de Santo Antão 96", "Lisboa", "PT")
	if err != nil {
		t.Fatalf("NewAddress() error = %v", err)
	}
	return addr
}

func aVenueID() domain.VenueID {
	return domain.NewVenueID(uuid.MustParse("0192f5e0-7c1a-7b3e-9d2a-3f4b5c6d7e8f"))
}
