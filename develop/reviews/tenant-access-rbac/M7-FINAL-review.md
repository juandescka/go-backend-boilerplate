# M7 — Final review and consolidated report

## A. Final status

**M7 BLOCKED** — membership, RBAC schema and exact-scope authorization mechanics are implemented and verified. The business permission catalog/matrix and required inheritance policy are not approved. They remain denied; no full RBAC completion claim is made. The user's recovered contract explicitly permits safe foundation work while marking these acceptance criteria BLOCKED.

M6 prerequisite: latest [M6 final PASS](../tenancy-organization/M6-FINAL-02-review.md) checked against actual source and fresh PostgreSQL17 execution before M7 SQL. M5 login/password/session/context/lifecycle verified by source and live regression. Initial working tree was clean; no earlier uncommitted work required separation.

## B. Milestone results

| Stage | Status | Review |
|---|---|---|
| M7A membership | PASS | [M7A-01](M7A-01-review.md) |
| M7B schema/fixed catalog | PASS for schema; business mappings uninitialized | [M7B-01](M7B-01-review.md) |
| M7C authorization | BLOCKED business acceptance; foundation PASS | [M7C-01](M7C-01-review.md) |
| M7D integration/security | BLOCKED business acceptance; executed foundation/regression checks PASS | [M7D-01](M7D-01-review.md) |
| M7 final | BLOCKED | This report |

Separate-context reviewer checked the canonical recovered contract before SQL, then actual A/B/C source and primary-agent execution evidence. Runtime commands were executed by the primary agent, not independently replicated by the reviewer. Final separate-context integrated review found no new runtime security or database-integrity defect: foundation PASS, overall M7 BLOCKED. The reviewer inspected final SQL, Go source, tests, documentation and execution evidence without independently rerunning tests. Report-only wording corrections were applied after this closing inspection.

## C. Database schema

Full columns, types, defaults, FK targets and technical-vs-approved decisions are in the [canonical schema](../../../documentation/database/tenant-membership-rbac-schema.md).

All audit fields: created_at/updated_at TIMESTAMPTZ default CURRENT_TIMESTAMP; nullable created_by/updated_by UUID FKs to global users. No timestamp trigger; writers maintain updates. UUID IDs generated except fixed roles. All20 M7 FKs are immediate/nondeferrable NO ACTION;9 indexes implement PK/unique contracts, no user triggers.

| Table | Columns beyond shared audit | Keys and ownership |
|---|---|---|
| tenant_memberships | id, tenant_id, user_id, is_active | UUID PK; tenant/user FKs; UNIQUE (tenant_id,user_id); association grants nothing |
| roles | id, code, name, role_scope | Fixed UUID PK; unique code; CHECK fixes six UUID/code/scope tuples; global |
| permissions | id, code, name | UUID PK; unique lowercase resource.action code; global, initially empty |
| role_permissions | role_id, permission_id | Composite PK; role/permission FKs; global, initially empty |
| role_assignments | id, tenant_id, user_id, role_id, scope_type, organization_id, branch_id, is_active | UUID PK; user/role/tenant FKs; composite membership/org/branch FKs; shape/compatibility CHECKs; NULLS NOT DISTINCT unique user/role/type/owner tuple |

Assignments have exactly12 approved columns including audit fields. **Neither assigned_at nor expires_at exists**, following the user's explicit correction. Nullable ownership is allowed only by the four approved scope shapes. PLATFORM needs no tenant membership; tenant scopes require association and active membership at request time. Multiple roles are allowed; inactive identities are retained for reactivation. No cross-tenant/org/branch references or implicit permissions.

## D. System roles

| Code | Actual scope | Deterministic UUID suffix |
|---|---|---|
| SUPER_ADMIN | PLATFORM | 000000000001 |
| TENANT_ADMIN | TENANT | 000000000002 |
| HR_ADMIN | TENANT | 000000000003 |
| MANAGER | TENANT | 000000000004 |
| SUPERVISOR | TENANT | 000000000005 |
| EMPLOYEE | TENANT | 000000000006 |

UUID prefix is 00000000-0000-4000-8000-. Fixed identity/scope CHECKs enforce compatibility without an extra assignment column or mutable-policy trigger. Migration initializes six role definitions, no user assignment or platform bypass.

## E. Actual permission matrix

**Implemented business permission definitions:0. Clean-migration permissions:0 and role_permissions mappings:0.** Every role starts with an empty grant set. No production database was deployed or inspected. No role name implies authority. The disposable-only fixture.inspect permission and its explicit test mappings prove mechanics; they are not approved business permission definitions and are never seeded by migrations. Positive approved business-policy tests are BLOCKED.

## F. Authorization architecture

M5 opaque bearer middleware supplies the existing typed global Identity. RequirePermission resolves a resource's verified owner tuple, then uses the authorization service and a small Store boundary. One parameterized database EXISTS statement checks active user/assignment, explicit role_permissions, exact scope_type/owner IDs and compatible role_scope. Tenant scopes additionally require active membership/tenant and active referenced organization/branch. PLATFORM is evaluated independently with explicit mapping, never as a tenant bypass.

Only success attaches typed authorized scope to the current request context. No cache, role-bearing session or hierarchy inference. The next statement after committed membership/assignment/mapping changes observes them; already-authorized in-flight work may finish. Unknown permissions/invalid or mismatched scope deny403, missing identity401, operational failures500; middleware aborts and returns no internal SQL/error details. Existing completion logging supplies safe status/error-code/request correlation without new duplicate logs.

No production business resource route exists. Future resource resolvers must derive ownership from stored resources before work; this foundation does not prove isolation of unimplemented HRIS APIs.

## G. Migration files and deployment

New pairs under migrations/postgres:

```text
000010_create_tenant_memberships.up.sql / .down.sql
000011_create_roles.up.sql / .down.sql
000012_create_permissions.up.sql / .down.sql
000013_create_role_permissions.up.sql / .down.sql
000014_create_role_assignments.up.sql / .down.sql
```

Only disposable databases were migrated. **mapatu_dev was not migrated.** Down14→9/reapply14 preserves existing M5/M6 data. Current actual CLI also passed full-source integration. No applied migration edited, force used, Docker volume removed, secret/config replaced or production seed run. Startup remains migration-free; operator deployment requires an explicitly verified target.

## H. API changes

**No new or changed production HTTP endpoints.** Existing login/me/logout/logout-all, health/readiness, development Scalar and Swagger JSON contracts are preserved. The new middleware/service is wired for future approved consumers. Fixture routes exist only inside tests. No annotations changed, so go generate was not needed; generated specs remain unchanged and Scalar regression passed.

## I. Testing actually executed

All Go commands used GOCACHE=/private/tmp/hris-m7-gocache to keep build writes within permitted temporary storage.

| Check | Status | Evidence |
|---|---|---|
| Build | PASS | go build ./... and current CLI executable |
| Vet | PASS | go vet ./... |
| Default tests | PASS | go test -count=1 ./...; opt-in DB checks separately executed |
| M6 prerequisite | PASS | M6_INTEGRATION=1 M6_STAGE=full M6_CLI_BINARY=/private/tmp/hris-m7-backend; PostgreSQL17, full9 lifecycle; m6_verify_fcb76166b4843032 cleaned |
| Membership integrity | PASS | Stage a, followed by expanded full checks; duplicate/FK/activation/audit isolation |
| RBAC scope integrity | PASS | Stage b/full; exact five-table types/defaults,20FK/9index/no triggers; all four shapes/role compatibility/NULL uniqueness |
| Permission enforcement mechanics | PASS | Full M7 fixture; explicit grant, unknown/missing mapping, inactive state and immediate removal |
| Approved business permission behavior | BLOCKED | No approved permission matrix; no speculative grants |
| Cross-tenant/platform isolation | PASS within foundation scope | Real bearer/permission HTTP route, tenant substitution403, wrong organization/branch, no inheritance or platform bypass |
| Security/session regression | PASS | Actual current CLI1–14; existing login/me/logout/all, concurrency/expiry/revocation/rollback/cancellation, privacy/health/ready/Scalar |
| Migration rollback/reapply | PASS | M7 runner14→9/reapply14; actual CLI regression14→3→2/reapply14 preserves identity data |
| Documentation/format | PASS | gofmt, local Markdown links, whitespace, migration pairs and git diff --check |

Reproduce the live checks using commands in [M7D review](M7D-01-review.md) and [CLI reference](../../../documentation/cli.md). Final full M7 database m7_verify_87566fc387774346a84e03a4dc3f03d2 and final auth database m5b_verify_d37cb32a3463ad2d were owner/OID-checked and removed. Initial sandboxed M6 connection failed before creating fixtures; authorized local-network escalation succeeded. No approval-review rejection remains.

## J. Review history and findings

One recorded iteration per stage, no recorded FAIL checkpoint. Contract was recovered from the user's new message before SQL and separately reviewed. A PASS; B schema PASS; C/D foundation checks PASS with business acceptance BLOCKED. Draft review refinements tightened repeat-up error checking, directly proved branch non-inheritance and typed handler context, and corrected stale documentation before checkpoint closure. No earlier review overwritten.

Open material findings: **B1** exact business permission codes and six-role mappings; **B2** required inheritance semantics. Assignment-management authority is also open; management APIs remain deferred. No observed unresolved runtime security defect within the implemented foundation.

Documentation synchronized: both READMEs, schema contract, architecture, CLI, operations, API standard, auth and identity references. No config key/dependency/generated-spec change. Inputs live under develop/inputs/tenant-access-rbac/ and reviews under develop/reviews/tenant-access-rbac/, as shown in the stage links. Existing migrations000001–000009 and authentication implementation/tests were preserved; only registry wiring integrates the new service.

## K. Seeder/bootstrap

**Seeder CLI: NOT IMPLEMENTED.** Six fixed role rows initialized deterministically by migration000011. Permissions, mappings, memberships and role assignments remain empty on clean migration. Privileged real-user/bootstrap workflow remains deferred; no default credentials, first-user elevation or automatic SUPER_ADMIN.

## L. Git recommendation

```text
feat(authorization): add tenant membership and scoped RBAC foundation

Add five PostgreSQL tables and the fixed global role catalog.
Enforce explicit permissions and exact active owner scopes without inheritance.
Verify disposable integrity and authentication regressions; document blocked business policy.
```

All actual task changes belong to M7; no earlier uncommitted milestone changes to split. Process artifacts may be a separate docs commit if desired. **Nothing staged or committed automatically.**

## M. Next milestone

M8 Security Hardening & Rate Limiting was not implemented. Full M7 dependency readiness is not yet satisfied: obtain approved business permissions/mapping and required inheritance rules, add scoped production-policy checks, then repeat impacted reviews and final acceptance. The implemented foundation and regression evidence can support later separately authorized hardening work; they do not waive the M7 gate.

Final formatting evidence:34 changed/new files,125 local Markdown links, whitespace and migration-pair checks passed; gofmt clean, git diff --check passed and staged diff empty. The initial link checker false positive on an inline generic signature was corrected without changing API code. No runtime rerun required for final report-only wording corrections.
