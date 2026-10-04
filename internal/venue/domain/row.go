package domain

import (
	"fmt"
	"strings"
)

// Row is a labelled line of seats in a seated section ("A", "B", … or "1", "2", …).
// It is a value object: two rows with the same label and seat count are the same row.
type Row struct {
	label string
	seats int
}

// NewRow builds a row with seats numbered 1..seats (VEN-3).
func NewRow(label string, seats int) (Row, error) {
	label = strings.TrimSpace(label)
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
