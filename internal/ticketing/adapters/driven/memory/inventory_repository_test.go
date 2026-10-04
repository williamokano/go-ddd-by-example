package memory_test

import (
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/memory"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application/inventoryrepotest"
)

func TestInventoryRepository_Contract(t *testing.T) {
	inventoryrepotest.Run(t, func(*testing.T) application.InventoryRepository { return memory.NewInventoryRepository() })
}
