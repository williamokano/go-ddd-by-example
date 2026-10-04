package trace_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/williamokano/go-ddd-by-example/internal/platform/trace"
)

func TestIDs_TravelInTheContext(t *testing.T) {
	ctx := context.Background()
	if trace.CorrelationID(ctx) != "" || trace.CausationID(ctx) != "" {
		t.Fatal("a bare context has IDs")
	}
	ctx = trace.WithCausationID(trace.WithCorrelationID(ctx, "corr-1"), "evt-9")
	if got := trace.CorrelationID(ctx); got != "corr-1" {
		t.Errorf("CorrelationID = %q", got)
	}
	if got := trace.CausationID(ctx); got != "evt-9" {
		t.Errorf("CausationID = %q", got)
	}
}

func TestHandler_AddsBothIDsToEveryLine(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(trace.NewHandler(slog.NewTextHandler(&buf, nil)))
	ctx := trace.WithCausationID(trace.WithCorrelationID(context.Background(), "corr-1"), "evt-9")

	logger.With("component", "relay").InfoContext(ctx, "hello")

	for _, want := range []string{"correlation_id=corr-1", "causation_id=evt-9", "component=relay"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("log line %q lacks %s", buf.String(), want)
		}
	}
}

func TestHandler_OmitsMissingIDs(t *testing.T) {
	var buf bytes.Buffer
	slog.New(trace.NewHandler(slog.NewTextHandler(&buf, nil))).InfoContext(context.Background(), "hello")
	if strings.Contains(buf.String(), "correlation_id") || strings.Contains(buf.String(), "causation_id") {
		t.Errorf("log line %q has empty IDs", buf.String())
	}
}

// 9.7: a log line names its OpenTelemetry span, so it links to the trace.
func TestHandler_AddsTheSpan(t *testing.T) {
	var buf bytes.Buffer
	ctx := oteltrace.ContextWithSpanContext(context.Background(), oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID: oteltrace.TraceID{0x4b, 0xf9}, SpanID: oteltrace.SpanID{0x01}, TraceFlags: oteltrace.FlagsSampled,
	}))

	slog.New(trace.NewHandler(slog.NewTextHandler(&buf, nil))).InfoContext(ctx, "hello")

	for _, want := range []string{"trace_id=4bf90000000000000000000000000000", "span_id=0100000000000000"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("log line %q lacks %s", buf.String(), want)
		}
	}
}
