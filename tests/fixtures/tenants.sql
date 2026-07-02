-- Dev/test tenant seed data.
-- Applied by scripts/seed.sh in development environments only.
-- NEVER run against production.

INSERT INTO api_keys (id, tenant_id, name, key_hash, scopes, created_at)
VALUES
  (
    'aaaaaaaa-0000-0000-0000-000000000001',
    'tenant-dev-001',
    'e2e-test-key',
    -- SHA-256 of literal "dev-api-key" — replace with real hash before use
    '6e340b9cffb37a989ca544e6bb780a2c78901d3fb33738768511a30617afa01d',
    ARRAY['send:email','send:sms','send:whatsapp','read:notifications','read:analytics'],
    NOW()
  )
ON CONFLICT (id) DO NOTHING;
