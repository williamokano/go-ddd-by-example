package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

func TestIssueTicket(t *testing.T) {
	id := domain.NewTicketID(uuid.MustParse("0192f5e0-0000-7000-8000-00000000abcd"))
	seat := refs(t, "ORCH/A/1")[0]
	order := newOrderID()

	ticket, err := domain.IssueTicket(id, showID, order, seat, now)

	if err != nil {
		t.Fatal(err)
	}
	if ticket.Status() != domain.ValidTicket || ticket.Seat() != seat || ticket.OrderID() != order {
		t.Errorf("ticket = %v %v %v", ticket.Status(), ticket.Seat(), ticket.OrderID())
	}
	if code := ticket.Code().String(); len(code) != 14 || code != domain.TicketCodeFor(id).String() {
		t.Errorf("Code() = %q, want 14 characters derived from the id (TKT-9)", code)
	}
	if ev := ticket.PullEvents(); len(ev) != 1 || ev[0].EventName() != "ticketing.TicketIssued" {
		t.Errorf("events = %v", ev)
	}
}

func TestTicketCodeFor_IsUniquePerTicket(t *testing.T) {
	seen := map[string]bool{}
	for range 1000 {
		code := domain.TicketCodeFor(domain.NewTicketID(uuid.New())).String()
		if seen[code] {
			t.Fatalf("duplicate code %s (TKT-9)", code)
		}
		seen[code] = true
	}
}

func TestTicket_Void(t *testing.T) {
	ticket, _ := domain.IssueTicket(domain.NewTicketID(uuid.New()), showID, newOrderID(), refs(t, "ORCH/A/1")[0], now)
	ticket.PullEvents()

	ticket.Void(now)
	ticket.Void(now) // the show-cancelled fact may arrive twice

	if ticket.Status() != domain.VoidedTicket || len(ticket.PullEvents()) != 1 {
		t.Errorf("status = %v; want voided with one TicketVoided (TKT-11)", ticket.Status())
	}
}

func TestSectionInventory_RejectConfirmation(t *testing.T) {
	inv := orch(t)
	hold := held(t, inv, ana, "ORCH/A/1")
	inv.PullEvents()
	order := newOrderID()

	inv.RejectConfirmation(hold, order, errors.New("hold expired"), now)

	failed, ok := inv.PullEvents()[0].(domain.HoldConfirmationFailed)
	if !ok || failed.OrderID != order || failed.HoldID != hold || failed.Reason != "hold expired" {
		t.Errorf("event = %+v, want HoldConfirmationFailed for the order (TKT-8)", failed)
	}
}

func TestOrder_MarkFulfilled_CarriesTheTickets(t *testing.T) {
	o := placed(t)
	ref, _ := domain.NewPaymentRef("pay_1")
	_ = o.MarkPaid(ref, now)
	o.PullEvents()
	tickets := []domain.IssuedTicket{{Seat: refs(t, "ORCH/A/1")[0], Code: domain.TicketCodeFor(domain.NewTicketID(uuid.New()))}}

	if err := o.MarkFulfilled(tickets, now); err != nil {
		t.Fatal(err)
	}

	ev := o.PullEvents()[0].(domain.OrderFulfilled)
	if len(ev.Tickets) != 1 || ev.ContactEmail.String() != "ana@example.com" {
		t.Errorf("OrderFulfilled = %+v", ev)
	}
}

func TestTicket_CheckIn(t *testing.T) {
	gate, _ := domain.NewGateID("north-1")
	issue := func(t *testing.T) *domain.Ticket {
		t.Helper()
		ticket, _ := domain.IssueTicket(domain.NewTicketID(uuid.New()), showID, newOrderID(), refs(t, "ORCH/A/1")[0], now)
		ticket.PullEvents()
		return ticket
	}
	showDay := startsAt.Add(-time.Hour) // doors open an hour before

	t.Run("lets the ticket in once and records the gate (TKT-13)", func(t *testing.T) {
		ticket := issue(t)

		if err := ticket.CheckIn(gate, startsAt, showDay); err != nil {
			t.Fatal(err)
		}

		if ticket.Status() != domain.CheckedInTicket || ticket.CheckedInAt() != showDay || ticket.Gate() != gate {
			t.Errorf("ticket = %v at %v through %v", ticket.Status(), ticket.CheckedInAt(), ticket.Gate())
		}
		want := domain.TicketCheckedIn{TicketID: ticket.ID(), ShowID: showID, Gate: gate, At: showDay}
		if ev := ticket.PullEvents(); len(ev) != 1 || ev[0] != sharedkernel.DomainEvent(want) {
			t.Errorf("events = %v, want %v", ev, want)
		}
	})

	t.Run("a second check-in is rejected, even through another gate (TKT-13)", func(t *testing.T) {
		ticket := issue(t)
		_ = ticket.CheckIn(gate, startsAt, showDay)
		other, _ := domain.NewGateID("south-2")

		err := ticket.CheckIn(other, startsAt, showDay.Add(time.Minute))

		if !errors.Is(err, domain.ErrAlreadyCheckedIn) {
			t.Errorf("error = %v, want %v", err, domain.ErrAlreadyCheckedIn)
		}
	})

	t.Run("a voided ticket is rejected (TKT-13)", func(t *testing.T) {
		ticket := issue(t)
		ticket.Void(now)

		if err := ticket.CheckIn(gate, startsAt, showDay); !errors.Is(err, domain.ErrTicketVoided) {
			t.Errorf("error = %v, want %v", err, domain.ErrTicketVoided)
		}
	})

	for name, at := range map[string]time.Time{
		"the day before": startsAt.Add(-12*time.Hour - time.Second),
		"the day after":  startsAt.Add(12*time.Hour + time.Second),
	} {
		t.Run("rejects "+name+" (TKT-13)", func(t *testing.T) {
			if err := issue(t).CheckIn(gate, startsAt, at); !errors.Is(err, domain.ErrNotShowDay) {
				t.Errorf("error = %v, want %v", err, domain.ErrNotShowDay)
			}
		})
	}

	t.Run("a gate needs a name", func(t *testing.T) {
		if _, err := domain.NewGateID("  "); !errors.Is(err, domain.ErrInvalidID) {
			t.Errorf("error = %v, want %v", err, domain.ErrInvalidID)
		}
	})
}
