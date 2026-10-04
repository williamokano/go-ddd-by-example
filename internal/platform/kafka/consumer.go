package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/williamokano/go-ddd-by-example/internal/platform/telemetry"
	"github.com/williamokano/go-ddd-by-example/internal/platform/trace"
)

// Handler handles one message. Returning nil commits it; an error retries it,
// unless it is Permanent. Handlers must be idempotent: delivery is at least
// once.
type Handler func(ctx context.Context, env Envelope) error

// permanentError marks a failure that retrying cannot fix.
type permanentError struct{ err error }

func (p permanentError) Error() string { return p.err.Error() }
func (p permanentError) Unwrap() error { return p.err }

// Permanent marks err as poison: a message that will never be handled, such as
// a payload that does not decode. The runner parks it in the DLQ at once.
// Anything else is transient (a database outage, a timeout) and is retried
// until it succeeds (8.5).
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return permanentError{err: err}
}

// IsPermanent reports whether err, or an error it wraps, is Permanent.
func IsPermanent(err error) bool {
	var p permanentError
	return errors.As(err, &p)
}

// ConsumerConfig configures a consumer-group runner.
type ConsumerConfig struct {
	Brokers    []string
	Group      string        // one group per consuming context
	Topics     []string      // their <topic>.dlq must exist
	Backoff    time.Duration // first retry delay; doubles each attempt
	MaxBackoff time.Duration // the longest delay between two attempts
}

// Run consumes until ctx is done, then returns nil. Each message is handled
// in partition order; its offset is committed only after the handler
// succeeds, or after a Permanent failure parked it in <topic>.dlq (a poison
// message must never block a partition forever). A transient failure blocks
// the partition instead, retrying with backoff: dead-lettering it would lose
// the event to an outage.
func Run(ctx context.Context, cfg ConsumerConfig, handle Handler, logger *slog.Logger) error {
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = 30 * time.Second
	}
	client, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ConsumerGroup(cfg.Group),
		kgo.ConsumeTopics(cfg.Topics...),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		return fmt.Errorf("kafka consumer %s: %w", cfg.Group, err)
	}
	defer client.Close()

	for {
		fetches := client.PollFetches(ctx)
		if ctx.Err() != nil {
			return nil
		}
		fetches.EachError(func(topic string, partition int32, err error) {
			logger.WarnContext(ctx, "kafka fetch", "group", cfg.Group, "topic", topic, "partition", partition, "error", err)
		})
		for _, rec := range fetches.Records() {
			if !process(ctx, client, cfg, handle, logger, rec) {
				return nil // stopped mid-retry: not committed, so redelivered later
			}
			if err := client.CommitRecords(ctx, rec); err != nil && ctx.Err() == nil {
				logger.WarnContext(ctx, "kafka commit", "group", cfg.Group, "error", err)
			}
		}
	}
}

// process handles rec until it succeeds or is parked in the DLQ. It returns
// false if ctx ended first.
func process(ctx context.Context, client *kgo.Client, cfg ConsumerConfig, handle Handler, logger *slog.Logger, rec *kgo.Record) bool {
	var env Envelope
	if err := json.Unmarshal(rec.Value, &env); err != nil {
		return deadLetter(ctx, client, cfg, logger, rec, fmt.Errorf("decode envelope: %w", err))
	}
	ctx = withTrace(ctx, env)
	ctx, span := startSpan(ctx, cfg, rec, env)
	defer span.End()
	backoff := cfg.Backoff
	for attempt := 1; ; attempt++ {
		err := handle(ctx, env)
		if err == nil {
			return true
		}
		if IsPermanent(err) {
			span.SetStatus(codes.Error, err.Error())
			return deadLetter(ctx, client, cfg, logger, rec, err)
		}
		logger.WarnContext(ctx, "kafka handler", "group", cfg.Group, "event_type", env.EventType,
			"event_id", env.EventID, "attempt", attempt, "retry_in", backoff, "error", err)
		select {
		case <-ctx.Done():
			return false
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, cfg.MaxBackoff)
	}
}

func deadLetter(ctx context.Context, client *kgo.Client, cfg ConsumerConfig, logger *slog.Logger, rec *kgo.Record, cause error) bool {
	dead := &kgo.Record{
		Topic:   rec.Topic + ".dlq",
		Key:     rec.Key,
		Value:   rec.Value,
		Headers: append(rec.Headers, kgo.RecordHeader{Key: "error", Value: []byte(cause.Error())}),
	}
	if err := client.ProduceSync(ctx, dead).FirstErr(); err != nil {
		logger.ErrorContext(ctx, "kafka dlq", "group", cfg.Group, "topic", dead.Topic, "error", err)
		return false
	}
	logger.ErrorContext(ctx, "kafka dead-lettered", "group", cfg.Group, "topic", rec.Topic, "offset", rec.Offset, "error", cause)
	return true
}

// withTrace continues the event's flow: its correlation ID (or the event
// itself, for an event published without one), and the event as the cause of
// whatever the handler does next (8.3).
func withTrace(ctx context.Context, env Envelope) context.Context {
	correlation := env.CorrelationID
	if correlation == "" {
		correlation = env.EventID
	}
	return trace.WithCausationID(trace.WithCorrelationID(ctx, correlation), env.EventID)
}

// startSpan continues the producer's trace from the record's traceparent
// header (9.7): one consumer span per message, however many attempts.
func startSpan(ctx context.Context, cfg ConsumerConfig, rec *kgo.Record, env Envelope) (context.Context, oteltrace.Span) {
	carrier := propagation.MapCarrier{}
	for _, h := range rec.Headers {
		if h.Key == "traceparent" {
			carrier["traceparent"] = string(h.Value)
		}
	}
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
	return otel.Tracer(telemetry.TracerName).Start(ctx, "consume "+env.EventType,
		oteltrace.WithSpanKind(oteltrace.SpanKindConsumer),
		oteltrace.WithAttributes(
			attribute.String("messaging.destination.name", rec.Topic),
			attribute.String("messaging.consumer.group.name", cfg.Group),
			attribute.String("messaging.message.id", env.EventID),
		))
}
