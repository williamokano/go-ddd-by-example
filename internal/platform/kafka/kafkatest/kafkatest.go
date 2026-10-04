//go:build integration

// Package kafkatest starts one real Kafka (KRaft) per test package and hands
// out fresh topics, so tests never see each other's messages.
//
//	func TestMain(m *testing.M) { os.Exit(kafkatest.Main(m)) }
package kafkatest

import (
	"context"
	"log"
	"testing"

	"github.com/google/uuid"
	tckafka "github.com/testcontainers/testcontainers-go/modules/kafka"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

var brokers []string

// Main starts the container, runs the tests and stops it.
func Main(m *testing.M) int {
	ctx := context.Background()
	container, err := tckafka.Run(ctx, "confluentinc/confluent-local:7.5.0", // the module drives this image (KRaft)
		tckafka.WithClusterID("stagehand-test"))
	if err != nil {
		log.Printf("kafkatest: start kafka: %v", err)
		return 1
	}
	defer func() {
		if err := container.Terminate(context.Background()); err != nil {
			log.Printf("kafkatest: stop kafka: %v", err)
		}
	}()
	brokers, err = container.Brokers(ctx)
	if err != nil {
		log.Printf("kafkatest: brokers: %v", err)
		return 1
	}
	return m.Run()
}

// Brokers returns the package's broker addresses.
func Brokers(t *testing.T) []string {
	t.Helper()
	if len(brokers) == 0 {
		t.Fatal("kafkatest: no kafka; call kafkatest.Main from TestMain")
	}
	return brokers
}

// Topic creates a fresh topic (and its <topic>.dlq) with one partition.
func Topic(t *testing.T) string {
	t.Helper()
	topic := "test-" + uuid.NewString()
	client, err := kgo.NewClient(kgo.SeedBrokers(Brokers(t)...))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	resp, err := kadm.NewClient(client).CreateTopics(context.Background(), 1, 1, nil, topic, topic+".dlq")
	if err == nil {
		err = resp.Error()
	}
	if err != nil {
		t.Fatalf("kafkatest: create topic: %v", err)
	}
	return topic
}

// Consume reads n records from topic, from the beginning.
func Consume(t *testing.T, topic string, n int) []*kgo.Record {
	t.Helper()
	client, err := kgo.NewClient(kgo.SeedBrokers(Brokers(t)...), kgo.ConsumeTopics(topic), kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20e9)
	defer cancel()
	var records []*kgo.Record
	for len(records) < n {
		fetches := client.PollFetches(ctx)
		if ctx.Err() != nil {
			t.Fatalf("kafkatest: got %d of %d records from %s", len(records), n, topic)
		}
		records = append(records, fetches.Records()...)
	}
	return records
}

// ProduceRaw writes value as-is to topic, bypassing the envelope.
func ProduceRaw(t *testing.T, topic, value string) {
	t.Helper()
	client, err := kgo.NewClient(kgo.SeedBrokers(Brokers(t)...))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := client.ProduceSync(context.Background(), &kgo.Record{Topic: topic, Value: []byte(value)}).FirstErr(); err != nil {
		t.Fatal(err)
	}
}

// Count returns how many records topic holds (its end offsets).
func Count(t *testing.T, topic string) int64 {
	t.Helper()
	client, err := kgo.NewClient(kgo.SeedBrokers(Brokers(t)...))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	offsets, err := kadm.NewClient(client).ListEndOffsets(context.Background(), topic)
	if err != nil {
		t.Fatalf("kafkatest: end offsets of %s: %v", topic, err)
	}
	var n int64
	offsets.Each(func(o kadm.ListedOffset) { n += o.Offset })
	return n
}
