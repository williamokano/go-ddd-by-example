// Package seatquerytest is the contract of application.SeatQueries.
package seatquerytest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application/inventoryrepotest"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// Run runs the contract against queries reading what repo stored.
func Run(t *testing.T, newQueries func(t *testing.T) (application.InventoryRepository, application.SeatQueries)) {
	t.Helper()
	ctx := context.Background()

	t.Run("lists every seat with its state and price, in layout order", func(t *testing.T) {
		repo, queries := newQueries(t)
		inv, floor := inventoryrepotest.Open(t)
		ref, _ := domain.ParseSeatRef("ORCH/A/2")
		at := time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)
		if err := inv.Hold(domain.NewHoldID(uuid.New()), domain.NewCustomerID(uuid.New()), []domain.SeatRef{ref}, at, time.Minute); err != nil {
			t.Fatal(err)
		}
		// Saved FLOOR first: the listing follows the layout, not the saves.
		for _, s := range []*domain.SectionInventory{floor, inv} {
			if err := repo.Save(ctx, s); err != nil {
				t.Fatal(err)
			}
		}

		rows, err := queries.ListSeats(ctx, inv.ShowID())

		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 5 || rows[0] != (application.SeatRow{Ref: "ORCH/A/1", State: "available", Amount: 4500, Currency: "EUR"}) ||
			rows[1].State != "held" || rows[4].Ref != "FLOOR/GA/0003" {
			t.Errorf("rows = %+v", rows)
		}
	})

	t.Run("an unknown show is ErrInventoryNotFound", func(t *testing.T) {
		_, queries := newQueries(t)

		if _, err := queries.ListSeats(ctx, domain.NewShowID(uuid.New())); !errors.Is(err, application.ErrInventoryNotFound) {
			t.Errorf("error = %v, want %v", err, application.ErrInventoryNotFound)
		}
	})
}
