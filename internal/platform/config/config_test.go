package config_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/williamokano/go-ddd-by-example/internal/platform/config"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestLoad(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		got, err := config.Load(env(map[string]string{"DATABASE_URL": "postgres://x"}))

		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		want := config.Config{
			HTTPAddr: ":8080", DatabaseURL: "postgres://x", LogLevel: slog.LevelInfo,
			KafkaBrokers: []string{"localhost:9092"}, OutboxPollInterval: 200 * time.Millisecond,
			HoldTTL: 10 * time.Minute, PaymentFakeMode: "approve", HoldSweepInterval: 5 * time.Second,
			SagaStyle: "orchestration", ShowSweepInterval: time.Minute,
			ContextDatabaseURLs: map[string]string{
				"venue": "postgres://x", "show": "postgres://x", "ticketing": "postgres://x", "notifications": "postgres://x",
			},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("config mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("overrides", func(t *testing.T) {
		got, err := config.Load(env(map[string]string{
			"DATABASE_URL": "postgres://x", "HTTP_ADDR": ":9000", "LOG_LEVEL": "debug",
			"KAFKA_BROKERS": "k1:9092, k2:9092", "OUTBOX_POLL_INTERVAL": "1s", "HOLD_TTL": "30s", "PAYMENT_FAKE_MODE": "decline",
			"SAGA_STYLE": "choreography",
		}))

		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if got.HTTPAddr != ":9000" || got.LogLevel != slog.LevelDebug || got.OutboxPollInterval != time.Second || got.HoldTTL != 30*time.Second || got.PaymentFakeMode != "decline" ||
			got.SagaStyle != "choreography" {
			t.Errorf("config = %+v", got)
		}
		if diff := cmp.Diff([]string{"k1:9092", "k2:9092"}, got.KafkaBrokers); diff != "" {
			t.Errorf("KafkaBrokers mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("each context may connect as its own role (9.6)", func(t *testing.T) {
		got, err := config.Load(env(map[string]string{
			"DATABASE_URL": "postgres://admin", "TICKETING_DATABASE_URL": "postgres://ticketing",
		}))

		if err != nil {
			t.Fatal(err)
		}
		if got.ContextDatabaseURLs["ticketing"] != "postgres://ticketing" || got.ContextDatabaseURLs["venue"] != "postgres://admin" {
			t.Errorf("ContextDatabaseURLs = %v", got.ContextDatabaseURLs)
		}
	})

	for name, vars := range map[string]map[string]string{
		"missing DATABASE_URL": {},
		"invalid LOG_LEVEL":    {"DATABASE_URL": "postgres://x", "LOG_LEVEL": "loud"},
		"invalid duration":     {"DATABASE_URL": "postgres://x", "OUTBOX_POLL_INTERVAL": "soon"},
		"invalid SAGA_STYLE":   {"DATABASE_URL": "postgres://x", "SAGA_STYLE": "telepathy"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := config.Load(env(vars)); err == nil {
				t.Error("Load() error = nil, want an error")
			}
		})
	}
}
