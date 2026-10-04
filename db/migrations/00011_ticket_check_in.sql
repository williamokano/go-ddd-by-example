-- TKT-13: a ticket is checked in once, at a gate, on the show's day.
-- +goose Up
ALTER TABLE ticketing.tickets
    ADD COLUMN checked_in_at TIMESTAMPTZ,
    ADD COLUMN gate          TEXT NOT NULL DEFAULT '';
-- status is now valid | checked_in | voided; check-in looks tickets up by code
-- (already UNIQUE, so indexed).

-- +goose Down
ALTER TABLE ticketing.tickets DROP COLUMN gate, DROP COLUMN checked_in_at;
