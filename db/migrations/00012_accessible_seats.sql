-- 9.3: a seat can have step-free access, from show.published.v2.
-- +goose Up
ALTER TABLE ticketing.seats ADD COLUMN accessible BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE ticketing.seats DROP COLUMN accessible;
