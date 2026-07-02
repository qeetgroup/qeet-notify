-- Append-only DPDP consent ledger (Modules 22 & 34). Every consent change
-- (opt-in / opt-out) appends a row here with its source, purpose and policy
-- version — an auditable trail distinct from the mutable `preferences` matrix.
-- Rows are retained even after subscriber erasure (erasure soft-deletes the
-- subscriber, so the ON DELETE CASCADE does not fire), satisfying DPDP's
-- consent-record retention requirement.
CREATE TABLE consent_ledger (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    subscriber_id UUID        NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE,
    channel       TEXT        NOT NULL,
    category      TEXT        NOT NULL DEFAULT 'all',
    opted_in      BOOLEAN     NOT NULL,
    source        TEXT        NOT NULL DEFAULT 'api', -- api | preference_center | import | unsubscribe_link | system
    purpose       TEXT,
    version       TEXT,                               -- consent policy version, if known
    actor         TEXT,                               -- who recorded it (api key fingerprint | subscriber | system)
    ip_address    TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE consent_ledger ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON consent_ledger
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);

CREATE INDEX idx_consent_ledger_subscriber
    ON consent_ledger(tenant_id, subscriber_id, created_at DESC);
