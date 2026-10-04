package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/notifications/adapters/driven/logsender"
	notificationsconsumer "github.com/williamokano/go-ddd-by-example/internal/notifications/adapters/driving/consumer"
	notificationsapp "github.com/williamokano/go-ddd-by-example/internal/notifications/application"
	"github.com/williamokano/go-ddd-by-example/internal/platform/clock"
	"github.com/williamokano/go-ddd-by-example/internal/platform/config"
	"github.com/williamokano/go-ddd-by-example/internal/platform/httpx"
	"github.com/williamokano/go-ddd-by-example/internal/platform/idgen"
	"github.com/williamokano/go-ddd-by-example/internal/platform/kafka"
	"github.com/williamokano/go-ddd-by-example/internal/platform/outbox"
	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/platform/trace"
	showids "github.com/williamokano/go-ddd-by-example/internal/show/adapters/driven/ids"
	showpg "github.com/williamokano/go-ddd-by-example/internal/show/adapters/driven/postgres"
	showconsumer "github.com/williamokano/go-ddd-by-example/internal/show/adapters/driving/consumer"
	showhttp "github.com/williamokano/go-ddd-by-example/internal/show/adapters/driving/httpapi"
	showapp "github.com/williamokano/go-ddd-by-example/internal/show/application"
	showcontracts "github.com/williamokano/go-ddd-by-example/internal/show/contracts"
	ticketingids "github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/ids"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/payment/fakegateway"
	ticketingpg "github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/postgres"
	ticketingconsumer "github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driving/consumer"
	ticketinghttp "github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driving/httpapi"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driving/scheduler"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/sagamsg"
	ticketingapp "github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	ticketingcontracts "github.com/williamokano/go-ddd-by-example/internal/ticketing/contracts"
	"github.com/williamokano/go-ddd-by-example/internal/venue/adapters/driven/ids"
	venuepg "github.com/williamokano/go-ddd-by-example/internal/venue/adapters/driven/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/venue/adapters/driving/httpapi"
	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	venuecontracts "github.com/williamokano/go-ddd-by-example/internal/venue/contracts"
)

// serve is the composition root: the only place that knows every concrete
// type. Plain constructor injection, top to bottom, no container.
func serve(ctx context.Context) error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	logger := slog.New(trace.NewHandler(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel})))

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer pool.Close()

	producer, err := kafka.NewProducer(cfg.KafkaBrokers)
	if err != nil {
		return fmt.Errorf("kafka: %w", err)
	}
	defer producer.Close()

	// Background work, tied to the root context: stops on SIGINT/SIGTERM.
	var background sync.WaitGroup
	defer background.Wait()
	for _, schema := range []string{"venue", "show", "ticketing"} {
		background.Go(func() { outbox.NewRelay(pool, schema, producer, logger).Run(ctx, cfg.OutboxPollInterval) })
	}

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

	// Show: its own repositories, its projection of Venue, its API and its
	// consumer of venue.events (group "show").
	shows := showpg.NewShowRepository(pool)
	layouts := showpg.NewVenueLayouts(pool)
	showAPI := showhttp.Routes(showhttp.UseCases{
		Draft:   showapp.NewDraftShowHandler(shows, layouts, showids.NewShowIDs(idgen.UUIDv7{}), clk),
		Price:   showapp.NewPriceShowHandler(shows, layouts, clk),
		Publish: showapp.NewPublishShowHandler(shows, layouts, clk),
		Cancel:  showapp.NewCancelShowHandler(shows, clk),
		Queries: showpg.NewShowQueries(pool),
	}, logger)
	venueEvents := showconsumer.NewVenueConsumer(
		showapp.NewOnVenueActivatedHandler(layouts),
		showapp.NewOnVenueRetiredHandler(layouts, shows, clk),
		logger,
	)
	ticketingEvents := showconsumer.NewTicketingConsumer(showapp.NewMarkShowSoldOutHandler(shows, clk), logger)
	consume(ctx, &background, cfg, "show", []string{venuecontracts.Topic, ticketingcontracts.Topic},
		byEventPrefix(map[string]kafka.Handler{"venue.": venueEvents.Handle, "ticketing.": ticketingEvents.Handle}), logger)

	// Ticketing: the inventory, its API, and its consumer of show.events.
	paymentMode, err := fakegateway.ParseMode(cfg.PaymentFakeMode)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	gateway := fakegateway.New(paymentMode)
	inventories := ticketingpg.NewInventoryRepository(pool)
	orders := ticketingpg.NewOrderRepository(pool)
	tickets := ticketingpg.NewTicketRepository(pool)
	ticketingIDs := ticketingids.New(idgen.UUIDv7{})
	ticketingAPI := ticketinghttp.Routes(ticketinghttp.UseCases{
		Hold:     ticketingapp.NewHoldSeatsHandler(inventories, ticketingIDs, clk, cfg.HoldTTL),
		Release:  ticketingapp.NewReleaseHoldHandler(inventories, clk),
		Seats:    ticketingpg.NewSeatQueries(pool),
		Checkout: ticketingapp.NewCheckoutHandler(inventories, orders, gateway, ticketingIDs, clk),
		Orders:   ticketingapp.NewOrderQueries(orders, tickets),
	}, logger)
	refund := ticketingapp.NewRefundOrderHandler(orders, gateway, clk)
	saga := ticketingconsumer.NewSagaConsumer(ticketingconsumer.SagaSteps{
		Confirm: ticketingapp.NewConfirmHoldHandler(inventories, clk),
		Issue:   ticketingapp.NewIssueTicketsHandler(orders, tickets, ticketingIDs, clk),
		Refund:  refund,
		Closed:  ticketingapp.NewOnInventoryClosedHandler(tickets, orders, refund, clk),
	}, logger)
	consume(ctx, &background, cfg, "ticketing-saga", []string{sagamsg.Topic}, saga.Handle, logger)

	// A scheduler drives the application too: expire lapsed holds (TKT-4).
	expireHolds := ticketingapp.NewExpireHoldsHandler(inventories, ticketingpg.NewExpiredHolds(pool), clk)
	sweep := time.NewTicker(cfg.HoldSweepInterval)
	defer sweep.Stop()
	background.Go(func() { scheduler.Run(ctx, sweep.C, expireHolds.Handle, logger) })
	showEvents := ticketingconsumer.NewShowConsumer(
		ticketingapp.NewOpenInventoryHandler(inventories, clk),
		ticketingapp.NewCloseInventoryHandler(inventories, clk),
		logger,
	)
	consume(ctx, &background, cfg, "ticketing", []string{showcontracts.Topic}, showEvents.Handle, logger)

	// Notifications: a transaction script per event, emails logged.
	sender := logsender.New(logger)
	notify := notificationsconsumer.NewTicketingConsumer(
		notificationsapp.NewSendTicketsHandler(sender),
		notificationsapp.NewSendRefundHandler(sender),
	)
	consume(ctx, &background, cfg, "notifications", []string{ticketingcontracts.Topic}, notify.Handle, logger)

	mux := http.NewServeMux()
	for _, pattern := range ticketinghttp.Patterns {
		mux.Handle(pattern, ticketingAPI)
	}
	mux.Handle("/venues", venueAPI)
	mux.Handle("/venues/", venueAPI)
	mux.Handle("/shows", showAPI)
	mux.Handle("/shows/", showAPI)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpx.Chain(mux, httpx.RequestID, httpx.Correlation, httpx.Recover(logger), httpx.AccessLog(logger), fakegateway.ModeHeader),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return runServer(ctx, server, logger)
}

// consume runs a consumer group in the background until ctx is done.
func consume(ctx context.Context, wg *sync.WaitGroup, cfg config.Config, group string, topics []string, handle kafka.Handler, logger *slog.Logger) {
	wg.Go(func() {
		err := kafka.Run(ctx, kafka.ConsumerConfig{
			Brokers: cfg.KafkaBrokers, Group: group, Topics: topics, MaxAttempts: 5, Backoff: 100 * time.Millisecond,
		}, handle, logger)
		if err != nil {
			logger.Error("consumer stopped", "group", group, "error", err)
		}
	})
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

// byEventPrefix routes one consumer group's records to the consumer of the
// topic they came from: a context has one group, whatever it listens to.
func byEventPrefix(routes map[string]kafka.Handler) kafka.Handler {
	return func(ctx context.Context, env kafka.Envelope) error {
		for prefix, handle := range routes {
			if strings.HasPrefix(env.EventType, prefix) {
				return handle(ctx, env)
			}
		}
		return nil
	}
}
