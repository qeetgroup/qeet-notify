-- National NDNC / DND registry (Module 32). This is NATIONAL reference data
-- (a number registered on the DND list applies to every sender), so it is
-- intentionally NOT tenant-scoped and — like the base `tenants` table — has no
-- RLS. Phone numbers are stored only as SHA-256 hashes, never plaintext.
-- Rows are populated by an operator import / TRAI sync (a later slice).
CREATE TABLE ndnc_registry (
    phone_hash TEXT        NOT NULL,
    category   TEXT        NOT NULL DEFAULT 'all',    -- 'all' or a specific DLT category
    source     TEXT        NOT NULL DEFAULT 'manual', -- 'manual' | 'trai_sync'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (phone_hash, category)
);
