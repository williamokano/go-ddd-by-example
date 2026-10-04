package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/williamokano/go-ddd-by-example/internal/platform/trace"
)

// Handler handles one message. Returning nil commits it; an error retries it.
// Handlers must be idempotent: delivery is at least once.
type Handler func(ctx context.Context, env Envelope) error

// ConsumerConfig configures a consumer-group runner.
type ConsumerConfig struct {
	Brokers     []string
	Group       string        // one group per consuming context
	Topics      []string      // their <topic>.dlq must exist
	MaxAttempts int           // tries per message before it goes to the DLQ
	Backoff     time.Duration // first retry delay; doubles each attempt
}

// Run consumes until ctx is done, then returns nil. Each message is handled
// in partition order; its offset is committed only after the handler
// succeeds, or after it was parked in <topic>.dlq following MaxAttempts
// failures (a poison message must never block a partition forever).
func Run(ctx context.Context, cfg ConsumerConfig, handle Handler, logger *slog.Logger) error {
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
	backoff := cfg.Backoff
	var err error
	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		if err = handle(ctx, env); err == nil {
			return true
		}
		logger.WarnContext(ctx, "kafka handler", "group", cfg.Group, "event_type", env.EventType,
			"event_id", env.EventID, "attempt", attempt, "error", err)
		if attempt < cfg.MaxAttempts {
			select {
			case <-ctx.Done():
				return false
			case <-time.After(backoff):
			}
			backoff *= 2
		}
	}
	return deadLetter(ctx, client, cfg, logger, rec, err)
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
