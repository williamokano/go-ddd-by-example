package httpapi

import (
	"errors"
	"net/http"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"

	"github.com/williamokano/go-ddd-by-example/internal/platform/httpx"
	"github.com/williamokano/go-ddd-by-example/internal/show/application"
	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

// errorStatuses maps core errors to HTTP: invalid input → 422, not the
// promoter → 403, not found → 404, state or conflict → 409, else 500.
var errorStatuses = []struct {
	err    error
	status int
	title  string
}{
	{domain.ErrInvalidID, http.StatusUnprocessableEntity, "Invalid id"},
	{domain.ErrInvalidTitle, http.StatusUnprocessableEntity, "Invalid title"},
	{domain.ErrInvalidSchedule, http.StatusUnprocessableEntity, "Invalid schedule"},
	{sharedkernel.ErrInvalidMoney, http.StatusUnprocessableEntity, "Invalid money"},
	{sharedkernel.ErrCurrencyMismatch, http.StatusUnprocessableEntity, "Currency mismatch"},
	{domain.ErrInvalidPriceList, http.StatusUnprocessableEntity, "Invalid price list"},
	{domain.ErrPriceListMismatch, http.StatusUnprocessableEntity, "Price list does not match the venue"},
	{domain.ErrInvalidCancellationReason, http.StatusUnprocessableEntity, "Invalid cancellation reason"},
	{application.ErrVenueUnknown, http.StatusUnprocessableEntity, "Unknown venue"},
	{application.ErrNotPromoter, http.StatusForbidden, "Not the show's promoter"},
	{application.ErrShowNotFound, http.StatusNotFound, "Show not found"},
	{domain.ErrVenueNotActive, http.StatusConflict, "Venue is not active"},
	{domain.ErrScheduleConflict, http.StatusConflict, "Schedule conflict"},
	{domain.ErrShowNotDraft, http.StatusConflict, "Show is not a draft"},
	{domain.ErrShowNotPriced, http.StatusConflict, "Show is not priced"},
	{domain.ErrInvalidShowTransition, http.StatusConflict, "Invalid show transition"},
	{application.ErrConcurrentModification, http.StatusConflict, "Concurrent modification"},
}

func (h *handlers) writeError(w http.ResponseWriter, r *http.Request, err error) {
	for _, e := range errorStatuses {
		if errors.Is(err, e.err) {
			httpx.WriteProblem(w, e.status, e.title, err.Error())
			return
		}
	}
	h.logger.ErrorContext(r.Context(), "show api", "error", err, "path", r.URL.Path, "request_id", httpx.RequestIDFrom(r.Context()))
	httpx.WriteProblem(w, http.StatusInternalServerError, "Internal error", "")
}
