package ids_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/platform/idgen"
	"github.com/williamokano/go-ddd-by-example/internal/venue/adapters/driven/ids"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

func TestVenueIDs(t *testing.T) {
	gen := ids.NewVenueIDs(idgen.NewSequence())

	got := gen.NewVenueID()

	want := domain.NewVenueID(uuid.MustParse("00000000-0000-7000-8000-000000000001"))
	if got != want {
		t.Errorf("NewVenueID() = %v, want %v", got, want)
	}
}
