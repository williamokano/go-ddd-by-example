package application_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

func TestAddSection(t *testing.T) {
	t.Run("adds a seated section to a draft venue", func(t *testing.T) {
		f := newFixture(t)
		id := f.draftVenue(t) // FLOOR, 100

		err := f.add.Handle(f.ctx, application.AddSection{
			VenueID: id.String(), Code: "orch", Name: "Orchestra", Kind: application.KindSeated,
			Rows: []application.RowSpec{{Label: "A", Seats: 10, Accessible: []int{1, 2}}, {Label: "B", Seats: 12}},
		})

		if err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
		if ev, ok := f.lastEvent(t).(domain.SectionAdded); !ok || len(ev.Section.Rows()[0].AccessibleSeats()) != 2 {
			t.Errorf("row A's accessible seats were lost: %+v", f.lastEvent(t))
		}
		if got, want := f.venue(t, id).Capacity(), 122; got != want {
			t.Errorf("Capacity() = %d, want %d", got, want)
		}
		if ev, ok := f.lastEvent(t).(domain.SectionAdded); !ok || ev.Section.Code().String() != "ORCH" {
			t.Errorf("last event = %+v, want SectionAdded for ORCH", f.lastEvent(t))
		}
	})

	t.Run("adds a general admission section", func(t *testing.T) {
		f := newFixture(t)
		id := f.draftVenue(t)

		err := f.add.Handle(f.ctx, application.AddSection{VenueID: id.String(), Code: "BALCONY", Kind: application.KindGA, Capacity: 50})

		if err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
		if got, want := f.venue(t, id).Capacity(), 150; got != want {
			t.Errorf("Capacity() = %d, want %d", got, want)
		}
	})

	tests := []struct {
		name    string
		cmd     func(id domain.VenueID) application.AddSection
		wantErr error
	}{
		{"unknown venue", func(domain.VenueID) application.AddSection {
			return application.AddSection{VenueID: uuid.NewString(), Code: "BOX", Kind: application.KindGA, Capacity: 10}
		}, application.ErrVenueNotFound},
		{"malformed venue id", func(domain.VenueID) application.AddSection {
			return application.AddSection{VenueID: "nope", Code: "BOX", Kind: application.KindGA, Capacity: 10}
		}, domain.ErrInvalidVenueID},
		{"invalid section code (VEN-2)", func(id domain.VenueID) application.AddSection {
			return application.AddSection{VenueID: id.String(), Code: "A B", Kind: application.KindGA, Capacity: 10}
		}, domain.ErrInvalidSectionCode},
		{"invalid row (VEN-3)", func(id domain.VenueID) application.AddSection {
			return application.AddSection{VenueID: id.String(), Code: "ORCH", Kind: application.KindSeated, Rows: []application.RowSpec{{Label: "A", Seats: 0}}}
		}, domain.ErrInvalidRow},
		{"unknown kind", func(id domain.VenueID) application.AddSection {
			return application.AddSection{VenueID: id.String(), Code: "BOX", Kind: "box", Capacity: 4}
		}, domain.ErrInvalidSection},
		{"duplicate code (VEN-2)", func(id domain.VenueID) application.AddSection {
			return application.AddSection{VenueID: id.String(), Code: "floor", Kind: application.KindGA, Capacity: 10}
		}, domain.ErrDuplicateSectionCode},
		{"active venue (VEN-4)", nil, domain.ErrVenueNotDraft},
	}
	for _, tt := range tests {
		t.Run(tt.name+" saves nothing", func(t *testing.T) {
			f := newFixture(t)
			var id domain.VenueID
			var cmd application.AddSection
			if tt.cmd == nil {
				id = f.activeVenue(t)
				cmd = application.AddSection{VenueID: id.String(), Code: "BOX", Kind: application.KindGA, Capacity: 4}
			} else {
				id = f.draftVenue(t)
				cmd = tt.cmd(id)
			}
			before := len(f.repo.Published())

			err := f.add.Handle(f.ctx, cmd)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Handle() error = %v, want %v", err, tt.wantErr)
			}
			if got := len(f.repo.Published()); got != before {
				t.Errorf("published %d new events, want none", got-before)
			}
		})
	}
}
