CREATE TABLE delivery_events (
    id                UUID        NOT NULL DEFAULT gen_random_uuid(),
    tenant_id         UUID        NOT NULL,
    notification_id   UUID        NOT NULL REFERENCES notifications(id),
    event_type        TEXT        NOT NULL,
    -- queued | sent | delivered | failed | opened | clicked
    -- bounced | complained | ndnc_blocked | suppressed | preference_skipped
    provider          TEXT,
    provider_response JSONB,
    occurred_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE delivery_events ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON delivery_events
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);

SELECT create_hypertable('delivery_events', 'occurred_at');

CREATE INDEX idx_delivery_events_notification
    ON delivery_events(notification_id, occurred_at DESC);
CREATE INDEX idx_delivery_events_tenant_time
    ON delivery_events(tenant_id, occurred_at DESC);
