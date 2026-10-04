package memory_test

import (
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/memory"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application/orderrepotest"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application/ticketrepotest"
)

func TestOrderRepository_Contract(t *testing.T) {
	orderrepotest.Run(t, func(*testing.T) application.OrderRepository { return memory.NewOrderRepository() })
}

func TestTicketRepository_Contract(t *testing.T) {
	ticketrepotest.Run(t, func(*testing.T) application.TicketRepository { return memory.NewTicketRepository() })
}
