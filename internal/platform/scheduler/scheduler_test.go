package scheduler_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/platform/scheduler"
	"github.com/williamokano/go-ddd-by-example/internal/platform/trace"
)

func TestRun_CallsTheUseCaseOncePerTickAndStopsOnCancel(t *testing.T) {
	ticks := make(chan time.Time)
	var calls atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		scheduler.Run(ctx, ticks, func(context.Context) error {
			calls.Add(1)
			return errors.New("logged, not fatal")
		}, slog.New(slog.NewTextHandler(io.Discard, nil)))
		close(done)
	}()

	ticks <- time.Now()
	ticks <- time.Now()
	ticks <- time.Now()
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not stop on cancel")
	}
	if calls.Load() != 3 {
		t.Errorf("calls = %d, want 3 (one per tick)", calls.Load())
	}
}

// A tick is a flow of its own: what it causes shares one correlation ID (8.3).
func TestRun_EachTickStartsANewFlow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ticks := make(chan time.Time)
	ids := make(chan string)
	go scheduler.Run(ctx, ticks, func(ctx context.Context) error {
		ids <- trace.CorrelationID(ctx)
		return nil
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	defer cancel()

	ticks <- time.Now()
	first := <-ids
	ticks <- time.Now()
	second := <-ids

	if first == "" || second == "" || first == second {
		t.Errorf("correlation IDs %q and %q; want two different, non-empty IDs", first, second)
	}
}
