package httpapi_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driving/httpapi"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

type checkoutStub struct {
	got *application.Checkout
	err error
}

func (c *checkoutStub) Handle(_ context.Context, cmd application.Checkout) (application.CheckoutResult, error) {
	c.got = &cmd
	return application.CheckoutResult{OrderID: domain.NewOrderID(uuid.MustParse("0192f5e0-0000-7000-8000-0000000000d1")), Status: domain.Paid}, c.err
}

type orderQueriesStub struct{}

func (orderQueriesStub) Get(context.Context, domain.OrderID) (application.OrderView, error) {
	return application.OrderView{ID: "o1", Status: "fulfilled", Amount: 10494, Subtotal: 9000, Fee: 900, VAT: 594, Currency: "EUR",
		Tickets: []application.TicketView{{Seat: "ORCH/A/1", Code: "ABCD-EFGH-IJKL", Status: "valid"}}}, nil
}

func ordersHandler(c *checkoutStub) http.Handler {
	s := &stubs{}
	return httpapi.Routes(httpapi.UseCases{Hold: holdStub{s}, Release: releaseStub{s}, Seats: seatsStub{s}, Checkout: c, Orders: orderQueriesStub{}},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestPlaceOrder(t *testing.T) {
	c := &checkoutStub{}

	w := do(ordersHandler(c), http.MethodPost, "/orders", `{"holdId":"h1","customerId":"`+customer+`","contactEmail":"ana@example.com"}`)

	if w.Code != http.StatusCreated || w.Header().Get("Location") != "/orders/0192f5e0-0000-7000-8000-0000000000d1" {
		t.Fatalf("status %d, Location %q (%s)", w.Code, w.Header().Get("Location"), w.Body)
	}
	if *c.got != (application.Checkout{HoldID: "h1", CustomerID: customer, ContactEmail: "ana@example.com"}) {
		t.Errorf("command = %+v", c.got)
	}
}

func TestPlaceOrder_Errors(t *testing.T) {
	for err, status := range map[error]int{
		domain.ErrInvalidContactEmail: 422, domain.ErrNotHoldOwner: 403, domain.ErrHoldExpired: 409, application.ErrHoldNotFound: 404,
	} {
		w := do(ordersHandler(&checkoutStub{err: errors.Join(errors.New("checkout"), err)}), http.MethodPost, "/orders", `{"holdId":"h1"}`)
		if w.Code != status {
			t.Errorf("%v → %d, want %d", err, w.Code, status)
		}
	}
}

func TestGetOrder(t *testing.T) {
	w := do(ordersHandler(&checkoutStub{}), http.MethodGet, "/orders/0192f5e0-0000-7000-8000-0000000000d1", "")

	want := `{"id":"o1","showId":"","status":"fulfilled","total":{"amount":10494,"currency":"EUR"},` +
		`"subtotal":{"amount":9000,"currency":"EUR"},"fee":{"amount":900,"currency":"EUR"},"vat":{"amount":594,"currency":"EUR"},"tickets":[{"seat":"ORCH/A/1","code":"ABCD-EFGH-IJKL","status":"valid"}]}` + "\n"
	if w.Code != http.StatusOK || w.Body.String() != want {
		t.Errorf("status %d, body %s", w.Code, w.Body)
	}
}

type returnStub struct {
	got *application.ReturnOrder
	err error
}

func (r *returnStub) Handle(_ context.Context, cmd application.ReturnOrder) error {
	r.got = &cmd
	return r.err
}

// TKT-14: the buyer returns an order; the rest happens asynchronously.
func TestReturnOrder(t *testing.T) {
	r := &returnStub{}
	h := httpapi.Routes(httpapi.UseCases{Return: r}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	w := do(h, http.MethodPost, "/orders/o1/return", `{"customerId":"`+customer+`"}`)

	if w.Code != http.StatusAccepted || *r.got != (application.ReturnOrder{OrderID: "o1", CustomerID: customer}) {
		t.Errorf("status %d, command %+v", w.Code, r.got)
	}
	for err, status := range map[error]int{domain.ErrNotOrderOwner: 403, domain.ErrSalesClosed: 409, domain.ErrInvalidOrderTransition: 409} {
		h := httpapi.Routes(httpapi.UseCases{Return: &returnStub{err: errors.Join(errors.New("return"), err)}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
		if w := do(h, http.MethodPost, "/orders/o1/return", `{"customerId":"`+customer+`"}`); w.Code != status {
			t.Errorf("%v → %d, want %d", err, w.Code, status)
		}
	}
}
