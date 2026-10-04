package memory_test

import (
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/memory"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application/checkoutrepotest"
)

func TestCheckoutProcessRepository(t *testing.T) {
	checkoutrepotest.Run(t, func(*testing.T) application.CheckoutProcessRepository { return memory.NewCheckoutProcessRepository() })
}
