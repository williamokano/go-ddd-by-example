package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/williamokano/go-ddd-by-example/internal/platform/outbox"
)

// Producer implements outbox.Publisher on Kafka.
type Producer struct{ client *kgo.Client }

// NewProducer connects to brokers. Every write waits for all in-sync replicas.
func NewProducer(brokers []string) (*Producer, error) {
	client, err := kgo.NewClient(kgo.SeedBrokers(brokers...), kgo.RequiredAcks(kgo.AllISRAcks()))
	if err != nil {
		return nil, fmt.Errorf("kafka producer: %w", err)
	}
	return &Producer{client: client}, nil
}

// Close flushes and disconnects.
func (p *Producer) Close() { p.client.Close() }

// Publish sends msgs and waits for the broker's acknowledgement of all of them.
func (p *Producer) Publish(ctx context.Context, msgs []outbox.Message) error {
	records := make([]*kgo.Record, 0, len(msgs))
	for _, m := range msgs {
		value, err := json.Marshal(Envelope{
			EventID: m.EventID.String(), EventType: m.Type, OccurredAt: m.OccurredAt,
			AggregateID: m.Key, Payload: m.Payload,
		})
		if err != nil {
			return fmt.Errorf("kafka: envelope %s: %w", m.Type, err)
		}
		records = append(records, &kgo.Record{
			Topic: m.Topic,
			Key:   []byte(m.Key),
			Value: value,
			Headers: []kgo.RecordHeader{
				{Key: "event_id", Value: []byte(m.EventID.String())},
				{Key: "event_type", Value: []byte(m.Type)},
				{Key: "occurred_at", Value: []byte(m.OccurredAt.UTC().Format(time.RFC3339Nano))},
				{Key: "aggregate_id", Value: []byte(m.Key)},
			},
		})
	}
	if err := p.client.ProduceSync(ctx, records...).FirstErr(); err != nil {
		return fmt.Errorf("kafka: produce: %w", err)
	}
	return nil
}
