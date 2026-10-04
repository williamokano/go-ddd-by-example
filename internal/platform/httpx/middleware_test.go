package httpx_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/williamokano/go-ddd-by-example/internal/platform/httpx"
	"github.com/williamokano/go-ddd-by-example/internal/platform/trace"
)

func TestRecover(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	h := httpx.Recover(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/venues", nil))

	assertProblem(t, w, http.StatusInternalServerError)
	if !strings.Contains(logs.String(), "boom") {
		t.Errorf("panic not logged: %q", logs.String())
	}
}

func TestRequestID(t *testing.T) {
	var seen string
	h := httpx.RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = httpx.RequestIDFrom(r.Context())
	}))

	t.Run("keeps the caller's id", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("X-Request-ID", "abc-123")

		h.ServeHTTP(w, r)

		if seen != "abc-123" || w.Header().Get("X-Request-ID") != "abc-123" {
			t.Errorf("id = %q, header = %q; want abc-123", seen, w.Header().Get("X-Request-ID"))
		}
	})

	t.Run("generates one when missing", func(t *testing.T) {
		w := httptest.NewRecorder()

		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

		if seen == "" || w.Header().Get("X-Request-ID") != seen {
			t.Errorf("id = %q, header = %q; want the same generated id", seen, w.Header().Get("X-Request-ID"))
		}
	})
}

func TestAccessLog(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	h := httpx.AccessLog(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/venues", nil))

	for _, want := range []string{"method=POST", "path=/venues", "status=418"} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("access log %q lacks %q", logs.String(), want)
		}
	}
}

func TestCorrelation(t *testing.T) {
	var seen string
	h := httpx.Correlation(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen = trace.CorrelationID(r.Context())
	}))

	t.Run("keeps the caller's id", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("X-Correlation-ID", "purchase-42")

		h.ServeHTTP(w, r)

		if seen != "purchase-42" || w.Header().Get("X-Correlation-ID") != "purchase-42" {
			t.Errorf("id = %q, header = %q; want purchase-42", seen, w.Header().Get("X-Correlation-ID"))
		}
	})

	t.Run("starts a new flow when missing", func(t *testing.T) {
		w := httptest.NewRecorder()

		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

		if seen == "" || w.Header().Get("X-Correlation-ID") != seen {
			t.Errorf("id = %q, header = %q; want the same generated id", seen, w.Header().Get("X-Correlation-ID"))
		}
	})
}

// 9.7: a request is a span, continuing the caller's trace when it sends a
// traceparent; without an X-Correlation-ID, the trace ID is the correlation.
func TestTracing(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder)))
	otel.SetTextMapPropagator(propagation.TraceContext{})
	var correlation string
	h := httpx.Tracing(httpx.Correlation(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		correlation = trace.CorrelationID(r.Context())
	})))
	r := httptest.NewRequest(http.MethodPost, "/orders", nil)
	r.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

	h.ServeHTTP(httptest.NewRecorder(), r)

	spans := recorder.Ended()
	if len(spans) != 1 || spans[0].Name() != "POST" || spans[0].Parent().SpanID().String() != "00f067aa0ba902b7" {
		t.Fatalf("spans = %v, want one POST span child of the caller's", spans)
	}
	if correlation != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("correlation = %q, want the trace ID", correlation)
	}
}
