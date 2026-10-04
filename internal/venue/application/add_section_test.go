package application_test

import (
	"context"
	"testing"

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
}
