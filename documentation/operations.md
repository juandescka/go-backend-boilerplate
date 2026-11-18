# Operations — Health, Readiness & Lifecycle

## 1. Operational endpoints

The server registers both probes on the root router, outside authentication and login rate limiting. No JWT or permissions are required. Responses use the existing JSON envelope.

| Endpoint | Purpose | Normal status |
|---|---|---|
| `GET /health` | Process liveness; no dependency checks | 200 |
| `GET /ready` | Primary PostgreSQL pool connectivity | 200 or 503 |

Health returns:

```json
{"success":true,"message":"Service is healthy.","data":{"status":"ok"}}
```

Ready returns on success:

```json
{"success":true,"message":"Service is ready.","data":{"status":"ready"}}
```

All readiness failures return HTTP 503:

```json
{"error":{"code":"SERVICE_NOT_READY","message":"Service is not ready."}}
```

## 2. Dependency checks and timeout

`DatabaseReadinessChecker.Check(ctx)` calls `PingContext(ctx)` on the registry-owned `*sql.DB`, shared with GORM and DB metrics. No pool is created per request. No business query, schema inspection, migration, Redis, or external API is involved. Connectivity does not prove business schema or workflows are correct.

The handler derives a **2-second deadline** from the request context, preserving earlier cancellation/deadlines. A failed ping, timeout, missing checker/pool, or shutdown state produces 503. Driver errors are never sent to clients. Internal logs contain bounded reason codes: `shutting_down`, `unavailable`, `timeout`, `ping_failed`.

| Condition | `/health` | `/ready` |
|---|---|---|
| HTTP running, DB healthy | 200 | 200 |
| DB fails after startup | 200 | 503 |
| DB ping exceeds deadline | 200 | 503 |
| Shutdown flag active, request still reaches handler | 200 | 503 |
| DB fails during startup | No listener | No listener |
| HTTP listener closed | Unavailable | Unavailable |

Readiness already in progress before the shutdown flag changes may still complete with 200. There is no drain delay or guarantee a load balancer receives 503 before the listener closes.

## 3. Startup

```text
Load and validate configuration
→ Initialize logger and existing metrics component
→ Create/configure one SQL pool
→ PingContext with database.queryTimeoutSeconds deadline
→ Attach GORM to the same pool (no automatic ping/version query)
→ Register DB stats, lifecycle, checker and operational handler
→ Set up global middleware and routes
→ Bind HTTP listener
→ Log application.server.started and serve HTTP
```

Startup is **fail-fast**: DB failure prevents HTTP listening. The connector closes its pool if initialization fails; registry initialization closes it if DB-stats registration fails. Errors exposed by the operational connector use safe fixed messages without credentials or raw driver details. No migrations run during server startup.

The connector uses pgx over TCP, requests UTF-8 and UTC, and bounds connection establishment and the total startup ping using `database.queryTimeoutSeconds` (default 5 seconds). Arbitrary future queries require caller contexts; no global read/write socket timeout or SQL statement timeout is added. See [configuration](configuration.md) for pool limits and retained inactive settings.

## 4. Shutdown

```text
SIGINT / SIGTERM
→ LifecycleState.BeginShutdown
→ HTTP server.Shutdown with http.shutdownTimeoutSeconds deadline
→ Force HTTP close if graceful shutdown fails
→ Registry.Close
→ Close shared SQL pool exactly once
```

`Registry.Close()` uses `sync.Once`, including repeated calls. Listener bind/serve failures also close registry resources. Signals are registered before binding. The shutdown deadline defaults to 15 seconds. Shutdown errors are returned to the CLI. Redis is not initialized.

## 5. Middleware and metrics

Global order:

```text
Request ID → Completion log → HTTP metrics (if enabled)
→ Panic recovery → Security headers → CORS → Body limit → Handler
```

Probe requests use these global middlewares. CORS may reject requests with disallowed `Origin` headers with HTTP 403; monitoring requests without `Origin` pass through. Panic/transport failures can produce other outcomes than the normal probe statuses. Requests receive `X-Request-ID`.

When enabled, the existing Prometheus component is exposed at `metrics.path` (default `/metrics`) with no auth. Its shared-pool collectors read `sql.DB.Stats()` without DB queries. The metrics endpoint is excluded from HTTP request metrics. No new collectors, dashboard, alerting, tracing, or SLA are provided. Endpoint exposure/network policy belongs to deployment configuration.

### 5.1 Structured logging

The existing Logrus logger writes to stdout. Exact `app.env=development` selects full-timestamp text; other environments use JSON. Production starts at INFO, other environments at DEBUG. In JSON, the human-readable native message is `msg`; no additional `message` field is emitted. Formatter, configuration and dependencies are unchanged.

`event_code` remains the stable event identity, such as `http.request.completed`. Optional `error_code` identifies the actual API error, such as `CORS_ORIGIN_NOT_ALLOWED`; it is a different classification. The response helper supplies only code/status through private request-local Gin metadata. Completion reads this metadata after the handler; it never captures/parses a response body or retains public error messages/details. Only the first helper response with an error status, matching final status and a safe code (uppercase ASCII initial, then uppercase letters/digits/underscores, maximum 64 bytes) supplies telemetry. Missing/unsafe codes and late writes are omitted rather than guessed; API serialization is unchanged. A success has no `error_code`.

| Boundary/context | Level and native message |
|---|---|
| Successful HTTP response | INFO, `HTTP request completed` |
| Sourced 401 `UNAUTHORIZED` | INFO, `HTTP request rejected` |
| Other 4xx, including CORS 403; unclassified 401 | WARNING, `HTTP request rejected` |
| HTTP 5xx, including readiness 503 | ERROR, `HTTP request failed` |
| Active authentication/session denial | INFO, e.g. `Authentication failed` or `Session authentication failed` |
| Active auth operation blocked by internal failure (`operation_failed`) | ERROR, e.g. `Authentication failed` or `Session logout failed` |
| Active auth verification capacity rejection | WARNING, `Authentication failed` |
| Readiness diagnostic | Existing WARNING, `Service readiness check failed` or `Readiness rejected during shutdown` |
| Panic diagnostic | ERROR, `HTTP request panicked`; existing type/stack retained, panic value omitted |

Other unclassified HTTP errors retain conservative status-based fallbacks. A 401 alone does not establish invalid credentials, and this API's generic authentication error is `UNAUTHORIZED`. Existing security decision codes/outcomes/reasons remain intact; unused security-event paths retain their previous fallback. Startup/shutdown retain existing codes, fields and levels with native messages such as `HTTP server started` and `HTTP server shutdown completed`. No new log emissions or logging subsystem are added.

Each request traversing the global middleware has one completion event, including middleware-aborted requests and recovered panics. Separate existing security events describe authentication decisions; readiness events describe dependency/shutdown state; panic events carry diagnostic type/stack. These serve different purposes from HTTP timing/status and share `request_id`; they are not extra completion events. No raw DB error is logged at every layer. The connector's safe fixed diagnostic messages and silent GORM SQL logger remain unchanged.

Selected fields from actual controlled HTTP fixture output (timestamps, timing, IP/agent and size omitted here):

```json
{"event_code":"http.request.completed","http_status":200,"level":"info","msg":"HTTP request completed","request_id":"log-fixture-success"}
{"error_code":"CORS_ORIGIN_NOT_ALLOWED","event_code":"http.request.completed","http_status":403,"level":"warning","msg":"HTTP request rejected","request_id":"log-fixture-cors"}
{"event_code":"auth.login.failed","level":"info","msg":"Authentication failed","request_id":"log-fixture-authentication"}
{"error_code":"UNAUTHORIZED","event_code":"http.request.completed","http_status":401,"level":"info","msg":"HTTP request rejected","request_id":"log-fixture-authentication"}
{"event_code":"auth.login.failed","level":"error","msg":"Authentication failed","request_id":"log-fixture-internal"}
{"error_code":"INTERNAL_SERVER_ERROR","event_code":"http.request.completed","http_status":500,"level":"error","msg":"HTTP request failed","request_id":"log-fixture-internal"}
```

Validated/generated `X-Request-ID` is returned in the response and propagated through Gin and standard request contexts; error metadata belongs only to that request. No request-ID generator or tracing framework is changed. Completion preserves its existing method/path/route/status/latency/size/IP/agent and optional principal fields. Paths are single-line and bounded to 2048 runes; user-agent is sanitized and bounded to 256 runes. Query strings, Origin, body/response payloads, header dumps, passwords/hashes, bearer/session tokens or hashes, Authorization/cookies, credentials and secret configuration must not be logged. These restrictions apply to future logging changes too; sanitized paths/agents remain untrusted metadata, never event codes.

## 6. Manual checks

After configuring and starting a development instance (see [CLI](cli.md)):

```bash
BASE_URL='http://localhost:8080' # match app.port; 8080 is the example value
curl -i "$BASE_URL/health"
curl -i "$BASE_URL/ready"
```

Verify DB outage behavior only against an isolated development/test instance. Do not stop a shared or production DB. Implementation verification evidence and review status belong in `develop/`, separate from this operational reference. These contracts do not imply production deployment verification.

## 7. Scalar API reference (development only)

| GET path | Development response |
|---|---|
| `/docs` | Scalar API reference HTML |
| `/swagger/doc.json` | Existing generated Swagger 2.0 specification |
| `/swagger/index.html` and old Swagger UI assets | Existing `ROUTE_NOT_FOUND` 404 envelope |

Set existing `app.env` / `APP_ENV` to `development` and restart. Open `http://localhost:8099/docs` when `app.port` is 8099; otherwise use the configured port. Scalar reads same-origin `/swagger/doc.json` directly from the registered generated Go package through `swag.ReadDoc()`. No repository filesystem access, second spec, Registry or external spec host is used at runtime. `/docs` is canonical; Gin's existing trailing-slash redirect behavior is not replaced with custom aliases.

The only UI is Scalar. Swagger UI's wildcard registration and dependencies are removed; JSON retains its existing URL for compatibility. Consumers found in this repository were the route tests and current documentation, which now target Scalar. Historical M3A inputs/reviews remain unchanged. No additional UI aliases or redirects to old Swagger pages were introduced.

Outside development both docs and JSON return the existing 404 envelope. Global request IDs, logs, recovery, optional metrics, security headers and CORS remain in effect; disallowed Origin may receive403. `GIN_MODE` does not control exposure. Metrics cannot register under `/docs` or `/swagger`; validation catches conflicts before HTTP startup.

Scalar v1.69.2 uses the official HTML/JavaScript integration via pinned jsDelivr ESM modules. JavaScript and internet/CDN access are required for rendering, including lazily loaded modules/styles; HTTP200 HTML alone does not prove rendering. System fonts avoid a separate font request. No frontend framework/build pipeline, Scalar account/key, Registry or runtime CLI is required. AI Agent, telemetry, credential persistence and external request proxy are disabled. API test requests go directly to the backend origin, without broadening CORS or weakening security middleware. Current security middleware sets no CSP; none was removed or relaxed.

The reference renders GET health/ready, POST /v1/auth/login, protected GET /v1/auth/me and protected POST /v1/auth/logout plus /v1/auth/logout-all. Shared envelopes remain, with concrete authentication DTOs and an opaque BearerAuth header scheme; readiness 503 SERVICE_NOT_READY is retained. Optional Prometheus/docs asset routes are outside this specification. Probe implementations, readiness deadline, shared DB pool, fail-fast startup and shutdown remain unchanged. DB connectivity is still required to start the backend, but a later DB outage does not prevent docs or health from responding while readiness returns503.

For unchanged generation see [CLI](cli.md#5-api-specification-generation). Verify the actual UI loads endpoints/schemas and can issue requests; check browser console and the development/production gate, rather than relying only on HTML status.

## 8. PostgreSQL development lifecycle (M4P)

Root `compose.yaml` runs only PostgreSQL 17; Go remains local and uses PostgreSQL after its backend connection settings are aligned. Container health checks readiness inside Docker; backend `/ready` checks its actual shared pool from the host. The existing probe responses and fail-fast startup contract are unchanged.

Use `up -d` to create/apply configuration, wait for healthy, and use `stop` / `start` for routine lifecycle. `restart: unless-stopped` permits restart after daemon/machine restart unless explicitly stopped. Named volume `postgres_data` persists across stops and container recreation under the same project. `down` removes containers/network while retaining the volume; `down -v` destroys its data and is not a routine operation. Docker Desktop groups the service as a Compose project and supports stop/start after initial creation.

Do not stop unrelated projects to free a port or reset volumes to change initialization credentials. See [configuration](configuration.md#7-postgresql-docker-development-m4p) and [Docker commands](cli.md#8-docker--postgresql-development-commands) for setup, precedence, credential behavior and SQL verification. No migration or business schema is created by this setup.

## 9. PostgreSQL startup troubleshooting (M4A)

Backend configuration uses `DB_*` / JSON, not Compose `POSTGRES_*`. Check the published host port (existing local development 5433), database/user, initialized password and explicit SSL mode without displaying secrets. The previous MySQL connector could not use the M4P PostgreSQL service. Startup diagnostics now distinguish `connection refused`, `authentication failed`, `database not found`, `timeout`, `canceled`, host-resolution failure and generic server/connection failure using fixed messages; raw driver details/DSNs are withheld. Invalid configuration fails validation before HTTP startup.

An unavailable database before startup means no HTTP listener. A database transport failure after startup leaves `/health` at 200 and makes `/ready` return the existing 503 `SERVICE_NOT_READY` within its request deadline. Test this only through an isolated application/proxy or test instance; do not stop a developer/shared PostgreSQL instance. Registry still closes the same pool once on shutdown. Migration CLI is separate from HTTP; no migration schema is initialized during server startup.

## 10. PostgreSQL migration operations (M4B)

Migration commands are now active through Cobra and use the same effective backend database settings. Check the target and reviewed SQL before running them. M5A supplies reviewed migration pairs for global `users` and `auth_sessions`; applying `up` changes the verified target schema. See the [identity/session schema](database/identity-session-schema.md). With M5B version3, one `down` drops only the token-hash index, retaining both tables/data at2. From version2, the next down drops sessions; another drops users. Rollback destroys those table records and must not be demonstrated on existing development data. Schema verification uses only an owned disposable database. `version` may initialize `schema_migrations` and acquire an advisory lock, so it is not guaranteed read-only. HTTP startup remains migration-free.

`up` applies pending versions; `down` rolls back one version only. Migration sessions use configured connection/statement/lock bounds and close their own source/connection/pool. A failed SQL operation marks dirty and blocks further up/down. Inspect migration SQL and real schema, plan reconciliation, then use explicit force only to match inspected state; never reset a database or automatically force past errors. Do not use down/force as routine setup or probe commands. See [migration CLI reference](cli.md#3-postgresql-migration-commands) for exact empty-source, version-zero/dirty-nil and transaction behavior. Destructive lifecycle verification uses only newly created disposable databases owned by the test.

## 11. Authentication operations (M5B)

Login dan bearer /me memakai shared PostgreSQL pool yang sama. Startup tetap hanya memvalidasi config/connectivity; tidak ada schema check/migration/seeder. Apply migration yang direview ke target terverifikasi sebelum memakai auth. Readiness200 tidak membuktikan schema auth tersedia. Session validity diperiksa pada tiap protected request; hanya metadata IP dari trusted-proxy resolution dan user-agent tersanitasi disimpan.

Wrong/unknown/inactive/no-password credentials menghasilkan generic401; invalid/expired/revoked/inactive bearer juga401. Operational DB/transaction failure menghasilkan500 aman; local Argon2 capacity exhaustion menghasilkan503, bukan invalid credentials. Query/transaction memakai DB timeout dan request cancellation. Tidak ada raw error, SQL, password atau token di response/log auth. Audit last_login_at/session insert commit atomik. `/health`, `/ready`, graceful shutdown dan middleware global tetap mempertahankan kontrak existing.

Login memakai batas fixed-window IP dan identifier per proses; lihat [konfigurasi](configuration.md#login-abuse-policy). Batas dua verifikasi Argon2 bersamaan tetap hanya resource bound. Deployment publik dengan beberapa instance memerlukan kebijakan abuse pada boundary bersama yang telah direview. Gunakan HTTPS dan redaksi credential pada proxy/client logs. Login tidak memberikan tenant/RBAC grant dan logout sesi tersedia melalui M5C. Provisioning lokal menggunakan [CLI terpisah](#local-seeding-and-bootstrap). Detail konfigurasi/token/failure dan verifikasi disposable ada di [authentication API](api/authentication.md).

## 12. Session revocation operations (M5C)

`POST /v1/auth/logout` mencabut current session; `POST /v1/auth/logout-all` mencabut session aktif milik authenticated user, termasuk caller. Keduanya mempertahankan row dan memakai envelope200/data:null; token yang dicabut ditolak pada request berikutnya. Repeat/stale caller dapat401. DB/transaction/deadline failures memberi500 aman tanpa partial revocation. Tidak ada scope user/tenant dari body.

Store memakai transaction READ COMMITTED dan lock user→caller-session, mengikuti urutan lock login. Timestamp server dibaca setelah lock waits untuk recheck expiry dan satu nilai revoked_at/updated_at; updated_by adalah user terautentikasi. Creations terurut sebelum logout-all ikut dicabut; password login terurut setelahnya adalah login baru yang dapat valid. Request biasa yang sudah terautentikasi sebelum commit dapat selesai; logout-all tidak menjamin account lockout/zero sessions selamanya. Semua writer aplikasi perlu mempertahankan urutan lock tersebut.

M5C tidak menambah migration atau config. Terapkan pending M5A/M5B SQL hanya ke target yang telah diverifikasi; startup tetap migration-free. Lifetime15 menit default tetap, tanpa idle/sliding renewal atau writes last_used_at per request. Renewal/rotation, session-management API dan cleanup/retention DEFER; invalid rows tetap ditolak walaupun masih tersimpan. Kontrol login-abuse, HTTPS/proxy redaction dan batas production readiness M5B tetap berlaku. Lihat [authentication API](api/authentication.md).

## Tenancy & organization migrations

Module A SQL000004–000009 adds six tables without changing server/authentication/configuration contracts. Use the [schema reference](database/tenancy-organization-schema.md) and [migration CLI](cli.md#3-postgresql-migration-commands). Verify the effective DB_* target before up/version; version can initialize metadata. Server startup never applies migrations or seeds records. Development deployment remains an explicit operator action after review.

Test up/down/reapply only on a newly created, owned disposable PostgreSQL17 database. Each down drops one table; rollback9→3 removes Module A and preserves users/auth_sessions/token index. It deletes organizational data on its target, so never demonstrate it on mapatu_dev. No CASCADE or Docker-volume removal is needed. Audit references may prevent user deletion. Multiple head offices and nullable branch locations are intentional; longer hierarchy cycles still require future write validation. Location fields do not implement attendance/geofence evaluation, and global authentication does not grant tenant access.

## Membership & RBAC migration boundary

Versions000010–000014 add five Module B tables and six fixed system roles. Version 000015 installs the v1.0 permission catalog and grants. An account gains access only after separate membership and assignment provisioning where applicable. Scope constraints prevent inconsistent references, while active-state authorization remains a request-time concern. See the [canonical contract](database/tenant-membership-rbac-schema.md). Down 15→14 removes v1.0 reference data only. A later mapping to a seeded permission blocks rollback and the migration runner can report a dirty version. Down 14→9 removes the M7 tables. Verify rollback in a disposable database. Startup remains migration-free and readiness is a connectivity check, not proof that authorization schema exists.

Authorization foundation uses the existing SQL pool with query timeout/cancellation and no cache. Database errors fail closed with safe500; the next query after committed deactivation/revocation observes the new state. In-flight work authorized before a change may finish. RequirePermission relies on the existing request completion event for status/error-code correlation; it introduces no credential/permission dump or duplicate security log. Assignment management remains deferred and denied.


## Local seeding and bootstrap

Verify the configured endpoint is a local development cluster, then apply migrations through at least 000015. Use `seed list` to see the registered datasets and run each required dataset by name. Server startup and migrations never run the seeders. Keep production data out of seeded databases. Local port forwarding to a remote cluster falls outside this workflow.

The demo seeders reserve IDs alongside the schema's business keys. Reruns preserve non-key edits, password state and activation state. If a reserved ID or business key conflicts, inspect the conflicting record before making changes. Missing dependencies require an explicit named run. Each run commits independently. After an ambiguous commit result, inspect the target and rerun the same named dataset only after confirming the target is correct.

Run initial administrator bootstrap separately from demo seeding. Use a trusted local operator terminal, a new email and a unique password. The transaction locks assignment writers before checking for prior bootstrap. An existing platform SUPER_ADMIN assignment blocks bootstrap even when inactive. The reserved bootstrap user also blocks reruns if the original assignment alone was deleted. Preserve bootstrap records. Direct removal of both records by a database administrator lies outside the CLI's one-time guarantee and is not an approved recovery procedure. Use a separately approved recovery process for lost access. Production bootstrap and future assignment management remain deferred. See the [CLI and command order](cli.md#4-development-seeding-and-administrator-bootstrap).

## Authentication abuse operations

A 429 `RATE_LIMIT_EXCEEDED` means this process rejected admission, not that an account exists or credentials were checked. Wait at least `Retry-After` seconds, add client backoff, and avoid automatic tight retries. One `auth.login.rate_limited` warning and one correlated `http.request.completed` event describe each throttled handler request. Internal reasons distinguish the IP/identifier admission stage; neither counters nor keys nor raw identifiers are returned or logged. Capacity rejection uses the same safe response. Inspect distinct-client load, trusted proxies, configured windows and record capacity before changing budgets. Successful logins also consume budgets.

Active records are never evicted to accommodate attacker churn. At capacity, new keys wait for the earliest expiry while known keys may use remaining budgets; sufficiently large attacks can temporarily reduce login availability. Rejections never extend deadlines. Counters expire without background goroutines and survive only within the process; restarting clears them. Do not repeatedly restart or enlarge limits as an automatic response to attacks. Health/readiness, bearer lookups and logout are outside login limits. Multi-instance production deployment needs a shared ingress/distributed abuse policy before relying on fleet-wide limits. No Redis/WAF setup is supplied here.

Forwarding trust must match actual proxy hops. Direct development uses `TRUSTED_PROXIES` empty. When behind a proxy, list only its specific IP/CIDR and ensure it overwrites/sanitizes forwarded headers and prevents direct access to the upstream. Never use blanket `0.0.0.0/0` or `::/0` trust for internet clients. The application uses Gin validated resolution for both login limits and stored/logged client IPs. See [Gin trusted proxies](https://gin-gonic.com/en/docs/server-config/trusted-proxies/).

## Bootstrap secret handling

The bootstrap CLI reads a confirmed password only from hidden interactive terminal input; it has no password-file/stdin/environment/HTTP provisioning interface. New bootstrap input requires at least **15 Unicode characters**, valid UTF-8 and at most **1024 bytes**. This tightens the historical five-character operator input rule; existing login credentials remain compatible. Argon2id, development-only loopback/live-target checks, new-account-only behavior, transactional first-admin guard and PLATFORM-only assignment remain unchanged. There is no default password or first-login elevation. Production bootstrap stays refused.

A locally retained `.bootstrap-admin-password` is not consumed by this CLI. Both that actual spelling and the reported `.bootstra-admin-password` typo are excluded from Git and Docker context; ignore rules alone do not secure a secret. Keep any necessary local secret regular, owned by the operator and mode 0600, outside shared folders/backups/build context. After provisioning, prefer a trusted password manager and explicitly remove an unnecessary local copy through an operator action. Do not delete it automatically or display/copy it into logs/reviews. If any credential has ever entered Git or other unauthorized storage, rotate it; do not rewrite history as an automatic remedy. A local history check cannot prove absence from remote/deleted history or other copies.

For browser failures, distinguish 403 `CORS_ORIGIN_NOT_ALLOWED` from 403 `CORS_PREFLIGHT_NOT_ALLOWED` and inspect the configured frontend origin/method/header names without logging headers or secret bodies. Preflights carry no bearer credential; successful CORS grants no authentication. Keep HTTPS enforcement, forwarded-header sanitation and HSTS policy at the trusted TLS ingress.

## Foundation development checklist

1. Preserve existing `.env`/`config.json`; align Compose `POSTGRES_*` with the separate effective backend `DB_*`/JSON settings. Validate with `docker compose config -q`, start with `docker compose up -d`, and wait for healthy using `docker compose ps`.
2. Verify the intended endpoint/database before `go run main.go migrate version` and `go run main.go migrate up`. Even `version` can initialize metadata. The current source ends at `000015_seed_rbac_policy_v1`; startup/readiness do not check this version. Never demonstrate down/force on existing development data.
3. Start `go run main.go server`; check `/health`, `/ready`, and development `/docs` plus `/swagger/doc.json` on the configured JSON port. Scalar rendering additionally needs its pinned CDN.
4. Use `go run main.go seed list` without a database. Execute only a deliberate named dataset with the effective `--target`, following the [dependency order](cli.md#4-development-seeding-and-administrator-bootstrap). Seeders grant no roles. Bootstrap is a separate development-only, hidden-terminal operation for a new first administrator; do not rerun it on an existing administrator.
5. Run build/vet/default tests and the relevant existing disposable PostgreSQL suites in [CLI](cli.md). Opt-in DB suites are skipped by ordinary `go test`; inspect their actual output and verified cleanup. The [canonical M9 review](../develop/reviews/foundation-readiness/M9-FINAL-review.md) records current evidence and readiness, rather than this checklist certifying a deployment.

Future tenant business APIs must resolve stored resource ownership, scope every query/write to the verified tenant, require explicit approved permissions after bearer authentication, and test cross-tenant/active-state denial. Current work-location authorization fixtures verify the foundation; Employee/Employment endpoints do not yet exist. Define their permission/lifecycle contracts before exposure. A hierarchy writer must reject longer organization/department cycles transactionally under concurrency; no current public writer can reparent these records. Session rows remain retained with fixed expiry/revocation; no cleanup/renewal worker or production account-recovery mechanism is supplied. Production deployment still requires the transport/proxy/shared-abuse/provisioning controls described above.
