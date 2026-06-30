CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

CREATE TABLE tenants (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name           TEXT        NOT NULL,
    slug           TEXT        NOT NULL UNIQUE,
    api_key_hash   TEXT        NOT NULL UNIQUE,  -- SHA-256 of the raw API key
    api_key_prefix TEXT        NOT NULL,          -- first 8 chars for display
    plan           TEXT        NOT NULL DEFAULT 'free',
    metadata       JSONB       NOT NULL DEFAULT '{}',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenants_api_key_hash ON tenants(api_key_hash);
