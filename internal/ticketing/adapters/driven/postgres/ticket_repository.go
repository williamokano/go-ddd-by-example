package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/postgres/sqlcgen"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// TicketRepository implements application.TicketRepository on ticketing.tickets.
type TicketRepository struct{ pool *pgxpool.Pool }

// NewTicketRepository returns a repository using pool.
func NewTicketRepository(pool *pgxpool.Pool) *TicketRepository { return &TicketRepository{pool: pool} }

// Save implements application.TicketRepository: a new ticket is inserted if
// absent (its ID derives from order + seat), an existing one updated.
func (r *TicketRepository) Save(ctx context.Context, t *domain.Ticket) error {
	q := sqlcgen.New(r.pool)
	defer t.PullEvents()
	if t.Version() == 0 {
		err := q.InsertTicket(ctx, sqlcgen.InsertTicketParams{
			ID: t.ID().UUID(), Code: t.Code().String(), ShowID: t.ShowID().UUID(), OrderID: t.OrderID().UUID(),
			SeatRef: t.Seat().String(), Status: t.Status().String(),
		})
		if err != nil {
			return fmt.Errorf("issue ticket %s: %w", t.ID(), err)
		}
		return nil
	}
	n, err := q.UpdateTicket(ctx, sqlcgen.UpdateTicketParams{ID: t.ID().UUID(), Status: t.Status().String(), ExpectedVersion: int32(t.Version())})
	if err != nil {
		return fmt.Errorf("save ticket %s: %w", t.ID(), err)
	}
	if n == 0 {
		return fmt.Errorf("%w: ticket %s", application.ErrConcurrentModification, t.ID())
	}
	return nil
}

// ListByOrder implements application.TicketRepository.
func (r *TicketRepository) ListByOrder(ctx context.Context, order domain.OrderID) ([]*domain.Ticket, error) {
	rows, err := sqlcgen.New(r.pool).ListTicketsByOrder(ctx, order.UUID())
	if err != nil {
		return nil, fmt.Errorf("tickets of order %s: %w", order, err)
	}
	out := make([]*domain.Ticket, 0, len(rows))
	for _, row := range rows {
		t, err := toTicket(ticketRow(row))
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

// ListByShow implements application.TicketRepository.
func (r *TicketRepository) ListByShow(ctx context.Context, show domain.ShowID) ([]*domain.Ticket, error) {
	rows, err := sqlcgen.New(r.pool).ListTicketsByShow(ctx, show.UUID())
	if err != nil {
		return nil, fmt.Errorf("tickets of show %s: %w", show, err)
	}
	out := make([]*domain.Ticket, 0, len(rows))
	for _, row := range rows {
		t, err := toTicket(ticketRow(row))
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

type ticketRow struct {
	ID              uuid.UUID
	Code            string
	ShowID, OrderID uuid.UUID
	SeatRef, Status string
	Version         int32
}

func toTicket(r ticketRow) (*domain.Ticket, error) {
	code, err := domain.NewTicketCode(r.Code)
	if err != nil {
		return nil, err
	}
	seat, err := domain.ParseSeatRef(r.SeatRef)
	if err != nil {
		return nil, err
	}
	status := domain.ValidTicket
	if r.Status == domain.VoidedTicket.String() {
		status = domain.VoidedTicket
	}
	return domain.RehydrateTicket(domain.TicketState{
		ID: domain.NewTicketID(r.ID), Code: code, ShowID: domain.NewShowID(r.ShowID), OrderID: domain.NewOrderID(r.OrderID),
		Seat: seat, Status: status, Version: int(r.Version),
	}), nil
}
