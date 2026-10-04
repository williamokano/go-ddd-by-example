package outbox

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/williamokano/go-ddd-by-example/internal/platform/trace"
)

// Write inserts msgs into <schema>.outbox inside tx, the transaction that also
// saves the aggregate: both commit, or neither does. Messages without trace
// IDs get the ones in ctx, so the domain never sees them (8.3).
func Write(ctx context.Context, tx pgx.Tx, schema string, msgs []Message) error {
	table := pgx.Identifier{schema, "outbox"}.Sanitize()
	for _, m := range msgs {
		if m.CorrelationID == "" {
			m.CorrelationID = trace.CorrelationID(ctx)
		}
		if m.CausationID == "" {
			m.CausationID = trace.CausationID(ctx)
		}
		_, err := tx.Exec(ctx,
			`INSERT INTO `+table+` (event_id, topic, msg_key, event_type, payload, occurred_at, correlation_id, causation_id)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			m.EventID, m.Topic, m.Key, m.Type, []byte(m.Payload), m.OccurredAt, m.CorrelationID, m.CausationID)
		if err != nil {
			return fmt.Errorf("outbox %s: insert %s: %w", schema, m.Type, err)
		}
	}
	return nil
}
