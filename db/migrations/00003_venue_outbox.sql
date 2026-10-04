-- The transactional outbox (ADR-004): integration events written in the same
-- transaction as the venue, published to Kafka later by the relay.
-- +goose Up
CREATE TABLE venue.outbox (
    id           BIGSERIAL   PRIMARY KEY,          -- relay order
    event_id     UUID        NOT NULL UNIQUE,      -- dedup key for consumers
    topic        TEXT        NOT NULL,
    msg_key      TEXT        NOT NULL,             -- aggregate id ⇒ per-aggregate ordering in Kafka
    event_type   TEXT        NOT NULL,             -- "venue.activated.v1"
    payload      JSONB       NOT NULL,
    occurred_at  TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ
);
CREATE INDEX outbox_unpublished ON venue.outbox (id) WHERE published_at IS NULL;

-- +goose Down
DROP TABLE venue.outbox;
