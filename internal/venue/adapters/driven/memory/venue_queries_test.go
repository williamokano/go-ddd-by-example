package memory_test

import (
	"context"
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
