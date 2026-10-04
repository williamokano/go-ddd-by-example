package contracts_test

import (
	"encoding/json"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/contracts"
)

var update = flag.Bool("update", false, "rewrite golden files")

var at = time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)

func TestGolden(t *testing.T) {
	for file, event := range map[string]any{
		"ticketing.inventory_sold_out.v1": contracts.InventorySoldOutV1{ShowID: "0192f5e0-0000-7000-8000-000000000001", SoldOutAt: at},
		"ticketing.tickets_issued.v1": contracts.TicketsIssuedV1{
			OrderID: "0192f5e0-0000-7000-8000-0000000000d1", ShowID: "0192f5e0-0000-7000-8000-000000000001",
			CustomerID: "0192f5e0-0000-7000-8000-0000000000c1", ContactEmail: "ana@example.com",
			Tickets: []contracts.TicketV1{{Seat: "ORCH/A/1", Code: "ABCD-EFGH-IJKL"}}, IssuedAt: at,
		},
		"ticketing.order_refunded.v1": contracts.OrderRefundedV1{
			OrderID: "0192f5e0-0000-7000-8000-0000000000d1", ShowID: "0192f5e0-0000-7000-8000-000000000001",
			CustomerID: "0192f5e0-0000-7000-8000-0000000000c1", ContactEmail: "ana@example.com",
			Amount: 9000, Currency: "EUR", RefundedAt: at,
		},
	} {
		t.Run(file, func(t *testing.T) { golden(t, "testdata/"+file+".golden.json", event) })
	}
}

func golden(t *testing.T, path string, v any) {
	t.Helper()
	got, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	if *update {
		if err := os.WriteFile(path, got, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run with -update to create it)", err)
	}
	if diff := cmp.Diff(string(want), string(got)); diff != "" {
		t.Errorf("the Published Language changed (-golden +got):\n%s", diff)
	}
}
