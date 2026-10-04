package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/platform/clock"
	"github.com/williamokano/go-ddd-by-example/internal/platform/config"
	"github.com/williamokano/go-ddd-by-example/internal/platform/httpx"
	"github.com/williamokano/go-ddd-by-example/internal/platform/idgen"
	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/venue/adapters/driven/ids"
	venuepg "github.com/williamokano/go-ddd-by-example/internal/venue/adapters/driven/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/venue/adapters/driving/httpapi"
	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
)

// serve is the composition root: the only place that knows every concrete
// type. Plain constructor injection, top to bottom, no container.
func serve(ctx context.Context) error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer pool.Close()

	// Driven adapters.
	clk := clock.System{}
	venues := venuepg.NewVenueRepository(pool)
	venueIDs := ids.NewVenueIDs(idgen.UUIDv7{})

	// Use cases, then the driving adapter.
	venueAPI := httpapi.Routes(httpapi.UseCases{
		Register:   application.NewRegisterVenueHandler(venues, venueIDs, clk),
		AddSection: application.NewAddSectionHandler(venues, clk),
		Activate:   application.NewActivateVenueHandler(venues, clk),
		Retire:     application.NewRetireVenueHandler(venues, clk),
		Queries:    venuepg.NewVenueQueries(pool),
	}, logger)

	mux := http.NewServeMux()
	mux.Handle("/venues", venueAPI)
	mux.Handle("/venues/", venueAPI)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpx.Chain(mux, httpx.RequestID, httpx.Recover(logger), httpx.AccessLog(logger)),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return runServer(ctx, server, logger)
}

// runServer serves until ctx is cancelled (SIGINT/SIGTERM), then drains
// in-flight requests for up to 10 seconds.
func runServer(ctx context.Context, server *http.Server, logger *slog.Logger) error {
	errc := make(chan error, 1)
	go func() {
		logger.Info("http listening", "addr", server.Addr)
		errc <- server.ListenAndServe()
	}()
	select {
	case err := <-errc:
		return fmt.Errorf("http: %w", err)
	case <-ctx.Done():
	}
	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}
	if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http: %w", err)
	}
	return nil
}
