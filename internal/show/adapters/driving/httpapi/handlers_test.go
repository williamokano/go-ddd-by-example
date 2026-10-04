package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/show/adapters/driving/httpapi"
	"github.com/williamokano/go-ddd-by-example/internal/show/application"
	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

var update = flag.Bool("update", false, "rewrite golden files")

var (
	showID   = domain.NewShowID(uuid.MustParse("0192f5e0-0000-7000-8000-000000000001"))
	promoter = "0192f5e0-0000-7000-8000-000000000003"
	start    = time.Date(2026, 12, 1, 20, 0, 0, 0, time.UTC)
)

type stubs struct {
	err       error
	drafted   *application.DraftShow
	priced    *application.PriceShow
	published *application.PublishShow
	cancelled *application.CancelShow
	view      application.ShowView
}

type draftStub struct{ s *stubs }

func (d draftStub) Handle(_ context.Context, cmd application.DraftShow) (domain.ShowID, error) {
	d.s.drafted = &cmd
	return showID, d.s.err
}

type priceStub struct{ s *stubs }

func (p priceStub) Handle(_ context.Context, cmd application.PriceShow) error {
	p.s.priced = &cmd
	return p.s.err
}

type publishStub struct{ s *stubs }

func (p publishStub) Handle(_ context.Context, cmd application.PublishShow) error {
	p.s.published = &cmd
	return p.s.err
}

type cancelStub struct{ s *stubs }

func (c cancelStub) Handle(_ context.Context, cmd application.CancelShow) error {
	c.s.cancelled = &cmd
	return c.s.err
}

type queriesStub struct{ s *stubs }

func (q queriesStub) Get(context.Context, domain.ShowID) (application.ShowView, error) {
	return q.s.view, q.s.err
}

func (s *stubs) handler() http.Handler {
	return httpapi.Routes(httpapi.UseCases{
		Draft: draftStub{s}, Price: priceStub{s}, Publish: publishStub{s}, Cancel: cancelStub{s}, Queries: queriesStub{s},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	return w
}

func TestDraftShow(t *testing.T) {
	s := &stubs{}

	w := do(s.handler(), http.MethodPost, "/shows", `{"promoterId":"`+promoter+`","venueId":"v1","title":"Fado",`+
		`"doorsOpen":"2026-12-01T19:00:00Z","startsAt":"2026-12-01T20:00:00Z","endsAt":"2026-12-01T22:00:00Z"}`)

	if w.Code != http.StatusCreated || w.Header().Get("Location") != "/shows/"+showID.String() {
		t.Fatalf("status = %d, Location %q (%s)", w.Code, w.Header().Get("Location"), w.Body)
	}
	want := &application.DraftShow{PromoterID: promoter, VenueID: "v1", Title: "Fado",
		DoorsOpen: start.Add(-time.Hour), StartsAt: start, EndsAt: start.Add(2 * time.Hour)}
	if diff := cmp.Diff(want, s.drafted); diff != "" {
		t.Errorf("command mismatch (-want +got):\n%s", diff)
	}
}

func TestPriceShow(t *testing.T) {
	s := &stubs{}

	w := do(s.handler(), http.MethodPost, "/shows/"+showID.String()+"/prices",
		`{"promoterId":"`+promoter+`","prices":[{"section":"ORCH","amount":4500,"currency":"EUR"}]}`)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d (%s)", w.Code, w.Body)
	}
	want := &application.PriceShow{ShowID: showID.String(), PromoterID: promoter,
		Prices: []application.PriceSpec{{Section: "ORCH", Amount: 4500, Currency: "EUR"}}}
	if diff := cmp.Diff(want, s.priced); diff != "" {
		t.Errorf("command mismatch (-want +got):\n%s", diff)
	}
}

func TestPublicationAndCancellation(t *testing.T) {
	s := &stubs{}
	h := s.handler()

	pub := do(h, http.MethodPost, "/shows/"+showID.String()+"/publication", `{"promoterId":"`+promoter+`"}`)
	cancel := do(h, http.MethodPost, "/shows/"+showID.String()+"/cancellation", `{"promoterId":"`+promoter+`","reason":"rain"}`)

	if pub.Code != http.StatusNoContent || *s.published != (application.PublishShow{ShowID: showID.String(), PromoterID: promoter}) {
		t.Errorf("publication: %d %+v", pub.Code, s.published)
	}
	if cancel.Code != http.StatusNoContent || *s.cancelled != (application.CancelShow{ShowID: showID.String(), PromoterID: promoter, Reason: "rain"}) {
		t.Errorf("cancellation: %d %+v", cancel.Code, s.cancelled)
	}
}

func TestMalformedJSON(t *testing.T) {
	s := &stubs{}

	if w := do(s.handler(), http.MethodPost, "/shows", `{`); w.Code != http.StatusBadRequest || s.drafted != nil {
		t.Errorf("status = %d, called %v; want 400, not called", w.Code, s.drafted != nil)
	}
}

func TestErrorMapping(t *testing.T) {
	tests := []struct {
		err    error
		status int
		title  string
	}{
		{domain.ErrInvalidID, 422, "Invalid id"},
		{domain.ErrInvalidTitle, 422, "Invalid title"},
		{domain.ErrInvalidSchedule, 422, "Invalid schedule"},
		{sharedkernel.ErrInvalidMoney, 422, "Invalid money"},
		{sharedkernel.ErrCurrencyMismatch, 422, "Currency mismatch"},
		{domain.ErrInvalidPriceList, 422, "Invalid price list"},
		{domain.ErrPriceListMismatch, 422, "Price list does not match the venue"},
		{domain.ErrInvalidCancellationReason, 422, "Invalid cancellation reason"},
		{application.ErrVenueUnknown, 422, "Unknown venue"},
		{application.ErrNotPromoter, 403, "Not the show's promoter"},
		{application.ErrShowNotFound, 404, "Show not found"},
		{domain.ErrVenueNotActive, 409, "Venue is not active"},
		{domain.ErrScheduleConflict, 409, "Schedule conflict"},
		{domain.ErrShowNotDraft, 409, "Show is not a draft"},
		{domain.ErrShowNotPriced, 409, "Show is not priced"},
		{domain.ErrInvalidShowTransition, 409, "Invalid show transition"},
		{application.ErrConcurrentModification, 409, "Concurrent modification"},
		{errors.New("boom"), 500, "Internal error"},
	}
	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			s := &stubs{err: errors.Join(errors.New("publish show"), tt.err)}

			w := do(s.handler(), http.MethodPost, "/shows/"+showID.String()+"/publication", `{"promoterId":"`+promoter+`"}`)

			var p struct{ Title string }
			_ = json.Unmarshal(w.Body.Bytes(), &p)
			if w.Code != tt.status || p.Title != tt.title {
				t.Errorf("got %d %q, want %d %q", w.Code, p.Title, tt.status, tt.title)
			}
		})
	}
}

func TestGetShow(t *testing.T) {
	s := &stubs{view: application.ShowView{
		ID: showID.String(), VenueID: "v1", PromoterID: promoter, Title: "Fado Night",
		DoorsOpen: start.Add(-time.Hour), StartsAt: start, EndsAt: start.Add(2 * time.Hour), Status: "published",
		Prices: []application.PriceView{{Section: "FLOOR", Amount: 2500, Currency: "EUR"}},
	}}

	w := do(s.handler(), http.MethodGet, "/shows/"+showID.String(), "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	path := "testdata/show.golden.json"
	if *update {
		_ = os.WriteFile(path, w.Body.Bytes(), 0o600)
	}
	want, _ := os.ReadFile(path)
	if diff := cmp.Diff(string(want), w.Body.String()); diff != "" {
		t.Errorf("response differs from %s (-want +got):\n%s", path, diff)
	}
}
