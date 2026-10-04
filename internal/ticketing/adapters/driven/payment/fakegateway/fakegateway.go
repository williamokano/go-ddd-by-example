// Package fakegateway is a fake payment provider implementing
// application.PaymentGateway: approve, decline, or approve after a delay
// (PAYMENT_FAKE_MODE). It records calls, so tests can spy on them.
package fakegateway

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// Mode says how the fake answers.
type Mode struct {
	Decline bool
	Delay   time.Duration
}

// ParseMode parses "approve", "decline" or "delay:3s".
func ParseMode(raw string) (Mode, error) {
	switch {
	case raw == "" || raw == "approve":
		return Mode{}, nil
	case raw == "decline":
		return Mode{Decline: true}, nil
	case strings.HasPrefix(raw, "delay:"):
		d, err := time.ParseDuration(strings.TrimPrefix(raw, "delay:"))
		if err != nil {
			return Mode{}, fmt.Errorf("payment fake mode %q: %w", raw, err)
		}
		return Mode{Delay: d}, nil
	default:
		return Mode{}, fmt.Errorf("payment fake mode %q: want approve, decline or delay:<duration>", raw)
	}
}

// Gateway is the fake provider.
type Gateway struct {
	mode    Mode
	mu      sync.Mutex
	charges map[domain.OrderID]domain.PaymentRef
	refunds []domain.PaymentRef
}

// New returns a fake answering according to mode.
func New(mode Mode) *Gateway {
	return &Gateway{mode: mode, charges: map[domain.OrderID]domain.PaymentRef{}}
}

// Charge implements application.PaymentGateway.
func (g *Gateway) Charge(ctx context.Context, orderID domain.OrderID, amount sharedkernel.Money) (domain.PaymentRef, error) {
	if g.mode.Delay > 0 {
		select {
		case <-ctx.Done():
			return domain.PaymentRef{}, fmt.Errorf("charge %s: %w", orderID, ctx.Err())
		case <-time.After(g.mode.Delay):
		}
	}
	if g.mode.Decline {
		return domain.PaymentRef{}, fmt.Errorf("%w: charge %s of %s", application.ErrPaymentDeclined, orderID, amount)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if ref, ok := g.charges[orderID]; ok {
		return ref, nil // the order ID is the idempotency key
	}
	ref, err := domain.NewPaymentRef("fake_" + orderID.String())
	if err != nil {
		return domain.PaymentRef{}, err
	}
	g.charges[orderID] = ref
	return ref, nil
}

// Refund implements application.PaymentGateway.
func (g *Gateway) Refund(_ context.Context, ref domain.PaymentRef, _ sharedkernel.Money) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.refunds = append(g.refunds, ref)
	return nil
}

// Charges returns the charged orders' references.
func (g *Gateway) Charges() []domain.PaymentRef {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]domain.PaymentRef, 0, len(g.charges))
	for _, ref := range g.charges {
		out = append(out, ref)
	}
	return out
}

// Refunds returns the refunded references, in order.
func (g *Gateway) Refunds() []domain.PaymentRef {
	g.mu.Lock()
	defer g.mu.Unlock()
	return slices.Clone(g.refunds)
}
