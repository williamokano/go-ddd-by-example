# OpenTelemetry (9.7)

One purchase is one trace in Jaeger (http://localhost:16686 after `make up`).
It runs from the `POST /orders` span through each outbox row and Kafka hop
(`consume ticketing.order_paid`, `consume ticketing.seats_sold`, …) to the
emails. `TestS1_OnePurchaseIsOneTrace` checks it through Jaeger's API.

How the trace travels:

1. `httpx.Tracing` starts a server span, continuing the caller's
   `traceparent` if it sends one.
2. `outbox.Write` stores the current span as a W3C `traceparent` in the row,
   next to the correlation and causation IDs (8.3), from the `ctx` the
   repository already passes.
3. The relay's producer sends it as a Kafka header.
4. The consumer runner extracts it and starts a consumer span per message.
   The handler's own writes go back to step 2.

The hand-rolled correlation ID is now the trace ID, unless a client sends
`X-Correlation-ID`. It stays in the envelope because it is part of the
Published Language, and log lines carry `trace_id` and `span_id` as well.

## Which layers changed? Only platform and the composition root

`git diff --stat lesson-9.6 lesson-9.7` lists:

- `internal/platform/{telemetry,httpx,outbox,kafka,trace,config}`
- `cmd/stagehand/serve.go` (setup and the middleware chain)
- `db/migrations/00015_outbox_trace_parent.sql`, and the regenerated sqlc models
- `docker-compose.yml` (Jaeger)

No `domain` and no `application` package changed, in any context, and no
repository either. Cross-cutting concerns live where the hexagon meets the
world. Because every use case already receives a `context.Context`, the trace
rides along for free.
