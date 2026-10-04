package application_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/payment/fakegateway"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

func TestCloseInventory(t *testing.T) {
	t.Run("closes the inventory and releases its holds (TKT-11)", func(t *testing.T) {
		f := newFixture(t)
		show := f.openShow(t)
		f.holdSeats(t, show, uuid.NewString(), "ORCH/A/1")

		if err := application.NewCloseInventoryHandler(f.inventories, f.clock).Handle(f.ctx, application.CloseInventory{ShowID: show}); err != nil {
			t.Fatal(err)
		}

		if f.seatStates(t, show)["ORCH/A/1"] != "available" || !published[domain.InventoryClosed](f.inventories.Published()) {
			t.Error("want the hold released and InventoryClosed recorded")
		}
	})

	t.Run("a show cancelled while still a draft has no inventory: a no-op", func(t *testing.T) {
		f := newFixture(t)

		err := application.NewCloseInventoryHandler(f.inventories, f.clock).Handle(f.ctx, application.CloseInventory{ShowID: uuid.NewString()})

		if err != nil {
			t.Errorf("error = %v, want a harmless no-op (not a DLQ entry)", err)
		}
	})
}

func TestOnInventoryClosed_VoidsTicketsAndRefundsOrders(t *testing.T) {
	f := newSaga(t, fakegateway.Mode{})
	show, hold, order := f.paidOrder(t)
	if err := f.confirm.Handle(f.ctx, application.ConfirmHold{ShowID: show, Section: "ORCH", HoldID: hold.String(), OrderID: order.String()}); err != nil {
		t.Fatal(err)
	}
	if err := f.issue.Handle(f.ctx, application.IssueTickets{ShowID: show, OrderID: order.String(), Seats: []string{"ORCH/A/1", "ORCH/A/2"}}); err != nil {
		t.Fatal(err)
	}
	onClosed := application.NewOnInventoryClosedHandler(f.tickets, f.orders, f.refund, f.clock)

	if err := onClosed.Handle(f.ctx, application.OnInventoryClosed{ShowID: show}); err != nil {
		t.Fatal(err)
	}
	if err := onClosed.Handle(f.ctx, application.OnInventoryClosed{ShowID: show}); err != nil { // redelivered
		t.Fatal(err)
	}

	tickets, _ := f.tickets.ListByOrder(f.ctx, order)
	for _, tk := range tickets {
		if tk.Status() != domain.VoidedTicket {
			t.Errorf("ticket %s is %v, want voided (TKT-11)", tk.Seat(), tk.Status())
		}
	}
	if f.order(t, order).Status() != domain.Refunded || len(f.gateway.Refunds()) != 1 {
		t.Errorf("order %v, %d refunds; want refunded once (TKT-11)", f.order(t, order).Status(), len(f.gateway.Refunds()))
	}
}
