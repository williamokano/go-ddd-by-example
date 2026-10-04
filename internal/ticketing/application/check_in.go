package application

import (
	"context"
	"fmt"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// CheckIn is the command a gate scanner sends.
type CheckIn struct {
	TicketCode string
	GateID     string
}

// CheckInHandler lets a ticket in (TKT-13). It changes only the Ticket
// aggregate, which is why Ticket was kept small: two gates scanning the same
// ticket race for one row, and nothing else is locked.
type CheckInHandler struct {
	tickets  TicketRepository
	schedule ShowSchedule
	clock    Clock
}

// NewCheckInHandler wires the use case.
func NewCheckInHandler(tickets TicketRepository, schedule ShowSchedule, clock Clock) *CheckInHandler {
	return &CheckInHandler{tickets: tickets, schedule: schedule, clock: clock}
}

// Handle checks the ticket in. A lost race with another gate is not retried:
// reloading would find the ticket checked in, which is the right answer.
func (h *CheckInHandler) Handle(ctx context.Context, cmd CheckIn) error {
	code, err := domain.NewTicketCode(cmd.TicketCode)
	if err != nil {
		return fmt.Errorf("check in: %w", err)
	}
	gate, err := domain.NewGateID(cmd.GateID)
	if err != nil {
		return fmt.Errorf("check in: %w", err)
	}
	ticket, err := h.tickets.GetByCode(ctx, code)
	if err != nil {
		return fmt.Errorf("check in: %w", err)
	}
	startsAt, err := h.schedule.StartsAt(ctx, ticket.ShowID())
	if err != nil {
		return fmt.Errorf("check in: %w", err)
	}
	if err := ticket.CheckIn(gate, startsAt, h.clock.Now()); err != nil {
		return fmt.Errorf("check in: %w", err)
	}
	if err := h.tickets.Save(ctx, ticket); err != nil {
		return fmt.Errorf("check in: %w", err)
	}
	return nil
}
