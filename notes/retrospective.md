# Retrospective (8.6)

Parts 0–8 are built: four contexts, six acceptance scenarios, four failure
drills. Below is what I would explain differently now, then the stretch goals.

## What I would explain differently

**Chapter 4: "Delivery guarantees and how we cope."** The chapter separates
transient failures (retry, don't commit) from poison (dead-letter it). The
5.4 runner merged them into "N attempts, then the DLQ", and every test passed.
Only drill 4 showed the DLQ swallowing `show.published.v1` during a Postgres
outage. The lesson should say plainly that a bounded retry turns an outage
into data loss. The runner should be built with `kafka.Permanent` from the
start, and the poison row should say "at once", not "after N attempts".

**ADR-004 → ADR-012: "Save is the transaction."** It was the right start.
Every use case so far touched one aggregate, so the repository could own the
transaction, and no port leaked `pgx`. The inbox broke that: the claim and the
handler's writes must commit together. The cheapest change was a `TxManager`
port plus repositories that *join* an ambient transaction through a savepoint
(`platform/postgres.Begin`). No repository signature changed, and none of the
contract suites noticed. That is the argument for starting simple: the
refactor cost one file.

**Chapter 3: correlation IDs and the pure domain.** The Discuss answer ("the
repository adapter reads it from ctx in Save") understates it. The adapter
doesn't even need to know. `outbox.Write` stamps the IDs from the `ctx` it
already receives, so no context's code mentions them.

**Part 7.6: the race test and ADR-005.** "One aggregate per show" is correct
for the last seat. The measurement in `notes/aggregate-design.md` (200 holds
for 200 different seats: 3 succeed within 3 retries) belongs in ADR-005's
consequences, next to the cure (retry with jitter, or per-section inventories
in Part 9).

**Smaller things I would fix in the pages:**

- 0.3: `go test ./...` with no packages exits 1. Say so, so nobody thinks the
  setup is broken.
- 1.x: the `wrapcheck` settings that keep domain sentinels unwrapped are
  part of the design. Show them in the lesson.
- 5.x: the Kafka testcontainers module needs the `confluentinc/confluent-local`
  image, not `apache/kafka`.
- 8.1: "make the fake mode switchable per request" lands naturally as HTTP
  middleware in the fakegateway package (`X-Fake-Payment-Mode`). Name it.
- 8.2: the duplicate-email test can only be written once the inbox exists. Move
  it to 8.4 rather than leaving a red test in the tree for two lessons.
- 8.5: `docker compose start` returns before the service is healthy. Drills
  should use `up -d --wait`.

## Stretch goals (Part 9)

I'll take them in order of what each one teaches per line of code:

1. **9.2 Ticket check-in.** A new behaviour on an aggregate that already
   exists, and a new invariant (a ticket enters once).
2. **9.3 `show.published.v2`.** Contract evolution, with both versions
   consumed side by side.
3. **9.5 Lifecycle completion.** The `Completed` state, driven by the clock
   (the scheduler pattern again).
4. **9.8 Fees and taxes.** A domain service: pricing that belongs to no single
   aggregate.
5. **9.6 Database-enforced boundaries.** One Postgres role per context, so the
   architecture test has a second line of defence.
6. **9.1 Per-section inventories.** The fix for the contention measured in 7.6.
7. **9.4 Orchestration.** The same saga with a coordinator, to compare it with
   choreography.
8. **9.7 OpenTelemetry.** Spans on top of the IDs that already travel.
