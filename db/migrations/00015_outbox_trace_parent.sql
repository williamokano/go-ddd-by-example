-- 9.7: the W3C trace context of the span that wrote each event, so the
-- consumer's span continues the producer's trace across Kafka.
-- +goose Up
ALTER TABLE venue.outbox ADD COLUMN trace_parent TEXT NOT NULL DEFAULT '';
ALTER TABLE show.outbox ADD COLUMN trace_parent TEXT NOT NULL DEFAULT '';
ALTER TABLE ticketing.outbox ADD COLUMN trace_parent TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE ticketing.outbox DROP COLUMN trace_parent;
ALTER TABLE show.outbox DROP COLUMN trace_parent;
ALTER TABLE venue.outbox DROP COLUMN trace_parent;
