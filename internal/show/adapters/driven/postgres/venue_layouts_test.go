//go:build integration

package postgres_test

import (
	"os"
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres/pgtest"
	"github.com/williamokano/go-ddd-by-example/internal/show/adapters/driven/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/show/application"
	"github.com/williamokano/go-ddd-by-example/internal/show/application/venuelayouttest"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }

func TestVenueLayouts_Contract(t *testing.T) {
	venuelayouttest.Run(t, func(t *testing.T) application.VenueLayouts { return postgres.NewVenueLayouts(pgtest.New(t)) })
}
