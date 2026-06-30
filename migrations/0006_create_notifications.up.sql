CREATE TABLE notifications (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    workflow_run_id     UUID        REFERENCES workflow_runs(id),
    subscriber_id       UUID        NOT NULL REFERENCES subscribers(id),
    channel             TEXT        NOT NULL
                            CHECK (channel IN ('email','sms','whatsapp','push','inapp','webhook')),
    template_id         UUID        REFERENCES templates(id),
    status              TEXT        NOT NULL DEFAULT 'pending'
                            CHECK (status IN ('pending','queued','sent','delivered','failed','skipped')),
    provider            TEXT,                              -- 'ses','resend','msg91','2factor','meta', etc.
    provider_message_id TEXT,
    content             JSONB       NOT NULL DEFAULT '{}', -- rendered subject + body
    metadata            JSONB       NOT NULL DEFAULT '{}',
    is_read             BOOLEAN     NOT NULL DEFAULT FALSE, -- in-app only
    read_at             TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON notifications
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);

CREATE INDEX idx_notifications_tenant_id    ON notifications(tenant_id);
CREATE INDEX idx_notifications_subscriber   ON notifications(tenant_id, subscriber_id, channel);
CREATE INDEX idx_notifications_status       ON notifications(status, created_at);
CREATE INDEX idx_notifications_inapp_unread ON notifications(tenant_id, subscriber_id, is_read)
    WHERE channel = 'inapp';
