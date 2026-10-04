// Package ticketrepotest is the contract of application.TicketRepository.
package ticketrepotest

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
func Run(t *testing.T, newRepo func(t *testing.T) application.TicketRepository) {
	t.Helper()
	ctx := context.Background()
	issue := func(t *testing.T, id domain.TicketID, show domain.ShowID, order domain.OrderID, seat string) *domain.Ticket {
		t.Helper()
		ref, _ := domain.ParseSeatRef(seat)
		ticket, err := domain.IssueTicket(id, show, order, ref, now)
		if err != nil {
			t.Fatal(err)
		}
		return ticket
	}

	t.Run("issued tickets are listed by order and by show", func(t *testing.T) {
		repo := newRepo(t)
		show, order := domain.NewShowID(uuid.New()), domain.NewOrderID(uuid.New())
		for _, seat := range []string{"ORCH/A/1", "ORCH/A/2"} {
			if err := repo.Save(ctx, issue(t, domain.NewTicketID(uuid.New()), show, order, seat)); err != nil {
				t.Fatal(err)
			}
		}

		byOrder, err1 := repo.ListByOrder(ctx, order)
		byShow, err2 := repo.ListByShow(ctx, show)

		if err := errors.Join(err1, err2); err != nil || len(byOrder) != 2 || len(byShow) != 2 {
			t.Errorf("by order %d, by show %d, %v", len(byOrder), len(byShow), err)
		}
	})

	t.Run("issuing the same ticket id twice keeps one ticket (TKT-9)", func(t *testing.T) {
		repo := newRepo(t)
		id, show, order := domain.NewTicketID(uuid.New()), domain.NewShowID(uuid.New()), domain.NewOrderID(uuid.New())

		err1 := repo.Save(ctx, issue(t, id, show, order, "ORCH/A/1"))
		err2 := repo.Save(ctx, issue(t, id, show, order, "ORCH/A/1"))

		got, _ := repo.ListByOrder(ctx, order)
		if err := errors.Join(err1, err2); err != nil || len(got) != 1 {
			t.Errorf("%d tickets, %v; want 1", len(got), err)
		}
	})

	t.Run("a voided ticket round-trips", func(t *testing.T) {
		repo := newRepo(t)
		show, order := domain.NewShowID(uuid.New()), domain.NewOrderID(uuid.New())
		if err := repo.Save(ctx, issue(t, domain.NewTicketID(uuid.New()), show, order, "ORCH/A/1")); err != nil {
			t.Fatal(err)
		}
		loaded, _ := repo.ListByShow(ctx, show)
		loaded[0].Void(now)

		if err := repo.Save(ctx, loaded[0]); err != nil {
			t.Fatal(err)
		}

		again, _ := repo.ListByShow(ctx, show)
		if again[0].Status() != domain.VoidedTicket {
			t.Errorf("status = %v, want voided", again[0].Status())
		}
	})
}
