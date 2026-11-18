# M5A-01 — Identity & Session Schema Implementation

- **Feature:** `identity/session-schema`
- **Iteration:** `M5A-01`
- **Date:** 16 November 2026 (Asia/Makassar)
- **Approval:** User-supplied MASTER PROMPT M5A authorizes audit, implementation, disposable verification, separate-context review and routine fixes. No automatic commit.
- **Prerequisite:** [M4C-01 PASS](../../../reviews/database/foundation-verification/M4C-01-review.md); M4P-03, M4A-01 and M4B-01 PASS records read and compared with current source.

## Objective and scope

Create only the approved global `users` and `auth_sessions` PostgreSQL tables using the existing golang-migrate CLI. Preserve M4C runtime, configuration, dependencies, HTTP, Docker, secrets and previous history. No authentication flow, tenant/employee/RBAC schema, model without a consumer, seeder or default account.

## Audit and decisions

| Area | Existing state | Decision |
|---|---|---|
| Migration directory | `migrations/postgres/.gitkeep`; no SQL | KEEP directory and placeholder; CREATE two paired migrations |
| Versioning | CLI documentation prescribes positive six-digit sequential versions; no highest version | KEEP; use 000001 users, 000002 auth_sessions |
| UUID strategy | No business schema or model convention | APPLY PostgreSQL 17 built-in `gen_random_uuid()` defaults; explicit UUIDs also accepted; no extension |
| Naming conventions | Plural snake_case tables, `_id` FK design; no existing constraint names | KEEP; explicit descriptive `_fkey`, `_uidx`, `_idx` names; approved audit actors remain `created_by`/`updated_by` |
| Users schema | Absent | CREATE exact 11 approved columns; global identity, no tenant key |
| Auth sessions schema | Absent | CREATE exact 13 approved columns; many per user, native INET, no is_active |
| Audit columns | No existing business timestamp mechanism | APPLY TIMESTAMPTZ creation defaults, nullable actor FKs; writer maintains updated_at thereafter; no trigger |
| Documentation | References currently describe empty migration source | UPDATE one official schema document, architecture, CLI, operations, bilingual README |

No separate final Module B ERD document was found in repository searches. The user's approved Module B/schema baseline is authoritative; do not substitute LMS/SPMB designs. Domain directories have no consumer, so Go models are deferred to M5B.

## Schema implementation plan

Create users first with self-referencing actor constraints, then sessions with three FKs to users. All five FKs use immediate, nondeferrable `NO ACTION` for delete/update; no cascading history deletion. Down pairs drop only their own table, sessions before users; no CASCADE/schema reset. Existing runner executes each multi-statement file atomically; no CLI semantics change.

One global unique expression index `lower(btrim(email))` handles case and ordinary leading/trailing spaces without a duplicate email index or extension. Preserve original email text. PostgreSQL locale controls lowercasing; broader whitespace/Unicode/email syntax/provider policy belongs to M5B. Password hash nullable does not authorize passwordless login. `is_active` has no default: account creation must explicitly choose it. No default credentials.

Only `auth_sessions(user_id)` is additionally indexed for required per-user lookup. Token hash algorithm/lookup/uniqueness, expiry/revocation filtering, nullable actor indexes and retention policy are deferred pending real M5B/M5C query contracts. No speculative token uniqueness or lifecycle checks.

## Verification strategy

Run `go build ./...`, `go vet ./...`, fresh `go test ./...`, `git diff --check`, `docker compose ps`. Reuse real config loader (process > .env > JSON > defaults), confirm safe effective identity/SSL/timeout without credentials, and override only database name for a uniquely created disposable DB. Verify owner/OID before cleanup, never reuse an existing name or force-drop active connections. All schema/data/metadata mutations target the owned disposable only; configured development database can be inspected read-only, never migrated (even version can initialize metadata).

Add opt-in migration integration beside existing runner tests using actual repository SQL and current CLI binary. Verify exact catalog types/nullability/defaults/PK/FK/indexes; normalized email insert/update collisions; required fields; nullable values; IPv4/IPv6; audit actor integrity; conservative deletion; multiple sessions; fresh up/version/repeat-up, two one-step downs, unrelated sentinel preservation, reapply, no business seed rows, and safe cleanup. Reuse the existing read-only shared-pool integration for M4C preservation. Separate-context reviewer evaluates actual final artifacts and attributed runtime evidence.

## Documentation impact and acceptance

The schema guide is the single contract for columns, normalization, timestamps, FK retention and deferred decisions. Update stale empty-source claims and link it from architecture/README/CLI/operations; preserve Swaggo, Scalar, Docker commands and unavailable-seeder boundary. Configuration/API standard and generated API need no change because runtime/config/HTTP behavior is preserved. Every subsequent fix must reassess these documents.

Acceptance: exact approved two-table schema; full disposable lifecycle and integrity checks PASS; Go checks PASS; development DB and local secrets preserved; accurate docs; final actual review PASS or truthful BLOCKED. Do not work on M5B.
