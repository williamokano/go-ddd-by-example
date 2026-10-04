package telemetry_test

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/williamokano/go-ddd-by-example/internal/platform/telemetry"
)

// Without an endpoint nothing is exported, but spans still get real IDs:
// the correlation ID is the trace ID (9.7), exporter or not.
func TestSetup_WithoutAnEndpointStillTraces(t *testing.T) {
	shutdown, err := telemetry.Setup(context.Background(), "stagehand", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = shutdown(context.Background()) }()

	ctx, span := otel.Tracer("test").Start(context.Background(), "work")
	defer span.End()

	if !span.SpanContext().IsValid() {
		t.Fatal("span has no trace ID")
	}
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	if carrier["traceparent"] == "" {
		t.Error("no W3C traceparent propagated")
	}
}
