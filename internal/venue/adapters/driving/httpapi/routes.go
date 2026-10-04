// Package httpapi is the Venue context's HTTP driving adapter. It translates
// HTTP into use-case calls and results back into HTTP, and decides nothing:
// every rule stays in the core.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

// The adapter declares only what it consumes: one method per use case, so its
// tests can pass stubs ("accept interfaces, define them where consumed").
type (
	registerVenue interface {
		Handle(context.Context, application.RegisterVenue) (domain.VenueID, error)
	}
	addSection interface {
		Handle(context.Context, application.AddSection) error
	}
	activateVenue interface {
		Handle(context.Context, application.ActivateVenue) error
	}
	retireVenue interface {
		Handle(context.Context, application.RetireVenue) error
	}
)

// UseCases are the driving ports the routes call.
type UseCases struct {
	Register   registerVenue
	AddSection addSection
	Activate   activateVenue
	Retire     retireVenue
	Queries    application.VenueQueries
}

// Routes returns the Venue API:
//
//	POST /venues                     RegisterVenue   201 + Location
//	POST /venues/{id}/sections       AddSection      204
//	POST /venues/{id}/activation     ActivateVenue   204
//	POST /venues/{id}/retirement     RetireVenue     204
//	GET  /venues/{id}                VenueQueries    200
//	GET  /venues?status=active       VenueQueries    200
func Routes(uc UseCases, logger *slog.Logger) http.Handler {
	h := &handlers{uc: uc, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /venues", h.register)
	mux.HandleFunc("POST /venues/{id}/sections", h.addSection)
	mux.HandleFunc("POST /venues/{id}/activation", h.activate)
	mux.HandleFunc("POST /venues/{id}/retirement", h.retire)
	mux.HandleFunc("GET /venues/{id}", h.get)
	mux.HandleFunc("GET /venues", h.list)
	return mux
}
