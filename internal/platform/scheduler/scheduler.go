// Package scheduler is a driving adapter driven by time: on every tick it
// calls a use case, exactly like an HTTP handler or a Kafka consumer does on
// a request or a message.
package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/platform/trace"
)

// Run calls job once per tick until ctx is done. Errors are logged; the next
// tick tries again. Production passes time.NewTicker(d).C; tests pass their
// own channel, so nothing sleeps. Each tick starts a flow of its own, with a
// fresh correlation ID (8.3).
func Run(ctx context.Context, ticks <-chan time.Time, job func(context.Context) error, logger *slog.Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
			tickCtx := trace.WithCorrelationID(ctx, uuid.NewString())
			if err := job(tickCtx); err != nil && ctx.Err() == nil {
				logger.WarnContext(tickCtx, "scheduled job failed", "error", err)
			}
		}
	}
}
