-- Dev/test subscriber seed data.
-- Applied by scripts/seed.sh in development environments only.

INSERT INTO subscribers (id, tenant_id, external_id, email, phone, created_at)
VALUES
  (
    'bbbbbbbb-0000-0000-0000-000000000001',
    'tenant-dev-001',
    'e2e-subscriber-001',
    'e2e@example.com',
    '+919876543210',
    NOW()
  ),
  (
    'bbbbbbbb-0000-0000-0000-000000000002',
    'tenant-dev-001',
    'e2e-suppressed-001',
    'suppressed@example.com',
    '+919876543211',
    NOW()
  )
ON CONFLICT (tenant_id, external_id) DO NOTHING;

-- Mark second subscriber as suppressed.
INSERT INTO suppression_list (id, tenant_id, channel, address, reason, created_at)
VALUES
  (
    'cccccccc-0000-0000-0000-000000000001',
    'tenant-dev-001',
    'email',
    'suppressed@example.com',
    'e2e-test-suppression',
    NOW()
  )
ON CONFLICT DO NOTHING;
