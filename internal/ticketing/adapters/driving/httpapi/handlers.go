package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/williamokano/go-ddd-by-example/internal/platform/httpx"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

type handlers struct {
	uc     UseCases
	logger *slog.Logger
}

func (h *handlers) hold(w http.ResponseWriter, r *http.Request) {
	var req holdRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	res, err := h.uc.Hold.Handle(r.Context(), application.HoldSeats{ShowID: r.PathValue("id"), CustomerID: req.CustomerID, Seats: req.Seats})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, holdResponse{HoldID: res.HoldID.String(), ExpiresAt: res.ExpiresAt})
}

func (h *handlers) release(w http.ResponseWriter, r *http.Request) {
	var req customerRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	if err := h.uc.Release.Handle(r.Context(), application.ReleaseHold{HoldID: r.PathValue("id"), CustomerID: req.CustomerID}); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) seats(w http.ResponseWriter, r *http.Request) {
	showID, err := domain.ParseShowID(r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	rows, err := h.uc.Seats.ListSeats(r.Context(), showID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	resp := make([]seatResponse, len(rows))
	for i, s := range rows {
		resp[i] = seatResponse{Ref: s.Ref, State: s.State, Price: priceDTO{Amount: s.Amount, Currency: s.Currency}}
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}
