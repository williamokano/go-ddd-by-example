// Package postgres implements Ticketing's driven ports on Postgres (schema
// ticketing). sqlc-generated types never leave this package.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/williamokano/go-ddd-by-example/internal/platform/idgen"
	"github.com/williamokano/go-ddd-by-example/internal/platform/outbox"
	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/postgres/sqlcgen"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// InventoryRepository implements application.InventoryRepository on
// ticketing.inventories, .seats and .holds, with the outbox (ADR-004).
type InventoryRepository struct {
	pool       *pgxpool.Pool
	newEventID func() uuid.UUID
}

// NewInventoryRepository returns a repository using pool.
func NewInventoryRepository(pool *pgxpool.Pool) *InventoryRepository {
	return &InventoryRepository{pool: pool, newEventID: idgen.UUIDv7{}.New}
}

// Get implements application.InventoryRepository.
func (r *InventoryRepository) Get(ctx context.Context, id domain.ShowID) (*domain.ShowInventory, error) {
	return load(ctx, sqlcgen.New(r.pool), id.UUID())
}

// GetByHold implements application.InventoryRepository.
func (r *InventoryRepository) GetByHold(ctx context.Context, id domain.HoldID) (*domain.ShowInventory, error) {
	q := sqlcgen.New(r.pool)
	showID, err := q.GetShowIDByHold(ctx, id.UUID())
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: %s", application.ErrHoldNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("find hold %s: %w", id, err)
	}
	return load(ctx, q, showID)
}

func load(ctx context.Context, q *sqlcgen.Queries, showID uuid.UUID) (*domain.ShowInventory, error) {
	row, err := q.GetInventory(ctx, showID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: show %s", application.ErrInventoryNotFound, showID)
	}
	if err != nil {
		return nil, fmt.Errorf("get inventory %s: %w", showID, err)
	}
	seats, err := q.ListSeats(ctx, showID)
	if err != nil {
		return nil, fmt.Errorf("get inventory %s seats: %w", showID, err)
	}
	holds, err := q.ListHolds(ctx, showID)
	if err != nil {
		return nil, fmt.Errorf("get inventory %s holds: %w", showID, err)
	}
	state := domain.InventoryState{
		ShowID: domain.NewShowID(row.ShowID), StartsAt: row.StartsAt,
		Closed: row.Closed, SoldOut: row.SoldOut, Version: int(row.Version),
	}
	for _, s := range seats {
		view, err := toSeatView(s)
		if err != nil {
			return nil, fmt.Errorf("inventory %s: %w", showID, err)
		}
		state.Seats = append(state.Seats, view)
	}
	for _, h := range holds {
		hs := domain.HoldState{ID: domain.NewHoldID(h.HoldID), Customer: domain.NewCustomerID(h.CustomerID), ExpiresAt: h.ExpiresAt}
		for _, raw := range h.Seats {
			ref, err := domain.ParseSeatRef(raw)
			if err != nil {
				return nil, fmt.Errorf("hold %s: %w", h.HoldID, err)
			}
			hs.Seats = append(hs.Seats, ref)
		}
		state.Holds = append(state.Holds, hs)
	}
	return domain.RehydrateInventory(state), nil
}

var seatStates = map[string]domain.SeatState{"available": domain.Available, "held": domain.Held, "sold": domain.Sold}

func toSeatView(s sqlcgen.ListSeatsRow) (domain.SeatView, error) {
	ref, err := domain.ParseSeatRef(s.SeatRef)
	if err != nil {
		return domain.SeatView{}, err
	}
	currency, err := sharedkernel.NewCurrency(s.Currency)
	if err != nil {
		return domain.SeatView{}, err
	}
	price, err := sharedkernel.NewMoney(s.PriceAmount, currency)
	if err != nil {
		return domain.SeatView{}, err
	}
	state, ok := seatStates[s.State]
	if !ok {
		return domain.SeatView{}, fmt.Errorf("seat %s: unknown stored state %q", s.SeatRef, s.State)
	}
	view := domain.SeatView{Ref: ref, Price: price, State: state}
	if s.HoldID.Valid {
		view.HoldID = domain.NewHoldID(s.HoldID.UUID)
	}
	if s.OrderID.Valid {
		view.OrderID = domain.NewOrderID(s.OrderID.UUID)
	}
	return view, nil
}

// Save implements application.InventoryRepository. A new inventory inserts
// every seat (COPY); afterwards only the seats its pending events mention are
// written: the events are the change log, so 50k seats aren't rewritten for a
// 2-seat hold.
func (r *InventoryRepository) Save(ctx context.Context, inv *domain.ShowInventory) (err error) {
	events := inv.PullEvents()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("save inventory %s: begin: %w", inv.ShowID(), err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()
	q := sqlcgen.New(tx)
	showID := inv.ShowID().UUID()

	if inv.Version() == 0 {
		err = insertInventory(ctx, q, inv)
	} else {
		err = updateInventory(ctx, q, inv, changedSeats(events))
	}
	if err != nil {
		return err
	}
	if err := q.DeleteHolds(ctx, showID); err != nil {
		return fmt.Errorf("save inventory %s: holds: %w", inv.ShowID(), err)
	}
	for _, h := range inv.Holds() {
		seats := make([]string, 0, len(h.Seats()))
		for _, ref := range h.Seats() {
			seats = append(seats, ref.String())
		}
		if err := q.InsertHold(ctx, sqlcgen.InsertHoldParams{
			HoldID: h.ID().UUID(), ShowID: showID, CustomerID: h.Customer().UUID(), Seats: seats, ExpiresAt: h.ExpiresAt(),
		}); err != nil {
			return fmt.Errorf("save inventory %s: hold: %w", inv.ShowID(), err)
		}
	}
	msgs, err := ToOutboxMessages(events, r.newEventID)
	if err != nil {
		return fmt.Errorf("save inventory %s: %w", inv.ShowID(), err)
	}
	if err := outbox.Write(ctx, tx, "ticketing", msgs); err != nil {
		return fmt.Errorf("save inventory %s: %w", inv.ShowID(), err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("save inventory %s: commit: %w", inv.ShowID(), err)
	}
	return nil
}

func insertInventory(ctx context.Context, q *sqlcgen.Queries, inv *domain.ShowInventory) error {
	n, err := q.InsertInventory(ctx, sqlcgen.InsertInventoryParams{
		ShowID: inv.ShowID().UUID(), StartsAt: inv.StartsAt(), Closed: inv.IsClosed(), SoldOut: inv.IsSoldOut(),
	})
	if err != nil {
		return fmt.Errorf("save inventory %s: %w", inv.ShowID(), err)
	}
	if n == 0 {
		return fmt.Errorf("%w: inventory %s already open", application.ErrConcurrentModification, inv.ShowID())
	}
	seats := inv.Seats()
	rows := make([]sqlcgen.InsertSeatsParams, len(seats))
	for i, s := range seats {
		rows[i] = seatRow(inv.ShowID(), s)
		rows[i].Position = int32(i)
	}
	if _, err := q.InsertSeats(ctx, rows); err != nil {
		return fmt.Errorf("save inventory %s: seats: %w", inv.ShowID(), err)
	}
	return nil
}

func updateInventory(ctx context.Context, q *sqlcgen.Queries, inv *domain.ShowInventory, changed map[domain.SeatRef]bool) error {
	n, err := q.UpdateInventory(ctx, sqlcgen.UpdateInventoryParams{
		ShowID: inv.ShowID().UUID(), Closed: inv.IsClosed(), SoldOut: inv.IsSoldOut(), ExpectedVersion: int32(inv.Version()),
	})
	if err != nil {
		return fmt.Errorf("save inventory %s: %w", inv.ShowID(), err)
	}
	if n == 0 {
		return fmt.Errorf("%w: inventory %s", application.ErrConcurrentModification, inv.ShowID())
	}
	for _, s := range inv.Seats() {
		if !changed[s.Ref] {
			continue
		}
		row := seatRow(inv.ShowID(), s)
		if err := q.UpdateSeat(ctx, sqlcgen.UpdateSeatParams{
			ShowID: row.ShowID, SeatRef: row.SeatRef, State: row.State, HoldID: row.HoldID, OrderID: row.OrderID,
		}); err != nil {
			return fmt.Errorf("save inventory %s: seat %s: %w", inv.ShowID(), s.Ref, err)
		}
	}
	return nil
}

func seatRow(showID domain.ShowID, s domain.SeatView) sqlcgen.InsertSeatsParams {
	row := sqlcgen.InsertSeatsParams{
		ShowID: showID.UUID(), SeatRef: s.Ref.String(), PriceAmount: s.Price.Amount(),
		Currency: s.Price.Currency().String(), State: s.State.String(),
	}
	if !s.HoldID.IsZero() {
		row.HoldID = uuid.NullUUID{UUID: s.HoldID.UUID(), Valid: true}
	}
	if !s.OrderID.IsZero() {
		row.OrderID = uuid.NullUUID{UUID: s.OrderID.UUID(), Valid: true}
	}
	return row
}

// changedSeats lists the seats the pending events touched.
func changedSeats(events []sharedkernel.DomainEvent) map[domain.SeatRef]bool {
	changed := map[domain.SeatRef]bool{}
	mark := func(refs []domain.SeatRef) {
		for _, r := range refs {
			changed[r] = true
		}
	}
	for _, ev := range events {
		switch e := ev.(type) {
		case domain.SeatsHeld:
			mark(e.Seats)
		case domain.HoldReleased:
			mark(e.Seats)
		case domain.HoldExpired:
			mark(e.Seats)
		case domain.SeatsSold:
			mark(e.Seats)
		case domain.InventoryClosed:
			mark(e.ReleasedSeats)
		}
	}
	return changed
}
