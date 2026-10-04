package domain_test

import (
	"errors"
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

func TestNewSeatedSection(t *testing.T) {
	t.Run("capacity is the number of seats in its rows (VEN-7)", func(t *testing.T) {
		rows := []domain.Row{mustRow(t, "A", 10), mustRow(t, "B", 12)}

		section, err := domain.NewSeatedSection(mustCode(t, "ORCH"), " Orchestra ", rows)

		if err != nil {
			t.Fatalf("NewSeatedSection() error = %v", err)
		}
		if got, want := section.Name(), "Orchestra"; got != want {
			t.Errorf("Name() = %q, want %q", got, want)
		}
		if got, want := section.Capacity(), 22; got != want {
			t.Errorf("Capacity() = %d, want %d", got, want)
		}
		if got, want := section.Kind(), domain.Seated; got != want {
			t.Errorf("Kind() = %v, want %v", got, want)
		}
	})

	t.Run("needs at least one row (VEN-3)", func(t *testing.T) {
		_, err := domain.NewSeatedSection(mustCode(t, "ORCH"), "Orchestra", nil)

		if !errors.Is(err, domain.ErrInvalidSection) {
			t.Errorf("NewSeatedSection() error = %v, want %v", err, domain.ErrInvalidSection)
		}
	})

	t.Run("row labels are unique within the section, ignoring case (VEN-3)", func(t *testing.T) {
		rows := []domain.Row{mustRow(t, "A", 10), mustRow(t, "a", 12)}

		_, err := domain.NewSeatedSection(mustCode(t, "ORCH"), "Orchestra", rows)

		if !errors.Is(err, domain.ErrDuplicateRowLabel) {
			t.Errorf("NewSeatedSection() error = %v, want %v", err, domain.ErrDuplicateRowLabel)
		}
	})
}

func TestNewGeneralAdmissionSection(t *testing.T) {
	t.Run("has a capacity and no rows (VEN-3, VEN-7)", func(t *testing.T) {
		section, err := domain.NewGeneralAdmissionSection(mustCode(t, "FLOOR"), "Floor", 500)

		if err != nil {
			t.Fatalf("NewGeneralAdmissionSection() error = %v", err)
		}
		if got, want := section.Capacity(), 500; got != want {
			t.Errorf("Capacity() = %d, want %d", got, want)
		}
		if got := section.Rows(); len(got) != 0 {
			t.Errorf("Rows() = %v, want none", got)
		}
		if got, want := section.Kind(), domain.GeneralAdmission; got != want {
			t.Errorf("Kind() = %v, want %v", got, want)
		}
	})

	t.Run("needs a capacity of at least one (VEN-3)", func(t *testing.T) {
		_, err := domain.NewGeneralAdmissionSection(mustCode(t, "FLOOR"), "Floor", 0)

		if !errors.Is(err, domain.ErrInvalidSection) {
			t.Errorf("NewGeneralAdmissionSection() error = %v, want %v", err, domain.ErrInvalidSection)
		}
	})
}

func TestSection_Rows_ReturnsACopy(t *testing.T) {
	section, err := domain.NewSeatedSection(mustCode(t, "ORCH"), "Orchestra", []domain.Row{mustRow(t, "A", 10)})
	if err != nil {
		t.Fatal(err)
	}

	rows := section.Rows()
	rows[0] = mustRow(t, "Z", 99)
	_ = append(rows, mustRow(t, "B", 12))

	if got, want := section.Capacity(), 10; got != want {
		t.Errorf("Capacity() = %d after mutating Rows(), want %d", got, want)
	}
}

func TestNewSeatedSection_CopiesTheGivenRows(t *testing.T) {
	rows := []domain.Row{mustRow(t, "A", 10)}
	section, err := domain.NewSeatedSection(mustCode(t, "ORCH"), "Orchestra", rows)
	if err != nil {
		t.Fatal(err)
	}

	rows[0] = mustRow(t, "Z", 99)

	if got, want := section.Capacity(), 10; got != want {
		t.Errorf("Capacity() = %d after mutating the caller's slice, want %d", got, want)
	}
}
