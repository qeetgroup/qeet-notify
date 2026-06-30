CREATE TABLE templates (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name       TEXT        NOT NULL,
    channel    TEXT        NOT NULL CHECK (channel IN ('email','sms','whatsapp','push','inapp','webhook')),
    locale     TEXT        NOT NULL DEFAULT 'en',
    subject    TEXT,                               -- email only
    body       TEXT        NOT NULL,               -- Handlebars template
    metadata   JSONB       NOT NULL DEFAULT '{}',  -- from_name, from_email, reply_to, etc.
    is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE templates ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON templates
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);

CREATE UNIQUE INDEX idx_templates_tenant_name_channel_locale
    ON templates(tenant_id, name, channel, locale);
CREATE INDEX idx_templates_tenant_id ON templates(tenant_id);
