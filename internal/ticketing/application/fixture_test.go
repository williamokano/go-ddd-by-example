package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/platform/clock"
	"github.com/williamokano/go-ddd-by-example/internal/platform/idgen"
	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/ids"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/memory"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

var fixedNow = time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)

const holdTTL = 10 * time.Minute

type fixture struct {
	ctx         context.Context
	inventories *memory.InventoryRepository
	clock       *clock.Fixed
	ids         ids.TicketingIDs

	open    *application.OpenInventoryHandler
	hold    *application.HoldSeatsHandler
	release *application.ReleaseHoldHandler
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	inventories, clk, gen := memory.NewInventoryRepository(), clock.NewFixed(fixedNow), ids.New(idgen.NewSequence())
	return &fixture{
		ctx: context.Background(), inventories: inventories, clock: clk, ids: gen,
		open:    application.NewOpenInventoryHandler(inventories, clk),
		hold:    application.NewHoldSeatsHandler(inventories, gen, clk, holdTTL),
		release: application.NewReleaseHoldHandler(inventories, clk),
	}
}

// openCmd is what the consumer builds from show.published.v1: ORCH A(2) at
// EUR 45, FLOOR 3 GA places at EUR 25, starting in a month.
func openCmd(showID string) application.OpenInventory {
	return application.OpenInventory{
		ShowID: showID, StartsAt: fixedNow.Add(30 * 24 * time.Hour), Country: "PT",
		Sections: []application.SectionSpec{
			{Code: "ORCH", Kind: "seated", Rows: []application.RowSpec{{Label: "A", Seats: 2, Accessible: []int{2}}}, Price: 4500, Currency: "EUR"},
			{Code: "FLOOR", Kind: "ga", Capacity: 3, Price: 2500, Currency: "EUR"},
		},
	}
}

// openShow opens a fresh show's inventory and returns its id.
func (f *fixture) openShow(t *testing.T) string {
	t.Helper()
	id := uuid.NewString()
	if err := f.open.Handle(f.ctx, openCmd(id)); err != nil {
		t.Fatal(err)
	}
	return id
}

func (f *fixture) holdSeats(t *testing.T, showID, customer string, seats ...string) domain.HoldID {
	t.Helper()
	res, err := f.hold.Handle(f.ctx, application.HoldSeats{ShowID: showID, CustomerID: customer, Seats: seats})
	if err != nil {
		t.Fatal(err)
	}
	return res.HoldID
}

func (f *fixture) seatStates(t *testing.T, showID string) map[string]string {
	t.Helper()
	id, _ := domain.ParseShowID(showID)
	sections, err := f.inventories.ListByShow(f.ctx, id)
	if err != nil || len(sections) == 0 {
		t.Fatalf("ListByShow() = %d sections, %v", len(sections), err)
	}
	out := map[string]string{}
	for _, inv := range sections {
		for _, s := range inv.Seats() {
			out[s.Ref.String()] = s.State.String()
		}
	}
	return out
}

func published[T sharedkernel.DomainEvent](events []sharedkernel.DomainEvent) bool {
	return count[T](events) > 0
}

func count[T sharedkernel.DomainEvent](events []sharedkernel.DomainEvent) int {
	n := 0
	for _, ev := range events {
		if _, ok := ev.(T); ok {
			n++
		}
	}
	return n
}
