package httpapi_test

import (
	"encoding/json"
	"errors"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

var update = flag.Bool("update", false, "rewrite golden files")

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	return w
}

func TestRegisterVenue(t *testing.T) {
	t.Run("maps the JSON to the command, answers 201 with a Location", func(t *testing.T) {
		s := &stubs{}

		w := do(t, s.handler(), http.MethodPost, "/venues",
			`{"name":"Coliseu","street":"Rua Portas de Santo Antão 96","city":"Lisboa","country":"PT"}`)

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201 (%s)", w.Code, w.Body)
		}
		want := application.RegisterVenue{Name: "Coliseu", Street: "Rua Portas de Santo Antão 96", City: "Lisboa", Country: "PT"}
		if s.registered == nil || *s.registered != want {
			t.Errorf("command = %+v, want %+v", s.registered, want)
		}
		if got := w.Header().Get("Location"); got != "/venues/"+venueID.String() {
			t.Errorf("Location = %q", got)
		}
		if !strings.Contains(w.Body.String(), venueID.String()) {
			t.Errorf("body %s lacks the id", w.Body)
		}
	})

	t.Run("malformed JSON is a 400 and the use case is not called", func(t *testing.T) {
		s := &stubs{}

		w := do(t, s.handler(), http.MethodPost, "/venues", `{"name":`)

		if w.Code != http.StatusBadRequest || s.registered != nil {
			t.Errorf("status = %d, called = %v; want 400, not called", w.Code, s.registered != nil)
		}
	})
}

func TestAddSection(t *testing.T) {
	s := &stubs{}

	w := do(t, s.handler(), http.MethodPost, "/venues/"+venueID.String()+"/sections",
		`{"code":"orch","name":"Orchestra","kind":"seated","rows":[{"label":"A","seats":20,"accessibleSeats":[1,2]}]}`)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (%s)", w.Code, w.Body)
	}
	want := application.AddSection{
		VenueID: venueID.String(), Code: "orch", Name: "Orchestra", Kind: "seated",
		Rows: []application.RowSpec{{Label: "A", Seats: 20, Accessible: []int{1, 2}}},
	}
	if diff := cmp.Diff(&want, s.added); diff != "" {
		t.Errorf("command mismatch (-want +got):\n%s", diff)
	}
}

func TestLifecycle(t *testing.T) {
	s := &stubs{}
	h := s.handler()

	activate := do(t, h, http.MethodPost, "/venues/"+venueID.String()+"/activation", "")
	retire := do(t, h, http.MethodPost, "/venues/"+venueID.String()+"/retirement", "")

	if activate.Code != http.StatusNoContent || s.activated == nil || s.activated.VenueID != venueID.String() {
		t.Errorf("activation: status %d, command %+v", activate.Code, s.activated)
	}
	if retire.Code != http.StatusNoContent || s.retired == nil || s.retired.VenueID != venueID.String() {
		t.Errorf("retirement: status %d, command %+v", retire.Code, s.retired)
	}
}

func TestErrorMapping(t *testing.T) {
	tests := []struct {
		err        error
		wantStatus int
		wantTitle  string
	}{
		{domain.ErrInvalidVenueID, 422, "Invalid venue id"},
		{domain.ErrInvalidVenueName, 422, "Invalid venue name"},
		{domain.ErrInvalidAddress, 422, "Invalid address"},
		{domain.ErrInvalidSectionCode, 422, "Invalid section code"},
		{domain.ErrInvalidSection, 422, "Invalid section"},
		{domain.ErrInvalidRow, 422, "Invalid row"},
		{domain.ErrDuplicateRowLabel, 422, "Duplicate row label"},
		{application.ErrVenueNotFound, 404, "Venue not found"},
		{domain.ErrDuplicateSectionCode, 409, "Duplicate section code"},
		{domain.ErrVenueNotDraft, 409, "Venue is not a draft"},
		{domain.ErrVenueHasNoSections, 409, "Venue has no sections"},
		{domain.ErrInvalidVenueTransition, 409, "Invalid venue transition"},
		{application.ErrConcurrentModification, 409, "Concurrent modification"},
		{errors.New("disk on fire"), 500, "Internal error"},
	}
	for _, tt := range tests {
		t.Run(tt.wantTitle, func(t *testing.T) {
			s := &stubs{err: wrap(tt.err)}

			w := do(t, s.handler(), http.MethodPost, "/venues/"+venueID.String()+"/activation", "")

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			var p struct{ Title string }
			_ = json.Unmarshal(w.Body.Bytes(), &p)
			if p.Title != tt.wantTitle {
				t.Errorf("title = %q, want %q", p.Title, tt.wantTitle)
			}
		})
	}
}

// wrap mimics a use case wrapping the error, as ours do.
func wrap(err error) error { return errors.Join(errors.New("activate venue"), err) }

func TestGetVenue(t *testing.T) {
	s := &stubs{view: application.VenueView{
		ID: venueID.String(), Name: "Coliseu dos Recreios", Street: "Rua Portas de Santo Antão 96",
		City: "Lisboa", Country: "PT", Status: "active", Capacity: 520,
		Sections: []application.SectionView{
			{Code: "ORCH", Name: "Orchestra", Kind: "seated", Capacity: 20, Rows: []application.RowView{{Label: "A", Seats: 20}}},
			{Code: "FLOOR", Name: "Floor", Kind: "ga", Capacity: 500},
		},
	}}

	w := do(t, s.handler(), http.MethodGet, "/venues/"+venueID.String(), "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	golden(t, "testdata/venue.golden.json", w.Body.Bytes())
}

func TestListVenues(t *testing.T) {
	s := &stubs{view: application.VenueView{ID: venueID.String(), Name: "Coliseu", Status: "active"}}

	w := do(t, s.handler(), http.MethodGet, "/venues?status=active", "")

	if w.Code != http.StatusOK || s.listed != "active" || !strings.Contains(w.Body.String(), venueID.String()) {
		t.Errorf("status = %d, listed %q, body %s", w.Code, s.listed, w.Body)
	}
}

func golden(t *testing.T, path string, got []byte) {
	t.Helper()
	if *update {
		if err := os.WriteFile(path, got, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run with -update to create it)", err)
	}
	if diff := cmp.Diff(string(want), string(got)); diff != "" {
		t.Errorf("response differs from %s (-want +got):\n%s", path, diff)
	}
}
