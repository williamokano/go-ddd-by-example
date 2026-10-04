// Package httpapi is Show's HTTP driving adapter. There is no login (Chapter
// 1): every request acting for a promoter names them in its body.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/williamokano/go-ddd-by-example/internal/show/application"
	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

type (
	draftShow interface {
		Handle(context.Context, application.DraftShow) (domain.ShowID, error)
	}
	priceShow interface {
		Handle(context.Context, application.PriceShow) error
	}
	publishShow interface {
		Handle(context.Context, application.PublishShow) error
	}
	cancelShow interface {
		Handle(context.Context, application.CancelShow) error
	}
)

// UseCases are the driving ports the routes call.
type UseCases struct {
	Draft   draftShow
	Price   priceShow
	Publish publishShow
	Cancel  cancelShow
	Queries application.ShowQueries
}

// Routes returns the Show API:
//
//	POST /shows                       DraftShow    201 + Location
//	POST /shows/{id}/prices           PriceShow    204
//	POST /shows/{id}/publication      PublishShow  204
//	POST /shows/{id}/cancellation     CancelShow   204
//	GET  /shows/{id}                  ShowQueries  200
func Routes(uc UseCases, logger *slog.Logger) http.Handler {
	h := &handlers{uc: uc, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /shows", h.draft)
	mux.HandleFunc("POST /shows/{id}/prices", h.price)
	mux.HandleFunc("POST /shows/{id}/publication", h.publish)
	mux.HandleFunc("POST /shows/{id}/cancellation", h.cancel)
	mux.HandleFunc("GET /shows/{id}", h.get)
	return mux
}
