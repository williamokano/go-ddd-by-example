package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driving/httpapi"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

var (
	holdID   = domain.NewHoldID(uuid.MustParse("0192f5e0-0000-7000-8000-000000000011"))
	customer = "0192f5e0-0000-7000-8000-0000000000c1"
	show     = "0192f5e0-0000-7000-8000-000000000001"
	expires  = time.Date(2026, 11, 1, 20, 10, 0, 0, time.UTC)
)

type stubs struct {
	err      error
	held     *application.HoldSeats
	released *application.ReleaseHold
	seats    []application.SeatRow
}

type holdStub struct{ s *stubs }

func (h holdStub) Handle(_ context.Context, cmd application.HoldSeats) (application.HoldResult, error) {
	h.s.held = &cmd
	return application.HoldResult{HoldID: holdID, ExpiresAt: expires}, h.s.err
}

type releaseStub struct{ s *stubs }

func (r releaseStub) Handle(_ context.Context, cmd application.ReleaseHold) error {
	r.s.released = &cmd
	return r.s.err
}

type seatsStub struct{ s *stubs }

func (q seatsStub) ListSeats(context.Context, domain.ShowID) ([]application.SeatRow, error) {
	return q.s.seats, q.s.err
}

func (s *stubs) handler() http.Handler {
	return httpapi.Routes(httpapi.UseCases{Hold: holdStub{s}, Release: releaseStub{s}, Seats: seatsStub{s}},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	return w
}

func TestHoldSeats(t *testing.T) {
	s := &stubs{}

	w := do(s.handler(), http.MethodPost, "/shows/"+show+"/holds", `{"customerId":"`+customer+`","seats":["ORCH/A/1","ORCH/A/2"]}`)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d (%s)", w.Code, w.Body)
	}
	want := &application.HoldSeats{ShowID: show, CustomerID: customer, Seats: []string{"ORCH/A/1", "ORCH/A/2"}}
	if diff := cmp.Diff(want, s.held); diff != "" {
		t.Errorf("command mismatch (-want +got):\n%s", diff)
	}
	var body struct {
		HoldID    string    `json:"holdId"`
		ExpiresAt time.Time `json:"expiresAt"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.HoldID != holdID.String() || !body.ExpiresAt.Equal(expires) {
		t.Errorf("body = %s", w.Body)
	}
}

func TestReleaseHold(t *testing.T) {
	s := &stubs{}

	w := do(s.handler(), http.MethodDelete, "/holds/"+holdID.String(), `{"customerId":"`+customer+`"}`)

	if w.Code != http.StatusNoContent || *s.released != (application.ReleaseHold{HoldID: holdID.String(), CustomerID: customer}) {
		t.Errorf("status %d, command %+v", w.Code, s.released)
	}
}

func TestListSeats(t *testing.T) {
	s := &stubs{seats: []application.SeatRow{{Ref: "ORCH/A/1", State: "available", Amount: 4500, Currency: "EUR"}}}

	w := do(s.handler(), http.MethodGet, "/shows/"+show+"/seats", "")

	want := `[{"ref":"ORCH/A/1","state":"available","price":{"amount":4500,"currency":"EUR"}}]` + "\n"
	if w.Code != http.StatusOK || w.Body.String() != want {
		t.Errorf("status %d, body %s", w.Code, w.Body)
	}
}

func TestErrorMapping(t *testing.T) {
	tests := []struct {
		err    error
		status int
		title  string
	}{
		{domain.ErrInvalidID, 422, "Invalid id"},
		{domain.ErrInvalidSeatRef, 422, "Invalid seat"},
		{domain.ErrInvalidHoldSize, 422, "Invalid hold size"},
		{domain.ErrDuplicateSeat, 422, "Duplicate seat"},
		{domain.ErrUnknownSeat, 422, "Unknown seat"},
		{domain.ErrNotHoldOwner, 403, "Not your hold"},
		{application.ErrInventoryNotFound, 404, "Inventory not found"},
		{application.ErrHoldNotFound, 404, "Hold not found"},
		{domain.ErrHoldNotFound, 404, "Hold not found"},
		{domain.ErrSeatUnavailable, 409, "Seat unavailable"},
		{domain.ErrCustomerAlreadyHolding, 409, "Customer already holding"},
		{domain.ErrSalesClosed, 409, "Sales closed"},
		{application.ErrConcurrentModification, 409, "Concurrent modification"},
		{errors.New("boom"), 500, "Internal error"},
	}
	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			s := &stubs{err: errors.Join(errors.New("hold seats"), tt.err)}

			w := do(s.handler(), http.MethodPost, "/shows/"+show+"/holds", `{"customerId":"`+customer+`","seats":["ORCH/A/1"]}`)

			var p struct{ Title string }
			_ = json.Unmarshal(w.Body.Bytes(), &p)
			if w.Code != tt.status || p.Title != tt.title {
				t.Errorf("got %d %q, want %d %q", w.Code, p.Title, tt.status, tt.title)
			}
		})
	}
}
