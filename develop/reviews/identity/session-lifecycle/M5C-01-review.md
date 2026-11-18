# M5C-01 — Session Lifecycle Review

- **Feature:** `identity/session-lifecycle`
- **Iteration:** `M5C-01`
- **Input:** [M5C-01 implementation](../../../inputs/identity/session-lifecycle/M5C-01-implementation.md)
- **Date:** 17 November 2026 (Asia/Makassar)
- **Verdict:** **PASS for the approved session lifecycle foundation; not a production-readiness verdict**
- **Review method:** Separate-context reviewer `/root/m5a_review` independently assessed the design, actual source/tests/docs/specification, prerequisite history and preservation fingerprints. Runtime/build/test results below were executed by the primary implementer and are explicitly attributed. The reviewer performed static/read-only checks, did not repeat builds, tests or database operations, and edited only this review.

## Scope and implementation assessment

Protected `POST /v1/auth/logout` and `POST /v1/auth/logout-all` extend the existing handler/usecase/store/routes. Both take ownership only from the private authenticated context, retain session rows/hashes, return the existing HTTP200/data:null envelope after commit, and distinguish generic401 from safe operational500. Optional body identifiers cannot redirect revocation. No token/hash/session count is returned or logged.

The existing Store gains two consumed methods using one bounded READ COMMITTED transaction: lock the active user, lock the owned caller session, sample database `clock_timestamp()` after both waits, reject expired/revoked callers, then issue one ownership/time-filtered UPDATE. Current logout additionally filters by session ID. All affected rows receive the same revoked_at/updated_at timestamp and authenticated updated_by; old revocations and expired rows remain untouched. Failure rolls back the operation.

This user-row lock conflicts with login's existing conditional user UPDATE. A creation committed before logout-all obtains the lock is visible to its later READ COMMITTED statement and is revoked if active. A password-authenticated creation ordered afterward is a new valid login, even when HTTP requests overlap. The documentation explicitly allows this case and already-authenticated in-flight requests; it does not promise account lockout or zero sessions after the response. Caller rechecks after lock waits prevent stale middleware context from revoking siblings.

The reviewer checked these semantics against official [PostgreSQL 17 row-lock conflicts](https://www.postgresql.org/docs/17/explicit-locking.html#LOCKING-ROWS), [READ COMMITTED snapshots](https://www.postgresql.org/docs/17/transaction-iso.html#XACT-READ-COMMITTED) and [current-time functions](https://www.postgresql.org/docs/17/functions-datetime.html#FUNCTIONS-DATETIME-CURRENT). No epoch column, repurposed audit timestamp or custom locking framework is needed for the stated contract.

Existing middleware, password/token primitives, login creation, lifetime/configuration, schema/migrations, shared pool, HTTP boundaries and PC01 rules are preserved. Renewal/rotation has no approved absolute-cap/replay/concurrency policy and is correctly deferred. Idle timeout, last_used_at writes, retention/cleanup and session listing/device APIs are also deferred; no speculative endpoint/worker is added. Requested OWASP/MDN references were checked, with broader hardening limitations disclosed rather than claiming full compliance.

## Independent inspection and preservation

| Reviewer check | Result |
|---|---|
| Current source, original callers, SQL ownership/lock/time/transaction behavior | PASS; consistent user→caller lock order, fresh-time recheck, no cross-user target or reactivation/deletion path |
| Actual grouped lifecycle test and existing integration changes | PASS; reuses the owned authentication fixture; real blocking-PID observations establish ordering, both creation/revocation directions and expiry during waits; old M5B assertions retained |
| HTTP/specification/documentation | PASS; both routes protected, six actual Swagger paths, correct responses/security, no renewal path, aligned EN/ID summaries and one existing auth guide |
| Baseline `/private/tmp/m5c-before.json` | Independently compared 120 files: exactly 21 expected baseline files changed, none missing; 23 changed/new files before this review agree with the manifest |
| Protected files and Git | Middleware/password/token/config/examples/SQL/dependencies/pool/CLI/PC rules/local secrets/prior history unchanged; HEAD and staged-entry fingerprints match initial `b657d3d`; nothing staged |
| Tracked/cached diff checks and code-aware Markdown local paths | Independently executed: PASS |

Two pre-review improvements were suggested and independently rechecked: qualify the shared helper's creation guarantee as logout-all-specific, and assign test expiry only after the actual blocking relation is observed. The latter ensures expiry truly occurs during the locked wait, rather than accidentally before it on a delayed run. Neither required a product-logic fix; final live race verification covers the strengthened fixtures.

## Actual executed evidence supplied by primary

The reviewer read `/private/tmp/m5c-verification-evidence.md` and the final file manifest. These execution results are attributed to the primary, not represented as independently repeated tests.

| Check | Actual result |
|---|---|
| `go generate .` | PASS with existing pinned Swaggo 1.16.6; existing three artifacts regenerated; six paths and both new POST BearerAuth contracts 200/401/413/415/500 |
| `go build ./...`, `go vet ./...`, `go test -count=1 ./...` | PASS; full ordinary suite, with opt-in PostgreSQL coverage separately executed below |
| Focused authentication/routes race tests | PASS, 14.957s/14.306s; lifecycle scope/error/cache/logging assertions and existing authentication/PC protections |
| Cumulative disposable authentication/lifecycle integration | PASS, 3.88s; actual current CLI and registry on owned `m5b_verify_f931939c239b2fe3`, ownership/OID-checked removal and absence confirmed |
| Initial live PostgreSQL race run | PASS, 5.82s; owned `m5b_verify_28491e3cc6bede8b` removed safely |
| Final PostgreSQL race run after test/comment improvements | PASS, 5.99s; owned `m5b_verify_068d888037b0d760` removed safely; expiry assigned only after observed user/caller blocking |
| Current/all HTTP behavior | Successful logout, subsequent/repeated old-token401, sibling survival, logout-all including caller, body IDs ignored for scope, exact other-user snapshot preservation, retained rows and audit/old-revocation timestamps |
| Real concurrency | Two preauthenticated current logouts yield one commit/one unauthenticated result; before-commit authentication may succeed, post-commit rejects; both CreateSession/logout-all lock orders behave as documented |
| Wait-time validity and failure safety | Expired/revoked/inactive caller after user wait and expired caller after session wait rejected without sibling revocation; CHECK failure atomically rolls back; 250ms context cancels lock wait without partial writes |
| Existing foundation regression | Login→me, normalization/Argon2/hash-only storage, generic denials, lifetime/metadata/audit, stale-account guard, insert rollback, timeouts, safe DB500, health200/ready503, current Scalar HTML/spec and index-only rollback/reapply retained |
| Logs, preservation and cleanup | Captured old/new credentials and hashes absent from logs; configured-secret byte scan clean; links/spec/diff checks pass; all fixture/migration writes disposable-only |
| Final read-only development observation | `mapatu_dev` remains version2 clean, no M5B index000003 deployed by this task; development catalogs unchanged before/after integrations; zero remaining verification databases |

An initial compound Docker-status/read-only metadata request did not execute because automatic approval review timed out. A narrower permitted read-only metadata retry succeeded; the timeout supplies no successful-check evidence and leaves no blocker. PostgreSQL17 availability is established by actual host integrations/metadata. No Docker/container/volume lifecycle operation, development migration/write/reset or seeding was performed.

## Sixteen-area acceptance assessment

| Required area | Assessment |
|---|---|
| 1. Prerequisite milestones | PASS — M4C/M5A/M5B/PC01 PASS history and current relevant source inspected |
| 2. Current logout correctness | PASS — owned caller only, durable revocation, old token rejected, siblings retained |
| 3. Logout-all ownership | PASS — authenticated identity only, includes caller, other-user snapshot unchanged |
| 4. Effective revocation | PASS — revoked rows retained, never cleared/reactivated; subsequent validations fail |
| 5. Expiration enforcement | PASS — unchanged server-side middleware plus fresh post-lock caller-time check |
| 6. Renewal decision/safety | PASS — explicitly deferred with options/risks; no unsafe extension or replacement credential |
| 7. Race conditions | PASS — shared per-user PostgreSQL order, stale-caller rechecks and deterministic separate-connection tests; later fresh-login boundary explicit |
| 8. PostgreSQL operations | PASS — one scoped UPDATE, consistent audit time, bounded transaction, cancellation/rollback and no schema change |
| 9. Middleware compatibility | PASS — existing bearer/header/context/active-user behavior unchanged |
| 10. API contracts | PASS — existing envelopes, protected POSTs, generic401/safe500, no body-based authority |
| 11. Security/token handling | PASS — no new token transport/hash exposure, cross-user scope or implicit tenant privileges |
| 12. Code quality/comments | PASS — small existing boundaries, selective accurate comments, no locking/testing framework |
| 13. Risk-based tests | PASS — shared fixture, real DB concurrency/time/failure coverage, retained M5B/PC tests and focused race checks |
| 14. Documentation | PASS — precise concurrency/expiry/idempotency, deferred policy and deployment limits, bilingual consistency |
| 15. OpenAPI synchronization | PASS — existing generator/artifacts/Scalar contract, exactly two new protected paths |
| 16. Preservation of existing changes | PASS — initially clean baseline, scoped manifest/fingerprints, unchanged protected contracts/HEAD/staged entries |

## Findings and completion

No remaining actionable finding. One formal iteration: **M5C-01 PASS**. No FAIL review was overwritten or invented; pre-review refinements were completed before this verdict. No automatic staging or commit occurred.

**No new migration is needed for M5C.** Existing M5B version000003 remains pending on development; its deliberate deployment to a verified target is separate from disposable verification. No production TLS/deployment or full browser-rendering rerun is claimed. Public login still requires appropriate abuse controls and HTTPS before exposure. Logout-all is an ordered revocation operation, not permanent prevention of new password logins or cancellation of already authenticated work.

Renewal/rotation, idle policy, retention/cleanup, session listing/device management, real-user bootstrap and broader hardening remain separately scoped. No refresh endpoint, worker, tenant/RBAC grant or default account is introduced. **Seeder CLI: NOT IMPLEMENTED.**
