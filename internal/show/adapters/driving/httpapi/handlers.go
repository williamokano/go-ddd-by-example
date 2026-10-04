package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/williamokano/go-ddd-by-example/internal/platform/httpx"
	"github.com/williamokano/go-ddd-by-example/internal/show/application"
	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

type handlers struct {
	uc     UseCases
	logger *slog.Logger
}

func (h *handlers) draft(w http.ResponseWriter, r *http.Request) {
	var req draftShowRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	id, err := h.uc.Draft.Handle(r.Context(), application.DraftShow{
		PromoterID: req.PromoterID, VenueID: req.VenueID, Title: req.Title,
		DoorsOpen: req.DoorsOpen, StartsAt: req.StartsAt, EndsAt: req.EndsAt,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Location", "/shows/"+id.String())
	httpx.WriteJSON(w, http.StatusCreated, createdResponse{ID: id.String()})
}

func (h *handlers) price(w http.ResponseWriter, r *http.Request) {
	var req priceShowRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	cmd := application.PriceShow{ShowID: r.PathValue("id"), PromoterID: req.PromoterID}
	for _, p := range req.Prices {
		cmd.Prices = append(cmd.Prices, application.PriceSpec{Section: p.Section, Amount: p.Amount, Currency: p.Currency})
	}
	h.noContent(w, r, h.uc.Price.Handle(r.Context(), cmd))
}

func (h *handlers) publish(w http.ResponseWriter, r *http.Request) {
	var req promoterRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	h.noContent(w, r, h.uc.Publish.Handle(r.Context(), application.PublishShow{ShowID: r.PathValue("id"), PromoterID: req.PromoterID}))
}

func (h *handlers) cancel(w http.ResponseWriter, r *http.Request) {
	var req cancelShowRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	h.noContent(w, r, h.uc.Cancel.Handle(r.Context(), application.CancelShow{
		ShowID: r.PathValue("id"), PromoterID: req.PromoterID, Reason: req.Reason,
	}))
}

func (h *handlers) get(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseShowID(r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	view, err := h.uc.Queries.Get(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toShowResponse(view))
}

func (h *handlers) noContent(w http.ResponseWriter, r *http.Request, err error) {
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
