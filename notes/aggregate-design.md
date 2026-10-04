# Ticketing: aggregate boundaries (lesson 7.2)

## Which data must be consistent *immediately* for each rule?

| Rule | Needs to see, in one transaction |
|---|---|
| TKT-1 open once | the show's inventory exists or not |
| TKT-2 1–8 seats, all available, all or nothing | the state of every requested seat |
| TKT-3 one active hold per customer per show | every active hold of the show |
| TKT-4 expiry | a hold's expiresAt and its seats |
| TKT-5 never sold twice, never held twice | every seat's state (the core invariant) |
| TKT-6 order for a valid, owned hold; total; email | the hold (read), the order |
| TKT-7 order lifecycle | the order only |
| TKT-8 confirm a paid order's hold | the hold and its seats (+ "has this order been confirmed?") |
| TKT-9 one ticket per sold seat, unique code | one ticket each (uniqueness by key) |
| TKT-10 sold out exactly once | every seat's state + a "sold out recorded" flag |
| TKT-11 close, void, refund | inventory; each ticket; each order — separately |
| TKT-12 no holds when closed or after start | inventory's closed flag and the show's start |

## Options for TKT-5

| Option | Invariant | 5,000 buyers in the first minute |
|---|---|---|
| **One ShowInventory per show** (ADR-005) | trivially correct in memory; multi-seat holds atomic | every hold on the show serialises on one version: many ErrConcurrentModification, retried (3×), some 409s; loading 50k seats per request is heavy |
| One inventory per section | correct within a section; a hold spans one section only | contention divided by the number of sections; "2 in ORCH + 2 in BALCONY" impossible in one hold |
| One aggregate per seat (or a DB unique constraint) | correct per seat; multi-seat hold = several aggregates → saga or DB transaction across rows | scales best; the invariant moves into the database (row locks / unique index) |

**Decision:** per show, as ADR-005 says. The course's shows are small and the
invariants are easiest to see and test. The cost is real and measured in 7.6;
Part 9.1 re-cuts it per section.

## The other aggregates

- **Order** — one checkout of one hold. Reads the hold (never modifies the
  inventory) when placed; its lifecycle (TKT-7) is its own.
- **Ticket** — one per sold seat. Tiny and independent, so voiding (TKT-11)
  or check-in (9.2) locks nothing else. Its ID is derived from order + seat,
  so redelivered "issue tickets" messages can't create duplicates (TKT-9).

## Where I disagree with ADR-005

Only on emphasis: the hot path is a hold, and holds conflict with *any*
other hold on the same show, even for different seats. That is the price of
a simple, obviously-correct TKT-5. Write the numbers from 7.6 into the ADR.

## Measured (lesson 7.6)

Real Postgres (testcontainers), one inventory, retries = 3:

- S6: 1 available seat, 50 concurrent holds → **1 win**, 49 "seat unavailable", never 2 winners.
- 1,000 seats, 200 concurrent holds for **200 different seats** → only **3/200 succeeded**,
  594 conflicts retried, p50 ≈ 0.8 s, p99 ≈ 0.83 s. Every hold races for the
  same inventory version, whatever seat it wants: the price of ADR-005. Part 9.1
  (an inventory per section) is the answer when this matters.
