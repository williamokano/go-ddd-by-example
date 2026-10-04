package outbox

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Write inserts msgs into <schema>.outbox inside tx, the transaction that also
// saves the aggregate: both commit, or neither does.
func Write(ctx context.Context, tx pgx.Tx, schema string, msgs []Message) error {
	table := pgx.Identifier{schema, "outbox"}.Sanitize()
	for _, m := range msgs {
		_, err := tx.Exec(ctx,
			`INSERT INTO `+table+` (event_id, topic, msg_key, event_type, payload, occurred_at)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			m.EventID, m.Topic, m.Key, m.Type, []byte(m.Payload), m.OccurredAt)
		if err != nil {
			return fmt.Errorf("outbox %s: insert %s: %w", schema, m.Type, err)
		}
	}
	return nil
}
