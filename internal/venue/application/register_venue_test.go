package application_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

func TestRegisterVenue(t *testing.T) {
	t.Run("registers a draft venue under a new id", func(t *testing.T) {
		f := newFixture(t)

		id, err := f.register.Handle(f.ctx, validRegisterVenue())

		if err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
		if want := domain.NewVenueID(uuid.MustParse("00000000-0000-7000-8000-000000000001")); id != want {
			t.Errorf("id = %v, want %v", id, want)
		}
		if got := f.venue(t, id).Status(); got != domain.Draft {
			t.Errorf("Status() = %v, want draft", got)
		}
	})

	t.Run("records VenueRegistered at the clock's time", func(t *testing.T) {
		f := newFixture(t)

		id, err := f.register.Handle(f.ctx, validRegisterVenue())

		if err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
		want := []domain.DomainEvent{domain.VenueRegistered{VenueID: id, Name: "Coliseu dos Recreios", At: fixedNow}}
		if diff := cmp.Diff(want, f.repo.Published(), cmp.AllowUnexported(domain.VenueID{})); diff != "" {
			t.Errorf("events mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("returns the domain error unchanged and saves nothing (VEN-1)", func(t *testing.T) {
		f := newFixture(t)
		cmd := validRegisterVenue()
		cmd.Country = "Portugal"

		_, err := f.register.Handle(f.ctx, cmd)

		if !errors.Is(err, domain.ErrInvalidAddress) {
			t.Errorf("Handle() error = %v, want %v", err, domain.ErrInvalidAddress)
		}
		if got := f.repo.Published(); len(got) != 0 {
			t.Errorf("saved %v, want nothing", got)
		}
	})
}
