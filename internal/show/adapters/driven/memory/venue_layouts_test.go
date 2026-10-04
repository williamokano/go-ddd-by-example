package memory_test

import (
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/show/adapters/driven/memory"
	"github.com/williamokano/go-ddd-by-example/internal/show/application"
	"github.com/williamokano/go-ddd-by-example/internal/show/application/venuelayouttest"
)

func TestVenueLayouts_Contract(t *testing.T) {
	venuelayouttest.Run(t, func(*testing.T) application.VenueLayouts { return memory.NewVenueLayouts() })
}
