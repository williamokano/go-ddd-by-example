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
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("config mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("overrides", func(t *testing.T) {
		got, err := config.Load(env(map[string]string{
			"DATABASE_URL": "postgres://x", "HTTP_ADDR": ":9000", "LOG_LEVEL": "debug",
			"KAFKA_BROKERS": "k1:9092, k2:9092", "OUTBOX_POLL_INTERVAL": "1s",
		}))

		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if got.HTTPAddr != ":9000" || got.LogLevel != slog.LevelDebug || got.OutboxPollInterval != time.Second {
			t.Errorf("config = %+v", got)
		}
		if diff := cmp.Diff([]string{"k1:9092", "k2:9092"}, got.KafkaBrokers); diff != "" {
			t.Errorf("KafkaBrokers mismatch (-want +got):\n%s", diff)
		}
	})

	for name, vars := range map[string]map[string]string{
		"missing DATABASE_URL": {},
		"invalid LOG_LEVEL":    {"DATABASE_URL": "postgres://x", "LOG_LEVEL": "loud"},
		"invalid duration":     {"DATABASE_URL": "postgres://x", "OUTBOX_POLL_INTERVAL": "soon"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := config.Load(env(vars)); err == nil {
				t.Error("Load() error = nil, want an error")
			}
		})
	}
}
