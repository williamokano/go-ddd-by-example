package ids_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/platform/idgen"
	"github.com/williamokano/go-ddd-by-example/internal/show/adapters/driven/ids"
	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

func TestShowIDs(t *testing.T) {
	got := ids.NewShowIDs(idgen.NewSequence()).NewShowID()

	if want := domain.NewShowID(uuid.MustParse("00000000-0000-7000-8000-000000000001")); got != want {
		t.Errorf("NewShowID() = %v, want %v", got, want)
	}
}
