package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

func TestOpenInventory(t *testing.T) {
	t.Run("opens every priced seat", func(t *testing.T) {
		f := newFixture(t)

		id := f.openShow(t)

		if s := f.seatStates(t, id); len(s) != 5 || s["FLOOR/GA/0003"] != "available" {
			t.Errorf("seats = %v", s)
		}
	})

	t.Run("receiving show.published.v1 twice opens it once (TKT-1)", func(t *testing.T) {
		f := newFixture(t)
		id := f.openShow(t)
		f.holdSeats(t, id, uuid.NewString(), "ORCH/A/1")

		err := f.open.Handle(f.ctx, openCmd(id))

		if err != nil || f.seatStates(t, id)["ORCH/A/1"] != "held" {
			t.Errorf("error = %v; the duplicate must not reopen (seat %s)", err, f.seatStates(t, id)["ORCH/A/1"])
		}
	})
}

func TestHoldSeats(t *testing.T) {
	t.Run("holds seats until now + TTL (TKT-4)", func(t *testing.T) {
		f := newFixture(t)
		show := f.openShow(t)

		res, err := f.hold.Handle(f.ctx, application.HoldSeats{ShowID: show, CustomerID: uuid.NewString(), Seats: []string{"ORCH/A/1", "ORCH/A/2"}})

		if err != nil || !res.ExpiresAt.Equal(fixedNow.Add(holdTTL)) {
			t.Fatalf("result = %+v, %v", res, err)
		}
		if s := f.seatStates(t, show); s["ORCH/A/1"] != "held" || s["ORCH/A/2"] != "held" {
			t.Errorf("seats = %v", s)
		}
	})

	tests := map[string]struct {
		cmd     func(show string) application.HoldSeats
		wantErr error
	}{
		"unknown show": {func(string) application.HoldSeats {
			return application.HoldSeats{ShowID: uuid.NewString(), CustomerID: uuid.NewString(), Seats: []string{"ORCH/A/1"}}
		}, application.ErrInventoryNotFound},
		"malformed seat": {func(show string) application.HoldSeats {
			return application.HoldSeats{ShowID: show, CustomerID: uuid.NewString(), Seats: []string{"ORCH"}}
		}, domain.ErrInvalidSeatRef},
		"missing customer": {func(show string) application.HoldSeats {
			return application.HoldSeats{ShowID: show, Seats: []string{"ORCH/A/1"}}
		}, domain.ErrInvalidID},
	}
	for name, tt := range tests {
		t.Run("rejects "+name, func(t *testing.T) {
			f := newFixture(t)

			if _, err := f.hold.Handle(f.ctx, tt.cmd(f.openShow(t))); !errors.Is(err, tt.wantErr) {
				t.Errorf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}

	t.Run("retries a conflict, then gives up after 3 attempts", func(t *testing.T) {
		f := newFixture(t)
		show := f.openShow(t)
		always := &conflicting{InventoryRepository: f.inventories, times: 99}
		handler := application.NewHoldSeatsHandler(always, f.ids, f.clock, holdTTL)

		_, err := handler.Handle(f.ctx, application.HoldSeats{ShowID: show, CustomerID: uuid.NewString(), Seats: []string{"ORCH/A/1"}})

		if !errors.Is(err, application.ErrConcurrentModification) || always.saves != 3 {
			t.Errorf("error = %v after %d saves; want ErrConcurrentModification after 3", err, always.saves)
		}
	})

	t.Run("a single conflict is retried transparently", func(t *testing.T) {
		f := newFixture(t)
		show := f.openShow(t)
		once := &conflicting{InventoryRepository: f.inventories, times: 1}
		handler := application.NewHoldSeatsHandler(once, f.ids, f.clock, holdTTL)

		if _, err := handler.Handle(f.ctx, application.HoldSeats{ShowID: show, CustomerID: uuid.NewString(), Seats: []string{"ORCH/A/1"}}); err != nil {
			t.Errorf("error = %v, want the retry to succeed", err)
		}
	})
}

func TestReleaseHold(t *testing.T) {
	f := newFixture(t)
	show := f.openShow(t)
	customer := uuid.NewString()
	hold := f.holdSeats(t, show, customer, "ORCH/A/1")

	if err := f.release.Handle(f.ctx, application.ReleaseHold{HoldID: hold.String(), CustomerID: uuid.NewString()}); !errors.Is(err, domain.ErrNotHoldOwner) {
		t.Errorf("stranger: error = %v, want %v", err, domain.ErrNotHoldOwner)
	}
	if err := f.release.Handle(f.ctx, application.ReleaseHold{HoldID: hold.String(), CustomerID: customer}); err != nil {
		t.Fatal(err)
	}
	if f.seatStates(t, show)["ORCH/A/1"] != "available" {
		t.Error("seat still held")
	}
	if err := f.release.Handle(f.ctx, application.ReleaseHold{HoldID: hold.String(), CustomerID: customer}); !errors.Is(err, application.ErrHoldNotFound) {
		t.Errorf("released twice: error = %v, want %v", err, application.ErrHoldNotFound)
	}
}

// conflicting loses the first `times` saves to a concurrent writer.
type conflicting struct {
	application.InventoryRepository
	times, saves int
}

func (c *conflicting) Save(ctx context.Context, inv *domain.ShowInventory) error {
	c.saves++
	if c.saves <= c.times {
		return application.ErrConcurrentModification
	}
	return c.InventoryRepository.Save(ctx, inv)
}
