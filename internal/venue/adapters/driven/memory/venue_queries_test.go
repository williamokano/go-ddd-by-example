package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/williamokano/go-ddd-by-example/internal/venue/adapters/driven/memory"
	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

func TestVenueQueries_Get(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewVenueRepository()
	venue := newDraftVenue(t)
	orch, _ := domain.NewSectionCode("ORCH")
	rowA, _ := domain.NewRow("A", 10)
	seated, _ := domain.NewSeatedSection(orch, "Orchestra", []domain.Row{rowA})
	floor, _ := domain.NewSectionCode("FLOOR")
	ga, _ := domain.NewGeneralAdmissionSection(floor, "Floor", 500)
	for _, s := range []domain.Section{seated, ga} {
		if err := venue.AddSection(s, fixedNow); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.Save(ctx, venue); err != nil {
		t.Fatal(err)
	}

	got, err := memory.NewVenueQueries(repo).Get(ctx, venue.ID())

	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	want := application.VenueView{
		ID: venue.ID().String(), Name: "Coliseu dos Recreios",
		Street: "Rua Portas de Santo Antão 96", City: "Lisboa", Country: "PT",
		Status: "draft", Capacity: 510,
		Sections: []application.SectionView{
			{Code: "ORCH", Name: "Orchestra", Kind: "seated", Capacity: 10, Rows: []application.RowView{{Label: "A", Seats: 10}}},
			{Code: "FLOOR", Name: "Floor", Kind: "ga", Capacity: 500},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("VenueView mismatch (-want +got):\n%s", diff)
	}
}

func TestVenueQueries_Get_UnknownID(t *testing.T) {
	_, err := memory.NewVenueQueries(memory.NewVenueRepository()).Get(context.Background(), newDraftVenue(t).ID())

	if !errors.Is(err, application.ErrVenueNotFound) {
		t.Errorf("Get() error = %v, want %v", err, application.ErrVenueNotFound)
	}
}

func TestVenueQueries_List(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewVenueRepository()
	draft := newDraftVenue(t)
	active := newDraftVenue(t)
	code, _ := domain.NewSectionCode("FLOOR")
	floor, _ := domain.NewGeneralAdmissionSection(code, "Floor", 10)
	if err := active.AddSection(floor, fixedNow); err != nil {
		t.Fatal(err)
	}
	if err := active.Activate(fixedNow); err != nil {
		t.Fatal(err)
	}
	for _, v := range []*domain.Venue{draft, active} {
		if err := repo.Save(ctx, v); err != nil {
			t.Fatal(err)
		}
	}
	queries := memory.NewVenueQueries(repo)

	tests := []struct {
		status  string
		wantIDs []string
	}{
		{"active", []string{active.ID().String()}},
		{"draft", []string{draft.ID().String()}},
		{"retired", nil},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			views, err := queries.List(ctx, tt.status)

			if err != nil {
				t.Fatalf("List() error = %v", err)
			}
			var gotIDs []string
			for _, v := range views {
				gotIDs = append(gotIDs, v.ID)
			}
			if diff := cmp.Diff(tt.wantIDs, gotIDs); diff != "" {
				t.Errorf("List(%q) ids mismatch (-want +got):\n%s", tt.status, diff)
			}
		})
	}
}
