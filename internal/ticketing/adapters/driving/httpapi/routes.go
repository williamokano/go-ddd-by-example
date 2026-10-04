// Package httpapi is Ticketing's HTTP driving adapter. There is no login
// (Chapter 1): every request acting for a customer names them in its body.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

type (
	holdSeats interface {
		Handle(context.Context, application.HoldSeats) (application.HoldResult, error)
	}
	releaseHold interface {
		Handle(context.Context, application.ReleaseHold) error
	}
	checkout interface {
		Handle(context.Context, application.Checkout) (application.CheckoutResult, error)
	}
	checkIn interface {
		Handle(context.Context, application.CheckIn) error
	}
	orderQueries interface {
		Get(context.Context, domain.OrderID) (application.OrderView, error)
	}
)

// UseCases are the driving ports the routes call.
type UseCases struct {
	Hold     holdSeats
	Release  releaseHold
	Seats    application.SeatQueries
	Checkout checkout
	Orders   orderQueries
	CheckIn  checkIn
}

// Routes returns the Ticketing API:
//
//	POST   /shows/{id}/holds   HoldSeats    201
//	DELETE /holds/{id}         ReleaseHold  204
//	GET    /shows/{id}/seats   SeatQueries  200
//	POST   /orders             Checkout     201 + Location
//	GET    /orders/{id}        OrderQueries 200
//	POST   /tickets/{code}/check-in CheckIn 204
func Routes(uc UseCases, logger *slog.Logger) http.Handler {
	h := &handlers{uc: uc, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /shows/{id}/holds", h.hold)
	mux.HandleFunc("DELETE /holds/{id}", h.release)
	mux.HandleFunc("GET /shows/{id}/seats", h.seats)
	mux.HandleFunc("POST /orders", h.placeOrder)
	mux.HandleFunc("GET /orders/{id}", h.getOrder)
	mux.HandleFunc("POST /tickets/{code}/check-in", h.checkIn)
	return mux
}

// Patterns lists the routes, so the composition root can mount exactly
// these next to other contexts' APIs on one mux.
var Patterns = []string{
	"POST /shows/{id}/holds", "DELETE /holds/{id}", "GET /shows/{id}/seats", "POST /orders", "GET /orders/{id}",
	"POST /tickets/{code}/check-in",
}
