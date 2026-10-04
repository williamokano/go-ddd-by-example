package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/williamokano/go-ddd-by-example/internal/platform/httpx"
	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

type handlers struct {
	uc     UseCases
	logger *slog.Logger
}

func (h *handlers) register(w http.ResponseWriter, r *http.Request) {
	var req registerVenueRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	id, err := h.uc.Register.Handle(r.Context(), application.RegisterVenue{
		Name: req.Name, Street: req.Street, City: req.City, Country: req.Country,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Location", "/venues/"+id.String())
	httpx.WriteJSON(w, http.StatusCreated, createdResponse{ID: id.String()})
}

func (h *handlers) addSection(w http.ResponseWriter, r *http.Request) {
	var req addSectionRequest
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	cmd := application.AddSection{
		VenueID: r.PathValue("id"), Code: req.Code, Name: req.Name, Kind: req.Kind, Capacity: req.Capacity,
	}
	for _, row := range req.Rows {
		cmd.Rows = append(cmd.Rows, application.RowSpec{Label: row.Label, Seats: row.Seats})
	}
	if err := h.uc.AddSection.Handle(r.Context(), cmd); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) activate(w http.ResponseWriter, r *http.Request) {
	if err := h.uc.Activate.Handle(r.Context(), application.ActivateVenue{VenueID: r.PathValue("id")}); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) retire(w http.ResponseWriter, r *http.Request) {
	if err := h.uc.Retire.Handle(r.Context(), application.RetireVenue{VenueID: r.PathValue("id")}); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) get(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseVenueID(r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	view, err := h.uc.Queries.Get(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toVenueResponse(view))
}

func (h *handlers) list(w http.ResponseWriter, r *http.Request) {
	views, err := h.uc.Queries.List(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	resp := make([]venueResponse, 0, len(views))
	for _, v := range views {
		resp = append(resp, toVenueResponse(v))
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}
