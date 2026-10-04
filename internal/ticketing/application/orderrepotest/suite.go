// Package orderrepotest is the contract of application.OrderRepository.
package orderrepotest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

var now = time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)

// Run runs the contract.
func Run(t *testing.T, newRepo func(t *testing.T) application.OrderRepository) {
	t.Helper()
	ctx := context.Background()

	t.Run("get of an unknown order is ErrOrderNotFound", func(t *testing.T) {
		if _, err := newRepo(t).Get(ctx, domain.NewOrderID(uuid.New())); !errors.Is(err, application.ErrOrderNotFound) {
			t.Errorf("error = %v, want %v", err, application.ErrOrderNotFound)
		}
	})

	t.Run("a placed, then paid order round-trips", func(t *testing.T) {
		repo := newRepo(t)
		order := Place(t, domain.NewShowID(uuid.New()))
		save(t, repo, order)
		loaded := get(t, repo, order.ID())
		ref, _ := domain.NewPaymentRef("pay_1")
		if err := loaded.MarkPaid(ref, now); err != nil {
			t.Fatal(err)
		}

		save(t, repo, loaded)

		assertSame(t, loaded, get(t, repo, order.ID()))
	})

	t.Run("a stale save is ErrConcurrentModification", func(t *testing.T) {
		repo := newRepo(t)
		order := Place(t, domain.NewShowID(uuid.New()))
		save(t, repo, order)
		first, second := get(t, repo, order.ID()), get(t, repo, order.ID())
		ref, _ := domain.NewPaymentRef("pay_1")
		_ = first.MarkPaid(ref, now)
		save(t, repo, first)
		_ = second.MarkPaymentFailed("declined", now)

		if err := repo.Save(ctx, second); !errors.Is(err, application.ErrConcurrentModification) {
			t.Errorf("error = %v, want %v", err, application.ErrConcurrentModification)
		}
	})

	t.Run("list paid for show returns Paid and Fulfilled orders only", func(t *testing.T) {
		repo := newRepo(t)
		show := domain.NewShowID(uuid.New())
		pending := Place(t, show)
		paid := Place(t, show)
		ref, _ := domain.NewPaymentRef("pay_2")
		_ = paid.MarkPaid(ref, now)
		for _, o := range []*domain.Order{pending, paid, Place(t, domain.NewShowID(uuid.New()))} {
			save(t, repo, o)
		}

		got, err := repo.ListPaidForShow(ctx, show)

		if err != nil || len(got) != 1 || got[0].ID() != paid.ID() {
			t.Errorf("ListPaidForShow() = %d orders, %v; want only %s", len(got), err, paid.ID())
		}
	})
}

// Place places an order of two seats for a new customer at the show.
func Place(t *testing.T, show domain.ShowID) *domain.Order {
	t.Helper()
	eur, _ := sharedkernel.NewCurrency("EUR")
	price, _ := sharedkernel.NewMoney(4500, eur)
	a, _ := domain.ParseSeatRef("ORCH/A/1")
	b, _ := domain.ParseSeatRef("ORCH/A/2")
	customer := domain.NewCustomerID(uuid.New())
	email, _ := domain.NewContactEmail("ana@example.com")
	view := domain.HoldView{
		HoldID: domain.NewHoldID(uuid.New()), ShowID: show, Customer: customer, ExpiresAt: now.Add(time.Minute),
		Lines: []domain.OrderLine{{Seat: a, Price: price}, {Seat: b, Price: price}},
	}
	o, err := domain.PlaceOrder(domain.NewOrderID(uuid.New()), customer, email, view, now)
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func save(t *testing.T, repo application.OrderRepository, o *domain.Order) {
	t.Helper()
	if err := repo.Save(context.Background(), o); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
}

func get(t *testing.T, repo application.OrderRepository, id domain.OrderID) *domain.Order {
	t.Helper()
	o, err := repo.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	return o
}

func assertSame(t *testing.T, want, got *domain.Order) {
	t.Helper()
	w, g := domain.OrderStateOf(want), domain.OrderStateOf(got)
	w.Version, g.Version = 0, 0
	opts := cmp.AllowUnexported(domain.OrderID{}, domain.ShowID{}, domain.HoldID{}, domain.CustomerID{}, domain.ContactEmail{},
		domain.SeatRef{}, domain.PaymentRef{}, sharedkernel.Money{}, sharedkernel.Currency{})
	if diff := cmp.Diff(w, g, opts); diff != "" {
		t.Errorf("loaded order mismatch (-saved +loaded):\n%s", diff)
	}
}
