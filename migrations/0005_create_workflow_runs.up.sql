CREATE TABLE workflow_runs (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    workflow_id       UUID        NOT NULL REFERENCES workflows(id),
    subscriber_id     UUID        REFERENCES subscribers(id),
    trigger_event     TEXT        NOT NULL,
    trigger_payload   JSONB       NOT NULL DEFAULT '{}',
    status            TEXT        NOT NULL DEFAULT 'running'
                          CHECK (status IN ('running','completed','failed','cancelled')),
    current_step_index INT        NOT NULL DEFAULT 0,
    resume_at         TIMESTAMPTZ,  -- set for delay steps; workflow engine re-enqueues
    error             TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE workflow_runs ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON workflow_runs
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);

CREATE INDEX idx_workflow_runs_tenant_id  ON workflow_runs(tenant_id);
CREATE INDEX idx_workflow_runs_running    ON workflow_runs(status) WHERE status = 'running';
CREATE INDEX idx_workflow_runs_resume_at  ON workflow_runs(resume_at) WHERE resume_at IS NOT NULL;
