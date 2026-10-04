package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

var t0 = time.Date(2026, 12, 1, 20, 0, 0, 0, time.UTC)

func h(n float64) time.Time { return t0.Add(time.Duration(n * float64(time.Hour))) }

func TestNewSchedule(t *testing.T) {
	tests := []struct {
		name              string
		doors, start, end time.Time
		wantErr           bool
	}{
		{"doors before start before end", h(-1), h(0), h(2), false},
		{"doors at start", h(0), h(0), h(2), false},
		{"exactly 12 hours", h(0), h(0), h(12), false},
		{"doors after start", h(1), h(0), h(2), true},
		{"end at start", h(-1), h(0), h(0), true},
		{"end before start", h(-1), h(0), h(-0.5), true},
		{"longer than 12 hours", h(-1), h(0), h(12.5), true},
		{"zero time", time.Time{}, h(0), h(2), true},
	}
	for _, tt := range tests {
		t.Run(tt.name+" (SHW-2)", func(t *testing.T) {
			s, err := domain.NewSchedule(tt.doors, tt.start, tt.end)

			if tt.wantErr {
				if !errors.Is(err, domain.ErrInvalidSchedule) {
					t.Errorf("error = %v, want %v", err, domain.ErrInvalidSchedule)
				}
				return
			}
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if !s.DoorsOpen().Equal(tt.doors) || !s.StartsAt().Equal(tt.start) || !s.EndsAt().Equal(tt.end) {
				t.Errorf("schedule = %v", s)
			}
		})
	}
}

func TestSchedule_Overlaps(t *testing.T) {
	base, _ := domain.NewSchedule(h(-1), h(0), h(2)) // occupies [-1h, 2h)
	tests := []struct {
		name              string
		doors, start, end time.Time
		want              bool
	}{
		{"ends when base opens its doors", h(-3), h(-2.5), h(-1), false},
		{"opens its doors when base ends", h(2), h(2.5), h(4), false},
		{"overlaps the start", h(-3), h(-2), h(0), true},
		{"overlaps the end", h(1), h(1.5), h(3), true},
		{"contained", h(0), h(0), h(1), true},
		{"contains", h(-2), h(-1.5), h(3), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			other, err := domain.NewSchedule(tt.doors, tt.start, tt.end)
			if err != nil {
				t.Fatal(err)
			}

			if got := base.Overlaps(other); got != tt.want {
				t.Errorf("Overlaps() = %v, want %v", got, tt.want)
			}
			if got := other.Overlaps(base); got != tt.want {
				t.Errorf("Overlaps() is not symmetric")
			}
		})
	}
}
