package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

func email(t *testing.T) domain.ContactEmail {
	t.Helper()
	e, err := domain.NewContactEmail("ana@example.com")
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestNewContactEmail(t *testing.T) {
	if e, err := domain.NewContactEmail("  Ana@Example.COM "); err != nil || e.String() != "ana@example.com" {
		t.Errorf("NewContactEmail = %q, %v; want ana@example.com", e, err)
	}
	for _, bad := range []string{"", "ana", "ana@", "@example.com", "ana@example", "a na@example.com"} {
		if _, err := domain.NewContactEmail(bad); !errors.Is(err, domain.ErrInvalidContactEmail) {
			t.Errorf("NewContactEmail(%q) error = %v (TKT-6)", bad, err)
		}
	}
}

// heldView holds FLOOR/GA/0001 + 0002 for Ana and returns the hold view.
func heldView(t *testing.T) domain.HoldView {
	t.Helper()
	inv := openSections(t)["FLOOR"]
	id := held(t, inv, ana, "FLOOR/GA/0001", "FLOOR/GA/0002")
	view, err := inv.HoldView(id)
	if err != nil {
		t.Fatal(err)
	}
	return view
}

func placed(t *testing.T) *domain.Order {
	t.Helper()
	o, err := domain.PlaceOrder(newOrderID(), ana, email(t), heldView(t), noFees, now)
	if err != nil {
		t.Fatal(err)
	}
	o.PullEvents()
	return o
}

// noFees prices at face value: the lifecycle tests are not about money.
var noFees = domain.PricingPolicy{VAT: map[string]int64{"PT": 0}}

// TKT-15: the order is priced by the PricingPolicy for the venue's country,
// and keeps the breakdown.
func TestPlaceOrder_RecordsThePricingBreakdown(t *testing.T) {
	policy := domain.PricingPolicy{
		Fees: []domain.FeeRule{{Name: "standard", When: domain.AnyOrder(), Percent: 1000, Minimum: eur(t, 150)}},
		VAT:  map[string]int64{"PT": 600},
	}

	o, err := domain.PlaceOrder(newOrderID(), ana, email(t), heldView(t), policy, now) // 2 FLOOR places, Portugal

	if err != nil {
		t.Fatal(err)
	}
	want := domain.PriceBreakdown{Subtotal: eur(t, 5000), Fee: eur(t, 500), VAT: eur(t, 330), Total: eur(t, 5830)}
	if o.Pricing() != want || o.Total() != want.Total {
		t.Errorf("pricing = %v, total %v; want %v", o.Pricing(), o.Total(), want)
	}
}

func TestPlaceOrder(t *testing.T) {
	t.Run("totals the lines and records OrderPlaced (TKT-6)", func(t *testing.T) {
		id := newOrderID()

		o, err := domain.PlaceOrder(id, ana, email(t), heldView(t), noFees, now)

		if err != nil {
			t.Fatal(err)
		}
		if o.Status() != domain.Pending || o.Total() != eur(t, 5000) || len(o.Lines()) != 2 {
			t.Errorf("order = %v total %v lines %d", o.Status(), o.Total(), len(o.Lines()))
		}
		if ev := o.PullEvents(); len(ev) != 1 || ev[0].EventName() != "ticketing.OrderPlaced" {
			t.Errorf("events = %v", ev)
		}
	})

	t.Run("someone else's hold is refused (TKT-6)", func(t *testing.T) {
		if _, err := domain.PlaceOrder(newOrderID(), bob, email(t), heldView(t), noFees, now); !errors.Is(err, domain.ErrNotHoldOwner) {
			t.Errorf("error = %v, want %v", err, domain.ErrNotHoldOwner)
		}
	})

	t.Run("an expired hold is refused (TKT-6)", func(t *testing.T) {
		if _, err := domain.PlaceOrder(newOrderID(), ana, email(t), heldView(t), noFees, now.Add(ttl)); !errors.Is(err, domain.ErrHoldExpired) {
			t.Errorf("error = %v, want %v", err, domain.ErrHoldExpired)
		}
	})

	t.Run("mixed currencies are refused (TKT-6)", func(t *testing.T) {
		view := heldView(t)
		usd, _ := sharedkernel.NewCurrency("USD")
		view.Lines[1].Price, _ = sharedkernel.NewMoney(100, usd)

		if _, err := domain.PlaceOrder(newOrderID(), ana, email(t), view, noFees, now); !errors.Is(err, sharedkernel.ErrCurrencyMismatch) {
			t.Errorf("error = %v, want %v", err, sharedkernel.ErrCurrencyMismatch)
		}
	})
}

// The order lifecycle (TKT-7) as one table.
func TestOrder_Lifecycle(t *testing.T) {
	ref, _ := domain.NewPaymentRef("pay_123")
	commands := map[string]func(o *domain.Order) error{
		"paid":      func(o *domain.Order) error { return o.MarkPaid(ref, now) },
		"failed":    func(o *domain.Order) error { return o.MarkPaymentFailed("card declined", now) },
		"fulfilled": func(o *domain.Order) error { return o.MarkFulfilled(nil, now) },
		"refunded":  func(o *domain.Order) error { return o.MarkRefunded(now) },
	}
	type outcome struct {
		state domain.OrderStatus
		err   bool
	}
	ok := func(s domain.OrderStatus) outcome { return outcome{state: s} }
	illegal := outcome{err: true}
	table := map[domain.OrderStatus]map[string]outcome{
		domain.Pending:       {"paid": ok(domain.Paid), "failed": ok(domain.PaymentFailed), "fulfilled": illegal, "refunded": illegal},
		domain.Paid:          {"paid": ok(domain.Paid), "failed": illegal, "fulfilled": ok(domain.Fulfilled), "refunded": ok(domain.Refunded)},
		domain.PaymentFailed: {"paid": illegal, "failed": ok(domain.PaymentFailed), "fulfilled": illegal, "refunded": illegal},
		domain.Fulfilled:     {"paid": illegal, "failed": illegal, "fulfilled": ok(domain.Fulfilled), "refunded": ok(domain.Refunded)},
		domain.Refunded:      {"paid": illegal, "failed": illegal, "fulfilled": illegal, "refunded": ok(domain.Refunded)},
	}
	reach := map[domain.OrderStatus][]string{
		domain.Pending: nil, domain.Paid: {"paid"}, domain.PaymentFailed: {"failed"},
		domain.Fulfilled: {"paid", "fulfilled"}, domain.Refunded: {"paid", "refunded"},
	}
	for state, row := range table {
		for name, want := range row {
			t.Run(state.String()+"/"+name, func(t *testing.T) {
				o := placed(t)
				for _, step := range reach[state] {
					if err := commands[step](o); err != nil {
						t.Fatal(err)
					}
				}

				err := commands[name](o)

				if want.err {
					if !errors.Is(err, domain.ErrInvalidOrderTransition) || o.Status() != state {
						t.Errorf("error = %v, status %v; want ErrInvalidOrderTransition, unchanged", err, o.Status())
					}
					return
				}
				if err != nil || o.Status() != want.state {
					t.Errorf("error = %v, status %v; want %v", err, o.Status(), want.state)
				}
			})
		}
	}
}

func TestOrder_RepeatedFactsRecordNothing(t *testing.T) {
	o := placed(t)
	ref, _ := domain.NewPaymentRef("pay_123")
	_ = o.MarkPaid(ref, now)
	_ = o.MarkFulfilled(nil, now)
	o.PullEvents()

	_ = o.MarkFulfilled(nil, now.Add(time.Minute)) // the saga redelivered SeatsSold

	if ev := o.PullEvents(); len(ev) != 0 {
		t.Errorf("events = %v, want none", ev)
	}
}

func TestOrder_RefundCarriesTheContactEmail(t *testing.T) {
	o := placed(t)
	ref, _ := domain.NewPaymentRef("pay_123")
	_ = o.MarkPaid(ref, now)
	o.PullEvents()

	_ = o.MarkRefunded(now)

	refunded := o.PullEvents()[0].(domain.OrderRefunded)
	if refunded.ContactEmail.String() != "ana@example.com" || refunded.Total != eur(t, 5000) {
		t.Errorf("OrderRefunded = %+v", refunded)
	}
}

// The saga confirms the hold in its section's inventory, and the hold may be
// gone by then: the paid order says which section (ADR-013).
func TestOrder_PaidCarriesTheSection(t *testing.T) {
	o := placed(t)
	ref, _ := domain.NewPaymentRef("pay_123")

	_ = o.MarkPaid(ref, now)

	paid, ok := o.PullEvents()[0].(domain.OrderPaid)
	if !ok || paid.Section != "FLOOR" {
		t.Errorf("OrderPaid = %+v, want section FLOOR", paid)
	}
}
