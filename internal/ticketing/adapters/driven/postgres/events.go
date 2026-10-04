package postgres

import (
	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/platform/outbox"
	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
)

// ToOutboxMessages translates Ticketing's domain events into messages for
// the outbox. (Nothing leaves the context yet: the saga's internal events
// and the public contracts arrive in 7.8 and 7.10.)
func ToOutboxMessages(_ []sharedkernel.DomainEvent, _ func() uuid.UUID) ([]outbox.Message, error) {
	return nil, nil
}
