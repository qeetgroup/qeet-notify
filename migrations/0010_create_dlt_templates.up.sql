CREATE TABLE dlt_templates (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    carrier         TEXT        NOT NULL,
    -- 'vodafone' | 'airtel' | 'jio' | 'bsnl' | 'meta' | 'all'
    channel         TEXT        NOT NULL DEFAULT 'sms'
                        CHECK (channel IN ('sms','whatsapp')),
    template_id_ext TEXT        NOT NULL, -- TRAI DLT template ID or Meta template name
    template_name   TEXT        NOT NULL,
    pe_id           TEXT,                 -- TRAI Principal Entity ID
    sender_id       TEXT,                 -- TRAI Sender Header / WhatsApp display name
    category        TEXT        NOT NULL DEFAULT 'transactional'
                        CHECK (category IN ('transactional','promotional','service_explicit','service_implicit')),
    body_regex      TEXT        NOT NULL, -- compiled at worker startup to match outgoing body
    status          TEXT        NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending','approved','rejected')),
    metadata        JSONB       NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE dlt_templates ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON dlt_templates
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);

CREATE INDEX idx_dlt_templates_tenant   ON dlt_templates(tenant_id, channel, carrier);
CREATE INDEX idx_dlt_templates_approved ON dlt_templates(tenant_id, channel, status)
    WHERE status = 'approved';
