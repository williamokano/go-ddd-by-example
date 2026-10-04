// Package config parses the environment once, into a typed struct that main
// passes down. Nothing below main reads environment variables.
package config

import (
	"errors"
	"fmt"
	"log/slog"
)

// Config is the whole runtime configuration (Chapter 6 → Configuration).
type Config struct {
	HTTPAddr    string     // HTTP_ADDR, default :8080
	DatabaseURL string     // DATABASE_URL, required
	LogLevel    slog.Level // LOG_LEVEL: debug, info, warn, error; default info
}

// Load builds the Config from getenv (os.Getenv in main, a map in tests).
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		HTTPAddr:    or(getenv("HTTP_ADDR"), ":8080"),
		DatabaseURL: getenv("DATABASE_URL"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("config: DATABASE_URL is required")
	}
	if err := cfg.LogLevel.UnmarshalText([]byte(or(getenv("LOG_LEVEL"), "info"))); err != nil {
		return Config{}, fmt.Errorf("config: LOG_LEVEL: %w", err)
	}
	return cfg, nil
}

func or(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
