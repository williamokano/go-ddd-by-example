// Package config parses the environment once, into a typed struct that main
// passes down. Nothing below main reads environment variables.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// Config is the whole runtime configuration (Chapter 6 → Configuration).
type Config struct {
	HTTPAddr    string     // HTTP_ADDR, default :8080
	DatabaseURL string     // DATABASE_URL, required
	LogLevel    slog.Level // LOG_LEVEL: debug, info, warn, error; default info

	KafkaBrokers       []string      // KAFKA_BROKERS, comma-separated; default localhost:9092
	OutboxPollInterval time.Duration // OUTBOX_POLL_INTERVAL, default 200ms
	HoldTTL            time.Duration // HOLD_TTL, default 10m (TKT-4); shorten it in e2e tests
	PaymentFakeMode    string        // PAYMENT_FAKE_MODE: approve | decline | delay:3s; default approve
	HoldSweepInterval  time.Duration // HOLD_SWEEP_INTERVAL, default 5s: how often holds are expired
	SagaStyle          string        // SAGA_STYLE: orchestration (default, 9.4) | choreography (ADR-010)
}

// Load builds the Config from getenv (os.Getenv in main, a map in tests).
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		HTTPAddr:        or(getenv("HTTP_ADDR"), ":8080"),
		DatabaseURL:     getenv("DATABASE_URL"),
		PaymentFakeMode: or(getenv("PAYMENT_FAKE_MODE"), "approve"),
		SagaStyle:       or(getenv("SAGA_STYLE"), "orchestration"),
	}
	if cfg.SagaStyle != "orchestration" && cfg.SagaStyle != "choreography" {
		return Config{}, fmt.Errorf("config: SAGA_STYLE %q: want orchestration or choreography", cfg.SagaStyle)
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("config: DATABASE_URL is required")
	}
	if err := cfg.LogLevel.UnmarshalText([]byte(or(getenv("LOG_LEVEL"), "info"))); err != nil {
		return Config{}, fmt.Errorf("config: LOG_LEVEL: %w", err)
	}
	for b := range strings.SplitSeq(or(getenv("KAFKA_BROKERS"), "localhost:9092"), ",") {
		cfg.KafkaBrokers = append(cfg.KafkaBrokers, strings.TrimSpace(b))
	}
	var err error
	if cfg.OutboxPollInterval, err = duration(getenv, "OUTBOX_POLL_INTERVAL", "200ms"); err != nil {
		return Config{}, err
	}
	if cfg.HoldTTL, err = duration(getenv, "HOLD_TTL", "10m"); err != nil {
		return Config{}, err
	}
	if cfg.HoldSweepInterval, err = duration(getenv, "HOLD_SWEEP_INTERVAL", "5s"); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func duration(getenv func(string) string, key, fallback string) (time.Duration, error) {
	d, err := time.ParseDuration(or(getenv(key), fallback))
	if err != nil {
		return 0, fmt.Errorf("config: %s: %w", key, err)
	}
	return d, nil
}

func or(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
