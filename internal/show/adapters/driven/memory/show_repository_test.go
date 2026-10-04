package memory_test

import (
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/show/adapters/driven/memory"
	"github.com/williamokano/go-ddd-by-example/internal/show/application"
	"github.com/williamokano/go-ddd-by-example/internal/show/application/showrepotest"
)

func TestShowRepository_Contract(t *testing.T) {
	showrepotest.Run(t, func(*testing.T) application.ShowRepository { return memory.NewShowRepository() })
}
