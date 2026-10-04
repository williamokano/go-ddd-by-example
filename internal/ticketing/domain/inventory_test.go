package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// One inventory per section (ADR-013): a hold in ORCH never contends with a
// hold on the FLOOR.
func TestOpenInventory_OpensOneInventoryPerSection(t *testing.T) {
	sections, err := domain.OpenInventory(showID, layout(t), startsAt, now)

	if err != nil {
		t.Fatal(err)
	}
	got := map[string][]string{}
	for _, inv := range sections {
		for _, s := range inv.Seats() {
			if s.State != domain.Available {
				t.Errorf("%s is %v, want available", s.Ref, s.State)
			}
			got[inv.Section()] = append(got[inv.Section()], s.Ref.String()+" "+s.Price.String())
		}
		events := inv.PullEvents()
		want := domain.InventoryOpened{ShowID: showID, Section: inv.Section(), Seats: 3, At: now}
		if len(events) != 1 || events[0] != sharedkernel.DomainEvent(want) {
			t.Errorf("%s events = %v, want %v", inv.Section(), events, want)
		}
	}
	if sections[0].Section() != "ORCH" || sections[0].Position() != 0 || sections[1].Position() != 1 {
		t.Errorf("sections out of layout order: %s@%d, %s@%d",
			sections[0].Section(), sections[0].Position(), sections[1].Section(), sections[1].Position())
	}
	want := map[string][]string{
		"ORCH":  {"ORCH/A/1 EUR 45.00", "ORCH/A/2 EUR 45.00", "ORCH/B/1 EUR 45.00"},
		"FLOOR": {"FLOOR/GA/0001 EUR 25.00", "FLOOR/GA/0002 EUR 25.00", "FLOOR/GA/0003 EUR 25.00"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("seats mismatch (-want +got):\n%s", diff)
	}
}

func TestOpenInventory_RejectsAnEmptyLayout(t *testing.T) {
	if _, err := domain.OpenInventory(showID, domain.InventoryLayout{}, startsAt, now); !errors.Is(err, domain.ErrInvalidLayout) {
		t.Errorf("error = %v, want %v", err, domain.ErrInvalidLayout)
	}
}

func TestOpenInventory_RejectsTheSameSectionTwice(t *testing.T) {
	l := layout(t)
	l.Sections = append(l.Sections, l.Sections[0])
	if _, err := domain.OpenInventory(showID, l, startsAt, now); !errors.Is(err, domain.ErrInvalidLayout) {
		t.Errorf("error = %v, want %v: two inventories would sell the same seats", err, domain.ErrInvalidLayout)
	}
}

func TestSectionInventory_Hold(t *testing.T) {
	t.Run("holds the seats and records SeatsHeld (TKT-2, TKT-4)", func(t *testing.T) {
		inv := orch(t)
		id := newHoldID()

		err := inv.Hold(id, ana, refs(t, "ORCH/A/1", "ORCH/B/1"), now, ttl)

		if err != nil {
			t.Fatal(err)
		}
		if s := states(inv); s["ORCH/A/1"] != "held" || s["ORCH/B/1"] != "held" || s["ORCH/A/2"] != "available" {
			t.Errorf("states = %v", s)
		}
		want := []sharedkernel.DomainEvent{domain.SeatsHeld{
			ShowID: showID, HoldID: id, CustomerID: ana, Seats: refs(t, "ORCH/A/1", "ORCH/B/1"),
			ExpiresAt: now.Add(ttl), At: now,
		}}
		if diff := cmp.Diff(want, inv.PullEvents(), cmp.AllowUnexported(domain.ShowID{}, domain.HoldID{}, domain.CustomerID{}, domain.SeatRef{})); diff != "" {
			t.Errorf("events mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("a seat already held makes the whole hold fail; nothing changes (TKT-2, TKT-5)", func(t *testing.T) {
		inv := orch(t)
		held(t, inv, bob, "ORCH/A/2")
		before := states(inv)
		inv.PullEvents()

		err := inv.Hold(newHoldID(), ana, refs(t, "ORCH/A/1", "ORCH/A/2"), now, ttl)

		if !errors.Is(err, domain.ErrSeatUnavailable) {
			t.Errorf("error = %v, want %v", err, domain.ErrSeatUnavailable)
		}
		if diff := cmp.Diff(before, states(inv)); diff != "" || len(inv.PullEvents()) != 0 {
			t.Errorf("a failed hold changed seats or recorded events:\n%s", diff)
		}
	})

	tooMany := []string{"ORCH/A/1", "ORCH/A/2", "ORCH/B/1", "FLOOR/GA/0001", "FLOOR/GA/0002", "FLOOR/GA/0003", "ORCH/A/1", "ORCH/A/2", "ORCH/B/1"}
	tests := []struct {
		name    string
		seats   []string
		wantErr error
	}{
		{"no seats (TKT-2)", nil, domain.ErrInvalidHoldSize},
		{"more than 8 seats (TKT-2)", tooMany, domain.ErrInvalidHoldSize},
		{"the same seat twice", []string{"ORCH/A/1", "orch/a/1"}, domain.ErrDuplicateSeat},
		{"a seat that does not exist", []string{"ORCH/Z/9"}, domain.ErrUnknownSeat},
		{"seats of another section (ADR-013)", []string{"ORCH/A/1", "FLOOR/GA/0001"}, domain.ErrHoldSpansSections},
	}
	for _, tt := range tests {
		t.Run("rejects "+tt.name, func(t *testing.T) {
			err := orch(t).Hold(newHoldID(), ana, refs(t, tt.seats...), now, ttl)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}

	t.Run("a second active hold by the same customer in the section is rejected (TKT-3, per section since ADR-013)", func(t *testing.T) {
		inv := orch(t)
		held(t, inv, ana, "ORCH/A/1")

		err := inv.Hold(newHoldID(), ana, refs(t, "ORCH/A/2"), now, ttl)

		if !errors.Is(err, domain.ErrCustomerAlreadyHolding) {
			t.Errorf("error = %v, want %v", err, domain.ErrCustomerAlreadyHolding)
		}
	})

	t.Run("no holds once the show has started (TKT-12)", func(t *testing.T) {
		err := orch(t).Hold(newHoldID(), ana, refs(t, "ORCH/A/1"), startsAt, ttl)

		if !errors.Is(err, domain.ErrSalesClosed) {
			t.Errorf("error = %v, want %v", err, domain.ErrSalesClosed)
		}
	})

	t.Run("seats of an expired, unswept hold can be held again (TKT-4)", func(t *testing.T) {
		inv := orch(t)
		held(t, inv, bob, "ORCH/A/1")
		later := now.Add(ttl + time.Second)

		err := inv.Hold(newHoldID(), ana, refs(t, "ORCH/A/1"), later, ttl)

		if err != nil {
			t.Errorf("error = %v, want the expired hold's seats to be free", err)
		}
	})
}
