ALTER TABLE database_connection_references
    ADD COLUMN IF NOT EXISTS secret_ref_env TEXT NOT NULL DEFAULT '';

ALTER TABLE database_connection_references
    ALTER COLUMN dsn_env DROP NOT NULL,
    ALTER COLUMN secret_ref DROP NOT NULL,
    ALTER COLUMN secret_ref_env DROP NOT NULL;

CREATE INDEX IF NOT EXISTS idx_database_connection_references_secret_ref_env
    ON database_connection_references (secret_ref_env);
