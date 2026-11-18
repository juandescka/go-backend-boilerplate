# M5B-01 — Authentication Foundation Review

- **Feature:** `identity/authentication`
- **Iteration:** `M5B-01`
- **Input:** [M5B-01 implementation](../../../inputs/identity/authentication/M5B-01-implementation.md)
- **Date:** 17 November 2026 (Asia/Makassar)
- **Verdict:** **PASS for the approved M5B foundation; not a production-readiness verdict**
- **Review method:** Separate-context reviewer `/root/m5a_review` independently inspected current source, SQL, tests, generated specification, documentation and preservation fingerprints. Runtime/build/test evidence below was executed by the primary implementer and is explicitly attributed. The reviewer did not repeat migrations, runtime tests, builds or Docker operations, and edited only this review.

## Scope and actual changes

`POST /v1/auth/login` verifies global email/password credentials and returns an opaque bearer token after durable session creation. `GET /v1/auth/me` uses bearer middleware to return safe current-user data. HTTP translation, authentication usecase, password/token operations and concrete GORM persistence are separated within the existing package structure. A three-method Store interface serves the actual authentication consumer; no generic repository or unrelated framework is introduced.

Passwords use fixed-policy Argon2id PHC v19, 64 MiB/three passes/four lanes, a random 16-byte salt and 32-byte result. Bounded canonical decoding rejects unsupported parameters before allocation; absent/malformed hashes perform fixed dummy work, with constant-time key comparison. Password input is bounded to 1024 bytes and service-owned verification concurrency to two. This capacity limit is **not** brute-force protection.

Tokens contain 256 random bits from crypto/rand, encoded as 43 canonical unpadded base64url characters. Only their 64-character hex SHA-256 verifier is persisted. Additive migration `000003_index_session_token_hash` supplies unique indexed lookup; its down removes only the index. Original M5A table SQL is unchanged. Login transactionally rechecks active state and the verified password hash, updates audit/login timestamps, and inserts the session. Persistence failure returns no token and rolls back both writes.

Middleware accepts one well-formed Authorization header, queries unexpired/unrevoked sessions joined to active users, and passes a private typed identity context without credentials or authorization grants. Lifetime defaults to 15 minutes through the existing loader's independent authentication setting. Legacy JWT/cookie validation remains compatible but does not activate JWT/cookie authentication. Registry and routes reuse the existing SQL pool, HTTP boundaries, response helpers and safe logging. Current source/generated docs and the bilingual/reference documentation describe these contracts and deployment limits.

## Independent inspection evidence

| Reviewer check | Result |
|---|---|
| Inspect actual password/token/service/store/handler/middleware/models and callers | PASS; bounded KDF, secure randomness, hash-only storage, parameterized contextual queries, transaction/recheck, safe errors/context and no implicit tenant privilege |
| Compare M5A SQL, new index pair and migration-test adaptation | PASS; only additive index SQL; M5A integration copies original versions 1/2 to preserve that milestone's lifecycle assertions |
| Inspect proxy, logging, response and global HTTP callers | PASS; canonical trusted-proxy IP, sanitized agent, no body/Authorization logging, unchanged CORS/security/operational contracts |
| Inspect generated Swagger and current annotations | PASS; required login email/password, masked password format, public login, protected /me, one BearerAuth header definition and four active endpoint paths |
| Recompute baseline SHA-256 using `/private/tmp/m5b-before.json` | 98 files; exactly 20 authorized existing files changed, none missing; original SQL, local secrets, connector/runner/CLI, HTTP boundaries, Docker, go.sum and prior history unchanged |
| Inspect 39-file pre-review scope manifest | PASS; source/tests/schema/config/generated/reference/input scope agrees with approval |
| `git diff --check` and code-aware Markdown local-path checks | PASS |

Two minor documentation nits were identified and corrected before this formal review: a duplicated lifetime example in `.env.example`, and an overbroad table-only rollback statement now qualified as M5A-specific. Their corrections were independently rechecked. No remaining actionable finding was found.

## Actual verification supplied by primary implementer

The reviewer read the secret-free execution summary `/private/tmp/m5b-verification-evidence.md`; the following are attributed results, not independently executed runtime claims.

| Check | Actual result |
|---|---|
| `go build ./...`, `go vet ./...`, `go test -count=1 ./...` | PASS using writable temporary Go cache; all ordinary packages pass |
| Focused race tests: password, authentication usecase, routes | PASS; 14.046s / 14.876s / 7.446s; includes concurrent admission, cancellation/context isolation, malformed credentials/tokens, known SHA-256 vector and safe failure/logging checks |
| Existing `go generate .` | PASS with pinned Swaggo 1.16.6; existing Go/JSON/YAML artifacts regenerated; generator and Scalar unchanged |
| Disposable `TestAuthenticationPostgresIntegration` with current CLI | PASS, 1.92s; owned `m5b_verify_08ff88554f136ba6` created fresh, owner/OID checked before exact-name DROP, absence verified |
| Actual registry/CLI/schema behavior | Fresh registry creates no tables/metadata; health/ready 200 before schema; absent-schema login safe 500; CLI applies 1–3 cleanly and repeat-up is no-change |
| Login/session/current-user flow | Normalized email login and bearer /me 200; identical 401 wrong/unknown/inactive/no-password denial with no failed-login sessions; only token hash stored; duplicate verifier rejected 23505 |
| Validity, audit and metadata | Unknown/expired/revoked/inactive bearer 401; query-only token rejected; correct lifetime/audit fields; untrusted forwarded IP ignored and agent sanitized; responses/logs omit credentials and hashes |
| Transaction and operational failures | Stale password/activation recheck rejects session creation; forced insert failure rolls back last_login_at and returns safe 500/no token; 100ms canceled row-lock wait rolls back; closed pool yields auth 500 while health stays 200/readiness 503 |
| Index rollback/reapply | Actual CLI 3→2 removes only token index, retaining users/session data; reapply reaches clean 3; no force |
| M5A original-source disposable regression | PASS, 1.33s; owned `m5a_verify_05fc5f6f8b6fde3d` removed safely; original 2→1→none/reapply lifecycle retained |
| Existing read-only shared-pool regression | PASS, 0.02s; configured pool identity/readiness/metrics/cleanup preserved |
| Local password benchmark | Approximately 63.6ms verification on Apple M1 Pro; local evidence only, not deployment sizing |
| Docker, development and cleanup | Existing PostgreSQL 17 healthy without lifecycle mutation; development catalog unchanged before/after both integrations; final read-only metadata SELECT reports version 2 clean and zero remaining M5A/M5B verification databases |
| Secrets/documentation/preservation | Configured-secret scan finds no nonignored-file matches; protected baseline files unchanged; link and diff checks pass |

Initial sandbox/offline generator attempts could not resolve proxy/deprecation metadata; the authorized network generation succeeded. An early route test ran before that generation completed and saw the prior operational-only spec; regenerated ordinary/race route checks then passed. A preliminary link utility also misread inline Go generic syntax and was corrected. These are recorded honestly, not counted as successful early attempts or invented formal FAIL iterations.

## Seventeen-area acceptance assessment

| Required area | Assessment |
|---|---|
| 1. M5A schema compatibility | PASS — prerequisite PASS/source verified; original table migrations preserved; consumed models match columns |
| 2. Password hashing security | PASS — bounded fixed Argon2id policy, random salt, canonical decoder, constant-time comparison and dummy work |
| 3. Login correctness | PASS — normalized lookup, generic denials, explicit active/password checks, durable session before response |
| 4. Token randomness | PASS — 32 crypto/rand bytes, canonical header-safe encoding |
| 5. Token hash storage | PASS — SHA-256 verifier only; unique indexed lookup and SQL evidence |
| 6. Session expiration | PASS — independent validated lifetime and database expiry check |
| 7. Session revocation checks | PASS — revoked_at checked on protected requests without adding revocation APIs |
| 8. Authentication middleware | PASS — strict single-header format, safe failure, private request identity and request isolation |
| 9. Active-user enforcement | PASS — login verification, transactional state/hash recheck and session/user join |
| 10. Error response security | PASS — generic 401, safe operational 500, bounded-capacity 503; no raw secrets/SQL/errors |
| 11. Tenant/RBAC separation | PASS — global identity only; no roles, permissions, memberships or default privileges |
| 12. Source architecture | PASS — consumed models/store boundary, existing package roles and explicit shared-pool wiring |
| 13. Database integrity | PASS — additive index, atomic writes, context timeout/cancellation and non-destructive original-schema preservation |
| 14. Documentation | PASS — auth guide, examples/config/reference/EN-ID synchronization; security/deferred limits explicit |
| 15. OpenAPI specification | PASS — generated current endpoints/DTOs/errors/header security; existing Swaggo/Scalar retained |
| 16. Test evidence | PASS — actual full Go checks, race tests, disposable integration and targeted regressions |
| 17. M4C preservation | PASS — same connector/pool/readiness/shutdown/CLI/HTTP boundaries, protected fingerprints and fresh regression evidence |

## Completion and limitations

One formal iteration: **M5B-01 PASS**. No post-review fix iteration is required; no historical FAIL was overwritten. No automatic commit was performed.

All M5B migration/fixture/business writes were **disposable-only**. Current `mapatu_dev` already contains `users`, `auth_sessions` and `schema_migrations` from M5A, at version 2 clean; it has **no M5B token-hash index 000003**. This development schema remained unchanged. Historical M5A's earlier empty-development observation is not the current state, and this review does not claim M5B was deployed there.

**Login must not be exposed to the public internet before appropriate login-abuse/brute-force controls are supplied.** There is no active rate limiter; two concurrent KDF slots only bound local work. Argon2 is bounded but not interruptible. HTTPS deployment, distributed abuse controls, real-user bootstrap, password-policy evolution and production sizing remain separate work. No production TLS/complete browser-rendering/full M4C outage rerun is claimed; current Scalar HTML/spec and operational components were checked proportionally.

Registration, logout, refresh/renewal, session management/cleanup, alternative providers, tenancy/RBAC and M5C are not implemented. **Seeder CLI: NOT IMPLEMENTED.** The reviewed authentication foundation is ready for separately authorized M5C work, after deliberate migration deployment to a verified target.
