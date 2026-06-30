CREATE TABLE provider_configs (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    channel          TEXT        NOT NULL
                         CHECK (channel IN ('email','sms','whatsapp','push','webhook')),
    provider         TEXT        NOT NULL,
    -- email: 'ses' | 'resend'
    -- sms:   'msg91' | '2factor'
    -- wa:    'meta'
    -- push:  'fcm' | 'apns'
    priority         INT         NOT NULL DEFAULT 1, -- 1 = primary, 2 = fallback
    config_encrypted TEXT        NOT NULL,           -- pgp_sym_encrypt(JSON::text, enc_key)
    is_active        BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, channel, provider)
);

ALTER TABLE provider_configs ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON provider_configs
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);

CREATE INDEX idx_provider_configs_lookup
    ON provider_configs(tenant_id, channel, priority) WHERE is_active;
