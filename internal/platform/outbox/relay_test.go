//go:build integration

package outbox_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/williamokano/go-ddd-by-example/internal/platform/outbox"
	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres/pgtest"
	"github.com/williamokano/go-ddd-by-example/internal/platform/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }

var discard = slog.New(slog.NewTextHandler(io.Discard, nil))

// newOutbox creates a private schema with an outbox table, so each test sees
// only its own rows.
func newOutbox(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	schema := "relay_" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
	_, err := pool.Exec(context.Background(), fmt.Sprintf(`
		CREATE SCHEMA %[1]s;
		CREATE TABLE %[1]s.outbox (
			id BIGSERIAL PRIMARY KEY, event_id UUID NOT NULL UNIQUE, topic TEXT NOT NULL,
			msg_key TEXT NOT NULL, event_type TEXT NOT NULL, payload JSONB NOT NULL,
			occurred_at TIMESTAMPTZ NOT NULL, published_at TIMESTAMPTZ,
			correlation_id TEXT NOT NULL DEFAULT '', causation_id TEXT NOT NULL DEFAULT '',
			trace_parent TEXT NOT NULL DEFAULT '')`, schema))
	if err != nil {
		t.Fatal(err)
	}
	return schema
}

func write(t *testing.T, pool *pgxpool.Pool, schema string, types ...string) {
	t.Helper()
	ctx := context.Background()
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		var msgs []outbox.Message
		for _, typ := range types {
			msgs = append(msgs, outbox.Message{
				EventID: uuid.New(), Topic: "venue.events", Key: "k", Type: typ,
				Payload: json.RawMessage(`{}`), OccurredAt: time.Now(),
			})
		}
		return outbox.Write(ctx, tx, schema, msgs)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func unpublished(t *testing.T, pool *pgxpool.Pool, schema string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM `+schema+`.outbox WHERE published_at IS NULL`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// recorder is a fake Publisher that records what it was given.
type recorder struct {
	mu   sync.Mutex
	sent []outbox.Message
	err  error
}

func (r *recorder) Publish(_ context.Context, msgs []outbox.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return r.err
	}
	r.sent = append(r.sent, msgs...)
	return nil
}

func (r *recorder) types() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	for _, m := range r.sent {
		out = append(out, m.Type)
	}
	return out
}

func TestRelay_PublishesInOrderAndMarksPublished(t *testing.T) {
	pool := pgtest.New(t)
	schema := newOutbox(t, pool)
	write(t, pool, schema, "a.v1", "b.v1", "c.v1")
	pub := &recorder{}

	n, err := outbox.NewRelay(pool, schema, pub, discard).Tick(context.Background())

	if err != nil || n != 3 {
		t.Fatalf("Tick() = %d, %v; want 3, nil", n, err)
	}
	if got := strings.Join(pub.types(), ","); got != "a.v1,b.v1,c.v1" {
		t.Errorf("published %s, want a.v1,b.v1,c.v1 (id order)", got)
	}
	if left := unpublished(t, pool, schema); left != 0 {
		t.Errorf("%d rows still unpublished", left)
	}
}

func TestRelay_APublisherErrorLeavesRowsForTheNextTick(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.New(t)
	schema := newOutbox(t, pool)
	write(t, pool, schema, "a.v1", "b.v1")
	pub := &recorder{err: errors.New("kafka down")}
	relay := outbox.NewRelay(pool, schema, pub, discard)

	if _, err := relay.Tick(ctx); err == nil {
		t.Fatal("Tick() error = nil, want the publisher's error")
	}
	if left := unpublished(t, pool, schema); left != 2 {
		t.Fatalf("%d rows unpublished after a failed tick, want 2", left)
	}

	pub.err = nil
	n, err := relay.Tick(ctx)

	if err != nil || n != 2 || unpublished(t, pool, schema) != 0 {
		t.Errorf("retry Tick() = %d, %v; want 2 rows published", n, err)
	}
}

// blocking publishes only once released, so a second relay runs while the
// first one still holds its rows.
type blocking struct {
	recorder
	started chan struct{}
	release chan struct{}
}

func (b *blocking) Publish(ctx context.Context, msgs []outbox.Message) error {
	close(b.started)
	<-b.release
	return b.recorder.Publish(ctx, msgs)
}

func TestRelay_TwoRelaysNeverPublishTheSameRow(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.New(t)
	schema := newOutbox(t, pool)
	write(t, pool, schema, "a.v1", "b.v1", "c.v1")
	slow := &blocking{started: make(chan struct{}), release: make(chan struct{})}
	fast := &recorder{}

	var wg sync.WaitGroup
	wg.Go(func() {
		if _, err := outbox.NewRelay(pool, schema, slow, discard).Tick(ctx); err != nil {
			t.Error(err)
		}
	})
	<-slow.started // the slow relay holds the rows (FOR UPDATE)
	n, err := outbox.NewRelay(pool, schema, fast, discard).Tick(ctx)
	close(slow.release)
	wg.Wait()

	if err != nil || n != 0 {
		t.Errorf("second relay Tick() = %d, %v; want 0 (rows are locked, SKIP LOCKED)", n, err)
	}
	if got := len(slow.types()) + len(fast.types()); got != 3 {
		t.Errorf("published %d messages in total, want exactly 3", got)
	}
}

// The repository's Save passes its ctx to Write: that is how the IDs reach
// the outbox without the domain knowing about them (8.3).
func TestWrite_StampsTheContextsIDsAndTheRelayCarriesThem(t *testing.T) {
	pool := pgtest.New(t)
	schema := newOutbox(t, pool)
	ctx := trace.WithCausationID(trace.WithCorrelationID(context.Background(), "purchase-42"), "evt-9")
	otel.SetTextMapPropagator(propagation.TraceContext{})
	ctx = oteltrace.ContextWithSpanContext(ctx, oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID: oteltrace.TraceID{0x4b, 0xf9}, SpanID: oteltrace.SpanID{0x01}, TraceFlags: oteltrace.FlagsSampled,
	}))
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		return outbox.Write(ctx, tx, schema, []outbox.Message{{
			EventID: uuid.New(), Topic: "t", Key: "k", Type: "a.v1", Payload: json.RawMessage(`{}`), OccurredAt: time.Now(),
		}})
	})
	if err != nil {
		t.Fatal(err)
	}
	pub := &recorder{}

	if _, err := outbox.NewRelay(pool, schema, pub, discard).Tick(context.Background()); err != nil {
		t.Fatal(err)
	}

	if len(pub.sent) != 1 || pub.sent[0].CorrelationID != "purchase-42" || pub.sent[0].CausationID != "evt-9" {
		t.Errorf("published %+v, want correlation purchase-42 and causation evt-9", pub.sent)
	}
	// 9.7: the writer's span travels too, as a W3C traceparent.
	if want := "00-4bf90000000000000000000000000000-0100000000000000-01"; len(pub.sent) == 1 && pub.sent[0].TraceParent != want {
		t.Errorf("traceparent = %q, want %q", pub.sent[0].TraceParent, want)
	}
}
