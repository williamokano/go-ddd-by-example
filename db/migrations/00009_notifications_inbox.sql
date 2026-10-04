-- Notifications has no model, but it has an inbox: which events each
-- consumer has already processed, so a redelivery sends no second email (8.4).
-- +goose Up
CREATE SCHEMA notifications;
CREATE TABLE notifications.inbox (
    consumer     TEXT        NOT NULL,
    event_id     TEXT        NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (consumer, event_id)
);

-- +goose Down
DROP SCHEMA notifications CASCADE;
