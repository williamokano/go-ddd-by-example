// Package inventoryrepotest is the contract of application.InventoryRepository.
package inventoryrepotest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

var now = time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)

// Run runs the contract.
func Run(t *testing.T, newRepo func(t *testing.T) application.InventoryRepository) {
	t.Helper()
	ctx := context.Background()

	t.Run("get of an unknown show is ErrInventoryNotFound", func(t *testing.T) {
		_, err := newRepo(t).Get(ctx, domain.NewShowID(uuid.New()))

		if !errors.Is(err, application.ErrInventoryNotFound) {
			t.Errorf("error = %v, want %v", err, application.ErrInventoryNotFound)
		}
	})

	t.Run("an opened inventory round-trips", func(t *testing.T) {
		repo := newRepo(t)
		inv := Open(t)

		save(t, repo, inv)

		assertSame(t, inv, get(t, repo, inv.ShowID()))
	})

	t.Run("holds, sales and closing round-trip", func(t *testing.T) {
		repo := newRepo(t)
		inv := Open(t)
		save(t, repo, inv)
		loaded := get(t, repo, inv.ShowID())
		hold(t, loaded, "ORCH/A/1")
		sold := hold(t, loaded, "FLOOR/GA/0002")
		if err := loaded.ConfirmHold(sold, domain.NewOrderID(uuid.New()), now); err != nil {
			t.Fatal(err)
		}
		save(t, repo, loaded)
		again := get(t, repo, inv.ShowID())
		assertSame(t, loaded, again)

		again.Close(now)
		save(t, repo, again)

		assertSame(t, again, get(t, repo, inv.ShowID()))
	})

	t.Run("get by hold finds the inventory; an unknown hold is ErrHoldNotFound", func(t *testing.T) {
		repo := newRepo(t)
		inv := Open(t)
		id := hold(t, inv, "ORCH/A/2")
		save(t, repo, inv)

		got, err := repo.GetByHold(ctx, id)
		if err != nil || got.ShowID() != inv.ShowID() {
			t.Errorf("GetByHold() = %v, %v", got, err)
		}
		if _, err := repo.GetByHold(ctx, domain.NewHoldID(uuid.New())); !errors.Is(err, application.ErrHoldNotFound) {
			t.Errorf("error = %v, want %v", err, application.ErrHoldNotFound)
		}
	})

	t.Run("opening the same show twice is ErrConcurrentModification (TKT-1)", func(t *testing.T) {
		repo := newRepo(t)
		inv := Open(t)
		save(t, repo, inv)
		twin, err := domain.OpenInventory(inv.ShowID(), Layout(t), inv.StartsAt(), now)
		if err != nil {
			t.Fatal(err)
		}

		if err := repo.Save(ctx, twin); !errors.Is(err, application.ErrConcurrentModification) {
			t.Errorf("error = %v, want %v", err, application.ErrConcurrentModification)
		}
	})

	t.Run("a stale save is ErrConcurrentModification", func(t *testing.T) {
		repo := newRepo(t)
		inv := Open(t)
		save(t, repo, inv)
		first, second := get(t, repo, inv.ShowID()), get(t, repo, inv.ShowID())
		hold(t, first, "ORCH/A/1")
		save(t, repo, first)
		hold(t, second, "ORCH/A/2")

		if err := repo.Save(ctx, second); !errors.Is(err, application.ErrConcurrentModification) {
			t.Errorf("error = %v, want %v", err, application.ErrConcurrentModification)
		}
	})

	t.Run("save drains the events", func(t *testing.T) {
		repo := newRepo(t)
		inv := Open(t)

		save(t, repo, inv)

		if left := inv.PullEvents(); len(left) != 0 {
			t.Errorf("%d events left", len(left))
		}
	})
}

// Layout is ORCH rows A(2) at EUR 45 and FLOOR 3 GA places at EUR 25.
func Layout(t *testing.T) domain.InventoryLayout {
	t.Helper()
	eur, _ := sharedkernel.NewCurrency("EUR")
	p45, _ := sharedkernel.NewMoney(4500, eur)
	p25, _ := sharedkernel.NewMoney(2500, eur)
	return domain.InventoryLayout{Sections: []domain.InventorySection{
		{Code: "ORCH", Kind: domain.KindSeated, Rows: []domain.InventoryRow{{Label: "A", Seats: 2}}, Price: p45},
		{Code: "FLOOR", Kind: domain.KindGA, Capacity: 3, Price: p25},
	}}
}

// Open opens a fresh inventory for a new show, starting in a month.
func Open(t *testing.T) *domain.ShowInventory {
	t.Helper()
	inv, err := domain.OpenInventory(domain.NewShowID(uuid.New()), Layout(t), now.Add(30*24*time.Hour), now)
	if err != nil {
		t.Fatal(err)
	}
	return inv
}

func hold(t *testing.T, inv *domain.ShowInventory, seat string) domain.HoldID {
	t.Helper()
	ref, _ := domain.ParseSeatRef(seat)
	id := domain.NewHoldID(uuid.New())
	if err := inv.Hold(id, domain.NewCustomerID(uuid.New()), []domain.SeatRef{ref}, now, 10*time.Minute); err != nil {
		t.Fatal(err)
	}
	return id
}

func save(t *testing.T, repo application.InventoryRepository, inv *domain.ShowInventory) {
	t.Helper()
	if err := repo.Save(context.Background(), inv); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
}

func get(t *testing.T, repo application.InventoryRepository, id domain.ShowID) *domain.ShowInventory {
	t.Helper()
	inv, err := repo.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	return inv
}

func assertSame(t *testing.T, want, got *domain.ShowInventory) {
	t.Helper()
	w, g := domain.StateOf(want), domain.StateOf(got)
	w.Version, g.Version = 0, 0
	opts := cmp.Options{
		cmp.AllowUnexported(domain.ShowID{}, domain.HoldID{}, domain.OrderID{}, domain.CustomerID{}, domain.SeatRef{},
			sharedkernel.Money{}, sharedkernel.Currency{}),
		cmp.Comparer(func(a, b time.Time) bool { return a.Equal(b) }),
	}
	if diff := cmp.Diff(w, g, opts); diff != "" {
		t.Errorf("loaded inventory mismatch (-saved +loaded):\n%s", diff)
	}
}
