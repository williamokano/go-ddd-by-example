-- +goose Up
CREATE SCHEMA show;

-- Show's local projection of Venue's facts, built from venue.events (the ACL).
CREATE TABLE show.venue_layouts (
    venue_id   UUID        PRIMARY KEY,
    name       TEXT        NOT NULL,
    active     BOOLEAN     NOT NULL,
    sections   JSONB       NOT NULL DEFAULT '[]',  -- [{code, kind, rows:[{label, seats}], capacity}]
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE show.venue_layouts;
DROP SCHEMA show;
