//go:build integration

package postgres_test

import (
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres/pgtest"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application/orderrepotest"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application/ticketrepotest"
)

func TestOrderRepository_Contract(t *testing.T) {
	orderrepotest.Run(t, func(t *testing.T) application.OrderRepository { return postgres.NewOrderRepository(pgtest.New(t)) })
}

func TestTicketRepository_Contract(t *testing.T) {
	ticketrepotest.Run(t, func(t *testing.T) application.TicketRepository { return postgres.NewTicketRepository(pgtest.New(t)) })
}
