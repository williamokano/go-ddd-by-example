package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

func eventNames(inv *domain.ShowInventory) []string {
	var out []string
	for _, ev := range inv.PullEvents() {
		out = append(out, ev.EventName())
	}
	return out
}

func newOrderID() domain.OrderID { return domain.NewOrderID(uuid.New()) }

func TestShowInventory_ReleaseHold(t *testing.T) {
	t.Run("the owner frees the seats", func(t *testing.T) {
		inv := openInventory(t)
		id := held(t, inv, ana, "ORCH/A/1")
		inv.PullEvents()

		err := inv.ReleaseHold(id, ana, now)

		if err != nil || states(inv)["ORCH/A/1"] != "available" {
			t.Errorf("error = %v, state %s", err, states(inv)["ORCH/A/1"])
		}
		if names := eventNames(inv); len(names) != 1 || names[0] != "ticketing.HoldReleased" {
			t.Errorf("events = %v", names)
		}
	})

	t.Run("someone else's hold cannot be released", func(t *testing.T) {
		inv := openInventory(t)
		id := held(t, inv, ana, "ORCH/A/1")

		if err := inv.ReleaseHold(id, bob, now); !errors.Is(err, domain.ErrNotHoldOwner) {
			t.Errorf("error = %v, want %v", err, domain.ErrNotHoldOwner)
		}
	})

	t.Run("an unknown hold is ErrHoldNotFound", func(t *testing.T) {
		if err := openInventory(t).ReleaseHold(newHoldID(), ana, now); !errors.Is(err, domain.ErrHoldNotFound) {
			t.Errorf("error = %v, want %v", err, domain.ErrHoldNotFound)
		}
	})
}

func TestShowInventory_ExpireHolds(t *testing.T) {
	inv := openInventory(t)
	held(t, inv, ana, "ORCH/A/1")
	if err := inv.Hold(newHoldID(), bob, refs(t, "ORCH/A/2"), now.Add(5*time.Minute), ttl); err != nil {
		t.Fatal(err)
	}
	inv.PullEvents()

	inv.ExpireHolds(now.Add(ttl)) // Ana's hold lapses exactly now; Bob's in 5 minutes

	if s := states(inv); s["ORCH/A/1"] != "available" || s["ORCH/A/2"] != "held" {
		t.Errorf("states = %v (TKT-4)", s)
	}
	if names := eventNames(inv); len(names) != 1 || names[0] != "ticketing.HoldExpired" {
		t.Errorf("events = %v, want one HoldExpired", names)
	}
}

func TestShowInventory_ConfirmHold(t *testing.T) {
	t.Run("sells the held seats and records SeatsSold (TKT-8)", func(t *testing.T) {
		inv := openInventory(t)
		id := held(t, inv, ana, "ORCH/A/1", "ORCH/A/2")
		inv.PullEvents()
		order := newOrderID()

		err := inv.ConfirmHold(id, order, now.Add(time.Minute))

		if err != nil {
			t.Fatal(err)
		}
		for _, s := range inv.Seats()[:2] {
			if s.State != domain.Sold || s.OrderID != order {
				t.Errorf("%s = %v for %v", s.Ref, s.State, s.OrderID)
			}
		}
		if names := eventNames(inv); len(names) != 1 || names[0] != "ticketing.SeatsSold" {
			t.Errorf("events = %v", names)
		}
	})

	t.Run("an expired hold, even unswept, cannot be confirmed (TKT-8)", func(t *testing.T) {
		inv := openInventory(t)
		id := held(t, inv, ana, "ORCH/A/1")
		inv.PullEvents()

		err := inv.ConfirmHold(id, newOrderID(), now.Add(ttl))

		if !errors.Is(err, domain.ErrHoldExpired) {
			t.Errorf("error = %v, want %v", err, domain.ErrHoldExpired)
		}
		if states(inv)["ORCH/A/1"] == "sold" || len(inv.PullEvents()) != 0 {
			t.Error("a failed confirmation sold the seat or recorded events")
		}
	})

	t.Run("a released hold cannot be confirmed", func(t *testing.T) {
		inv := openInventory(t)
		id := held(t, inv, ana, "ORCH/A/1")
		_ = inv.ReleaseHold(id, ana, now)

		if err := inv.ConfirmHold(id, newOrderID(), now); !errors.Is(err, domain.ErrHoldNotFound) {
			t.Errorf("error = %v, want %v", err, domain.ErrHoldNotFound)
		}
	})

	t.Run("confirming the same order twice is a no-op (redelivery)", func(t *testing.T) {
		inv := openInventory(t)
		id := held(t, inv, ana, "ORCH/A/1")
		order := newOrderID()
		_ = inv.ConfirmHold(id, order, now)
		inv.PullEvents()

		err := inv.ConfirmHold(id, order, now)

		if err != nil || len(inv.PullEvents()) != 0 {
			t.Errorf("second confirm: error = %v; want a silent no-op", err)
		}
	})
}

func TestShowInventory_SoldOutExactlyOnce(t *testing.T) {
	inv := openInventory(t)
	first := held(t, inv, ana, "ORCH/A/1", "ORCH/A/2", "ORCH/B/1")
	second := held(t, inv, bob, "FLOOR/GA/0001", "FLOOR/GA/0002", "FLOOR/GA/0003")
	order1, order2 := newOrderID(), newOrderID()
	_ = inv.ConfirmHold(first, order1, now)
	inv.PullEvents()

	_ = inv.ConfirmHold(second, order2, now)
	_ = inv.ConfirmHold(second, order2, now) // redelivered

	names := eventNames(inv)
	if len(names) != 2 || names[0] != "ticketing.SeatsSold" || names[1] != "ticketing.InventorySoldOut" {
		t.Errorf("events = %v, want SeatsSold then InventorySoldOut once (TKT-10)", names)
	}
}

func TestShowInventory_Close(t *testing.T) {
	inv := openInventory(t)
	held(t, inv, ana, "ORCH/A/1")
	inv.PullEvents()

	inv.Close(now)
	inv.Close(now) // idempotent

	if states(inv)["ORCH/A/1"] != "available" || len(inv.Holds()) != 0 || !inv.IsClosed() {
		t.Errorf("after Close: states %v, %d holds", states(inv), len(inv.Holds()))
	}
	if names := eventNames(inv); len(names) != 1 || names[0] != "ticketing.InventoryClosed" {
		t.Errorf("events = %v, want InventoryClosed once", names)
	}
	if err := inv.Hold(newHoldID(), bob, refs(t, "ORCH/A/2"), now, ttl); !errors.Is(err, domain.ErrSalesClosed) {
		t.Errorf("Hold after Close: error = %v, want %v (TKT-11, TKT-12)", err, domain.ErrSalesClosed)
	}
}
