package contracts_test

import (
	"encoding/json"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/williamokano/go-ddd-by-example/internal/show/contracts"
)

var update = flag.Bool("update", false, "rewrite golden files")

var at = time.Date(2026, 12, 1, 20, 0, 0, 0, time.UTC)

func TestShowPublishedV1_Golden(t *testing.T) {
	golden(t, "testdata/show.published.v1.golden.json", contracts.ShowPublishedV1{
		ShowID: "0192f5e0-0000-7000-8000-000000000001", VenueID: "0192f5e0-0000-7000-8000-000000000002",
		Title: "Fado Night", DoorsOpen: at.Add(-time.Hour), StartsAt: at, EndsAt: at.Add(2 * time.Hour),
		Sections: []contracts.SectionV1{
			{Code: "ORCH", Kind: contracts.KindSeated, Rows: []contracts.RowV1{{Label: "A", Seats: 10}},
				Price: contracts.PriceV1{Amount: 4500, Currency: "EUR"}},
			{Code: "FLOOR", Kind: contracts.KindGA, Capacity: 500, Price: contracts.PriceV1{Amount: 2500, Currency: "EUR"}},
		},
		PublishedAt: at.Add(-30 * 24 * time.Hour),
	})
}

// v2 lists every seat, so each can carry its own flags (9.3). Rows of
// counts can't, which is why this is a new version and not a new field.
func TestShowPublishedV2_Golden(t *testing.T) {
	golden(t, "testdata/show.published.v2.golden.json", contracts.ShowPublishedV2{
		ShowID: "0192f5e0-0000-7000-8000-000000000001", VenueID: "0192f5e0-0000-7000-8000-000000000002",
		Title: "Fado Night", DoorsOpen: at.Add(-time.Hour), StartsAt: at, EndsAt: at.Add(2 * time.Hour),
		Sections: []contracts.SectionV2{
			{Code: "ORCH", Kind: contracts.KindSeated, Price: contracts.PriceV1{Amount: 4500, Currency: "EUR"},
				Seats: []contracts.SeatV2{{Row: "A", Number: 1, Accessible: true}, {Row: "A", Number: 2}}},
			{Code: "FLOOR", Kind: contracts.KindGA, Capacity: 500, Price: contracts.PriceV1{Amount: 2500, Currency: "EUR"}},
		},
		PublishedAt: at.Add(-30 * 24 * time.Hour),
	})
}

func TestShowCancelledV1_Golden(t *testing.T) {
	golden(t, "testdata/show.cancelled.v1.golden.json", contracts.ShowCancelledV1{
		ShowID: "0192f5e0-0000-7000-8000-000000000001", VenueID: "0192f5e0-0000-7000-8000-000000000002",
		Reason: "venue_retired", CancelledAt: at,
	})
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
