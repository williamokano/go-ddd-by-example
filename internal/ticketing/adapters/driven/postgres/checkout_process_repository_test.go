//go:build integration

package postgres_test

import (
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres/pgtest"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application/checkoutrepotest"
)

func TestCheckoutProcessRepository(t *testing.T) {
	checkoutrepotest.Run(t, func(t *testing.T) application.CheckoutProcessRepository {
		return postgres.NewCheckoutProcessRepository(pgtest.New(t))
	})
}
