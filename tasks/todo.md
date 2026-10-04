# Course progress

Source of truth for *where we are* in the course. Lessons live in `docs/part-*.html`.
Tick a lesson when its "Done when" checks pass; add a one-line note (date, anything notable).

**Current lesson:** 0.1 — Event storming on paper

## Part 0 — Orientation & setup (`docs/part-0-orientation.html`)
- [ ] 0.1 Event storming on paper
- [ ] 0.2 Read the reference chapters
- [ ] 0.3 Go module, Makefile, linter
- [ ] 0.4 Working agreement: branches, commits, CI

## Part 1 — The Venue domain model (`docs/part-1-venue-domain.html`)
- [ ] 1.1 Create the domain package and a typed ID
- [ ] 1.2 Value object: Address
- [ ] 1.3 Value objects: SectionCode and Row
- [ ] 1.4 Entity: Section
- [ ] 1.5 The aggregate root: registering a Venue
- [ ] 1.6 Behaviour through the root: AddSection and capacity
- [ ] 1.7 Lifecycle: Activate and Retire
- [ ] 1.8 Domain events
- [ ] 1.9 Reconstitution and version
- [ ] 1.10 Checkpoint: read your domain as a domain expert would

## Part 2 — The application layer: use cases and ports (`docs/part-2-venue-application.html`)
- [ ] 2.1 Create the application package and declare the driven ports
- [ ] 2.2 Your first driven adapters: in-memory fakes
- [ ] 2.3 Use case: RegisterVenue
- [ ] 2.4 Use cases: AddSection, ActivateVenue, RetireVenue
- [ ] 2.5 The query side: GetVenue
- [ ] 2.6 Repository contract suite

## Part 3 — Persistence: Postgres, goose, sqlc, testcontainers (`docs/part-3-persistence.html`)
- [ ] 3.1 Docker Compose with Postgres
- [ ] 3.2 Migrations with goose, embedded, and a migrate subcommand
- [ ] 3.3 sqlc: write SQL, get Go
- [ ] 3.4 testcontainers: a real Postgres per test package
- [ ] 3.5 The Postgres VenueRepository
- [ ] 3.6 Optimistic concurrency under real contention
- [ ] 3.7 The query side in Postgres

## Part 4 — Driving adapters & the composition root (`docs/part-4-driving-adapters.html`)
- [ ] 4.1 HTTP plumbing in platform/httpx
- [ ] 4.2 The Venue HTTP adapter
- [ ] 4.3 Composition root: stagehand serve
- [ ] 4.4 Dockerfile and the app in Compose
- [ ] 4.5 The first end-to-end test
- [ ] 4.6 The architecture test

## Part 5 — Integration events: outbox, relay, Kafka (`docs/part-5-messaging.html`)
- [ ] 5.1 Kafka in Compose
- [ ] 5.2 The Published Language: venue/contracts
- [ ] 5.3 Writing the outbox atomically
- [ ] 5.4 The relay
- [ ] 5.5 The consumer runner
- [ ] 5.6 Wire it and watch it

## Part 6 — The Show context (`docs/part-6-show-context.html`)
- [ ] 6.1 The outer loop: the acceptance test first
- [ ] 6.2 Value objects: ShowID, Schedule, Money, PriceList
- [ ] 6.3 The Show aggregate and its state machine
- [ ] 6.4 Domain service: SchedulingPolicy
- [ ] 6.5 Anti-corruption layer: the local VenueLayout projection
- [ ] 6.6 Use cases, adapters, contracts, and the outer loop goes green

## Part 7 — Ticketing: the core domain (`docs/part-7-ticketing-core.html`)
- [ ] 7.1 Extract the shared kernel
- [ ] 7.2 Design workshop: aggregate boundaries
- [ ] 7.3 ShowInventory: opening and holding seats
- [ ] 7.4 Time-based behaviour: release, expire, confirm, sold out, close
- [ ] 7.5 Inventory application, persistence, HTTP, and reacting to Show
- [ ] 7.6 The race for the last seat (S6)
- [ ] 7.7 The Order aggregate and the payment port
- [ ] 7.8 Checkout and the saga
- [ ] 7.9 A scheduler is a driving adapter too
- [ ] 7.10 Closing the loops with Show

## Part 8 — End-to-end & hardening (`docs/part-8-end-to-end.html`)
- [ ] 8.1 All acceptance scenarios
- [ ] 8.2 Notifications: when a transaction script is the right answer
- [ ] 8.3 Correlation and causation IDs
- [ ] 8.4 Inbox and a transaction manager
- [ ] 8.5 Failure drills
- [ ] 8.6 Retrospective

## Part 9 — Stretch goals (`docs/part-9-stretch.html`)
- [ ] 9.1 Re-cut the inventory aggregate per section
- [ ] 9.2 Ticket check-in at the gate
- [ ] 9.3 Contract evolution: show.published.v2
- [ ] 9.4 Orchestration instead of choreography
- [ ] 9.5 Show lifecycle completion and back-on-sale
- [ ] 9.6 Database-enforced boundaries
- [ ] 9.7 Distributed tracing with OpenTelemetry
- [ ] 9.8 Fees and taxes as a domain service

## Review notes

 (Filled in as parts are completed: what went well, what was hard, decisions changed.)
