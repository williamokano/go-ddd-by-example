package httpx_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/platform/httpx"
)

type payload struct {
	Name string `json:"name"`
}

func TestDecodeJSON(t *testing.T) {
	t.Run("decodes a valid body", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Coliseu"}`))
		var got payload

		ok := httpx.DecodeJSON(w, r, &got)

		if !ok || got.Name != "Coliseu" {
			t.Errorf("DecodeJSON() = %v, %+v; want true, Coliseu", ok, got)
		}
	})

	tests := map[string]string{
		"malformed JSON":  `{"name":`,
		"unknown field":   `{"name":"x","capacity":3}`,
		"trailing data":   `{"name":"x"} {"name":"y"}`,
		"body over 1 MiB": `{"name":"` + strings.Repeat("x", 1<<20) + `"}`,
	}
	for name, body := range tests {
		t.Run("rejects "+name+" with a 400 problem", func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
			var got payload

			ok := httpx.DecodeJSON(w, r, &got)

			if ok {
				t.Fatal("DecodeJSON() = true, want false")
			}
			assertProblem(t, w, http.StatusBadRequest)
		})
	}
}

func assertProblem(t *testing.T, w *httptest.ResponseRecorder, status int) httpx.Problem {
	t.Helper()
	if w.Code != status {
		t.Errorf("status = %d, want %d", w.Code, status)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}
	var p httpx.Problem
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatalf("body is not a problem: %v (%s)", err, w.Body)
	}
	if p.Status != status || p.Title == "" {
		t.Errorf("problem = %+v, want status %d and a title", p, status)
	}
	return p
}
