package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

func TestRegisterVenue(t *testing.T) {
	t.Run("registers a draft venue (VEN-1)", func(t *testing.T) {
		id := domain.NewVenueID(uuid.MustParse("0192f5e0-7c1a-7b3e-9d2a-3f4b5c6d7e8f"))
		addr := mustAddress(t)

		venue, err := domain.RegisterVenue(id, "  Coliseu dos Recreios ", addr)

		if err != nil {
			t.Fatalf("RegisterVenue() error = %v", err)
		}
		if venue.ID() != id {
			t.Errorf("ID() = %v, want %v", venue.ID(), id)
		}
		if got, want := venue.Name(), "Coliseu dos Recreios"; got != want {
			t.Errorf("Name() = %q, want %q", got, want)
		}
		if venue.Address() != addr {
			t.Errorf("Address() = %+v, want %+v", venue.Address(), addr)
		}
		if got, want := venue.Status(), domain.Draft; got != want {
			t.Errorf("Status() = %v, want %v", got, want)
		}
	})

	t.Run("rejects a blank or too long name (VEN-1)", func(t *testing.T) {
		tests := []struct{ name, venueName string }{
			{"blank", "   "},
			{"121 characters", strings.Repeat("x", 121)},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := domain.RegisterVenue(aVenueID(), tt.venueName, mustAddress(t))

				if !errors.Is(err, domain.ErrInvalidVenueName) {
					t.Errorf("RegisterVenue() error = %v, want %v", err, domain.ErrInvalidVenueName)
				}
			})
		}
	})

	t.Run("accepts a name of exactly 120 characters, counted in runes (VEN-1)", func(t *testing.T) {
		_, err := domain.RegisterVenue(aVenueID(), strings.Repeat("é", 120), mustAddress(t))

		if err != nil {
			t.Errorf("RegisterVenue() error = %v", err)
		}
	})

	t.Run("rejects a zero id", func(t *testing.T) {
		_, err := domain.RegisterVenue(domain.VenueID{}, "Coliseu", mustAddress(t))

		if !errors.Is(err, domain.ErrInvalidVenueID) {
			t.Errorf("RegisterVenue() error = %v, want %v", err, domain.ErrInvalidVenueID)
		}
	})

	t.Run("rejects a zero address (VEN-1)", func(t *testing.T) {
		_, err := domain.RegisterVenue(aVenueID(), "Coliseu", domain.Address{})

		if !errors.Is(err, domain.ErrInvalidAddress) {
			t.Errorf("RegisterVenue() error = %v, want %v", err, domain.ErrInvalidAddress)
		}
	})
}

func TestVenue_AddSection(t *testing.T) {
	t.Run("a draft venue accepts a new section", func(t *testing.T) {
		venue := newDraftVenue(t)

		err := venue.AddSection(seatedSection(t, "ORCH", mustRow(t, "A", 20)))

		if err != nil {
			t.Fatalf("AddSection() error = %v", err)
		}
		if diff := cmp.Diff([]string{"ORCH"}, sectionCodes(venue.Sections())); diff != "" {
			t.Errorf("Sections() codes mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("rejects a duplicate section code and stays unchanged (VEN-2)", func(t *testing.T) {
		venue := newDraftVenue(t, withSection(seatedSection(t, "ORCH", mustRow(t, "A", 20))))

		err := venue.AddSection(gaSection(t, "orch", 300))

		if !errors.Is(err, domain.ErrDuplicateSectionCode) {
			t.Errorf("AddSection() error = %v, want %v", err, domain.ErrDuplicateSectionCode)
		}
		if diff := cmp.Diff([]string{"ORCH"}, sectionCodes(venue.Sections())); diff != "" {
			t.Errorf("Sections() codes mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestVenue_AddSection_RejectsAZeroSection(t *testing.T) {
	venue := newDraftVenue(t)

	err := venue.AddSection(domain.Section{})

	if !errors.Is(err, domain.ErrInvalidSection) {
		t.Errorf("AddSection() error = %v, want %v", err, domain.ErrInvalidSection)
	}
}

func TestVenue_Capacity(t *testing.T) {
	venue := newDraftVenue(t,
		withSection(seatedSection(t, "ORCH", mustRow(t, "A", 10), mustRow(t, "B", 12))),
		withSection(gaSection(t, "FLOOR", 500)),
	)

	if got, want := venue.Capacity(), 522; got != want {
		t.Errorf("Capacity() = %d, want %d (VEN-7)", got, want)
	}
}

func TestVenue_Sections_ReturnsACopy(t *testing.T) {
	venue := newDraftVenue(t, withSection(seatedSection(t, "ORCH", mustRow(t, "A", 20))))

	sections := venue.Sections()
	sections[0] = gaSection(t, "HACK", 9999)

	if diff := cmp.Diff([]string{"ORCH"}, sectionCodes(venue.Sections())); diff != "" {
		t.Errorf("mutating Sections() changed the venue (-want +got):\n%s", diff)
	}
}

func TestVenue_Activate(t *testing.T) {
	t.Run("a draft venue with a section becomes active (VEN-5)", func(t *testing.T) {
		venue := newDraftVenue(t, withSection(gaSection(t, "FLOOR", 500)))

		err := venue.Activate()

		if err != nil {
			t.Fatalf("Activate() error = %v", err)
		}
		if got, want := venue.Status(), domain.Active; got != want {
			t.Errorf("Status() = %v, want %v", got, want)
		}
	})

	t.Run("fails without sections and stays draft (VEN-5)", func(t *testing.T) {
		venue := newDraftVenue(t)

		err := venue.Activate()

		if !errors.Is(err, domain.ErrVenueHasNoSections) {
			t.Errorf("Activate() error = %v, want %v", err, domain.ErrVenueHasNoSections)
		}
		if got, want := venue.Status(), domain.Draft; got != want {
			t.Errorf("Status() = %v, want %v", got, want)
		}
	})
}

func TestVenue_Retire(t *testing.T) {
	t.Run("an active venue becomes retired (VEN-6)", func(t *testing.T) {
		venue := newActiveVenue(t)

		err := venue.Retire()

		if err != nil {
			t.Fatalf("Retire() error = %v", err)
		}
		if got, want := venue.Status(), domain.Retired; got != want {
			t.Errorf("Status() = %v, want %v", got, want)
		}
	})
}

func TestVenue_IllegalTransitions(t *testing.T) {
	tests := []struct {
		name       string
		venue      func(t *testing.T) *domain.Venue
		transition func(v *domain.Venue) error
		wantStatus domain.Status
	}{
		{"activate when active (VEN-5)", newActiveVenue, (*domain.Venue).Activate, domain.Active},
		{"activate when retired (VEN-6)", newRetiredVenue, (*domain.Venue).Activate, domain.Retired},
		{"retire when draft (VEN-6)", newDraftVenueWithSection, (*domain.Venue).Retire, domain.Draft},
		{"retire when retired (VEN-6)", newRetiredVenue, (*domain.Venue).Retire, domain.Retired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			venue := tt.venue(t)

			err := tt.transition(venue)

			if !errors.Is(err, domain.ErrInvalidVenueTransition) {
				t.Errorf("error = %v, want %v", err, domain.ErrInvalidVenueTransition)
			}
			if got := venue.Status(); got != tt.wantStatus {
				t.Errorf("Status() = %v, want %v (unchanged)", got, tt.wantStatus)
			}
		})
	}
}
