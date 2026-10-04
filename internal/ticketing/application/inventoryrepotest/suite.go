// Package inventoryrepotest is the contract of application.InventoryRepository.
package inventoryrepotest

import (
	"context"
	"errors"
	"slices"
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

	t.Run("get of an unknown section is ErrInventoryNotFound", func(t *testing.T) {
		_, err := newRepo(t).Get(ctx, domain.NewShowID(uuid.New()), "ORCH")

		if !errors.Is(err, application.ErrInventoryNotFound) {
			t.Errorf("error = %v, want %v", err, application.ErrInventoryNotFound)
		}
	})

	t.Run("an unknown show lists no sections", func(t *testing.T) {
		got, err := newRepo(t).ListByShow(ctx, domain.NewShowID(uuid.New()))

		if err != nil || len(got) != 0 {
			t.Errorf("ListByShow() = %v, %v; want none", got, err)
		}
	})

	t.Run("opened sections round-trip, one by one and per show", func(t *testing.T) {
		repo := newRepo(t)
		orch, floor := Open(t)
		save(t, repo, orch)
		save(t, repo, floor)

		assertSame(t, orch, get(t, repo, orch.ShowID(), "ORCH"))
		all, err := repo.ListByShow(ctx, orch.ShowID())
		if err != nil {
			t.Fatal(err)
		}
		var codes []string
		for _, inv := range all {
			codes = append(codes, inv.Section())
		}
		slices.Sort(codes)
		if diff := cmp.Diff([]string{"FLOOR", "ORCH"}, codes); diff != "" {
			t.Errorf("sections mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("holds, sales, sold out and closing round-trip", func(t *testing.T) {
		repo := newRepo(t)
		orch, _ := Open(t)
		save(t, repo, orch)
		loaded := get(t, repo, orch.ShowID(), "ORCH")
		held := hold(t, loaded, "ORCH/A/1")
		sold := hold(t, loaded, "ORCH/A/2")
		if err := loaded.ConfirmHold(sold, domain.NewOrderID(uuid.New()), now); err != nil {
			t.Fatal(err)
		}
		save(t, repo, loaded)
		again := get(t, repo, orch.ShowID(), "ORCH")
		assertSame(t, loaded, again)

		if err := again.ConfirmHold(held, domain.NewOrderID(uuid.New()), now); err != nil {
			t.Fatal(err)
		}
		again.Close(now)
		save(t, repo, again)

		final := get(t, repo, orch.ShowID(), "ORCH")
		assertSame(t, again, final)
		if !final.IsSoldOut() || !final.IsClosed() {
			t.Errorf("sold out %v, closed %v; want both", final.IsSoldOut(), final.IsClosed())
		}
	})

	t.Run("get by hold finds the section; an unknown hold is ErrHoldNotFound", func(t *testing.T) {
		repo := newRepo(t)
		_, floor := Open(t)
		id := hold(t, floor, "FLOOR/GA/0002")
		save(t, repo, floor)

		got, err := repo.GetByHold(ctx, id)
		if err != nil || got.ShowID() != floor.ShowID() || got.Section() != "FLOOR" {
			t.Errorf("GetByHold() = %v, %v", got, err)
		}
		if _, err := repo.GetByHold(ctx, domain.NewHoldID(uuid.New())); !errors.Is(err, application.ErrHoldNotFound) {
			t.Errorf("error = %v, want %v", err, application.ErrHoldNotFound)
		}
	})

	t.Run("opening the same section twice is ErrConcurrentModification (TKT-1)", func(t *testing.T) {
		repo := newRepo(t)
		orch, _ := Open(t)
		save(t, repo, orch)
		twins, err := domain.OpenInventory(orch.ShowID(), Layout(t), orch.StartsAt(), now)
		if err != nil {
			t.Fatal(err)
		}

		if err := repo.Save(ctx, twins[0]); !errors.Is(err, application.ErrConcurrentModification) {
			t.Errorf("error = %v, want %v", err, application.ErrConcurrentModification)
		}
	})

	t.Run("a stale save is ErrConcurrentModification", func(t *testing.T) {
		repo := newRepo(t)
		orch, _ := Open(t)
		save(t, repo, orch)
		first, second := get(t, repo, orch.ShowID(), "ORCH"), get(t, repo, orch.ShowID(), "ORCH")
		hold(t, first, "ORCH/A/1")
		save(t, repo, first)
		hold(t, second, "ORCH/A/2")

		if err := repo.Save(ctx, second); !errors.Is(err, application.ErrConcurrentModification) {
			t.Errorf("error = %v, want %v", err, application.ErrConcurrentModification)
		}
	})

	t.Run("two sections of one show never conflict (ADR-013)", func(t *testing.T) {
		repo := newRepo(t)
		orch, floor := Open(t)
		save(t, repo, orch)
		save(t, repo, floor)
		o, f := get(t, repo, orch.ShowID(), "ORCH"), get(t, repo, orch.ShowID(), "FLOOR")
		hold(t, o, "ORCH/A/1")
		hold(t, f, "FLOOR/GA/0001")

		save(t, repo, o)
		save(t, repo, f)
	})

	t.Run("save drains the events", func(t *testing.T) {
		repo := newRepo(t)
		orch, _ := Open(t)

		save(t, repo, orch)

		if left := orch.PullEvents(); len(left) != 0 {
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

// Open opens a fresh show's sections, starting in a month: ORCH and FLOOR.
func Open(t *testing.T) (orch, floor *domain.SectionInventory) {
	t.Helper()
	sections, err := domain.OpenInventory(domain.NewShowID(uuid.New()), Layout(t), now.Add(30*24*time.Hour), now)
	if err != nil {
		t.Fatal(err)
	}
	return sections[0], sections[1]
}

func hold(t *testing.T, inv *domain.SectionInventory, seat string) domain.HoldID {
	t.Helper()
	ref, _ := domain.ParseSeatRef(seat)
	id := domain.NewHoldID(uuid.New())
	if err := inv.Hold(id, domain.NewCustomerID(uuid.New()), []domain.SeatRef{ref}, now, 10*time.Minute); err != nil {
		t.Fatal(err)
	}
	return id
}

func save(t *testing.T, repo application.InventoryRepository, inv *domain.SectionInventory) {
	t.Helper()
	if err := repo.Save(context.Background(), inv); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
}

func get(t *testing.T, repo application.InventoryRepository, id domain.ShowID, section string) *domain.SectionInventory {
	t.Helper()
	inv, err := repo.Get(context.Background(), id, section)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	return inv
}

func assertSame(t *testing.T, want, got *domain.SectionInventory) {
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
