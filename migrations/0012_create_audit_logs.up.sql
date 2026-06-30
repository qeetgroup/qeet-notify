CREATE TABLE audit_logs (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL,
    actor_type    TEXT        NOT NULL
                      CHECK (actor_type IN ('api_key','dashboard_user','system')),
    actor_id      TEXT        NOT NULL,
    action        TEXT        NOT NULL,  -- e.g. 'subscriber.created', 'template.updated'
    resource_type TEXT        NOT NULL,
    resource_id   UUID,
    old_value     JSONB,
    new_value     JSONB,
    ip_address    TEXT,
    user_agent    TEXT,
    occurred_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON audit_logs
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);

CREATE INDEX idx_audit_logs_tenant_time ON audit_logs(tenant_id, occurred_at DESC);
CREATE INDEX idx_audit_logs_resource    ON audit_logs(tenant_id, resource_type, resource_id);
