// Package telemetry sets up OpenTelemetry tracing (9.7): a tracer provider
// that exports spans over OTLP/HTTP (to Jaeger in Compose), and W3C trace
// context propagation, which HTTP, the outbox and Kafka carry.
package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// TracerName names Stagehand's instrumentation.
const TracerName = "github.com/williamokano/go-ddd-by-example"

// Setup installs the global tracer provider and propagator. endpoint is an
// OTLP/HTTP base URL ("http://jaeger:4318"); empty means spans are created
// (so every request has a trace ID) but not exported. The returned function
// flushes and stops the provider.
func Setup(ctx context.Context, service, endpoint string) (func(context.Context) error, error) {
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(resource.NewSchemaless(semconv.ServiceName(service))),
	}
	if endpoint != "" {
		exporter, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(endpoint+"/v1/traces"))
		if err != nil {
			return nil, fmt.Errorf("telemetry: exporter: %w", err)
		}
		opts = append(opts, sdktrace.WithBatcher(exporter))
	}
	provider := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return provider.Shutdown, nil
}
