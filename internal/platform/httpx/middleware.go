package httpx

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/williamokano/go-ddd-by-example/internal/platform/telemetry"
	"github.com/williamokano/go-ddd-by-example/internal/platform/trace"
)

// Middleware wraps a handler.
type Middleware func(http.Handler) http.Handler

// Chain applies middleware so the first one listed is the outermost.
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// Recover turns a panic into a logged 500 problem instead of a dropped
// connection.
func Recover(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					logger.ErrorContext(r.Context(), "panic", "panic", v, "path", r.URL.Path, "request_id", RequestIDFrom(r.Context()))
					WriteProblem(w, http.StatusInternalServerError, "Internal error", "")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

type requestIDKey struct{}

// RequestID propagates the caller's X-Request-ID, or generates one, and puts
// it in the request context and the response header.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

// RequestIDFrom returns the request ID set by RequestID, or "".
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// Tracing makes each request an OpenTelemetry server span (9.7), the child
// of the caller's span when it sends a W3C traceparent.
func Tracing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		ctx, span := otel.Tracer(telemetry.TracerName).Start(ctx, r.Method, oteltrace.WithSpanKind(oteltrace.SpanKindServer))
		defer span.End()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r.WithContext(ctx))
		span.SetAttributes(attribute.String("url.path", r.URL.Path),
			attribute.Int("http.response.status_code", rec.status))
		if rec.status >= http.StatusInternalServerError {
			span.SetStatus(codes.Error, http.StatusText(rec.status))
		}
	})
}

// Correlation propagates the caller's X-Correlation-ID, or starts a new flow
// named after the request's trace (9.7), and puts it in the context
// (platform/trace), where the outbox writer finds it. Unlike the request ID,
// it outlives the request: it travels with every event the request causes.
func Correlation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Correlation-ID")
		if sc := oteltrace.SpanContextFromContext(r.Context()); id == "" && sc.HasTraceID() {
			id = sc.TraceID().String()
		}
		if id == "" {
			id = uuid.NewString()
		}
		w.Header().Set("X-Correlation-ID", id)
		next.ServeHTTP(w, r.WithContext(trace.WithCorrelationID(r.Context(), id)))
	})
}

// AccessLog logs one line per request.
func AccessLog(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			logger.InfoContext(r.Context(), "http",
				"method", r.Method, "path", r.URL.Path, "status", rec.status,
				"duration", time.Since(start), "request_id", RequestIDFrom(r.Context()))
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
