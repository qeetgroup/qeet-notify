CREATE TABLE preferences (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    subscriber_id UUID        NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE,
    channel       TEXT        NOT NULL
                      CHECK (channel IN ('email','sms','whatsapp','push','inapp','webhook','all')),
    category      TEXT        NOT NULL DEFAULT 'all',
    -- 'transactional' | 'marketing' | 'security' | 'all'
    is_opted_in   BOOLEAN     NOT NULL DEFAULT TRUE,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, subscriber_id, channel, category)
);

ALTER TABLE preferences ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON preferences
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);

CREATE INDEX idx_preferences_subscriber ON preferences(tenant_id, subscriber_id);
