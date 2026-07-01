CREATE TABLE api_keys (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        TEXT        NOT NULL,
    key_hash    TEXT        NOT NULL UNIQUE,  -- SHA-256 of the raw key
    prefix      TEXT        NOT NULL,          -- first 12 chars for display (e.g. "qn_live_xxxx")
    scope       TEXT        NOT NULL DEFAULT 'full'
                                CHECK (scope IN ('full','read','send')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at  TIMESTAMPTZ
);

ALTER TABLE api_keys ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON api_keys
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);

CREATE INDEX idx_api_keys_tenant_id ON api_keys(tenant_id);
CREATE INDEX idx_api_keys_hash      ON api_keys(key_hash) WHERE revoked_at IS NULL;
