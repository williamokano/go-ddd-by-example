package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/platform/clock"
	"github.com/williamokano/go-ddd-by-example/internal/platform/idgen"
	"github.com/williamokano/go-ddd-by-example/internal/venue/adapters/driven/ids"
	"github.com/williamokano/go-ddd-by-example/internal/venue/adapters/driven/memory"
	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

func TestAddSection(t *testing.T) {
	t.Run("adds a seated section to a draft venue", func(t *testing.T) {
		ctx := context.Background()
		repo := memory.NewVenueRepository()
		clk := clock.NewFixed(fixedNow)
		id, err := application.NewRegisterVenueHandler(repo, ids.NewVenueIDs(idgen.NewSequence()), clk).
			Handle(ctx, validRegisterVenue())
		if err != nil {
			t.Fatal(err)
		}
		handler := application.NewAddSectionHandler(repo, clk)

		err = handler.Handle(ctx, application.AddSection{
			VenueID: id.String(), Code: "orch", Name: "Orchestra", Kind: "seated",
			Rows: []application.RowSpec{{Label: "A", Seats: 10}, {Label: "B", Seats: 12}},
		})

		if err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
		venue, _ := repo.Get(ctx, id)
		if got, want := venue.Capacity(), 22; got != want {
			t.Errorf("Capacity() = %d, want %d", got, want)
		}
		events := repo.Published()
		if _, ok := events[len(events)-1].(domain.SectionAdded); !ok {
			t.Errorf("last event = %T, want domain.SectionAdded", events[len(events)-1])
		}
	})

	t.Run("adds a general admission section", func(t *testing.T) {
		ctx, repo, id, handler := setupAddSection(t)

		err := handler.Handle(ctx, application.AddSection{VenueID: id.String(), Code: "FLOOR", Name: "Floor", Kind: "ga", Capacity: 500})

		if err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
		venue, _ := repo.Get(ctx, id)
		if got, want := venue.Capacity(), 500; got != want {
			t.Errorf("Capacity() = %d, want %d", got, want)
		}
	})

	tests := []struct {
		name    string
		cmd     func(id domain.VenueID) application.AddSection
		wantErr error
	}{
		{"unknown venue", func(domain.VenueID) application.AddSection {
			return application.AddSection{VenueID: uuid.NewString(), Code: "FLOOR", Kind: "ga", Capacity: 10}
		}, application.ErrVenueNotFound},
		{"malformed venue id", func(domain.VenueID) application.AddSection {
			return application.AddSection{VenueID: "nope", Code: "FLOOR", Kind: "ga", Capacity: 10}
		}, domain.ErrInvalidVenueID},
		{"invalid section code (VEN-2)", func(id domain.VenueID) application.AddSection {
			return application.AddSection{VenueID: id.String(), Code: "A B", Kind: "ga", Capacity: 10}
		}, domain.ErrInvalidSectionCode},
		{"invalid row (VEN-3)", func(id domain.VenueID) application.AddSection {
			return application.AddSection{VenueID: id.String(), Code: "ORCH", Kind: "seated", Rows: []application.RowSpec{{Label: "A", Seats: 0}}}
		}, domain.ErrInvalidRow},
		{"unknown kind", func(id domain.VenueID) application.AddSection {
			return application.AddSection{VenueID: id.String(), Code: "BOX", Kind: "box", Capacity: 4}
		}, domain.ErrInvalidSection},
		{"duplicate code (VEN-2)", func(id domain.VenueID) application.AddSection {
			return application.AddSection{VenueID: id.String(), Code: "orch", Kind: "ga", Capacity: 10}
		}, domain.ErrDuplicateSectionCode},
	}
	for _, tt := range tests {
		t.Run(tt.name+" saves nothing", func(t *testing.T) {
			ctx, repo, id, handler := setupAddSection(t)
			if err := handler.Handle(ctx, application.AddSection{VenueID: id.String(), Code: "ORCH", Kind: "ga", Capacity: 100}); err != nil {
				t.Fatal(err)
			}
			before := len(repo.Published())

			err := handler.Handle(ctx, tt.cmd(id))

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Handle() error = %v, want %v", err, tt.wantErr)
			}
			if got := len(repo.Published()); got != before {
				t.Errorf("published %d new events, want none", got-before)
			}
		})
	}
}

func setupAddSection(t *testing.T) (context.Context, *memory.VenueRepository, domain.VenueID, *application.AddSectionHandler) {
	t.Helper()
	ctx := context.Background()
	repo := memory.NewVenueRepository()
	clk := clock.NewFixed(fixedNow)
	id, err := application.NewRegisterVenueHandler(repo, ids.NewVenueIDs(idgen.NewSequence()), clk).Handle(ctx, validRegisterVenue())
	if err != nil {
		t.Fatal(err)
	}
	return ctx, repo, id, application.NewAddSectionHandler(repo, clk)
}

func clockAt(t time.Time) *clock.Fixed { return clock.NewFixed(t) }
