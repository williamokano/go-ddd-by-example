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
		ctx, repo, id, addSection := setupAddSection(t)
		if err := addSection.Handle(ctx, application.AddSection{VenueID: id.String(), Code: "FLOOR", Kind: "ga", Capacity: 100}); err != nil {
			t.Fatal(err)
		}
		handler := application.NewActivateVenueHandler(repo, clockAt(fixedNow))

		err := handler.Handle(ctx, application.ActivateVenue{VenueID: id.String()})

		if err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
		venue, _ := repo.Get(ctx, id)
		if venue.Status() != domain.Active {
			t.Errorf("Status() = %v, want active", venue.Status())
		}
		events := repo.Published()
		if _, ok := events[len(events)-1].(domain.VenueActivated); !ok {
			t.Errorf("last event = %T, want domain.VenueActivated", events[len(events)-1])
		}
	})

	for name, tt := range map[string]struct {
		venueID func(domain.VenueID) string
		wantErr error
	}{
		"unknown venue":           {func(domain.VenueID) string { return uuid.NewString() }, application.ErrVenueNotFound},
		"malformed venue id":      {func(domain.VenueID) string { return "nope" }, domain.ErrInvalidVenueID},
		"no sections, VEN-5 rule": {func(id domain.VenueID) string { return id.String() }, domain.ErrVenueHasNoSections},
	} {
		t.Run(name+" saves nothing", func(t *testing.T) {
			ctx, repo, id, _ := setupAddSection(t)
			before := len(repo.Published())

			err := application.NewActivateVenueHandler(repo, clockAt(fixedNow)).
				Handle(ctx, application.ActivateVenue{VenueID: tt.venueID(id)})

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Handle() error = %v, want %v", err, tt.wantErr)
			}
			if got := len(repo.Published()); got != before {
				t.Errorf("published %d new events, want none", got-before)
			}
		})
	}
}

func TestRetireVenue(t *testing.T) {
	t.Run("retires an active venue (VEN-6)", func(t *testing.T) {
		ctx, repo, id, addSection := setupAddSection(t)
		if err := addSection.Handle(ctx, application.AddSection{VenueID: id.String(), Code: "FLOOR", Kind: "ga", Capacity: 100}); err != nil {
			t.Fatal(err)
		}
		if err := application.NewActivateVenueHandler(repo, clockAt(fixedNow)).Handle(ctx, application.ActivateVenue{VenueID: id.String()}); err != nil {
			t.Fatal(err)
		}
		handler := application.NewRetireVenueHandler(repo, clockAt(fixedNow))

		err := handler.Handle(ctx, application.RetireVenue{VenueID: id.String()})

		if err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
		venue, _ := repo.Get(ctx, id)
		if venue.Status() != domain.Retired {
			t.Errorf("Status() = %v, want retired", venue.Status())
		}
		events := repo.Published()
		if _, ok := events[len(events)-1].(domain.VenueRetired); !ok {
			t.Errorf("last event = %T, want domain.VenueRetired", events[len(events)-1])
		}
	})

	for name, tt := range map[string]struct {
		venueID func(domain.VenueID) string
		wantErr error
	}{
		"unknown venue":           {func(domain.VenueID) string { return uuid.NewString() }, application.ErrVenueNotFound},
		"malformed venue id":      {func(domain.VenueID) string { return "nope" }, domain.ErrInvalidVenueID},
		"draft venue, VEN-6 rule": {func(id domain.VenueID) string { return id.String() }, domain.ErrInvalidVenueTransition},
	} {
		t.Run(name+" saves nothing", func(t *testing.T) {
			ctx, repo, id, _ := setupAddSection(t)
			before := len(repo.Published())

			err := application.NewRetireVenueHandler(repo, clockAt(fixedNow)).
				Handle(ctx, application.RetireVenue{VenueID: tt.venueID(id)})

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Handle() error = %v, want %v", err, tt.wantErr)
			}
			if got := len(repo.Published()); got != before {
				t.Errorf("published %d new events, want none", got-before)
			}
		})
	}
}
