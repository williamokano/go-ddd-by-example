# Course progress

Source of truth for *where we are* in the course. Lessons live in `docs/part-*.html`.
Tick a lesson when its "Done when" checks pass; add a one-line note (date, anything notable).

**Current lesson:** 9.4 — Orchestration instead of choreography

## Part 0 — Orientation & setup (`docs/part-0-orientation.html`)
- [x] 0.1 Event storming on paper
- [x] 0.2 Read the reference chapters
- [x] 0.3 Go module, Makefile, linter
  - `make test` exits 1 with "no packages to test" until 1.1 adds the first package; golangci-lint runs in CI.
- [x] 0.4 Working agreement: branches, commits, CI

## Part 1 — The Venue domain model (`docs/part-1-venue-domain.html`)
- [x] 1.1 Create the domain package and a typed ID
- [x] 1.2 Value object: Address
- [x] 1.3 Value objects: SectionCode and Row
- [x] 1.4 Entity: Section
- [x] 1.5 The aggregate root: registering a Venue
  - `now` and the `newDraftVenue` builder arrive when a test needs them (1.8 and 1.6).
- [x] 1.6 Behaviour through the root: AddSection and capacity
- [x] 1.7 Lifecycle: Activate and Retire
- [x] 1.8 Domain events
  - Events recorder is a named field, not embedded, so `Record` is not promoted onto `*Venue`.
- [x] 1.9 Reconstitution and version
- [x] 1.10 Checkpoint: read your domain as a domain expert would
  - 100% coverage; imports: stdlib + uuid only.

## Part 2 — The application layer: use cases and ports (`docs/part-2-venue-application.html`)
- [x] 2.1 Create the application package and declare the driven ports
- [x] 2.2 Your first driven adapters: in-memory fakes
  - platform/idgen returns plain UUIDs (platform imports no context); venue/adapters/driven/ids wraps them as VenueIDs.
- [x] 2.3 Use case: RegisterVenue
  - Handlers wrap errors (`register venue: %w`): wrapcheck wants it, errors.Is still works.
- [x] 2.4 Use cases: AddSection, ActivateVenue, RetireVenue
- [x] 2.5 The query side: GetVenue
- [x] 2.6 Repository contract suite

## Part 3 — Persistence: Postgres, goose, sqlc, testcontainers (`docs/part-3-persistence.html`)
- [x] 3.1 Docker Compose with Postgres
- [x] 3.2 Migrations with goose, embedded, and a migrate subcommand
- [x] 3.3 sqlc: write SQL, get Go
- [x] 3.4 testcontainers: a real Postgres per test package
- [x] 3.5 The Postgres VenueRepository
- [x] 3.6 Optimistic concurrency under real contention
  - The race test needs a barrier: both writers load before either saves, or the second just loads the first one's result.
- [x] 3.7 The query side in Postgres

## Part 4 — Driving adapters & the composition root (`docs/part-4-driving-adapters.html`)
- [x] 4.1 HTTP plumbing in platform/httpx
- [x] 4.2 The Venue HTTP adapter
- [x] 4.3 Composition root: stagehand serve
- [x] 4.4 Dockerfile and the app in Compose
  - Distroless has no curl: `stagehand healthcheck` is the container healthcheck.
- [x] 4.5 The first end-to-end test
- [x] 4.6 The architecture test
  - Also enforces 1.9: only adapters/driven may call domain.Rehydrate*.

## Part 5 — Integration events: outbox, relay, Kafka (`docs/part-5-messaging.html`)
- [x] 5.1 Kafka in Compose
- [x] 5.2 The Published Language: venue/contracts
- [x] 5.3 Writing the outbox atomically
- [x] 5.4 The relay
  - Kafka tests use confluentinc/confluent-local (the testcontainers module drives that image); Compose runs apache/kafka.
- [x] 5.5 The consumer runner
- [x] 5.6 Wire it and watch it
  - Kafka stopped: activation still 204, backlog 1 in venue.outbox; Kafka back: drained in ~6s.

## Part 6 — The Show context (`docs/part-6-show-context.html`)
- [x] 6.1 The outer loop: the acceptance test first
- [x] 6.2 Value objects: ShowID, Schedule, Money, PriceList
- [x] 6.3 The Show aggregate and its state machine
  - DraftShow/Price/Publish take the local VenueLayout, so SHW-1 (active venue) stays a domain rule.
- [x] 6.4 Domain service: SchedulingPolicy
- [x] 6.5 Anti-corruption layer: the local VenueLayout projection
- [x] 6.6 Use cases, adapters, contracts, and the outer loop goes green
  - Authorisation (only the drafting promoter) is an application rule: ErrNotPromoter → 403.

## Part 7 — Ticketing: the core domain (`docs/part-7-ticketing-core.html`)
- [x] 7.1 Extract the shared kernel
  - Money, Currency, DomainEvent and Events moved to `internal/sharedkernel`; a goimports pass followed in 7.10.
- [x] 7.2 Design workshop: aggregate boundaries
  - `notes/aggregate-design.md`: one ShowInventory per show, Order separate, Ticket separate.
- [x] 7.3 ShowInventory: opening and holding seats
- [x] 7.4 Time-based behaviour: release, expire, confirm, sold out, close
- [x] 7.5 Inventory application, persistence, HTTP, and reacting to Show
- [x] 7.6 The race for the last seat (S6)
  - S6 proven: one winner for the last seat. 200 holds for 200 *different* seats: only 3 succeeded within 3 retries (594 conflicts, p50 ≈ 0.8 s). See `notes/aggregate-design.md` and ADR-005.
- [x] 7.7 The Order aggregate and the payment port
- [x] 7.8 Checkout and the saga
  - Choreographed saga on `ticketing.internal`: OrderPaid → ConfirmHold → SeatsSold → IssueTickets; HoldConfirmationFailed → RefundOrder.
- [x] 7.9 A scheduler is a driving adapter too
- [x] 7.10 Closing the loops with Show
  - Show's consumer group reads venue.events and ticketing.events, routed by event-type prefix.

## Part 8 — End-to-end & hardening (`docs/part-8-end-to-end.html`)
- [x] 8.1 All acceptance scenarios
  - The `X-Fake-Payment-Mode` header switches the fake gateway per request; `make test-e2e` runs with `HOLD_TTL=3s`.
- [x] 8.2 Notifications: when a transaction script is the right answer
  - No domain package. The duplicate-email test lives in 8.4 with the inbox it needs.
- [x] 8.3 Correlation and causation IDs
  - `outbox.Write` stamps the IDs from ctx, so no context's code mentions them.
- [x] 8.4 Inbox and a transaction manager
  - ADR-012: `TxManager` + `postgres.Begin` joins an ambient transaction through a savepoint.
- [x] 8.5 Failure drills
  - Drill 4 caught the DLQ swallowing events during an outage; only `kafka.Permanent` errors are dead-lettered now. `notes/failure-drills.md`.
- [x] 8.6 Retrospective
  - `README.md` (concept → file map) and `notes/retrospective.md`.

## Part 9 — Stretch goals (`docs/part-9-stretch.html`)
- [x] 9.1 Re-cut the inventory aggregate per section
  - ADR-013: TKT-3 per section; TKT-10 is a domain service. 10 sections: 30/200 holds vs 3/200 (`notes/aggregate-design.md`).
- [x] 9.2 Ticket check-in at the gate
  - TKT-13: once, never voided, within 12h of the start (no venue time zone yet). Only the Ticket is written; a `ShowSchedule` read port gives the start.
- [x] 9.3 Contract evolution: show.published.v2
  - Additive `accessible_seats` kept venue.activated at v1; seat-by-seat restructure made show.published.v2. Ticketing skips v1 once it reads v2 (v1 arrives first on the same key).
- [ ] 9.4 Orchestration instead of choreography
- [ ] 9.5 Show lifecycle completion and back-on-sale
- [ ] 9.6 Database-enforced boundaries
- [ ] 9.7 Distributed tracing with OpenTelemetry
- [ ] 9.8 Fees and taxes as a domain service

## Review notes

 (Filled in as parts are completed: what went well, what was hard, decisions changed.)
