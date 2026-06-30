CREATE TABLE workflows (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name          TEXT        NOT NULL,
    trigger_event TEXT        NOT NULL,              -- e.g. "user.welcome", "auth.otp"
    steps         JSONB       NOT NULL DEFAULT '[]', -- [{id, type, channel, template_id, delay_seconds, conditions}]
    is_active     BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE workflows ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON workflows
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);

CREATE INDEX idx_workflows_tenant_id ON workflows(tenant_id);
CREATE INDEX idx_workflows_trigger
    ON workflows(tenant_id, trigger_event) WHERE is_active;
