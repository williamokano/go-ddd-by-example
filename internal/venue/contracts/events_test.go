package contracts_test

import (
	"encoding/json"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/williamokano/go-ddd-by-example/internal/venue/contracts"
)

var update = flag.Bool("update", false, "rewrite golden files")

var at = time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)

func TestVenueActivatedV1_Golden(t *testing.T) {
	event := contracts.VenueActivatedV1{
		VenueID: "0192f5e0-7c1a-7b3e-9d2a-3f4b5c6d7e8f",
		Name:    "Coliseu dos Recreios",
		Sections: []contracts.SectionV1{
			{Code: "ORCH", Kind: contracts.KindSeated, Rows: []contracts.RowV1{{Label: "A", Seats: 10}, {Label: "B", Seats: 12}}},
			{Code: "FLOOR", Kind: contracts.KindGA, Capacity: 500},
		},
		ActivatedAt: at,
	}

	golden(t, "testdata/venue.activated.v1.golden.json", event)
}

func TestVenueRetiredV1_Golden(t *testing.T) {
	event := contracts.VenueRetiredV1{VenueID: "0192f5e0-7c1a-7b3e-9d2a-3f4b5c6d7e8f", RetiredAt: at}

	golden(t, "testdata/venue.retired.v1.golden.json", event)
}

// Consumers must survive a producer adding a field (additive change, same version).
func TestVenueRetiredV1_IgnoresUnknownFields(t *testing.T) {
	payload := `{"venue_id":"0192f5e0-7c1a-7b3e-9d2a-3f4b5c6d7e8f","retired_at":"2026-11-01T20:00:00Z","reason":"demolished"}`

	var got contracts.VenueRetiredV1
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if got.VenueID != "0192f5e0-7c1a-7b3e-9d2a-3f4b5c6d7e8f" || !got.RetiredAt.Equal(at) {
		t.Errorf("got %+v", got)
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
