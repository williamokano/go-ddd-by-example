package domain_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

func TestRehydrateInventory_RoundTripsThroughItsState(t *testing.T) {
	inv := openInventory(t)
	held(t, inv, ana, "ORCH/A/1")
	sold := held(t, inv, bob, "FLOOR/GA/0001")
	_ = inv.ConfirmHold(sold, newOrderID(), now)
	state := domain.StateOf(inv)
	state.Version = 4

	again := domain.RehydrateInventory(state)

	opts := cmp.AllowUnexported(domain.ShowID{}, domain.HoldID{}, domain.OrderID{}, domain.CustomerID{}, domain.SeatRef{}, domain.Hold{})
	if diff := cmp.Diff(state, domain.StateOf(again), opts, cmp.Comparer(func(a, b interface{ String() string }) bool { return a.String() == b.String() })); diff != "" {
		t.Errorf("state mismatch (-want +got):\n%s", diff)
	}
	if again.Version() != 4 || len(again.PullEvents()) != 0 {
		t.Errorf("version %d, events after rehydrate", again.Version())
	}
}

func TestInventoryClosed_ListsTheReleasedSeats(t *testing.T) {
	inv := openInventory(t)
	held(t, inv, ana, "ORCH/A/1", "ORCH/A/2")
	inv.PullEvents()

	inv.Close(now)

	closed := inv.PullEvents()[0].(domain.InventoryClosed)
	if len(closed.ReleasedSeats) != 2 {
		t.Errorf("ReleasedSeats = %v, want the 2 held seats", closed.ReleasedSeats)
	}
}
