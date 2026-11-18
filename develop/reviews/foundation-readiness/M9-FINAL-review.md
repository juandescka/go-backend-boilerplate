# M9 — Canonical Foundation Readiness Review

- **Project:** Mapatu HRIS, `backend/hris-core-service`
- **Date:** 2026-11-18 (Asia/Makassar)
- **Scope:** M9A–D, foundation integration/security validation and readiness for M10.
- **Baseline:** Clean working tree/index. No pre-existing user changes. No automatic staging or commit.

## A. Overall status

**M9 COMPLETE — PASS.**

Backend foundation verified as ready for M10 — Employee & Employment development. This is not a production deployment certification.

No confirmed production-code defect required a runtime change. Corrected one test cleanup safety gap, strengthened an existing HTTP authorization boundary test, and synchronized technical documentation. Schema, privileges, authentication architecture and dependencies remain unchanged.

## B. Milestones and review method

| Stage | Status | Review |
|---|---|---|
| M9A | PASS after one fix iteration | [Initial FAIL](M9A-01-review.md), [M9A-02 PASS](M9A-02-review.md) |
| M9B | PASS | [Integration](M9B-01-review.md) |
| M9C | PASS | [Readiness](M9C-01-review.md) |
| M9D | PASS | [Final verification](M9D-01-review.md) |
| M9 Final | PASS | This canonical record |

Primary agent executed build/vet/default/DB/HTTP/CLI checks. Separate-context reviewer `/root/foundation_review` independently inspected current source/diff/docs and ran focused config, middleware, routes, security, observability and migration default tests. Reviewer inspected primary-supplied PostgreSQL evidence; it did not independently rerun database/real-server/PTY suites. Primary consolidated stage decisions and final acceptance. This is separate-context source/evidence review with explicitly attributed execution, not external production/security certification. Original FAIL and all historical milestone reviews remain intact.

Historical baseline was resolved from actual reviews: M1 operational wiring originally used MySQL, superseded by M4 PostgreSQL; M3 contains Swagger/Scalar/HTTP-standard stages; M4 comprises P/A/B/C; M5 A/B/C identity/auth/lifecycle; M6 and M7 revised final reports supersede earlier blocked records; M7E approved policy v1.0; M7S provisioning; M8 hardening supersedes historical five-character bootstrap policy with15. No missing milestone or planned feature was invented.

## C. Verified foundation

| Foundation | Current verified behavior |
|---|---|
| PostgreSQL/runtime | Actual17.12; Compose postgres:17 healthy on127.0.0.1:5433. Shared pgx SQL/GORM/readiness/metrics pool, context deadlines and repeated-close cleanup |
| Migration lifecycle | Current15 pairs through `000015_seed_rbac_policy_v1`, explicit CLI only, fresh up/no-change, rollback/reapply, dirty-state/timeout/lock behavior and safe errors |
| Health/readiness | Root unauthenticated probes; DB-independent health, bounded shared-pool readiness, safe503, fail-fast initialization and shutdown |
| OpenAPI/Scalar | Six operations (two probes/four auth), login429, generated Swagger2.0 served from binary; development-only Scalar/spec routes |
| Identity/authentication | Global account, DB-normalized email, bounded Argon2id, opaque random bearer, hash-only persistence, active-user enforcement, typed identity context |
| Sessions | Fixed default15-minute expiry, current/all revocation, retained rows, per-request validity, transactional concurrency/cancellation/rollback |
| Tenancy/organization | Six tables, scoped uniqueness/composite ownership FKs, optional location branch, multiple head offices, conservative audit/delete behavior |
| Membership/RBAC | Global fixed6 roles/8 permissions/19 grants, active membership plus explicit assignment/grant, direct containment, no child traversal or platform tenant bypass |
| Stored ownership | Real stored work-location resolver fixture rejects forged tenant/resource combinations through bearer→permission middleware. Employee production routes remain unimplemented |
| Seeder/bootstrap | Four named idempotent development datasets, dependency/ownership/target guards, no implicit grant; separate atomic one-time PLATFORM bootstrap with hidden15-character minimum input |
| HTTP security/logging | Trusted proxy boundary, process-local IP/identifier admission, generic429/Retry-After, strict CORS, no-store, body/server limits, safe envelopes and correlated private diagnostics |

## D. Fresh integration evidence and reproducibility

Run from backend root with configured local PostgreSQL17/create-database permission and Python3 for the PTY fixture. Use existing loader, not invented connection variables. Local network access was required outside the restrictive sandbox.

```bash
export GOCACHE=/private/tmp/hris-m9-go-cache
go build -o /private/tmp/hris-m9-backend .
go build ./...
go vet ./...
go test -count=1 ./...
M4B_INTEGRATION=1 M4B_CLI_BINARY=/private/tmp/hris-m9-backend go test -count=1 -run '^TestPostgresMigrationIntegration$' -v ./internal/infrastructure/migration
M5A_INTEGRATION=1 M5A_CLI_BINARY=/private/tmp/hris-m9-backend go test -count=1 -run '^TestIdentitySessionSchemaIntegration$' -v ./internal/infrastructure/migration
M6_INTEGRATION=1 M6_STAGE=full M6_CLI_BINARY=/private/tmp/hris-m9-backend go test -count=1 -run '^TestTenancyOrganizationSchemaIntegration$' -v ./internal/infrastructure/migration
M7_INTEGRATION=1 M7_STAGE=full go test -count=1 -run '^TestTenantRBACPostgresIntegration$' -v ./internal/infrastructure/migration
M5B_INTEGRATION=1 M5B_CLI_BINARY=/private/tmp/hris-m9-backend go test -count=1 -run '^TestAuthenticationPostgresIntegration$' -v ./internal/registry
M7S_INTEGRATION=1 M7S_CLI_BINARY=/private/tmp/hris-m9-backend go test -count=1 -run '^TestProvisioningPostgresIntegration$' -v ./cmd
M4A_INTEGRATION=1 go test -count=1 -run '^TestPostgresSharedPoolIntegration$' -v ./internal/registry
git diff --check
```

All commands passed. Full default suite was rerun after the final test edit and passed; default runs skip opt-in DB suites, whose separate execution is listed above. Primary DB test durations: M4B5.24s, M5A0.76s, M6full0.98s, M7full1.22s, M5B9.58s, M7S2.18s, M4A0.02s. [M9B evidence](M9B-01-review.md) records exact fixture names and coverage. M5B includes M5C lock/lifecycle and M8 actual-server helpers without a new flag.

Reused all existing suites and ownership fixtures. Extended `TestPermissionBoundary`'s existing eight cases with global headers, safe401/403/500 codes, request-ID/error-code/status correlation, exactly one completion event and withheld private store/resolver diagnostics. No duplicate integration harness/business endpoint. M4B cleanup now captures/rechecks owner/OID, rejects a configured-name collision and confirms removal.

The actual HTTP server exercised valid/invalid preflight, auth no-store, generated429, docs/spec, incomplete-header timeout, successful login/throttle/window recovery, health isolation and graceful interrupt shutdown. PostgreSQL tests exercised missing/wrong/inactive credentials, bearer format/expiry/revocation, hash-only session writes, atomic failure/canceled locks, both login/logout-all orderings and active-state rechecks. M7 exercises each requested scope/grant/membership/owner boundary; provisioning exercises CLI/PTY/one-time/concurrent/rollback behavior with generated fixture credentials.

Initial sandbox Docker inspection and M7 network connection were denied; authorized escalations succeeded. Initial M7 failure occurred before creation and was not counted as PASS. No unresolved blocked check. Race testing was not repeated: no production/concurrency algorithm changed. Scalar's actual browser CDN JavaScript rendering and `go generate .` were not rerun; annotations/generated assets are unchanged and current default/live route/spec checks passed. No production ingress, certificate, fleet, backup/recovery or load certification is claimed.

Final separate-context review returned PASS for M9D and M9 Final. Primary preservation checks matched32 fingerprints (two local configuration files plus30 migrations). Reviewer independently confirmed the18-file inventory, all local targets in16 Markdown files, unchanged migration/protected production Git paths, clean whitespace and empty index. No remaining actionable finding; database/transport/PTY executions remain primary-agent evidence.

## E. Database integrity and safety

Fresh current-schema auth integration applies1–15 and clean no-change. Exact M5A catalogs/defaults/PK/5FK/4 initial indexes pass; unique session verifier index is separately verified on current schema. M6 full validates6 tables/24 FKs/19 indexes; M7 full validates5 tables/20 FKs/9 indexes. All49 foundation FKs are immediate NO ACTION; global identity normalization, scoped business keys, membership/assignment uniqueness (including NULL-aware assignment uniqueness), audit actors and active-state defaults match SQL. Users deliberately have no activation default. `role_assignments` remains12 columns without assigned_at/expires_at; role/permission catalogs stay global.

Exact initial reference counts and matrix are6 roles,8 permission codes,19 explicit pairs. Development seeders create zero assignments; bootstrap creates only one PLATFORM assignment in its isolated fixture. No new catalog grant or implicit inheritance.

M5A rolls2→1→none/reapplies; M6 rolls9→3/reapplies while retaining identity/unrelated objects; M7 rolls15→9/reapplies and refuses dependent custom mappings; M5B rolls later modules down to its token-index-only3→2 transition/reapplies15 while retaining identity/session data. M4B synthetic SQL tests dirty-state/explicit force semantics only; no actual repository migration failure was bypassed.

All destructive operations stayed in six fresh current-workflow databases. Owner/OID/name validated before exact DROP; each suite reports cleanup. Final read-only catalog query of all six exact names returned0. Local `.env`/`config.json` and30 migration-file fingerprints match baseline. No existing development data/administrator was seeded, migrated, revoked or reset; no real bootstrap password file was read; no Docker lifecycle/volume mutation. Tests with development snapshots reported unchanged catalogs. This evidence does not claim a complete before/after row audit of externally concurrent activity.

## F. Security findings and remaining risk

| Finding | Severity | Classification/control | Required before |
|---|---|---|---|
| M4B cleanup trusted a name/boolean | Medium test safety | RESOLVED: live owner/OID guard and successful cleanup execution | Completed M9A/B |
| Stale abuse/spec/package documentation | Low | RESOLVED: synchronized current references | Completed M9C |
| Public HTTPS/HSTS, exact proxy trust/header sanitation/upstream restrictions and database TLS | High if exposed unsafely | BLOCKING FOR PRODUCTION: local controls are not deployment evidence | Target deployment |
| Per-process rate counters/reset/replica-independent budgets | Medium | BLOCKING FOR PRODUCTION where fleet protection required; atomic bounded local controls retained | Reviewed shared ingress abuse/capacity design |
| Production admin provisioning/access recovery | High operational | BLOCKING FOR PRODUCTION: development bootstrap stays refused outside development | Approved operator procedure |
| Session retention/cleanup/renewal policy | Medium operational | DEFERRED NONBLOCKING for M10; fixed expiry/revocation and retained hash-only history | Production retention/privacy/capacity policy |
| Longer hierarchy cycles | Medium at future writer | DEFERRED NONBLOCKING: same-owner/self-parent SQL controls; no exposed hierarchy writer | First hierarchy writer with concurrent transactional validation |
| Future business ownership/permission integration | High if omitted | DEFERRED NONBLOCKING foundation gate; current stored-location fixture passes | Before M10 Employee/Employment APIs |
| Legacy multipart/DTO/pagination helpers | Low current | DEFERRED NONBLOCKING: no business consumers exposed | First consumer's explicit validation/size/pagination contract |

**M10-blocking risks: none unresolved in the current foundation.** No privilege expansion, automatic SUPER_ADMIN elevation, implicit tenant bypass or security redesign occurred. Rate defaults remain10/60s IP and5/300s identifier, with429/Retry-After; technical starting points, not universal throughput or distributed guarantees.

Secret metadata: actual `.bootstrap-admin-password` is regular/operator-owned0600, untracked and excluded by Git/Docker; `.bootstra-admin-password` absent and also excluded. Neither exact path appears in accessible local Git history. Contents were never read/copied/uploaded/deleted. Retain necessary credentials in a trusted password manager and manually remove unnecessary plaintext copies. Local metadata/ignore checks cannot prove remote/deleted history or other-copy safety.

## G. Operational readiness

Development startup/configuration/CLI/logging are ready for continued development. Existing [operations checklist](../../../documentation/operations.md#foundation-development-checklist) covers Compose, distinct backend configuration, explicit migration status/up, server/probes/docs, seeder dependencies and bootstrap safeguards. Real CLI help/list succeeds without config; actual named/migration/server commands pass disposable execution. Config validation/default/precedence matches runtime; JSON HTTP port and required legacy JWT/cookie validation remain documented. No local file overwrite or default administrator.

Logging uses one completion event, request correlation, actual public code metadata, safe DB/readiness categories and separate auth/rate events. Authorization denies use the completion403/error code; no new authz security-event subsystem or production audit-retention claim. Session expiry/revocation is immediate on subsequent lookup; already authenticated requests may finish and fresh post-logout-all password logins remain valid. No renewal, retention worker or recovery credential is invented.

Production remains dependent on target TLS/ingress/proxy/shared-abuse/provisioning/recovery controls, secure log handling and retention/capacity policies. No distributed infrastructure or production bootstrap bypass was introduced.

## H. M10 readiness answers

1. **Tenant-scoped Employee records:** Yes, foundation supports them, subject to M10's approved tenant ownership/FKs/scoped-query contracts and tests. User identity remains separate from employee identity.
2. **Organization/branch reuse:** Yes, existing composite tenant/organization/branch keys support ownership-safe relations. Do not change multiple-head-office semantics or infer child-organization permissions.
3. **Authentication identity reuse:** Yes, existing bearer middleware supplies verified global typed identity/session context. Linkage to employees is a separate M10 design decision; authentication does not establish membership/permission.
4. **Resource permissions without redesign:** Yes, stored-owner resolution plus existing explicit scoped evaluator can be reused. M10 must approve its Employee/Employment operation permissions; current8-code catalog does not silently authorize new actions. M9 adds none.
5. **Migration/seeder/test foundation:** Yes, reversible versioned SQL, stable named seed registry, isolated fixtures and real HTTP/DB regression suites suffice. New seed datasets must remain explicit/idempotent/nonprivileged.
6. **Before exposing APIs:** Define Employee/Employment/lifecycle and explicit permission contracts; resolve stored ownership (including parent ownership for create), tenant-scope all reads/writes, validate active states/relationships, maintain audit fields and atomic writes, and verify forged IDs/cross-tenant/membership/grant/scope denial through actual routes. Define pagination/input policy for first consumers; hierarchy changes require cycle/concurrency validation.
7. **Deferred issues:** Deployment transport/proxy/shared-limiter/admin recovery, retention/renewal and future hierarchy writers are separate gates. Preserve multiple employments per employee, employee activation independent of employment activation, assignment records per employment, and manually supplied nullable employee number with tenant-scoped uniqueness. Any remaining final field/ERD decisions belong to M10. No M10 table, endpoint, dataset or final ERD was created.

## I. Exact changed-file inventory

Production code: **none**. Runtime configuration/examples/dependencies/generated OpenAPI/SQL migrations: **none**.

Tests:

- `internal/infrastructure/migration/postgres_integration_test.go` — cleanup identity guard.
- `internal/middlewares/authorization_test.go` — global HTTP error/log correlation assertions.

Technical documentation:

- `README.md`
- `README.id.md`
- `documentation/architecture.md`
- `documentation/operations.md`
- `documentation/database/identity-session-schema.md`

Development artifacts:

- `develop/inputs/foundation-readiness/M9A-01-audit.md`
- `develop/inputs/foundation-readiness/M9A-02-fix.md`
- `develop/inputs/foundation-readiness/M9B-01-integration.md`
- `develop/inputs/foundation-readiness/M9C-01-readiness.md`
- `develop/inputs/foundation-readiness/M9D-01-final-verification.md`
- `develop/reviews/foundation-readiness/M9A-01-review.md` — preserved FAIL.
- `develop/reviews/foundation-readiness/M9A-02-review.md`
- `develop/reviews/foundation-readiness/M9B-01-review.md`
- `develop/reviews/foundation-readiness/M9C-01-review.md`
- `develop/reviews/foundation-readiness/M9D-01-review.md`
- `develop/reviews/foundation-readiness/M9-FINAL-review.md`

All18 changed/new files belong to M9; no prior uncommitted work. Temporary CLI/cache/preservation metadata live outside the repository, with no secret contents in review artifacts.

## J. Documentation disposition

Updated README translations/status/package/spec coverage, architecture actual inventory, operations compact setup/M10 gates, and identity schema active-login-controls wording. Commands, schemas and policies were checked against current source.

Assessed unchanged: `documentation/configuration.md` (correct keys/defaults/precedence/deployment limitations), `documentation/cli.md` (existing server/migrate/seed/bootstrap/generator commands and opt-in suites), `documentation/api/api-standard.md` (correct errors/envelopes/headers/request IDs), `documentation/api/authentication.md` (correct sessions/privacy/abuse/retention), tenancy/RBAC schema references (unchanged contracts/catalog), AGENTS EN/ID and development template/workflow/FIRST_PROMPT (existing evidence/safety/preservation rules sufficient). No historical review rewritten, no competing final-readiness guide.

## K. Conventional Commit recommendation

```text
test(integration): verify backend foundation readiness

Guard disposable migration cleanup and verify authorization error correlation.
Record PostgreSQL, HTTP, session and provisioning regression evidence and
synchronize foundation documentation and M10 implementation gates.
```

Recommendation only. Nothing staged or committed.

## L. Next milestone

**M10 — Employee & Employment Foundation.** Begin only as a separately scoped task using the contracts/gates above. M9 does not begin M10 or certify production deployment.
