package config_test

import (
	"log/slog"
	"testing"

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
		want := config.Config{HTTPAddr: ":8080", DatabaseURL: "postgres://x", LogLevel: slog.LevelInfo}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("config mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("overrides", func(t *testing.T) {
		got, err := config.Load(env(map[string]string{
			"DATABASE_URL": "postgres://x", "HTTP_ADDR": ":9000", "LOG_LEVEL": "debug",
		}))

		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if got.HTTPAddr != ":9000" || got.LogLevel != slog.LevelDebug {
			t.Errorf("config = %+v", got)
		}
	})

	for name, vars := range map[string]map[string]string{
		"missing DATABASE_URL": {},
		"invalid LOG_LEVEL":    {"DATABASE_URL": "postgres://x", "LOG_LEVEL": "loud"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := config.Load(env(vars)); err == nil {
				t.Error("Load() error = nil, want an error")
			}
		})
	}
}
