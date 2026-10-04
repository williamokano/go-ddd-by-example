# Stagehand

A ticketing platform built to learn DDD, hexagonal architecture and TDD in Go.
Venues publish their layouts, promoters schedule and price shows, and customers
hold seats, pay, and receive tickets. If anything goes wrong, they get their
money back.

The course is in [`docs/index.html`](docs/index.html) (`make docs`, then open
http://localhost:8000). This branch, `solution`, is its reference
implementation, one tag per lesson (`git tag -l 'lesson-*'`). `main` holds only
the course.

## Run it

You need Docker, and Go 1.27 (`mise install` picks it up from `mise.toml`).

```sh
make up                 # Postgres, Kafka (+ topics), migrations, the app on :8080, Kafka UI on :8081
make test               # domain, application, HTTP, contract and architecture tests: seconds, no Docker
make test-integration   # Postgres and Kafka adapters against testcontainers
make test-e2e           # S1–S6 against the running stack (holds last 3s here)
make drills             # failure drills: stops and starts Compose services on purpose
make down
```

[`api/venue.http`](api/venue.http) has requests to try by hand. Every response
carries an `X-Correlation-ID`, and every log line and outbox row of that flow
carries the same one.

Configuration is environment variables, parsed once in
[`internal/platform/config`](internal/platform/config/config.go). The
interesting ones are `HOLD_TTL` (10m), `HOLD_SWEEP_INTERVAL` (5s) and
`PAYMENT_FAKE_MODE` (`approve`, `decline`, `delay:3s`). The `X-Fake-Payment-Mode`
header overrides the payment mode for a single request.

## The map: each concept, and where it lives

### Strategic design

| Concept | Where |
|---|---|
| Bounded contexts | [`internal/venue`](internal/venue) (supporting), [`internal/show`](internal/show) (supporting), [`internal/ticketing`](internal/ticketing) (core), [`internal/notifications`](internal/notifications) (generic) |
| Ubiquitous language, rule IDs | [`docs/01-the-domain.html`](docs/01-the-domain.html); the error sentinels read like the rulebook, e.g. [`venue/domain/errors.go`](internal/venue/domain/errors.go) |
| Published language | each context's `contracts` package, e.g. [`ticketing/contracts`](internal/ticketing/contracts/events.go), pinned by golden-file tests |
| Anti-corruption layer | Show's [`venue_consumer.go`](internal/show/adapters/driving/consumer/venue_consumer.go) translates `venue.activated.v1` into Show's own `VenueLayout` |
| Conformist | Notifications takes Ticketing's events as they are: [`notifications/adapters/driving/consumer`](internal/notifications/adapters/driving/consumer/ticketing_consumer.go) |
| Shared kernel | [`internal/sharedkernel`](internal/sharedkernel): `Money`, `Currency`, `DomainEvent` |
| Transaction script (a generic subdomain done simply) | [`notifications/application/send.go`](internal/notifications/application/send.go): no domain package at all |

### Tactical design

| Concept | Where |
|---|---|
| Value object | [`venue/domain/address.go`](internal/venue/domain/address.go), [`section_code.go`](internal/venue/domain/section_code.go), [`sharedkernel/money.go`](internal/sharedkernel/money.go) |
| Entity inside an aggregate | [`venue/domain/section.go`](internal/venue/domain/section.go) |
| Aggregate root and invariants | [`venue/domain/venue.go`](internal/venue/domain/venue.go) (VEN-1…7), [`ticketing/domain/inventory.go`](internal/ticketing/domain/inventory.go) (TKT-1…5) |
| Aggregate boundaries, decided on purpose | [`notes/aggregate-design.md`](notes/aggregate-design.md): one `ShowInventory` per show, plus `Order` and `Ticket` |
| Domain events | [`venue/domain/events.go`](internal/venue/domain/events.go), recorded by [`sharedkernel/events.go`](internal/sharedkernel/events.go) |
| Rehydration (repositories only) | [`venue/domain/rehydrate.go`](internal/venue/domain/rehydrate.go), enforced by [`archtest`](internal/archtest/archtest_test.go) |
| Optimistic concurrency + retry | `version` columns; [`venue/application/retry.go`](internal/venue/application/retry.go); the race for the last seat in [`ticketing/adapters/driven/postgres/race_test.go`](internal/ticketing/adapters/driven/postgres/race_test.go) |
| Choreographed saga with compensation | [`ticketing/application/saga.go`](internal/ticketing/application/saga.go) on `ticketing.internal` (ADR-010) |

### Hexagonal architecture

| Concept | Where |
|---|---|
| Pure domain (no `context`, no I/O) | `internal/*/domain`, checked by [`archtest`](internal/archtest/archtest_test.go) |
| Use cases and the ports they declare | [`venue/application/ports.go`](internal/venue/application/ports.go), [`ticketing/application/ports.go`](internal/ticketing/application/ports.go) |
| Driving adapters | HTTP: [`venue/adapters/driving/httpapi`](internal/venue/adapters/driving/httpapi); Kafka: `*/adapters/driving/consumer`; time: [`ticketing/adapters/driving/scheduler`](internal/ticketing/adapters/driving/scheduler/scheduler.go) |
| Driven adapters | Postgres: `*/adapters/driven/postgres` (sqlc); memory fakes: `*/adapters/driven/memory`; payment: [`fakegateway`](internal/ticketing/adapters/driven/payment/fakegateway/fakegateway.go); email: [`logsender`](internal/notifications/adapters/driven/logsender/logsender.go) |
| One contract test suite, every adapter | e.g. [`venue/application/venuerepotest`](internal/venue/application/venuerepotest), run by the memory and the Postgres repository |
| Composition root | [`cmd/stagehand/serve.go`](cmd/stagehand/serve.go): the only place that knows every concrete type |
| The dependency rule as a test | [`internal/archtest`](internal/archtest/archtest_test.go) |

### Events and reliability

| Concept | Where |
|---|---|
| Transactional outbox + relay | [`platform/outbox`](internal/platform/outbox), written in each repository's `Save` |
| Consumer runner: commit after success, DLQ for poison, retry for outages | [`platform/kafka/consumer.go`](internal/platform/kafka/consumer.go) |
| Inbox + a use case's own transaction (ADR-012) | [`platform/postgres/tx.go`](internal/platform/postgres/tx.go), used by Notifications |
| Correlation and causation IDs | [`platform/trace`](internal/platform/trace/trace.go), [`httpx.Correlation`](internal/platform/httpx/middleware.go) |
| Failure drills and what they taught | [`notes/failure-drills.md`](notes/failure-drills.md) |

### Tests, outside in

| Level | Where | Needs |
|---|---|---|
| Domain and application (memory fakes) | `internal/*/domain`, `internal/*/application` | nothing |
| HTTP adapters | `internal/*/adapters/driving/httpapi` | nothing |
| Contracts (golden files), architecture | `internal/*/contracts`, `internal/archtest` | nothing |
| Postgres and Kafka adapters | build tag `integration` | Docker (testcontainers) |
| Acceptance scenarios S1–S6 | [`test/e2e`](test/e2e), build tag `e2e` | `make up` |
| Failure drills | [`test/e2e/drills_test.go`](test/e2e/drills_test.go), tags `e2e,drills` | `make up` |
