package domain

import (
	"fmt"
	"slices"
	"strings"
)

// Row is a labelled line of seats in a seated section ("A", "B", … or "1", "2", …).
// It is a value object: two rows with the same label, seat count and
// accessible seats are the same row.
type Row struct {
	label      string
	seats      int
	accessible []int // sorted seat numbers with step-free access (9.3)
}

// NewRow builds a row with seats numbered 1..seats (VEN-3). The label is
// trimmed and upper-cased: row "a" and row "A" are the same row.
func NewRow(label string, seats int) (Row, error) {
	label = strings.ToUpper(strings.TrimSpace(label))
	if label == "" {
		return Row{}, fmt.Errorf("%w: label is blank", ErrInvalidRow)
	}
	if seats < 1 {
		return Row{}, fmt.Errorf("%w: row %q has %d seats, needs at least 1", ErrInvalidRow, label, seats)
	}
	return Row{label: label, seats: seats}, nil
}

// Label returns the row label.
func (r Row) Label() string { return r.label }

// Seats returns the number of seats in the row.
func (r Row) Seats() int { return r.seats }

// SeatNumbers returns the seat numbers of the row, 1..Seats().
func (r Row) SeatNumbers() []int {
	numbers := make([]int, r.seats)
	for i := range numbers {
		numbers[i] = i + 1
	}
	return numbers
}

// WithAccessibleSeats returns a copy of the row with these seats marked
// accessible (wheelchair spaces, step-free access). Each must be a seat of
// the row, listed once (VEN-3).
func (r Row) WithAccessibleSeats(numbers ...int) (Row, error) {
	sorted := slices.Clone(numbers)
	slices.Sort(sorted)
	for i, n := range sorted {
		if n < 1 || n > r.seats {
			return Row{}, fmt.Errorf("%w: row %s has no seat %d", ErrInvalidRow, r.label, n)
		}
		if i > 0 && sorted[i-1] == n {
			return Row{}, fmt.Errorf("%w: row %s lists accessible seat %d twice", ErrInvalidRow, r.label, n)
		}
	}
	r.accessible = sorted
	return r, nil
}

// AccessibleSeats returns a copy of the row's accessible seat numbers.
func (r Row) AccessibleSeats() []int { return slices.Clone(r.accessible) }
