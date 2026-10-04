package domain_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

func TestNewRow(t *testing.T) {
	t.Run("valid label and seat count", func(t *testing.T) {
		row, err := domain.NewRow(" A ", 20)

		if err != nil {
			t.Fatalf("NewRow() error = %v", err)
		}
		if got, want := row.Label(), "A"; got != want {
			t.Errorf("Label() = %q, want %q", got, want)
		}
		if got, want := row.Seats(), 20; got != want {
			t.Errorf("Seats() = %d, want %d", got, want)
		}
	})

	t.Run("rejects a blank label or fewer than one seat (VEN-3)", func(t *testing.T) {
		tests := []struct {
			name  string
			label string
			seats int
		}{
			{"blank label", "  ", 10},
			{"zero seats", "A", 0},
			{"negative seats", "A", -1},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := domain.NewRow(tt.label, tt.seats)

				if !errors.Is(err, domain.ErrInvalidRow) {
					t.Errorf("NewRow(%q, %d) error = %v, want %v", tt.label, tt.seats, err, domain.ErrInvalidRow)
				}
			})
		}
	})
}

func TestRow_SeatNumbers(t *testing.T) {
	row, err := domain.NewRow("A", 4)
	if err != nil {
		t.Fatal(err)
	}

	got := row.SeatNumbers()

	if diff := cmp.Diff([]int{1, 2, 3, 4}, got); diff != "" {
		t.Errorf("SeatNumbers() mismatch (-want +got):\n%s", diff)
	}
}
