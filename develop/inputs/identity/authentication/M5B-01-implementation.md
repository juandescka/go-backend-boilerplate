# M5B-01 — Authentication Foundation

- **Feature:** `identity/authentication`
- **Iteration:** `M5B-01`
- **Date:** 16 November 2026 (Asia/Makassar)
- **Approval:** User's MASTER PROMPT M5B authorizes audit, minimal implementation, isolated verification and autonomous review/fix. No automatic commit.
- **Prerequisites:** [M5A-01 PASS](../../../reviews/identity/session-schema/M5A-01-review.md), [M4C-01 PASS](../../../reviews/database/foundation-verification/M4C-01-review.md), compared with current source. Working tree clean at start.

## Objective and scope

Implement `POST /v1/auth/login`, database-backed opaque bearer authentication and `GET /v1/auth/me`, with the existing success/error envelopes. Preserve global identity, M5A migrations, runtime pool/readiness/shutdown, configuration precedence, CORS/security headers, Scalar and migration CLI. No registration, logout, refresh, tenancy/RBAC, seeder, default account or frontend.

## Existing component audit

| Area | Observed state | Decision |
|---|---|---|
| Identity schema | Exact users11/sessions13; M5A reviewed PASS, tested disposable only | KEEP; no deployment to mapatu_dev |
| Password/token/auth middleware | None active; no approved algorithm conflict | IMPLEMENT Argon2id and crypto/rand opaque bearer |
| Domain models/persistence | Empty domain directories; GORM wraps shared pgx pool | IMPLEMENT consumed identity models, concrete GORM store, small usecase storage boundary; no generic CRUD |
| Response helpers | Existing 200 success, explicit error codes/envelope | REUSE; 400 malformed JSON, 422 validation, 401 invalid credentials/session, 500 operational failures, 503 local crypto capacity |
| Request validation | Existing bind helper, field formatter and validator; default JSON decoder lacks EOF check | REUSE validator conventions; endpoint-specific bounded JSON parsing/EOF check, no global strictness change |
| Configuration | Process > .env > required JSON > defaults; retained inactive JWT/cookie/rate config | KEEP precedence and legacy validation; ADD authentication.sessionLifetimeMinutes / AUTH_SESSION_LIFETIME_MINUTES, default15, range1–1440 |
| Observability | Completion log excludes body/query/Authorization; security-event recorder and sanitization available | REUSE safe metadata/events; no raw errors/credentials/identifiers |
| Trusted proxies/security/CORS | Existing Gin configured proxy resolution, boundaries | KEEP; session IP from canonical ClientIP only; no cookie auth |
| Rate limiting | Config/metrics vocabulary only, no implemented limiter | DEFER; explicitly forbid public internet exposure until brute-force/login-abuse controls are supplied |
| Session lookup index | Only per-user index, token contract previously deferred | ADAPT via new 000003 unique token_hash index; do not alter M5A SQL |

## Design and security decisions

Use the already pinned golang.org/x/crypto Argon2id implementation (promote direct dependency, no version change). PHC v19 format with fixed supported m=65536 KiB,t=3,p=4, random16-byte salt and32-byte key; verify exact supported encoding/parameter bounds before allocation and compare with constant-time routine. Passwords are UTF-8, nonempty, at most1024 bytes, never trimmed. Unknown/missing/malformed password hashes take a dummy KDF path so credential denial does not have an obvious cheap exit. At most two service-owned password verifications run concurrently (128 MiB KDF working memory); excess returns503. This resource bound is not rate limiting.

Generate32 random bytes using crypto/rand, canonical unpadded base64url bearer token, hash its encoded form with SHA-256 to64 lowercase hex characters. Raw token returned only after durable session creation. Canonical token format checked before lookup, no query/cookie fallback. Unique token_hash index makes single-session lookup unambiguous. Existing duplicate hashes cause migration failure requiring inspected remediation, never automatic data cleanup/force.

Normalize email lookup in PostgreSQL using the exact M5A lower(btrim()) expression on both sides, preserving database-locale semantics. Trim only ordinary edge spaces at validation; reject invalid email syntax/other whitespace through validation, no provider rewriting. Generic401 for wrong/unknown/inactive/no-password accounts; DB failures remain500.

Login session creation and user last_login_at/audit update are one bounded transaction. Recheck active state and the verified password hash in the conditional user update before inserting the session, preventing a stale credential read from creating a session after password/activation changes. User/session audit actors are the authenticated user. Device name null; IP canonicalized through trusted proxies, user agent sanitized/bounded. No last_used_at maintenance or cleanup job yet.

Middleware joins users/sessions on the unique hash and accepts only unexpired, unrevoked sessions of active users. It passes a private typed standard-request context containing user/session IDs and the safe identity fields needed by /me; no secret hash enters that context. Identity resolution has a configured DB query timeout and cancellation propagation. Authentication grants no tenant permission.

## Verification and documentation impact

Run build/vet/fresh tests/diff checks. Unit checks cover password encodings/bounds/dummy behavior, token generation/canonical format/hash, generic login denial, persistence failures, input validation, middleware/context isolation and operational errors. Disposable PostgreSQL integration applies actual migrations through the CLI and uses internally hashed random fixtures, tests login→me, SQL hash-only storage, expiration/revocation/inactive users, rollback of failed session writes, lookup index, no startup migration, health/readiness/docs, exact cleanup ownership and untouched development catalog. No raw credentials/tokens printed. M5A test source is pinned to its original two migrations so the new index does not rewrite that milestone's lifecycle contract.

Update one authentication API guide, schema deferred-index notes, architecture/configuration/operations/API standard/CLI and bilingual READMEs/examples where behavior changes. Annotate endpoints and bearer scheme, regenerate only existing docs with `go generate .`. Preserve generated/output locations. Every subsequent fix reassesses affected documentation. Final structured review must use actual evidence and state whether independent review occurred.

## Acceptance

All user M5B criteria: schema-compatible login/password verification; secure opaque token/hash persistence; expiration/revocation/active-user checks; protected current user; safe HTTP/errors/logs; no implicit authorization or out-of-scope feature; relevant unit/disposable integration PASS; current generated spec/reference docs; preserved M4C behavior; final honest PASS or BLOCKED.
