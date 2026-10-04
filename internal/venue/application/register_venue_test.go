package application_test

import (
	"context"
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

var fixedNow = time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)

func TestRegisterVenue(t *testing.T) {
	t.Run("registers a draft venue under a new id", func(t *testing.T) {
		repo := memory.NewVenueRepository()
		handler := application.NewRegisterVenueHandler(repo, ids.NewVenueIDs(idgen.NewSequence()), clock.NewFixed(fixedNow))

		id, err := handler.Handle(context.Background(), application.RegisterVenue{
			Name: "Coliseu dos Recreios", Street: "Rua Portas de Santo Antão 96", City: "Lisboa", Country: "PT",
		})

		if err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
		want := domain.NewVenueID(uuid.MustParse("00000000-0000-7000-8000-000000000001"))
		if id != want {
			t.Errorf("id = %v, want %v", id, want)
		}
		venue, err := repo.Get(context.Background(), id)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if venue.Status() != domain.Draft {
			t.Errorf("Status() = %v, want draft", venue.Status())
		}
	})
}
