package httpapi

import (
	"errors"
	"net/http"

	"github.com/williamokano/go-ddd-by-example/internal/platform/httpx"
	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// errorStatuses maps core errors to HTTP: invalid input → 422, someone
// else's hold → 403, not found → 404, unavailable or conflict → 409.
var errorStatuses = []struct {
	err    error
	status int
	title  string
}{
	{domain.ErrInvalidID, http.StatusUnprocessableEntity, "Invalid id"},
	{domain.ErrInvalidSeatRef, http.StatusUnprocessableEntity, "Invalid seat"},
	{domain.ErrInvalidHoldSize, http.StatusUnprocessableEntity, "Invalid hold size"},
	{domain.ErrDuplicateSeat, http.StatusUnprocessableEntity, "Duplicate seat"},
	{domain.ErrUnknownSeat, http.StatusUnprocessableEntity, "Unknown seat"},
	{sharedkernel.ErrInvalidMoney, http.StatusUnprocessableEntity, "Invalid money"},
	{domain.ErrNotHoldOwner, http.StatusForbidden, "Not your hold"},
	{application.ErrInventoryNotFound, http.StatusNotFound, "Inventory not found"},
	{application.ErrHoldNotFound, http.StatusNotFound, "Hold not found"},
	{domain.ErrHoldNotFound, http.StatusNotFound, "Hold not found"},
	{domain.ErrSeatUnavailable, http.StatusConflict, "Seat unavailable"},
	{domain.ErrCustomerAlreadyHolding, http.StatusConflict, "Customer already holding"},
	{domain.ErrSalesClosed, http.StatusConflict, "Sales closed"},
	{domain.ErrHoldExpired, http.StatusConflict, "Hold expired"},
	{application.ErrConcurrentModification, http.StatusConflict, "Concurrent modification"},
}

func (h *handlers) writeError(w http.ResponseWriter, r *http.Request, err error) {
	for _, e := range errorStatuses {
		if errors.Is(err, e.err) {
			httpx.WriteProblem(w, e.status, e.title, err.Error())
			return
		}
	}
	h.logger.ErrorContext(r.Context(), "ticketing api", "error", err, "path", r.URL.Path, "request_id", httpx.RequestIDFrom(r.Context()))
	httpx.WriteProblem(w, http.StatusInternalServerError, "Internal error", "")
}
