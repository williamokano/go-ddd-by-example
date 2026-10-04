# Reading notes (lesson 0.2)

Answers to the "Discuss" questions, written before re-checking the chapters.

**Why is the aggregate root the root?** Invariants span the whole cluster, and
only the object that sees the whole cluster can enforce them. VEN-2 ("section
codes are unique within a venue") can't be checked by a `Section`, because it
doesn't know its siblings. Only `Venue.AddSection` can reject the duplicate.

**Domain service vs application service.** A domain service holds a business
rule that doesn't fit one aggregate, and it is pure: `SchedulingPolicy`
(SHW-3, no overlapping shows at a venue). An application service holds no rules.
It runs the recipe "parse → load → call the domain → save" through ports:
`PublishShowHandler`.

**Who owns a driven port's interface?** The core (`application`). The core
declares what it needs (`VenueRepository`), and the adapter (Postgres) implements
it. Dependencies point inward, so the core can be tested with a fake and
Postgres can be swapped without touching a rule.

**Why does the domain not import `context`?** `context` exists for
cancellation, deadlines and I/O-scoped values. A pure domain does no I/O, so a
`ctx` parameter would be an invitation to start doing some. Time and IDs come in
as plain values (ADR-008).
