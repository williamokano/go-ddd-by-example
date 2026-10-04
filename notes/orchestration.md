# Orchestration vs choreography (9.4)

The checkout saga now exists twice, switched by `SAGA_STYLE`:

- `choreography` (ADR-010): each step's handler reacts to the previous
  step's fact. Nobody knows the whole flow.
- `orchestration` (the default since 9.4): `domain.CheckoutProcess`, one per
  order, is a state machine (`started → awaiting_seats → completed |
  compensated`). The `CheckoutOrchestrator` gives it each fact, the process
  names the next step, and the orchestrator runs it.

Both run the same steps (`ConfirmHold`, `IssueTickets`, `RefundOrder`) on the
same messages, so the e2e suite passes either way (`SAGA_STYLE=choreography
make test-e2e`).

| | Choreography | Orchestration |
|---|---|---|
| **Readability** | The flow is spread over three handlers and the outbox translator. You read it in Chapter 4's diagram, not in code. | `CheckoutProcess` *is* the diagram: three methods, one transition each. |
| **"Where is order X?"** | Infer it from the order's status plus the seats' states. | One row: `ticketing.checkout_processes.state`. |
| **Testability** | Each step alone; the flow only end to end. | The flow is a pure domain test (`TestCheckoutProcess`); the orchestrator is tested with the memory fakes. |
| **Failure handling** | Each step must be idempotent, and is. | Same requirement: the step runs before the process is saved, so a crash in between re-decides and re-runs an idempotent step. The process adds a guard against out-of-order facts. |
| **Coupling** | Steps know their successors only through events. | Steps know nothing. The orchestrator knows every step, which is the point, and the cost. |
| **Moving parts** | None extra. | A table, a repository, a migration path: orders paid before the switch have no process, so `drive` runs their step directly. |

Verdict for Stagehand: three steps in one context is small enough for
choreography to stay readable. Orchestration pays off as soon as the flow
branches more (fraud check, partial refunds) or someone asks "where is my
order?" every day. Kept as the default to exercise it; ADR-014 records it.
