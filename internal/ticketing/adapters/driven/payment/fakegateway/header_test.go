package fakegateway_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/payment/fakegateway"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// The e2e scenarios share one app: a test-only header switches the fake's
// mode for one request (S3 needs a late payment, S1 a prompt one).
func TestModeHeader_OverridesTheModeForOneRequest(t *testing.T) {
	gw := fakegateway.New(fakegateway.Mode{})
	var chargeErr error
	h := fakegateway.ModeHeader(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, chargeErr = gw.Charge(r.Context(), domain.NewOrderID(uuid.New()), amount(t))
	}))

	req := httptest.NewRequest(http.MethodPost, "/orders", nil)
	req.Header.Set(fakegateway.HeaderName, "decline")
	h.ServeHTTP(httptest.NewRecorder(), req)
	if !errors.Is(chargeErr, application.ErrPaymentDeclined) {
		t.Fatalf("charge with %s: decline = %v, want ErrPaymentDeclined", fakegateway.HeaderName, chargeErr)
	}

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/orders", nil))
	if chargeErr != nil {
		t.Errorf("charge without the header = %v, want the default mode (approve)", chargeErr)
	}
}

func TestModeHeader_RejectsAnInvalidMode(t *testing.T) {
	h := fakegateway.ModeHeader(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("the request reached the handler")
	}))
	req := httptest.NewRequest(http.MethodPost, "/orders", nil)
	req.Header.Set(fakegateway.HeaderName, "delay:soon")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestWithMode_IsReadByCharge(t *testing.T) {
	gw := fakegateway.New(fakegateway.Mode{Decline: true})
	ctx := fakegateway.WithMode(context.Background(), fakegateway.Mode{})
	if _, err := gw.Charge(ctx, domain.NewOrderID(uuid.New()), amount(t)); err != nil {
		t.Errorf("charge = %v, want the context's mode (approve)", err)
	}
}
