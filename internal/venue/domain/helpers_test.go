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

// venueOption customises the venue built by newDraftVenue.
type venueOption func(*venueSpec)

type venueSpec struct {
	id       domain.VenueID
	name     string
	sections []domain.Section
}

func withSection(s domain.Section) venueOption {
	return func(spec *venueSpec) { spec.sections = append(spec.sections, s) }
}

// newDraftVenue registers a draft venue, hiding every detail the test doesn't
// care about. Options add only what matters to the test.
func newDraftVenue(t *testing.T, opts ...venueOption) *domain.Venue {
	t.Helper()
	spec := venueSpec{id: aVenueID(), name: "Coliseu dos Recreios"}
	for _, opt := range opts {
		opt(&spec)
	}
	venue, err := domain.RegisterVenue(spec.id, spec.name, mustAddress(t))
	if err != nil {
		t.Fatalf("RegisterVenue() error = %v", err)
	}
	for _, s := range spec.sections {
		if err := venue.AddSection(s); err != nil {
			t.Fatalf("AddSection(%s) error = %v", s.Code(), err)
		}
	}
	return venue
}

func seatedSection(t *testing.T, code string, rows ...domain.Row) domain.Section {
	t.Helper()
	s, err := domain.NewSeatedSection(mustCode(t, code), code, rows)
	if err != nil {
		t.Fatalf("NewSeatedSection(%s) error = %v", code, err)
	}
	return s
}

func gaSection(t *testing.T, code string, capacity int) domain.Section {
	t.Helper()
	s, err := domain.NewGeneralAdmissionSection(mustCode(t, code), code, capacity)
	if err != nil {
		t.Fatalf("NewGeneralAdmissionSection(%s) error = %v", code, err)
	}
	return s
}

// sectionCodes lists the codes of the given sections, in order.
func sectionCodes(sections []domain.Section) []string {
	codes := make([]string, len(sections))
	for i, s := range sections {
		codes[i] = s.Code().String()
	}
	return codes
}
