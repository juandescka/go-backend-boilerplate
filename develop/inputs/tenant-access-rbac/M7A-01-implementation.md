# M7A-01 — Tenant membership implementation

- Status: APPROVED by the user's M7 request and recovered contract of 2026-11-18.
- Prerequisite: latest M6 final PASS confirmed against source and fresh disposable PostgreSQL17 execution. M5 authentication/session regression also PASS.

Implement migration000010 with global user/tenant references, UNIQUE (tenant_id,user_id), activation and existing audit conventions. First record the [canonical contract](../../../documentation/database/tenant-membership-rbac-schema.md); preserve existing SQL, secrets and uncommitted work. Verify positive/negative membership constraints and up/down/reapply on a newly created owned disposable database only. No implicit role, endpoint, seeder, staging or commit. Review separately and continue autonomous in-scope fixes. Documentation impact: canonical schema, CLI, architecture and bilingual overview after schema implementation.
