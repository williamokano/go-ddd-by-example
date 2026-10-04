//go:build integration

package kafka_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/platform/kafka"
	"github.com/williamokano/go-ddd-by-example/internal/platform/kafka/kafkatest"
	"github.com/williamokano/go-ddd-by-example/internal/platform/outbox"
	"github.com/williamokano/go-ddd-by-example/internal/platform/trace"
)

var discard = slog.New(slog.NewTextHandler(io.Discard, nil))

func produce(t *testing.T, topic string, types ...string) {
	t.Helper()
	producer, err := kafka.NewProducer(kafkatest.Brokers(t))
	if err != nil {
		t.Fatal(err)
	}
	defer producer.Close()
	for _, typ := range types {
		msg := outbox.Message{EventID: uuid.New(), Topic: topic, Key: "k", Type: typ, Payload: json.RawMessage(`{}`), OccurredAt: time.Now()}
		if err := producer.Publish(context.Background(), []outbox.Message{msg}); err != nil {
			t.Fatal(err)
		}
	}
}

func config(t *testing.T, topic, group string) kafka.ConsumerConfig {
	t.Helper()
	return kafka.ConsumerConfig{
		Brokers: kafkatest.Brokers(t), Group: group, Topics: []string{topic},
		MaxAttempts: 3, Backoff: 10 * time.Millisecond,
	}
}

// runUntil runs the consumer until done() or a timeout, and returns Run's error.
func runUntil(t *testing.T, cfg kafka.ConsumerConfig, handler kafka.Handler, done func() bool) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	errc := make(chan error, 1)
	go func() { errc <- kafka.Run(ctx, cfg, handler, discard) }()
	for !done() {
		if ctx.Err() != nil {
			t.Fatal("timed out waiting for the consumer")
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	return <-errc
}

type seen struct {
	mu    sync.Mutex
	types []string
}

func (s *seen) add(env kafka.Envelope) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.types = append(s.types, env.EventType)
}

func (s *seen) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.types)
}

func TestRun_DeliversDecodedEnvelopes(t *testing.T) {
	topic := kafkatest.Topic(t)
	produce(t, topic, "venue.activated.v1")
	got := &seen{}

	err := runUntil(t, config(t, topic, uuid.NewString()), func(_ context.Context, env kafka.Envelope) error {
		got.add(env)
		return nil
	}, func() bool { return got.count() == 1 })

	if err != nil {
		t.Errorf("Run() error = %v, want nil after cancel", err)
	}
	if got.types[0] != "venue.activated.v1" {
		t.Errorf("handled %v", got.types)
	}
}

func TestRun_CommitsOnlyAfterSuccess(t *testing.T) {
	topic := kafkatest.Topic(t)
	produce(t, topic, "venue.activated.v1")
	cfg := config(t, topic, uuid.NewString())
	cfg.MaxAttempts = 1000 // never reach the DLQ in this test
	failures := &seen{}

	// First run: the handler keeps failing; we stop the consumer meanwhile.
	_ = runUntil(t, cfg, func(_ context.Context, env kafka.Envelope) error {
		failures.add(env)
		return errors.New("database down")
	}, func() bool { return failures.count() >= 1 })

	// Restart with the same group: the message comes again.
	again := &seen{}
	_ = runUntil(t, cfg, func(_ context.Context, env kafka.Envelope) error {
		again.add(env)
		return nil
	}, func() bool { return again.count() == 1 })
}

func TestRun_PoisonMessagesGoToTheDLQAndConsumptionContinues(t *testing.T) {
	topic := kafkatest.Topic(t)
	produce(t, topic, "poison.v1", "fine.v1")
	attempts := 0
	handled := &seen{}

	_ = runUntil(t, config(t, topic, uuid.NewString()), func(_ context.Context, env kafka.Envelope) error {
		if env.EventType == "poison.v1" {
			attempts++
			return errors.New("cannot handle this")
		}
		handled.add(env)
		return nil
	}, func() bool { return handled.count() == 1 })

	if attempts != 3 {
		t.Errorf("poison message tried %d times, want 3 (MaxAttempts)", attempts)
	}
	dead := kafkatest.Consume(t, topic+".dlq", 1)[0]
	var errHeader string
	for _, h := range dead.Headers {
		if h.Key == "error" {
			errHeader = string(h.Value)
		}
	}
	if errHeader != "cannot handle this" {
		t.Errorf("dlq error header = %q", errHeader)
	}
}

func TestRun_UndecodableMessagesGoStraightToTheDLQ(t *testing.T) {
	topic := kafkatest.Topic(t)
	kafkatest.ProduceRaw(t, topic, "not json")
	produce(t, topic, "fine.v1")
	handled := &seen{}

	_ = runUntil(t, config(t, topic, uuid.NewString()), func(_ context.Context, env kafka.Envelope) error {
		handled.add(env)
		return nil
	}, func() bool { return handled.count() == 1 })

	if dead := kafkatest.Consume(t, topic+".dlq", 1); string(dead[0].Value) != "not json" {
		t.Errorf("dlq value = %q", dead[0].Value)
	}
}

// The runner continues the flow: the consumed event's correlation ID, and the
// event itself as the cause of whatever the handler does (8.3).
func TestRun_PutsTheIDsBackIntoTheContext(t *testing.T) {
	topic := kafkatest.Topic(t)
	producer, err := kafka.NewProducer(kafkatest.Brokers(t))
	if err != nil {
		t.Fatal(err)
	}
	defer producer.Close()
	msg := outbox.Message{
		EventID: uuid.New(), Topic: topic, Key: "k", Type: "a.v1", Payload: json.RawMessage(`{}`),
		OccurredAt: time.Now(), CorrelationID: "purchase-42",
	}
	if err := producer.Publish(context.Background(), []outbox.Message{msg}); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var correlation, causation string
	handled := false
	err = runUntil(t, config(t, topic, "ids"), func(ctx context.Context, _ kafka.Envelope) error {
		mu.Lock()
		defer mu.Unlock()
		correlation, causation, handled = trace.CorrelationID(ctx), trace.CausationID(ctx), true
		return nil
	}, func() bool { mu.Lock(); defer mu.Unlock(); return handled })
	if err != nil {
		t.Fatal(err)
	}
	if correlation != "purchase-42" || causation != msg.EventID.String() {
		t.Errorf("correlation %q, causation %q; want purchase-42 and the event id", correlation, causation)
	}
}
