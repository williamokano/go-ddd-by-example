package idgen_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/platform/idgen"
)

func TestUUIDv7(t *testing.T) {
	a, b := idgen.UUIDv7{}.New(), idgen.UUIDv7{}.New()

	if a.Version() != 7 {
		t.Errorf("Version() = %d, want 7 (time-ordered, ADR-008)", a.Version())
	}
	if a == b {
		t.Error("two calls returned the same UUID")
	}
}

func TestSequence(t *testing.T) {
	seq := idgen.NewSequence()

	got := []uuid.UUID{seq.New(), seq.New()}

	want := []uuid.UUID{
		uuid.MustParse("00000000-0000-7000-8000-000000000001"),
		uuid.MustParse("00000000-0000-7000-8000-000000000002"),
	}
	if got[0] != want[0] || got[1] != want[1] {
		t.Errorf("New() = %v, want %v", got, want)
	}
}
