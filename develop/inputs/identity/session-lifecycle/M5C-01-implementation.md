# M5C-01 — Session Lifecycle & Revocation Foundation

- **Feature:** `identity/session-lifecycle`
- **Iteration:** `M5C-01`
- **Date:** 17 November 2026 (Asia/Makassar)
- **Approval:** User's MASTER PROMPT M5C authorizes targeted implementation, disposable verification and autonomous review/fix. No staging or commit.
- **Prerequisites:** [M4C PASS](../../../reviews/database/foundation-verification/M4C-01-review.md), [M5A PASS](../../../reviews/identity/session-schema/M5A-01-review.md), [M5B PASS](../../../reviews/identity/authentication/M5B-01-review.md), [PC01 PASS](../../../reviews/maintenance/pre-commit-quality/PC01-01-review.md), compared with current source. Initial working tree clean at `b657d3d`; `/private/tmp/m5c-before.json` records 120 file hashes, HEAD and staged-entry fingerprint without secret values.

## Objective and scope

Add protected `POST /v1/auth/logout` and `POST /v1/auth/logout-all` through existing handler/usecase/store/routes. Retain session records and token hashes; revoke using existing audit/revocation columns. Preserve login/me, bearer middleware, password/token primitives, current lifetime, shared pool/configuration, tenancy boundary, migration CLI and PC01 conventions. No schema/migration/config/dependency changes, renewal/rotation endpoint, idle timeout, cleanup worker, RBAC or seeder.

## Session audit

| Component | Existing behavior | Decision |
|---|---|---|
| Creation | Argon2id password verification, crypto/rand256-bit opaque token, SHA-256 verifier; conditional user UPDATE and session INSERT in one transaction | KEEP; existing user-row lock is reused for ordering |
| Lookup | Unique token-hash index, join to active user, `revoked_at IS NULL`, `expires_at > CURRENT_TIMESTAMP` | KEEP middleware and SQL validity check |
| Context | Private typed identity holds user/session IDs and safe /me fields | REUSE; no body/tenant/user target accepted |
| Audit/metadata | UTC explicit login timestamps, trusted-proxy IP, bounded agent; last_used_at initially null | KEEP; no per-request writes or new idle policy |
| Store timeouts/errors | Context-bounded GORM store on shared pool; generic401 vs operational500 | REUSE |
| Logout operations | Absent | IMPLEMENT two consumed store/usecase operations and protected routes |
| Renewal/rotation | Explicitly deferred, no complete approved absolute/extension/replay/concurrency policy | DEFER; require fresh password login after fixed expiry |
| Cleanup/retention | No approved retention, no cleanup job | DEFER; rows remain audit history while invalid sessions are rejected |
| Schema | M5A users/sessions and M5B unique token index, highest version000003 | KEEP all SQL; no new migration |

## Revocation and concurrency design

Extend the small existing Store boundary with current/all revocation. Use a bounded PostgreSQL READ COMMITTED transaction, lock the active `users` row FOR UPDATE, then lock the authenticated caller's owned session. All revocation writers follow user→caller-session order. Revalidate active user and caller state; sample `clock_timestamp()` after both locks and reject expired/revoked caller state against that fresh time. Transaction-start CURRENT_TIMESTAMP must not authorize a caller that expired during a lock wait.

Use one conditional bulk UPDATE (current ID additionally scoped for logout, user ID for logout-all), only unrevoked/unexpired rows at the sampled DB time. Set revoked_at and updated_at to the same server time and updated_by to the authenticated user. Preserve existing revocations/expired rows and physical records. No query per target session. Commit before success; rollback on context/SQL/commit failure. Return401 for a caller invalidated while waiting,500 for operational failure, never raw errors.

The existing CreateSession user UPDATE acquires the same row lock. Therefore login creation and logout-all have a defined database order: creations committed before logout-all's lock are included; successful password-authenticated creations ordered afterward are new logins and may remain valid, even when HTTP requests overlap. This is not permanent account lockout or cancellation of future password login. Already authenticated in-flight ordinary requests may finish; subsequent validations after revocation commit reject old tokens. Document these boundaries rather than claiming zero sessions forever after the response. Separate-context design review confirmed this is the smallest sufficient mechanism, without an epoch column or repurposed timestamps.

## HTTP and observability

Both routes require existing bearer middleware. No request payload or free user/session identifier controls scope; handler reads only typed identity. Success uses existing HTTP200 envelope with data:null; no token/hash/count. Repeat logout with the now-revoked bearer normally returns401; concurrent stale callers also fail on store recheck. Reuse safe security events/request metadata, including logout success and operational/denial outcomes; no token/body/header/raw SQL logging. No cacheable auth responses or privilege grants.

## Renewal and cleanup decisions

Recommend fresh password authentication after the unchanged configurable fixed lifetime. Rotation-only could reduce token age but would require old-token replay/in-flight request rules without extending expiry; renewal additionally needs an approved absolute cap and revocation/race policy. A separate refresh credential introduces more scope/storage/abuse controls. None is sufficiently approved now, so no endpoint/config is added. Retain expired/revoked rows until retention/anonymization/cleanup policy is approved; do not infer idle timeout from last_used_at. OWASP/MDN session guidance read alongside PostgreSQL17 locking/time documentation; existing project token transport remains authoritative.

## Affected files and verification

Change existing usecase/store/handler/routes/security-event code; extend existing HTTP/usecase fakes and spec checks. Add one grouped lifecycle integration helper consuming the current owned disposable authentication fixture, rather than duplicating database setup or creating a test per source file. Preserve old assertions; snapshot row counts before the existing index-only rollback after new fixtures are added.

Verify successful current logout and /me401 afterward, sibling session survival, logout-all including caller, other-user isolation, preserved row count/old revocations, invalid/expired/revoked/inactive callers, safe DB errors and audit timestamps. Use deterministic separate PostgreSQL connections/lock observation for repeated logout, authentication during/after revoke, both login/all orderings, expiration/revocation while waiting, deadline cancellation and transaction rollback. No sleeps as proof of lock ordering. Capture logs without printing tokens and verify generated API includes exactly the two new protected routes.

Run existing `go generate .`, build/vet/fresh full tests/diff checks and focused race tests on auth/routes/database integration as justified. Apply existing migrations and all fixture mutations only inside the existing newly created disposable database with owner/OID-checked cleanup. Development database access remains read-only; no migration version CLI on it. No Docker/container/volume lifecycle mutation or user-data deletion.

## Documentation and acceptance

Update existing authentication guide (single contract), architecture/operations/schema references, API standard and bilingual README; CLI/config documentation only to clarify current/deferred policy if needed. Keep commands/Swaggo/Scalar and unavailable-seeder boundary. Regenerate existing docs artifacts; no parallel spec or auth UI. Every fix reassesses docs.

Final review must cover the16 requested areas with actual evidence, scoped concurrency guarantees and truthful deferred decisions. Preserve any formal FAIL and make paired fixes; do not force PASS or modify M5B/PC01 history. Recommend a Conventional Commit only after inspecting actual final changes.
