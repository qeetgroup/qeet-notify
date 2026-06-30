CREATE TABLE subscribers (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    external_id       TEXT        NOT NULL,                  -- caller's own user ID
    email_encrypted   TEXT,                                  -- pgp_sym_encrypt(email, enc_key)
    phone_encrypted   TEXT,                                  -- pgp_sym_encrypt(phone, enc_key)
    whatsapp_id       TEXT,
    push_tokens       JSONB       NOT NULL DEFAULT '[]',     -- [{provider, token, platform}]
    locale            TEXT        NOT NULL DEFAULT 'en',
    timezone          TEXT        NOT NULL DEFAULT 'UTC',
    metadata          JSONB       NOT NULL DEFAULT '{}',
    is_deleted        BOOLEAN     NOT NULL DEFAULT FALSE,
    deleted_at        TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, external_id)
);

ALTER TABLE subscribers ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON subscribers
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);

CREATE INDEX idx_subscribers_tenant_id       ON subscribers(tenant_id);
CREATE INDEX idx_subscribers_tenant_external ON subscribers(tenant_id, external_id);
