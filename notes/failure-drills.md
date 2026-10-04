# Failure drills (8.5)

Each drill is a test in `test/e2e/drills_test.go` (build tags `e2e,drills`):
`make drills` starts the stack with the real 10-minute holds and runs them.
They stop, kill and start Compose services, so they never run with the e2e suite.

| Drill | What happened | What saved us |
|---|---|---|
| 1. Kafka stops in the middle of S1 | The checkout still answered 201 and the order was `paid`. `OrderPaid` waited in `ticketing.outbox`. When Kafka was back, the relay published it, the consumers rejoined their groups, and the order became `fulfilled`. | **Outbox.** The order and its event committed together. Publishing is a separate step that retries on every tick. |
| 2. The app is killed while purchases are in flight | On restart, the relay republished the rows it had not marked yet, and the consumers re-read every uncommitted offset. Every order ended `fulfilled` with exactly 2 tickets. | **Outbox + idempotent handlers.** `ConfirmHold` on a confirmed hold is a no-op, ticket IDs are deterministic per seat, and the notifications inbox absorbs the duplicates. |
| 3. A malformed message on `show.events` | `decode envelope` failed, the record went to `show.events.dlq` with an `error` header, and the next show still opened its inventory. | **DLQ.** A poison message is parked, so it never blocks the partition. |
| 4. Postgres stops | HTTP answered 500 (problem+json, details logged). A `show.published.v1` that arrived during the outage was handled once Postgres was back. | **Offsets committed only after handling, plus retrying transient errors.** See the finding below. |

## Finding: the DLQ was losing work to outages

The first run of drill 4 failed. The runner tried a message 5 times with
backoff from 100 ms (about 1.5 s in total) and then dead-lettered it. Ten seconds
of Postgres downtime sent `show.published.v1` to the DLQ, so the inventory never
opened. It also sent a `tickets_issued.v1` there, so that email was never sent.
A DLQ is for messages that can *never* be handled. An outage is not that.

The fix ([c428d3f], [31b9944]):

- `kafka.Permanent(err)` marks poison. Consumers wrap payload decode errors with it,
  and the runner dead-letters a Permanent error at once.
- Every other error is transient. The runner retries it with backoff capped at
  `MaxBackoff` (10 s in `serve`) until it succeeds. Meanwhile the partition is
  blocked, which is the price of keeping per-aggregate order.
- The trade-off: a bug that returns a plain error for one message now blocks its
  partition until someone fixes the code, not just until the next deploy.
  The `kafka handler` warning names the event and the attempt number. Alert on
  it. A business outcome (an expired hold, an already-cancelled show) is
  modelled as a result, not an error, so it never gets here.

## Notes

- With `HOLD_TTL=3s` (the e2e setting), drills 1 and 2 end with the order
  `refunded` rather than `fulfilled`. The outage outlives the hold, so the saga
  takes the S3 path. That is correct, and it is why `make drills` uses the
  real TTL.
- `docker compose start` returns before Postgres or Kafka accepts connections.
  The drills use `up -d --wait` so the healthchecks decide when they are back.

[c428d3f]: https://github.com/williamokano/go-ddd-by-example/commit/c428d3f
[31b9944]: https://github.com/williamokano/go-ddd-by-example/commit/31b9944
