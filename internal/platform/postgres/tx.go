package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX is what sqlc-generated queries run on: a pool or a transaction.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type txKey struct{}

// TxManager lets a use case open a transaction that the repositories it calls
// join (ADR-012). Without one, each Save is its own transaction (ADR-004).
type TxManager struct{ pool *pgxpool.Pool }

// NewTxManager returns a TxManager on pool.
func NewTxManager(pool *pgxpool.Pool) *TxManager { return &TxManager{pool: pool} }

// WithinTx runs fn in a transaction carried by its context: it commits if fn
// returns nil and rolls back otherwise. Inside an ambient transaction, it
// simply joins it.
func (m *TxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()
	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// Begin starts a repository's transaction. Inside an ambient transaction it
// starts a savepoint instead, so the repository's Commit and Rollback still
// work, and the ambient transaction decides what is finally committed.
func Begin(ctx context.Context, pool *pgxpool.Pool) (pgx.Tx, error) {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		nested, err := tx.Begin(ctx)
		if err != nil {
			return nil, fmt.Errorf("savepoint: %w", err)
		}
		return nested, nil
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin: %w", err)
	}
	return tx, nil
}

// Conn returns the ambient transaction, or pool: reads inside a use case's
// transaction see its own uncommitted writes.
func Conn(ctx context.Context, pool *pgxpool.Pool) DBTX {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool
}

// ErrNoTransaction is returned by Inbox.Claim outside WithinTx.
var ErrNoTransaction = errors.New("postgres: no transaction in context")

// Inbox records which events a consumer has processed, in <schema>.inbox, so
// a redelivered event is skipped (8.4). Its claim commits or rolls back with
// the handler's own work, because both run in one transaction.
type Inbox struct {
	table    string
	consumer string
}

// NewInbox returns consumer's inbox in schema.
func NewInbox(schema, consumer string) *Inbox {
	return &Inbox{table: pgx.Identifier{schema, "inbox"}.Sanitize(), consumer: consumer}
}

// Claim records eventID as processed and reports whether this is its first
// delivery. It must run inside WithinTx.
func (i *Inbox) Claim(ctx context.Context, eventID string) (bool, error) {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	if !ok {
		return false, ErrNoTransaction
	}
	tag, err := tx.Exec(ctx,
		`INSERT INTO `+i.table+` (consumer, event_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, i.consumer, eventID)
	if err != nil {
		return false, fmt.Errorf("inbox %s: claim %s: %w", i.consumer, eventID, err)
	}
	return tag.RowsAffected() == 1, nil
}
