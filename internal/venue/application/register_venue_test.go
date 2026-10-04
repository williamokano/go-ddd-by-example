package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
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

	t.Run("records VenueRegistered at the clock's time", func(t *testing.T) {
		repo := memory.NewVenueRepository()
		handler := application.NewRegisterVenueHandler(repo, ids.NewVenueIDs(idgen.NewSequence()), clock.NewFixed(fixedNow))

		id, err := handler.Handle(context.Background(), validRegisterVenue())

		if err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
		want := []domain.DomainEvent{domain.VenueRegistered{VenueID: id, Name: "Coliseu dos Recreios", At: fixedNow}}
		if diff := cmp.Diff(want, repo.Published(), cmp.AllowUnexported(domain.VenueID{})); diff != "" {
			t.Errorf("events mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("returns the domain error unchanged and saves nothing (VEN-1)", func(t *testing.T) {
		repo := memory.NewVenueRepository()
		handler := application.NewRegisterVenueHandler(repo, ids.NewVenueIDs(idgen.NewSequence()), clock.NewFixed(fixedNow))
		cmd := validRegisterVenue()
		cmd.Country = "Portugal"

		_, err := handler.Handle(context.Background(), cmd)

		if !errors.Is(err, domain.ErrInvalidAddress) {
			t.Errorf("Handle() error = %v, want %v", err, domain.ErrInvalidAddress)
		}
		if got := repo.Published(); len(got) != 0 {
			t.Errorf("saved %v, want nothing", got)
		}
	})
}

func validRegisterVenue() application.RegisterVenue {
	return application.RegisterVenue{
		Name: "Coliseu dos Recreios", Street: "Rua Portas de Santo Antão 96", City: "Lisboa", Country: "PT",
	}
}
