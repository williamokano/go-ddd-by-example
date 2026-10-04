// Package checkoutrepotest is the contract of application.CheckoutProcessRepository.
package checkoutrepotest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

var now = time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)

// Run runs the contract.
func Run(t *testing.T, newRepo func(t *testing.T) application.CheckoutProcessRepository) {
	t.Helper()
	ctx := context.Background()
	start := func(t *testing.T) *domain.CheckoutProcess {
		t.Helper()
		p, err := domain.StartCheckout(domain.NewOrderID(uuid.New()), domain.NewShowID(uuid.New()), now)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}

	t.Run("an unknown order is ErrCheckoutNotFound", func(t *testing.T) {
		if _, err := newRepo(t).Get(ctx, domain.NewOrderID(uuid.New())); !errors.Is(err, application.ErrCheckoutNotFound) {
			t.Errorf("error = %v, want %v", err, application.ErrCheckoutNotFound)
		}
	})

	t.Run("a process round-trips through every state", func(t *testing.T) {
		repo := newRepo(t)
		p := start(t)
		if err := repo.Save(ctx, p); err != nil {
			t.Fatal(err)
		}
		for _, advance := range []func(*domain.CheckoutProcess, time.Time) (domain.CheckoutStep, error){
			(*domain.CheckoutProcess).OrderPaid, (*domain.CheckoutProcess).SeatsSold,
		} {
			loaded, err := repo.Get(ctx, p.OrderID())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := advance(loaded, now); err != nil {
				t.Fatal(err)
			}
			if err := repo.Save(ctx, loaded); err != nil {
				t.Fatal(err)
			}
		}

		got, err := repo.Get(ctx, p.OrderID())
		if err != nil || got.State() != domain.CheckoutCompleted || got.ShowID() != p.ShowID() {
			t.Errorf("Get() = %v %v, %v", got.State(), got.ShowID(), err)
		}
	})

	t.Run("a stale save is ErrConcurrentModification", func(t *testing.T) {
		repo := newRepo(t)
		p := start(t)
		if err := repo.Save(ctx, p); err != nil {
			t.Fatal(err)
		}
		first, _ := repo.Get(ctx, p.OrderID())
		second, _ := repo.Get(ctx, p.OrderID())
		_, _ = first.OrderPaid(now)
		_, _ = second.OrderPaid(now)
		if err := repo.Save(ctx, first); err != nil {
			t.Fatal(err)
		}

		if err := repo.Save(ctx, second); !errors.Is(err, application.ErrConcurrentModification) {
			t.Errorf("error = %v, want %v", err, application.ErrConcurrentModification)
		}
	})
}
