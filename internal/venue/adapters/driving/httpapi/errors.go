package httpapi

import (
	"errors"
	"net/http"

	"github.com/williamokano/go-ddd-by-example/internal/platform/httpx"
	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

// errorStatuses is the one place that maps core errors to HTTP: invalid
// input → 422, not found → 404, state or conflict → 409. Anything else is a
// bug or an outage: 500, logged, details hidden.
var errorStatuses = []struct {
	err    error
	status int
	title  string
}{
	{domain.ErrInvalidVenueID, http.StatusUnprocessableEntity, "Invalid venue id"},
	{domain.ErrInvalidVenueName, http.StatusUnprocessableEntity, "Invalid venue name"},
	{domain.ErrInvalidAddress, http.StatusUnprocessableEntity, "Invalid address"},
	{domain.ErrInvalidSectionCode, http.StatusUnprocessableEntity, "Invalid section code"},
	{domain.ErrInvalidSection, http.StatusUnprocessableEntity, "Invalid section"},
	{domain.ErrInvalidRow, http.StatusUnprocessableEntity, "Invalid row"},
	{domain.ErrDuplicateRowLabel, http.StatusUnprocessableEntity, "Duplicate row label"},
	{application.ErrVenueNotFound, http.StatusNotFound, "Venue not found"},
	{domain.ErrDuplicateSectionCode, http.StatusConflict, "Duplicate section code"},
	{domain.ErrVenueNotDraft, http.StatusConflict, "Venue is not a draft"},
	{domain.ErrVenueHasNoSections, http.StatusConflict, "Venue has no sections"},
	{domain.ErrInvalidVenueTransition, http.StatusConflict, "Invalid venue transition"},
	{application.ErrConcurrentModification, http.StatusConflict, "Concurrent modification"},
}

func (h *handlers) writeError(w http.ResponseWriter, r *http.Request, err error) {
	for _, e := range errorStatuses {
		if errors.Is(err, e.err) {
			httpx.WriteProblem(w, e.status, e.title, err.Error())
			return
		}
	}
	h.logger.ErrorContext(r.Context(), "venue api", "error", err, "path", r.URL.Path, "request_id", httpx.RequestIDFrom(r.Context()))
	httpx.WriteProblem(w, http.StatusInternalServerError, "Internal error", "")
}
