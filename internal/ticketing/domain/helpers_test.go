package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

var (
	now      = time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)
	startsAt = now.Add(30 * 24 * time.Hour)
	ttl      = 10 * time.Minute
	showID   = domain.NewShowID(uuid.MustParse("0192f5e0-0000-7000-8000-000000000001"))
	ana      = domain.NewCustomerID(uuid.MustParse("0192f5e0-0000-7000-8000-0000000000a1"))
	bob      = domain.NewCustomerID(uuid.MustParse("0192f5e0-0000-7000-8000-0000000000b0"))
)

func eur(t *testing.T, minor int64) sharedkernel.Money {
	t.Helper()
	c, _ := sharedkernel.NewCurrency("EUR")
	m, err := sharedkernel.NewMoney(minor, c)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// layout is the snapshot from show.published.v1: ORCH rows A(2), B(1) at
// EUR 45; FLOOR 3 GA places at EUR 25. Six sellable units.
func layout(t *testing.T) domain.InventoryLayout {
	t.Helper()
	return domain.InventoryLayout{Sections: []domain.InventorySection{
		{Code: "ORCH", Kind: domain.KindSeated, Rows: []domain.InventoryRow{{Label: "A", Seats: 2}, {Label: "B", Seats: 1}}, Price: eur(t, 4500)},
		{Code: "FLOOR", Kind: domain.KindGA, Capacity: 3, Price: eur(t, 2500)},
	}}
}

func openInventory(t *testing.T) *domain.ShowInventory {
	t.Helper()
	inv, err := domain.OpenInventory(showID, layout(t), startsAt, now)
	if err != nil {
		t.Fatal(err)
	}
	inv.PullEvents()
	return inv
}

func refs(t *testing.T, raw ...string) []domain.SeatRef {
	t.Helper()
	out := make([]domain.SeatRef, len(raw))
	for i, r := range raw {
		ref, err := domain.ParseSeatRef(r)
		if err != nil {
			t.Fatal(err)
		}
		out[i] = ref
	}
	return out
}

func newHoldID() domain.HoldID { return domain.NewHoldID(uuid.New()) }

// held holds seats for customer and returns the hold's id.
func held(t *testing.T, inv *domain.ShowInventory, customer domain.CustomerID, seats ...string) domain.HoldID {
	t.Helper()
	id := newHoldID()
	if err := inv.Hold(id, customer, refs(t, seats...), now, ttl); err != nil {
		t.Fatal(err)
	}
	return id
}

// states maps every seat to its state, for compact assertions.
func states(inv *domain.ShowInventory) map[string]string {
	out := map[string]string{}
	for _, s := range inv.Seats() {
		out[s.Ref.String()] = s.State.String()
	}
	return out
}
