package application_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

func TestActivateVenue(t *testing.T) {
	t.Run("activates a draft venue with a section (VEN-5)", func(t *testing.T) {
		f := newFixture(t)
		id := f.draftVenue(t)

		err := f.activate.Handle(f.ctx, application.ActivateVenue{VenueID: id.String()})

		if err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
		if got := f.venue(t, id).Status(); got != domain.Active {
			t.Errorf("Status() = %v, want active", got)
		}
		if _, ok := f.lastEvent(t).(domain.VenueActivated); !ok {
			t.Errorf("last event = %T, want domain.VenueActivated", f.lastEvent(t))
		}
	})

	tests := map[string]struct {
		venueID func(f *fixture, t *testing.T) string
		wantErr error
	}{
		"unknown venue":      {func(*fixture, *testing.T) string { return uuid.NewString() }, application.ErrVenueNotFound},
		"malformed venue id": {func(*fixture, *testing.T) string { return "nope" }, domain.ErrInvalidVenueID},
		"no sections (VEN-5)": {func(f *fixture, t *testing.T) string {
			id, err := f.register.Handle(f.ctx, validRegisterVenue())
			if err != nil {
				t.Fatal(err)
			}
			return id.String()
		}, domain.ErrVenueHasNoSections},
	}
	for name, tt := range tests {
		t.Run(name+" saves nothing", func(t *testing.T) {
			f := newFixture(t)
			venueID := tt.venueID(f, t)
			before := len(f.repo.Published())

			err := f.activate.Handle(f.ctx, application.ActivateVenue{VenueID: venueID})

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Handle() error = %v, want %v", err, tt.wantErr)
			}
			if got := len(f.repo.Published()); got != before {
				t.Errorf("published %d new events, want none", got-before)
			}
		})
	}
}

func TestRetireVenue(t *testing.T) {
	t.Run("retires an active venue (VEN-6)", func(t *testing.T) {
		f := newFixture(t)
		id := f.activeVenue(t)

		err := f.retire.Handle(f.ctx, application.RetireVenue{VenueID: id.String()})

		if err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
		if got := f.venue(t, id).Status(); got != domain.Retired {
			t.Errorf("Status() = %v, want retired", got)
		}
		if _, ok := f.lastEvent(t).(domain.VenueRetired); !ok {
			t.Errorf("last event = %T, want domain.VenueRetired", f.lastEvent(t))
		}
	})

	tests := map[string]struct {
		venueID func(f *fixture, t *testing.T) string
		wantErr error
	}{
		"unknown venue":      {func(*fixture, *testing.T) string { return uuid.NewString() }, application.ErrVenueNotFound},
		"malformed venue id": {func(*fixture, *testing.T) string { return "nope" }, domain.ErrInvalidVenueID},
		"draft venue (VEN-6)": {func(f *fixture, t *testing.T) string {
			return f.draftVenue(t).String()
		}, domain.ErrInvalidVenueTransition},
	}
	for name, tt := range tests {
		t.Run(name+" saves nothing", func(t *testing.T) {
			f := newFixture(t)
			venueID := tt.venueID(f, t)
			before := len(f.repo.Published())

			err := f.retire.Handle(f.ctx, application.RetireVenue{VenueID: venueID})

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Handle() error = %v, want %v", err, tt.wantErr)
			}
			if got := len(f.repo.Published()); got != before {
				t.Errorf("published %d new events, want none", got-before)
			}
		})
	}
}
