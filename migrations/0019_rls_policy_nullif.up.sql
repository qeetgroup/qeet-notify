-- Module 36 hardening: make the tenant_isolation policies tolerate an empty
-- app.tenant_id. The original policies used
--   current_setting('app.tenant_id', TRUE)::uuid
-- but once the custom GUC has been SET on a backend, current_setting returns
-- '' (empty string) rather than NULL when unset — and ''::uuid raises
-- "invalid input syntax for type uuid". That means a query with no tenant set
-- would ERROR instead of returning zero rows. Wrapping in NULLIF(...,'') makes
-- an unset/empty tenant evaluate to NULL, so the predicate is simply false
-- (no rows) — the intended fail-closed behavior. Applied to every table that
-- has a tenant_isolation policy.
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
      'CREATE POLICY tenant_isolation ON %I USING (tenant_id = NULLIF(current_setting(''app.tenant_id'', TRUE), '''')::uuid)',
      t);
  END LOOP;
END $$;
