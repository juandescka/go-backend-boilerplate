# Authentication Foundation

M5B implements global email/password login and database-backed opaque bearer authentication. M5C adds current-session logout and logout-all with server-side revocation. It does not grant tenant access, roles or permissions. Registration, refresh/renewal, session listing/device-management APIs, password reset and alternative providers are not implemented.

## Deployment boundary

Login has process-local fixed-window IP and database-normalized identifier admission limits. HTTP 429 uses `RATE_LIMIT_EXCEEDED` with generic text and `Retry-After` in whole seconds (rounded up). Limits count admitted attempts including success, with no account-state-dependent reset; no existence or password correctness is disclosed by throttle details. See [configuration](../configuration.md#login-abuse-policy). Multiple replicas and public ingress need a reviewed deployment abuse policy; counters are not distributed. A two-verification process-local capacity bound protects Argon2 memory use; it is not a rate limiter, per-account throttle or distributed defense. This foundation is not a claim of production readiness.

Use HTTPS outside controlled local development. Authorization headers and password/token values must be redacted in any ingress/proxy/client logging. The application completion logger records no request/response body, query string or Authorization header. Existing security-event logging records generic outcomes and safe request metadata without email, password, bearer token or hashes. CORS and security headers apply before authentication; deployment must supply HTTPS.

## POST /v1/auth/login

Public JSON endpoint:

```json
{"email":"user@example.com","password":"<password>"}
```

The password shown is a placeholder, not an account credential. Local operators can initialize the first platform administrator through the [interactive bootstrap CLI](../cli.md#initial-platform-administrator). Production provisioning remains deferred.

Login limits its body to 16 KiB, parses one JSON value and rejects trailing JSON. Unknown JSON properties follow the existing permissive convention. Email is required, valid under the existing validator's `email` rule and at most 255 UTF-8 bytes after trimming ordinary U+0020 edge spaces. Other whitespace that fails email validation is rejected. Password is required, valid UTF-8 and at most 1024 bytes; it is never trimmed or truncated. These are login input limits, not a registration/password-strength policy. Malformed JSON/types give400; invalid fields give422; oversized bodies give413; the global media-type boundary gives415.

Email lookup uses parameterized PostgreSQL `lower(btrim(email)) = lower(btrim(input))`, exactly matching the M5A unique expression and database locale. Stored email text is preserved in responses. No Go-only case conversion, Unicode/IDNA normalization, provider dot/plus rewriting or tenant key changes identity.

Success is HTTP200 with the existing envelope:

```json
{
  "success":true,
  "message":"Login successful.",
  "data":{
    "access_token":"<opaque-access-token>",
    "token_type":"Bearer",
    "expires_at":"2026-11-16T12:00:00Z",
    "user":{
      "id":"00000000-0000-4000-8000-000000000001",
      "email":"user@example.com",
      "display_name":"Example User",
      "email_verified_at":null
    }
  }
}
```

The raw token is returned only by successful login. No token is returned unless the database transaction commits. Responses use `Cache-Control: no-store` and `Pragma: no-cache`. Existing helpers omit `meta` when unused.

Unknown email, wrong password, inactive account and null/malformed/unsupported password hashes all receive the same HTTP401 envelope:

```json
{"error":{"code":"UNAUTHORIZED","message":"Invalid email or password."}}
```

These account-denial paths perform one bounded Argon2id verification, including dummy work for absent/unusable hashes. This removes an intentional cheap account-existence path, without claiming exact wall-clock equivalence across database/network/scheduler behavior. Database/transaction failures remain500, not invalid credentials. Saturated local password-verification capacity gives503 `AUTHENTICATION_UNAVAILABLE`.

## Password and token contracts

Passwords use Argon2id PHC v19, currently accepting exactly `m=65536,t=3,p=4`: 64 MiB, three passes, four lanes, 16 random salt bytes and32 derived-key bytes. Verification uses a constant-time comparison. Unsupported algorithm/version/parameter/encoding/length values fail closed, with fixed bounded dummy work; arbitrary database PHC parameters cannot allocate arbitrary resources. Password parameters are a reviewed code policy, not runtime environment settings. The existing pinned `golang.org/x/crypto` dependency supplies Argon2id; no second password algorithm or automatic rehash is implemented. See [Go Argon2 documentation](https://pkg.go.dev/golang.org/x/crypto@v0.41.0/argon2) and [OWASP password storage guidance](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html).

Tokens contain32 `crypto/rand` bytes (256 bits), encoded as43 canonical unpadded base64url characters. Only lowercase hex SHA-256 of that encoded token is persisted as `auth_sessions.token_hash`. SHA-256 is used for random-token verification only, never password hashing. The unique `auth_sessions_token_hash_uidx` makes lookup indexed and unambiguous. There is no signing key, JWT, refresh token, cookie authentication or query-token transport.

Successful login atomically rechecks active state and the verified password hash, updates `users.last_login_at`/`updated_at`/`updated_by`, and inserts one session. Failed persistence rolls back both writes. Session audit timestamps are explicit UTC values, with user ID as both actor fields. IP comes from Gin's configured trusted-proxy resolution, never directly from an untrusted forwarded header. User agent is sanitized and bounded using the existing helper. Device name, revoked_at and last_used_at initially remain null. Session expiry comes from one configured lifetime; there is no sliding expiry or per-request last_used_at write.

## GET /v1/auth/me

Send exactly one header:

```http
Authorization: Bearer <access_token>
```

The Bearer scheme is case-insensitive; the token is case-sensitive and must use its exact canonical format. Duplicate Authorization fields, other schemes, missing/empty/malformed/unknown tokens are rejected. Tokens supplied only via query parameters or cookies are not accepted.

Middleware joins sessions with users and requires `revoked_at IS NULL`, `expires_at > CURRENT_TIMESTAMP` and active user state on every protected request. Missing user/session records also fail closed. The private typed request context contains user/session IDs and the safe identity data needed by /me. It never contains password/token hashes or the raw bearer token, and it is not a tenant authorization context.

Success is200:

```json
{
  "success":true,
  "message":"Current user retrieved.",
  "data":{
    "id":"00000000-0000-4000-8000-000000000001",
    "email":"user@example.com",
    "display_name":"Example User",
    "email_verified_at":null
  }
}
```

Invalid authentication always gives401 `UNAUTHORIZED`, `WWW-Authenticate: Bearer`, and the generic message `Authentication is required.` Database failures give500 with a safe message, not401. Responses include `Cache-Control: no-store`. No tenant list, roles, permissions or sensitive session internals are returned.

## Configuration and operations

`authentication.sessionLifetimeMinutes` defaults to15, accepts1–1440, and supports `AUTH_SESSION_LIFETIME_MINUTES`. Existing precedence remains process environment > `.env` > required JSON > defaults. `database.queryTimeoutSeconds` bounds each store lookup and the complete login persistence transaction, with request cancellation propagated. Argon2 itself cannot be interrupted; cancellation is checked before/after its bounded computation.

Legacy JWT/cookie configuration validation is preserved for compatibility; those fields do not control opaque session lifetime, sign tokens or activate cookies. Keep existing local secrets/configuration. See [configuration](../configuration.md).

Apply reviewed migrations explicitly to the correct target before using auth. Server startup constructs dependencies and checks connectivity but does not inspect/create auth tables, run migrations, or seed users. Consequently health/readiness may be200 on a database without schema while auth returns a safe500. See [CLI](../cli.md) and the [identity/session schema](../database/identity-session-schema.md).

## Verification and deferred work

Unit tests cover password/token bounds, generic login denial, safe errors/logs, middleware format/context and current-user data. The opt-in database test builds the real registry, runs the actual migration CLI only against a newly created ownership/OID-checked disposable database, and exercises login→me, hash-only storage, validity, logout/all ownership, deterministic concurrent creation/revocation and lock-wait expiry, transaction rollback/cancellation and safe cleanup:

```bash
go build -o /private/tmp/hris-m5b-backend .
M5B_INTEGRATION=1 M5B_CLI_BINARY=/private/tmp/hris-m5b-backend go test -count=1 -run '^TestAuthenticationPostgresIntegration$' -v ./internal/registry
```

The disposable integration uses the full current migration source, including Module A. Before its original token-index rollback check, it rolls later module versions back without changing identity/session rows, then reapplies the full source. This preserves the existing authentication regression checks as schema modules are added.

OpenAPI remains Swaggo Swagger2.0 with an API-key header security definition describing `Bearer <access_token>`. Scalar reads the same generated spec and does not persist credentials. Use the existing `go generate .`; no frontend authentication UI was added.

M5C implements logout and logout-all below. Renewal/rotation, cleanup and session listing/device-management APIs remain deferred. Distributed abuse controls, production account provisioning, assignment management, production deployment and broader email/provider policies require separately scoped work. Membership and RBAC enforcement mechanics have a [separate contract](../database/tenant-membership-rbac-schema.md). Authentication alone grants no HRIS data permission.

## POST /v1/auth/logout

Requires the same single `Authorization: Bearer <access_token>` header as /me. No body is required. User/session IDs in an optional JSON body are ignored; only the private authenticated context controls ownership.

The endpoint revalidates the caller under database locks, sets the current row's `revoked_at`, `updated_at` and authenticated `updated_by`, and retains the row/hash. Other sessions are unaffected. Success is200 with the existing envelope:

```json
{"success":true,"message":"Logged out.","data":null}
```

The old token subsequently receives401 on /me or either logout route. Repeating logout normally gives401 because the bearer is no longer valid. Concurrent callers that passed middleware before another logout committed also receive401 when the locked persistence recheck finds revocation. No operation clears `revoked_at` or reactivates a session.

## POST /v1/auth/logout-all

Requires bearer authentication and no request payload. Revokes this user's active sessions, including the calling credential, using one bulk update rather than a query per session. It never targets another user or takes user/tenant identity from request fields. Already-revoked and already-expired rows are retained without rewriting their audit/revocation fields.

Success is200:

```json
{"success":true,"message":"All sessions logged out.","data":null}
```

All tokens revoked by that operation fail subsequent authentication. Other users' sessions stay unchanged. Responses contain no token, hash, session ID or session count. Protected responses use `Cache-Control: no-store`; authenticated handler responses also set `Pragma: no-cache`. Invalid/no-longer-valid callers get generic401 `UNAUTHORIZED` with `WWW-Authenticate: Bearer`, while database/timeout/commit failures get safe500 `INTERNAL_SERVER_ERROR`. Global request-size/media-type limits still apply. Clients may discard local credentials after success, but server revocation is authoritative.

## Concurrency and expiration contract

Revocation uses a context-bounded READ COMMITTED PostgreSQL transaction. It locks the active user's row, then the owned caller-session row, always in that order. Login's existing conditional user UPDATE acquires a conflicting lock on the same user row. This orders session creation and logout-all across application processes without an in-memory lock, revocation epoch, schema change or long transaction around password verification.

- A session creation that commits before logout-all obtains the user lock is included in the revocation.
- A successful password-authenticated creation ordered after logout-all is a **new login** and may remain valid, even if the HTTP requests overlapped or it completes before the logout-all response arrives. Logout-all is not account lockout or cancellation of future password authentication; no guarantee of zero sessions after the response is made.
- A request authenticated before revocation commits may finish. The next authentication lookup after commit rejects a revoked token; no cached bearer identity is accepted across requests.
- Caller user/session state is revalidated after lock waits. `clock_timestamp()` is sampled after both locks; expiry at or before that time is rejected. Transaction-start CURRENT_TIMESTAMP would be stale after a wait, so it is not used for this persistence decision. All affected rows use the same database revocation/audit timestamp.

Every bulk update is scoped by user ownership and current-session logout additionally by session ID. Persistence failure rolls back the complete revocation, preserving audit state; deadlines cancel lock waits. These ordering guarantees apply to application writes using the existing store, not arbitrary external SQL bypassing its lock protocol. See [PostgreSQL row locking](https://www.postgresql.org/docs/17/explicit-locking.html#LOCKING-ROWS) and [current-time functions](https://www.postgresql.org/docs/17/functions-datetime.html#FUNCTIONS-DATETIME-CURRENT).

Absolute lifetime remains the configured fixed expiry from login, default15 minutes (range1–1440). Middleware still enforces `expires_at > CURRENT_TIMESTAMP`, unrevoked state and active user on each request. No automatic token issuance, sliding renewal, new timeout key, idle timeout or per-request last_used_at write is introduced. This preserves the reviewed M5B behavior while enforcing server-side invalidation; see [OWASP session expiration](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html#session-expiration) and [MDN session lifetime](https://developer.mozilla.org/en-US/docs/Web/Security/Authentication/Session_management#session_lifetime). The broader OWASP idle-timeout recommendation remains future hardening pending an explicit Mapatu policy.

## Renewal, rotation and retention decisions

| Option | Tradeoff / required design | Status |
|---|---|---|
| Fresh password login after fixed expiry | Keeps a bounded lifetime and the existing credential checks, without a second token protocol | Recommended current approach |
| Rotation without extending expiry | Needs old-token replay, in-flight request and concurrent rotation/revocation rules | DEFER |
| Renewal extending expiry | Needs an approved absolute cap, reauthentication/abuse controls and concurrent renewal policy | NEEDS DECISION; not implemented |
| Separate refresh credential | Adds credential/storage/replay/rotation policy beyond the approved bearer model | DEFER |

There is no /refresh, /renew or automatic token replacement. Expired/revoked tokens cannot renew themselves. Cleanup is also DEFER: no retention/anonymization policy is approved, so expired/revoked rows remain stored as history but fail authentication. No cron, periodic goroutine, worker or physical session deletion is introduced. Session listing/device management, production account provisioning, assignment management and deployment-specific public-exposure controls remain separately scoped.
