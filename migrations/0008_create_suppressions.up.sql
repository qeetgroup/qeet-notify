CREATE TABLE suppressions (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    channel    TEXT        NOT NULL
                   CHECK (channel IN ('email','sms','whatsapp','push','inapp','webhook')),
    value_hash TEXT        NOT NULL, -- SHA-256 of the email/phone (PII never stored plaintext)
    reason     TEXT        NOT NULL
                   CHECK (reason IN ('hard_bounce','spam_complaint','manual','unsubscribe','ndnc')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, channel, value_hash)
);

ALTER TABLE suppressions ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON suppressions
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);

CREATE INDEX idx_suppressions_lookup ON suppressions(tenant_id, channel, value_hash);
