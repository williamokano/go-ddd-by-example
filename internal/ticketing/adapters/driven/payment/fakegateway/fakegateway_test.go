package fakegateway_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/payment/fakegateway"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

func amount(t *testing.T) sharedkernel.Money {
	t.Helper()
	eur, _ := sharedkernel.NewCurrency("EUR")
	m, _ := sharedkernel.NewMoney(9000, eur)
	return m
}

func TestParseMode(t *testing.T) {
	for raw, want := range map[string]fakegateway.Mode{
		"approve": {}, "": {}, "decline": {Decline: true}, "delay:3s": {Delay: 3 * time.Second},
	} {
		if got, err := fakegateway.ParseMode(raw); err != nil || got != want {
			t.Errorf("ParseMode(%q) = %+v, %v; want %+v", raw, got, err, want)
		}
	}
	if _, err := fakegateway.ParseMode("delay:soon"); err == nil {
		t.Error("ParseMode(delay:soon) error = nil")
	}
}

func TestGateway(t *testing.T) {
	ctx := context.Background()
	order := domain.NewOrderID(uuid.New())

	t.Run("approve charges and records the call; charging twice charges once", func(t *testing.T) {
		g := fakegateway.New(fakegateway.Mode{})

		ref, err := g.Charge(ctx, order, amount(t))
		again, _ := g.Charge(ctx, order, amount(t))

		if err != nil || ref.String() == "" || again != ref || len(g.Charges()) != 1 {
			t.Errorf("ref %v, again %v, err %v, charges %d", ref, again, err, len(g.Charges()))
		}
	})

	t.Run("decline is ErrPaymentDeclined", func(t *testing.T) {
		if _, err := fakegateway.New(fakegateway.Mode{Decline: true}).Charge(ctx, order, amount(t)); !errors.Is(err, application.ErrPaymentDeclined) {
			t.Errorf("error = %v, want %v", err, application.ErrPaymentDeclined)
		}
	})

	t.Run("delay waits before approving", func(t *testing.T) {
		g := fakegateway.New(fakegateway.Mode{Delay: 50 * time.Millisecond})
		start := time.Now()

		_, err := g.Charge(ctx, order, amount(t))

		if err != nil || time.Since(start) < 50*time.Millisecond {
			t.Errorf("err = %v after %s", err, time.Since(start))
		}
	})

	t.Run("refunds are recorded, so tests can spy on compensation", func(t *testing.T) {
		g := fakegateway.New(fakegateway.Mode{})
		ref, _ := g.Charge(ctx, order, amount(t))

		if err := g.Refund(ctx, ref, amount(t)); err != nil {
			t.Fatal(err)
		}

		if refunds := g.Refunds(); len(refunds) != 1 || refunds[0] != ref {
			t.Errorf("refunds = %v", refunds)
		}
	})
}
