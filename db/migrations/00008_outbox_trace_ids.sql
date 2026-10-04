-- Correlation and causation IDs travel with every integration event (8.3).
-- '' for rows written before this migration, or by a command with no flow.
-- +goose Up
ALTER TABLE venue.outbox
    ADD COLUMN correlation_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN causation_id   TEXT NOT NULL DEFAULT '';
ALTER TABLE show.outbox
    ADD COLUMN correlation_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN causation_id   TEXT NOT NULL DEFAULT '';
ALTER TABLE ticketing.outbox
    ADD COLUMN correlation_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN causation_id   TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE ticketing.outbox DROP COLUMN causation_id, DROP COLUMN correlation_id;
ALTER TABLE show.outbox DROP COLUMN causation_id, DROP COLUMN correlation_id;
ALTER TABLE venue.outbox DROP COLUMN causation_id, DROP COLUMN correlation_id;
