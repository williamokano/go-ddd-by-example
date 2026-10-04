package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/williamokano/go-ddd-by-example/internal/platform/idgen"
	"github.com/williamokano/go-ddd-by-example/internal/platform/outbox"
	pgplatform "github.com/williamokano/go-ddd-by-example/internal/platform/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/postgres/sqlcgen"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// OrderRepository implements application.OrderRepository on ticketing.orders.
type OrderRepository struct {
	pool       *pgxpool.Pool
	newEventID func() uuid.UUID
}

// NewOrderRepository returns a repository using pool.
func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{pool: pool, newEventID: idgen.UUIDv7{}.New}
}

type lineJSON struct {
	Seat     string `json:"seat"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type orderRow struct {
	ID, ShowID, HoldID, CustomerID uuid.UUID
	ContactEmail                   string
	Lines                          []byte
	TotalAmount                    int64
	Currency, Status, PaymentRef   string
	Version                        int32
	SubtotalAmount, FeeAmount      int64
	VatAmount                      int64
}

var orderStatuses = map[string]domain.OrderStatus{}

func init() {
	for _, s := range []domain.OrderStatus{domain.Pending, domain.Paid, domain.PaymentFailed, domain.Fulfilled, domain.Refunded} {
		orderStatuses[s.String()] = s
	}
}

// Get implements application.OrderRepository.
func (r *OrderRepository) Get(ctx context.Context, id domain.OrderID) (*domain.Order, error) {
	row, err := sqlcgen.New(pgplatform.Conn(ctx, r.pool)).GetOrder(ctx, id.UUID())
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: %s", application.ErrOrderNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("get order %s: %w", id, err)
	}
	return toOrder(orderRow(row))
}

// ListPaidForShow implements application.OrderRepository.
func (r *OrderRepository) ListPaidForShow(ctx context.Context, show domain.ShowID) ([]*domain.Order, error) {
	rows, err := sqlcgen.New(pgplatform.Conn(ctx, r.pool)).ListPaidOrdersForShow(ctx, show.UUID())
	if err != nil {
		return nil, fmt.Errorf("list orders of show %s: %w", show, err)
	}
	out := make([]*domain.Order, 0, len(rows))
	for _, row := range rows {
		o, err := toOrder(orderRow(row))
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, nil
}

func toOrder(r orderRow) (*domain.Order, error) {
	email, err := domain.NewContactEmail(r.ContactEmail)
	if err != nil {
		return nil, fmt.Errorf("order %s: %w", r.ID, err)
	}
	var pricing domain.PriceBreakdown
	for _, m := range []struct {
		into   *sharedkernel.Money
		amount int64
	}{{&pricing.Subtotal, r.SubtotalAmount}, {&pricing.Fee, r.FeeAmount}, {&pricing.VAT, r.VatAmount}, {&pricing.Total, r.TotalAmount}} {
		if *m.into, err = money(m.amount, r.Currency); err != nil {
			return nil, fmt.Errorf("order %s: %w", r.ID, err)
		}
	}
	var stored []lineJSON
	if err := json.Unmarshal(r.Lines, &stored); err != nil {
		return nil, fmt.Errorf("order %s lines: %w", r.ID, err)
	}
	state := domain.OrderState{
		ID: domain.NewOrderID(r.ID), ShowID: domain.NewShowID(r.ShowID), HoldID: domain.NewHoldID(r.HoldID),
		Customer: domain.NewCustomerID(r.CustomerID), Email: email, Pricing: pricing,
		Status: orderStatuses[r.Status], Version: int(r.Version),
	}
	for _, l := range stored {
		seat, err := domain.ParseSeatRef(l.Seat)
		if err != nil {
			return nil, err
		}
		price, err := money(l.Amount, l.Currency)
		if err != nil {
			return nil, err
		}
		state.Lines = append(state.Lines, domain.OrderLine{Seat: seat, Price: price})
	}
	if r.PaymentRef != "" {
		if state.PaymentRef, err = domain.NewPaymentRef(r.PaymentRef); err != nil {
			return nil, err
		}
	}
	return domain.RehydrateOrder(state), nil
}

func money(amount int64, currency string) (sharedkernel.Money, error) {
	c, err := sharedkernel.NewCurrency(currency)
	if err != nil {
		return sharedkernel.Money{}, err
	}
	return sharedkernel.NewMoney(amount, c)
}

// Save implements application.OrderRepository.
func (r *OrderRepository) Save(ctx context.Context, o *domain.Order) (err error) {
	tx, err := pgplatform.Begin(ctx, r.pool)
	if err != nil {
		return fmt.Errorf("save order %s: begin: %w", o.ID(), err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()
	q := sqlcgen.New(tx)
	var n int64
	if o.Version() == 0 {
		lines := make([]lineJSON, 0, len(o.Lines()))
		for _, l := range o.Lines() {
			lines = append(lines, lineJSON{Seat: l.Seat.String(), Amount: l.Price.Amount(), Currency: l.Price.Currency().String()})
		}
		b, err := json.Marshal(lines)
		if err != nil {
			return fmt.Errorf("save order %s: lines: %w", o.ID(), err)
		}
		n, err = q.InsertOrder(ctx, sqlcgen.InsertOrderParams{
			ID: o.ID().UUID(), ShowID: o.ShowID().UUID(), HoldID: o.HoldID().UUID(), CustomerID: o.Customer().UUID(),
			ContactEmail: o.ContactEmail().String(), Lines: b, TotalAmount: o.Total().Amount(),
			Currency: o.Total().Currency().String(), Status: o.Status().String(), PaymentRef: o.PaymentRef().String(),
			SubtotalAmount: o.Pricing().Subtotal.Amount(), FeeAmount: o.Pricing().Fee.Amount(), VatAmount: o.Pricing().VAT.Amount(),
		})
		if err != nil {
			return fmt.Errorf("save order %s: %w", o.ID(), err)
		}
	} else {
		n, err = q.UpdateOrder(ctx, sqlcgen.UpdateOrderParams{
			ID: o.ID().UUID(), Status: o.Status().String(), PaymentRef: o.PaymentRef().String(), ExpectedVersion: int32(o.Version()),
		})
		if err != nil {
			return fmt.Errorf("save order %s: %w", o.ID(), err)
		}
	}
	if n == 0 {
		return fmt.Errorf("%w: order %s", application.ErrConcurrentModification, o.ID())
	}
	msgs, err := ToOutboxMessages(o.PullEvents(), r.newEventID)
	if err != nil {
		return fmt.Errorf("save order %s: %w", o.ID(), err)
	}
	if err := outbox.Write(ctx, tx, "ticketing", msgs); err != nil {
		return fmt.Errorf("save order %s: %w", o.ID(), err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("save order %s: commit: %w", o.ID(), err)
	}
	return nil
}
