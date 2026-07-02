-- Dev/test notification template seed data.
-- Applied by scripts/seed.sh in development environments only.

INSERT INTO notification_templates (id, tenant_id, name, channel, subject, body, status, created_at)
VALUES
  (
    'dddddddd-0000-0000-0000-000000000001',
    'tenant-dev-001',
    'e2e-email-template',
    'email',
    'Hello {{name}}',
    '<p>This is a test email for {{name}}.</p>',
    'published',
    NOW()
  ),
  (
    'dddddddd-0000-0000-0000-000000000002',
    'tenant-dev-001',
    'e2e-sms-template',
    'sms',
    NULL,
    'Test SMS for {{name}}',
    'published',
    NOW()
  )
ON CONFLICT (id) DO NOTHING;
