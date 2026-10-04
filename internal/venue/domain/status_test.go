package domain_test

import (
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

func TestStatus_String(t *testing.T) {
	tests := []struct {
		status domain.Status
		want   string
	}{
		{domain.Draft, "draft"},
		{domain.Active, "active"},
		{domain.Retired, "retired"},
		{domain.Status(0), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Errorf("Status(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}
