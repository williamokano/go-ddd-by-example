package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	pgplatform "github.com/williamokano/go-ddd-by-example/internal/platform/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/postgres/sqlcgen"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// CheckoutProcessRepository implements application.CheckoutProcessRepository
// on ticketing.checkout_processes. The process's events are its own log, not
// messages: nothing goes to the outbox.
type CheckoutProcessRepository struct{ pool *pgxpool.Pool }

// NewCheckoutProcessRepository returns a repository using pool.
func NewCheckoutProcessRepository(pool *pgxpool.Pool) *CheckoutProcessRepository {
	return &CheckoutProcessRepository{pool: pool}
}

// Get implements application.CheckoutProcessRepository.
func (r *CheckoutProcessRepository) Get(ctx context.Context, id domain.OrderID) (*domain.CheckoutProcess, error) {
	row, err := sqlcgen.New(pgplatform.Conn(ctx, r.pool)).GetCheckoutProcess(ctx, id.UUID())
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: order %s", application.ErrCheckoutNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("checkout %s: %w", id, err)
	}
	state, err := domain.ParseCheckoutState(row.State)
	if err != nil {
		return nil, fmt.Errorf("checkout %s: %w", id, err)
	}
	return domain.RehydrateCheckoutProcess(domain.CheckoutProcessState{
		OrderID: domain.NewOrderID(row.OrderID), ShowID: domain.NewShowID(row.ShowID), State: state, Version: int(row.Version),
	}), nil
}

// Save implements application.CheckoutProcessRepository.
func (r *CheckoutProcessRepository) Save(ctx context.Context, p *domain.CheckoutProcess) error {
	q := sqlcgen.New(pgplatform.Conn(ctx, r.pool))
	defer p.PullEvents()
	var (
		n   int64
		err error
	)
	if p.Version() == 0 {
		n, err = q.InsertCheckoutProcess(ctx, sqlcgen.InsertCheckoutProcessParams{
			OrderID: p.OrderID().UUID(), ShowID: p.ShowID().UUID(), State: p.State().String(),
		})
	} else {
		n, err = q.UpdateCheckoutProcess(ctx, sqlcgen.UpdateCheckoutProcessParams{
			OrderID: p.OrderID().UUID(), State: p.State().String(), ExpectedVersion: int32(p.Version()),
		})
	}
	if err != nil {
		return fmt.Errorf("save checkout %s: %w", p.OrderID(), err)
	}
	if n == 0 {
		return fmt.Errorf("%w: checkout %s", application.ErrConcurrentModification, p.OrderID())
	}
	return nil
}
