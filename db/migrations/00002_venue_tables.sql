-- +goose Up
CREATE TABLE venue.venues (
    id         UUID        PRIMARY KEY,
    name       TEXT        NOT NULL,
    street     TEXT        NOT NULL,
    city       TEXT        NOT NULL,
    country    CHAR(2)     NOT NULL,
    status     TEXT        NOT NULL,
    version    INT         NOT NULL,          -- optimistic concurrency (ADR-011)
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- A section belongs to exactly one venue. Its rows are value objects with no
-- identity, never queried on their own and always loaded with the section,
-- so they live in a JSONB column: [{"label":"A","seats":20}, ...].
CREATE TABLE venue.sections (
    venue_id UUID    NOT NULL REFERENCES venue.venues (id) ON DELETE CASCADE,
    code     TEXT    NOT NULL,
    position INT     NOT NULL,                -- keeps the order sections were added in
    name     TEXT    NOT NULL,
    kind     TEXT    NOT NULL,                -- 'seated' | 'ga'
    capacity INT     NOT NULL,
    rows     JSONB   NOT NULL DEFAULT '[]',
    PRIMARY KEY (venue_id, code)
);

-- +goose Down
DROP TABLE venue.sections;
DROP TABLE venue.venues;
