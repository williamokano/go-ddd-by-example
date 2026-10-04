//go:build integration

package kafka_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/platform/kafka"
	"github.com/williamokano/go-ddd-by-example/internal/platform/kafka/kafkatest"
	"github.com/williamokano/go-ddd-by-example/internal/platform/outbox"
)

func TestMain(m *testing.M) { os.Exit(kafkatest.Main(m)) }

func TestProducer_PublishesTheEnvelopeKeyedByAggregate(t *testing.T) {
	topic := kafkatest.Topic(t)
	producer, err := kafka.NewProducer(kafkatest.Brokers(t))
	if err != nil {
		t.Fatal(err)
	}
	defer producer.Close()
	msg := outbox.Message{
		EventID: uuid.New(), Topic: topic, Key: "venue-42", Type: "venue.activated.v1",
		Payload: json.RawMessage(`{"venue_id":"venue-42"}`), OccurredAt: time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC),
		CorrelationID: "purchase-42", CausationID: "evt-9",
		TraceParent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
	}

	if err := producer.Publish(context.Background(), []outbox.Message{msg}); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	rec := kafkatest.Consume(t, topic, 1)[0]
	if string(rec.Key) != "venue-42" {
		t.Errorf("key = %q, want the aggregate id", rec.Key)
	}
	headers := map[string]string{}
	for _, h := range rec.Headers {
		headers[h.Key] = string(h.Value)
	}
	if headers["event_type"] != "venue.activated.v1" || headers["event_id"] != msg.EventID.String() || headers["traceparent"] != msg.TraceParent {
		t.Errorf("headers = %v", headers)
	}
	var env kafka.Envelope
	if err := json.Unmarshal(rec.Value, &env); err != nil {
		t.Fatal(err)
	}
	if env.EventID != msg.EventID.String() || env.AggregateID != "venue-42" || string(env.Payload) != `{"venue_id":"venue-42"}` ||
		env.CorrelationID != "purchase-42" || env.CausationID != "evt-9" {
		t.Errorf("envelope = %+v", env)
	}
}
