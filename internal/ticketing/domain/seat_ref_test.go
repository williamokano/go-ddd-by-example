package domain_test

import (
	"errors"
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

func TestParseSeatRef(t *testing.T) {
	for raw, want := range map[string]string{
		"ORCH/A/12":     "ORCH/A/12",
		"orch/a/12":     "ORCH/A/12",
		"FLOOR/GA/0457": "FLOOR/GA/0457",
		"FLOOR/GA/7":    "FLOOR/GA/0007",
	} {
		t.Run(raw, func(t *testing.T) {
			ref, err := domain.ParseSeatRef(raw)

			if err != nil || ref.String() != want {
				t.Errorf("ParseSeatRef(%q) = %q, %v; want %q", raw, ref, err, want)
			}
		})
	}

	for _, bad := range []string{"", "ORCH", "ORCH/A", "ORCH/A/0", "ORCH/A/x", "ORCH//1", "A B/A/1", "ORCH/A/1/2"} {
		t.Run("rejects "+bad, func(t *testing.T) {
			if _, err := domain.ParseSeatRef(bad); !errors.Is(err, domain.ErrInvalidSeatRef) {
				t.Errorf("error = %v, want %v", err, domain.ErrInvalidSeatRef)
			}
		})
	}
}

func TestSeatRef_Section(t *testing.T) {
	ref, _ := domain.ParseSeatRef("FLOOR/GA/0457")

	if ref.Section() != "FLOOR" || !ref.IsGeneralAdmission() {
		t.Errorf("Section() = %q, IsGeneralAdmission() = %v", ref.Section(), ref.IsGeneralAdmission())
	}
}
