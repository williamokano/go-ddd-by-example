-- 9.6: one role per bounded context, with privileges on its own schema only.
-- The architecture test keeps imports apart; this keeps the data apart: a
-- query from Ticketing's pool that touches venue.* is refused by Postgres.
-- Roles are cluster-wide, so they are created only if missing. The password
-- equals the role name: development credentials only. In production the
-- roles and their secrets are managed outside migrations, and this file
-- keeps only the GRANTs.
-- +goose Up
-- +goose StatementBegin
DO $$
DECLARE
    ctx  text;
    role text;
BEGIN
    FOREACH ctx IN ARRAY ARRAY['venue', 'show', 'ticketing', 'notifications'] LOOP
        role := 'stagehand_' || ctx;
        IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = role) THEN
            EXECUTE format('CREATE ROLE %I LOGIN PASSWORD %L', role, role);
        END IF;
        EXECUTE format('GRANT USAGE ON SCHEMA %I TO %I', ctx, role);
        EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA %I TO %I', ctx, role);
        EXECUTE format('GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA %I TO %I', ctx, role);
        EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA %I GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO %I', ctx, role);
        EXECUTE format('ALTER DEFAULT PRIVILEGES IN SCHEMA %I GRANT USAGE, SELECT ON SEQUENCES TO %I', ctx, role);
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
DECLARE
    ctx text;
BEGIN
    FOREACH ctx IN ARRAY ARRAY['venue', 'show', 'ticketing', 'notifications'] LOOP
        EXECUTE format('DROP OWNED BY %I', 'stagehand_' || ctx);
        EXECUTE format('DROP ROLE %I', 'stagehand_' || ctx);
    END LOOP;
END $$;
-- +goose StatementEnd
