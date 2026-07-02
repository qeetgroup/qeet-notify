-- Module 36: make row-level security a real backstop, not just defense-in-depth.
-- These tables already ENABLE RLS with a `tenant_isolation` policy, but the app
-- connects as the table owner, which bypasses RLS unless FORCE'd. With FORCE,
-- every query (owner included) is filtered by the policy
--   USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid)
-- so a query that forgets its WHERE tenant_id (or runs on a connection without
-- app.tenant_id set) returns zero rows / is rejected — no cross-tenant leak.
--
-- The application sets app.tenant_id per request/job (SET LOCAL inside a tx via
-- the TenantTx middleware / database.RunInTenant). Explicit WHERE tenant_id
-- filters are kept as belt-and-suspenders.
--
-- SCOPE: we FORCE the PII + compliance tables (subscribers, suppressions,
-- preferences, consent_ledger, audit_logs) — the data whose cross-tenant leak
-- would be most damaging. The whole data layer has already been retrofitted to
-- run tenant-scoped (querier-from-context), so extending FORCE to the config /
-- execution tables (templates, workflows, workflow_runs, dlt_templates,
-- provider_configs) is a one-line follow-up per table.
--
-- NOT forced (intentional): `tenants` (root, no policy), `api_keys` (auth
-- resolves a key hash → tenant BEFORE a tenant is known), `ndnc_registry`
-- (national reference data), and `notifications` / `delivery_events` (the
-- provider webhook looks these up cross-tenant by provider_message_id, and SSE
-- serves them by subscriber). Those retain explicit-WHERE isolation.
ALTER TABLE subscribers    FORCE ROW LEVEL SECURITY;
ALTER TABLE suppressions   FORCE ROW LEVEL SECURITY;
ALTER TABLE preferences    FORCE ROW LEVEL SECURITY;
ALTER TABLE consent_ledger FORCE ROW LEVEL SECURITY;
ALTER TABLE audit_logs     FORCE ROW LEVEL SECURITY;
