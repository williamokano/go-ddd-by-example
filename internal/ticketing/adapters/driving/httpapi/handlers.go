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

// returnOrder answers 202: the refund is done, the seats and tickets follow
// through the saga.
func (h *handlers) returnOrder(w http.ResponseWriter, r *http.Request) {
	var req customerRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	if err := h.uc.Return.Handle(r.Context(), application.ReturnOrder{OrderID: r.PathValue("id"), CustomerID: req.CustomerID}); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *handlers) checkIn(w http.ResponseWriter, r *http.Request) {
	var req checkInRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	if err := h.uc.CheckIn.Handle(r.Context(), application.CheckIn{TicketCode: r.PathValue("code"), GateID: req.GateID}); err != nil {
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
		resp[i] = seatResponse{Ref: s.Ref, State: s.State, Price: priceDTO{Amount: s.Amount, Currency: s.Currency}, Accessible: s.Accessible}
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *handlers) placeOrder(w http.ResponseWriter, r *http.Request) {
	var req placeOrderRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	res, err := h.uc.Checkout.Handle(r.Context(), application.Checkout{
		HoldID: req.HoldID, CustomerID: req.CustomerID, ContactEmail: req.ContactEmail,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Location", "/orders/"+res.OrderID.String())
	httpx.WriteJSON(w, http.StatusCreated, placeOrderResponse{ID: res.OrderID.String(), Status: res.Status.String()})
}

func (h *handlers) getOrder(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseOrderID(r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	v, err := h.uc.Orders.Get(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	resp := orderResponse{ID: v.ID, ShowID: v.ShowID, Status: v.Status, Total: priceDTO{Amount: v.Amount, Currency: v.Currency}, Tickets: []ticketResponse{}}
	for _, t := range v.Tickets {
		resp.Tickets = append(resp.Tickets, ticketResponse{Seat: t.Seat, Code: t.Code, Status: t.Status})
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}
