package outbox

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Publisher sends messages to the broker. The relay declares the port; Kafka
// is one adapter (platform/kafka), a fake is another.
type Publisher interface {
	Publish(ctx context.Context, msgs []Message) error
}

// batchSize bounds one tick.
const batchSize = 100

// Relay moves rows from one <schema>.outbox to the publisher, at least once:
// a crash after publishing but before marking the rows publishes them again,
// so consumers must tolerate duplicates.
type Relay struct {
	pool   *pgxpool.Pool
	schema string
	pub    Publisher
	logger *slog.Logger
}

// NewRelay returns a relay for schema's outbox.
func NewRelay(pool *pgxpool.Pool, schema string, pub Publisher, logger *slog.Logger) *Relay {
	return &Relay{pool: pool, schema: schema, pub: pub, logger: logger}
}

// Run ticks every interval until ctx is done. Errors are logged and retried
// on the next tick: the rows simply stay unpublished.
func (r *Relay) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if n, err := r.Tick(ctx); err != nil && ctx.Err() == nil {
			r.logger.WarnContext(ctx, "outbox relay", "schema", r.schema, "error", err)
		} else if n > 0 {
			r.logger.DebugContext(ctx, "outbox relay", "schema", r.schema, "published", n)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Tick publishes one batch of unpublished rows, in id order, and marks them.
// The rows stay locked (FOR UPDATE SKIP LOCKED) until the transaction ends, so
// two relays never publish the same row in the same tick. It returns how many
// rows it published.
func (r *Relay) Tick(ctx context.Context) (n int, err error) {
	table := pgx.Identifier{r.schema, "outbox"}.Sanitize()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("relay %s: begin: %w", r.schema, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	rows, err := tx.Query(ctx, `
		SELECT id, event_id, topic, msg_key, event_type, payload, occurred_at
		FROM `+table+`
		WHERE published_at IS NULL
		ORDER BY id
		LIMIT $1
		FOR UPDATE SKIP LOCKED`, batchSize)
	if err != nil {
		return 0, fmt.Errorf("relay %s: select: %w", r.schema, err)
	}
	var (
		ids  []int64
		msgs []Message
	)
	for rows.Next() {
		var (
			id int64
			m  Message
		)
		if err := rows.Scan(&id, &m.EventID, &m.Topic, &m.Key, &m.Type, &m.Payload, &m.OccurredAt); err != nil {
			rows.Close()
			return 0, fmt.Errorf("relay %s: scan: %w", r.schema, err)
		}
		ids = append(ids, id)
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("relay %s: rows: %w", r.schema, err)
	}
	if len(msgs) == 0 {
		if err := tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("relay %s: commit: %w", r.schema, err)
		}
		return 0, nil
	}

	if err := r.pub.Publish(ctx, msgs); err != nil {
		return 0, fmt.Errorf("relay %s: publish: %w", r.schema, err)
	}
	if _, err := tx.Exec(ctx, `UPDATE `+table+` SET published_at = now() WHERE id = ANY($1)`, ids); err != nil {
		return 0, fmt.Errorf("relay %s: mark published: %w", r.schema, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("relay %s: commit: %w", r.schema, err)
	}
	return len(msgs), nil
}
