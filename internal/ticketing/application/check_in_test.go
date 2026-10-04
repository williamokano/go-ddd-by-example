package application_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/memory"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// TKT-13 touches only the Ticket aggregate: the show's start comes from a
// read port, so the gate locks nothing but the ticket it scans.
func TestCheckIn(t *testing.T) {
	setup := func(t *testing.T) (*fixture, *application.CheckInHandler, domain.TicketCode) {
		t.Helper()
		f := newFixture(t)
		tickets := memory.NewTicketRepository()
		show := f.openShow(t)
		showID, _ := domain.ParseShowID(show)
		seat, _ := domain.ParseSeatRef("ORCH/A/1")
		ticket, _ := domain.IssueTicket(domain.NewTicketID(uuid.New()), showID, domain.NewOrderID(uuid.New()), seat, fixedNow)
		if err := tickets.Save(f.ctx, ticket); err != nil {
			t.Fatal(err)
		}
		return f, application.NewCheckInHandler(tickets, memory.NewShowSchedule(f.inventories), f.clock), ticket.Code()
	}

	t.Run("lets a valid ticket in once", func(t *testing.T) {
		f, checkIn, code := setup(t)
		f.clock.Advance(30*24*time.Hour - time.Hour) // the show starts in an hour
		cmd := application.CheckIn{TicketCode: code.String(), GateID: "north-1"}

		if err := checkIn.Handle(f.ctx, cmd); err != nil {
			t.Fatal(err)
		}
		if err := checkIn.Handle(f.ctx, cmd); !errors.Is(err, domain.ErrAlreadyCheckedIn) {
			t.Errorf("second scan: error = %v, want %v", err, domain.ErrAlreadyCheckedIn)
		}
	})

	t.Run("an unknown code is ErrTicketNotFound", func(t *testing.T) {
		f, checkIn, _ := setup(t)
		unknown := domain.TicketCodeFor(domain.NewTicketID(uuid.New()))

		err := checkIn.Handle(f.ctx, application.CheckIn{TicketCode: unknown.String(), GateID: "north-1"})

		if !errors.Is(err, application.ErrTicketNotFound) {
			t.Errorf("error = %v, want %v", err, application.ErrTicketNotFound)
		}
	})

	t.Run("not on the day of the show", func(t *testing.T) {
		f, checkIn, code := setup(t) // a month early

		err := checkIn.Handle(f.ctx, application.CheckIn{TicketCode: code.String(), GateID: "north-1"})

		if !errors.Is(err, domain.ErrNotShowDay) {
			t.Errorf("error = %v, want %v", err, domain.ErrNotShowDay)
		}
	})
}
