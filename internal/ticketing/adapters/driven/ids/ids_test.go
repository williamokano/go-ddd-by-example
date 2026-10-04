package ids_test

import (
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/platform/idgen"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/ids"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

func TestTicketingIDs(t *testing.T) {
	gen := ids.New(idgen.NewSequence())

	hold, order := gen.NewHoldID(), gen.NewOrderID()

	if hold.String() != "00000000-0000-7000-8000-000000000001" || order.String() != "00000000-0000-7000-8000-000000000002" {
		t.Errorf("ids = %v, %v", hold, order)
	}
}

func TestTicketIDFor_IsDeterministic(t *testing.T) {
	gen := ids.New(idgen.NewSequence())
	order := gen.NewOrderID()
	a, _ := domain.ParseSeatRef("ORCH/A/1")
	b, _ := domain.ParseSeatRef("ORCH/A/2")

	if gen.TicketIDFor(order, a) != gen.TicketIDFor(order, a) || gen.TicketIDFor(order, a) == gen.TicketIDFor(order, b) {
		t.Error("TicketIDFor must be the same for the same seat and differ across seats")
	}
}
