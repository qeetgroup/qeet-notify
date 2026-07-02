-- Revert to the original (non-NULLIF) tenant_isolation policy expression.
DO $$
DECLARE t text;
BEGIN
  FOREACH t IN ARRAY ARRAY[
    'subscribers','templates','workflows','workflow_runs','notifications',
    'delivery_events','suppressions','preferences','dlt_templates',
    'provider_configs','audit_logs','api_keys','consent_ledger'
  ] LOOP
    EXECUTE format('DROP POLICY IF EXISTS tenant_isolation ON %I', t);
    EXECUTE format(
      'CREATE POLICY tenant_isolation ON %I USING (tenant_id = current_setting(''app.tenant_id'', TRUE)::uuid)',
      t);
  END LOOP;
END $$;
