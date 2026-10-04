-- One schema per bounded context. Contexts never join or reference each
-- other's tables (no cross-schema foreign keys).
-- +goose Up
CREATE SCHEMA venue;

-- +goose Down
DROP SCHEMA venue;
