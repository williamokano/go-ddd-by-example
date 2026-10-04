package ids_test

import (
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/platform/idgen"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/ids"
)

func TestTicketingIDs(t *testing.T) {
	gen := ids.New(idgen.NewSequence())

	hold, order := gen.NewHoldID(), gen.NewOrderID()

	if hold.String() != "00000000-0000-7000-8000-000000000001" || order.String() != "00000000-0000-7000-8000-000000000002" {
		t.Errorf("ids = %v, %v", hold, order)
	}
}
