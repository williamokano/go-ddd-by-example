//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres/pgtest"
)

// scratch creates a private schema with an inbox and a "writes" table that
// stands in for a handler's own writes.
func scratch(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	schema := "tx_" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
	_, err := pool.Exec(context.Background(), `
		CREATE SCHEMA `+schema+`;
		CREATE TABLE `+schema+`.writes (v TEXT NOT NULL);
		CREATE TABLE `+schema+`.inbox (
			consumer TEXT NOT NULL, event_id TEXT NOT NULL,
			processed_at TIMESTAMPTZ NOT NULL DEFAULT now(), PRIMARY KEY (consumer, event_id))`)
	if err != nil {
		t.Fatal(err)
	}
	return schema
}

func count(t *testing.T, pool *pgxpool.Pool, table string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM `+table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// write is a "repository": it opens its own transaction with postgres.Begin,
// which joins the ambient one when there is one.
func write(ctx context.Context, pool *pgxpool.Pool, schema, v string) error {
	tx, err := postgres.Begin(ctx, pool)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO `+schema+`.writes VALUES ($1)`, v); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}

func TestWithinTx_CommitsTheInboxAndTheHandlersWritesTogether(t *testing.T) {
	pool := pgtest.New(t)
	schema := scratch(t, pool)
	inbox := postgres.NewInbox(schema, "notifications")
	txm := postgres.NewTxManager(pool)

	err := txm.WithinTx(context.Background(), func(ctx context.Context) error {
		if first, err := inbox.Claim(ctx, "evt-1"); err != nil || !first {
			t.Fatalf("Claim = %v, %v; want true", first, err)
		}
		return write(ctx, pool, schema, "a")
	})

	if err != nil {
		t.Fatal(err)
	}
	if count(t, pool, schema+".inbox") != 1 || count(t, pool, schema+".writes") != 1 {
		t.Error("the inbox row and the write did not both commit")
	}
}

func TestWithinTx_RollsTheInboxAndTheHandlersWritesBackTogether(t *testing.T) {
	pool := pgtest.New(t)
	schema := scratch(t, pool)
	inbox := postgres.NewInbox(schema, "notifications")
	boom := errors.New("email provider down")

	err := postgres.NewTxManager(pool).WithinTx(context.Background(), func(ctx context.Context) error {
		if _, err := inbox.Claim(ctx, "evt-1"); err != nil {
			return err
		}
		if err := write(ctx, pool, schema, "a"); err != nil {
			return err
		}
		return boom
	})

	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the handler's error", err)
	}
	if count(t, pool, schema+".inbox") != 0 || count(t, pool, schema+".writes") != 0 {
		t.Error("a failed handler left rows behind: the redelivery would be skipped")
	}
}

func TestInbox_ASecondDeliveryIsNotFirst(t *testing.T) {
	pool := pgtest.New(t)
	schema := scratch(t, pool)
	inbox := postgres.NewInbox(schema, "notifications")
	txm := postgres.NewTxManager(pool)
	claim := func(consumer *postgres.Inbox) bool {
		var first bool
		if err := txm.WithinTx(context.Background(), func(ctx context.Context) (err error) {
			first, err = consumer.Claim(ctx, "evt-1")
			return err
		}); err != nil {
			t.Fatal(err)
		}
		return first
	}

	if !claim(inbox) {
		t.Fatal("first delivery: Claim = false")
	}
	if claim(inbox) {
		t.Error("second delivery: Claim = true, want false (skip it)")
	}
	if !claim(postgres.NewInbox(schema, "another-consumer")) {
		t.Error("another consumer's first delivery: Claim = false; the inbox is per consumer")
	}
}

func TestInbox_ClaimOutsideATransactionIsAnError(t *testing.T) {
	pool := pgtest.New(t)
	if _, err := postgres.NewInbox(scratch(t, pool), "c").Claim(context.Background(), "evt-1"); err == nil {
		t.Error("err = nil: a claim that commits on its own would skip a redelivery after a failed handler")
	}
}
