package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/williamokano/go-ddd-by-example/internal/platform/idgen"
	"github.com/williamokano/go-ddd-by-example/internal/platform/outbox"
	pgplatform "github.com/williamokano/go-ddd-by-example/internal/platform/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
)

// EventPublisher implements application.EventPublisher: it writes events no
// aggregate recorded to ticketing.outbox, through the same translation as
// the repositories, joining the caller's transaction when there is one.
type EventPublisher struct {
	pool       *pgxpool.Pool
	newEventID func() uuid.UUID
}

// NewEventPublisher returns a publisher using pool.
func NewEventPublisher(pool *pgxpool.Pool) *EventPublisher {
	return &EventPublisher{pool: pool, newEventID: idgen.UUIDv7{}.New}
}

// Publish implements application.EventPublisher.
func (p *EventPublisher) Publish(ctx context.Context, events ...sharedkernel.DomainEvent) (err error) {
	msgs, err := ToOutboxMessages(events, p.newEventID)
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	tx, err := pgplatform.Begin(ctx, p.pool)
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()
	if err := outbox.Write(ctx, tx, "ticketing", msgs); err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("publish: commit: %w", err)
	}
	return nil
}
