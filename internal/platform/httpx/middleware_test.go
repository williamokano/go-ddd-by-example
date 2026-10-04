package httpx_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
