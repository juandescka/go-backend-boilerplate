# Configuration Guide

## 1. Sources and precedence

`config.LoadConfig()` loads `.env` without replacing existing process variables, applies defaults, decodes required `config.json`, applies supported environment overrides, then validates:

```text
Process environment > .env > config.json > defaults
```

This precedence applies only to fields with explicit overrides. `config.json` remains required; application HTTP port is JSON-only. JSON `${VARIABLE}` placeholders are not expanded. The loader uses existing godotenv, with no additional loader or PostgreSQL-specific fallback. Restart the process after changing settings; repeated loads in one process retain variables already loaded into its environment.

## 2. Local setup

Use the examples only for new files; preserve existing local files. `.env` and `config.json` are gitignored. The JSON example is development-only: HTTP 8080, PostgreSQL `127.0.0.1:5432`, database `mapatu_dev`, user `mapatu`, explicit `sslMode=disable`, and no password/JWT secret.

1. Start the existing Compose database using [Docker commands](cli.md#8-docker--postgresql-development-commands).
2. Configure backend JSON or supported `DB_*` overrides. The existing M4P local database is `127.0.0.1:5433`, database `mapatu_dev`, user `mapatu`; its initialized password is stored only in local `.env`. Set `DB_PASSWORD` to that password and match the backend port to `POSTGRES_PORT`. There is **no automatic mapping** from `POSTGRES_*` to `DB_*`, nor fallback to MySQL. Changing Docker initialization variables does not update credentials in an existing volume.
3. Set `JWT_AUTH_SECRET_KEY` or the existing JSON secret. Generate a local value for a new setup with `openssl rand -hex 32`; never commit it.
4. Run `go run main.go server` from the repository root. Check `/health` and `/ready` at the configured HTTP port (existing local setup 8099).

M5B opaque bearer authentication is implemented; existing JWT lifetime and cookie validation remains in force for compatibility, without activating JWT/cookie authentication. Login rate limiting is active; see the policy below. Production still requires secure refresh cookies. Only database settings/validation change in M4A.

## 3. PostgreSQL and pool

| JSON key under `database` | Default | Behavior |
|---|---|---|
| `driver` | None | Required; exactly `postgres` |
| `host`, `port` | None | Required host; port 1–65535; no hardcoded local port |
| `name`, `username` | None | Required, nonblank |
| `password` | None | Required, nonempty; supply actual local secret |
| `sslMode` | None | Required: `disable`, `allow`, `prefer`, `require`, `verify-ca`, `verify-full` |
| `maxOpenConnections` | 25 | Shared pool; range 1–1000 |
| `maxIdleConnections` | 10 | Shared pool; range 0–max open |
| `maxLifetimeSeconds` | 1800 | Applied as seconds; range 1–86400 |
| `maxIdleTime` | 300 | Applied as seconds; range 1–max lifetime |
| `queryTimeoutSeconds` | 5 | Startup ping deadline and pgx connection establishment timeout; range 1–60 |
| `transactionTimeoutSeconds` | 10 | Bounds seeder and bootstrap write transactions |
| `slowQueryThresholdMilliseconds` | 500 | Validated/retained; no slow-query instrumentation wired |
| `dbDebug` | false | Retained; GORM uses silent SQL logging |

One pgx `database/sql` pool is attached to the official GORM PostgreSQL dialector and shared with readiness, metrics and shutdown. URL construction escapes credentials/database names and handles IPv6; raw parse/driver errors and DSNs are not logged. Startup uses explicit bounded `PingContext`; GORM automatic ping is disabled. Sessions request UTF-8 and UTC. No version/schema/business query or migration is added to application connectivity checks.

Readiness retains its own 2-second request deadline. Unlike the old MySQL socket settings, pgx does not impose a global read/write deadline on arbitrary future GORM queries; callers must supply query contexts. No SQL `statement_timeout` or transaction wrapper is introduced into backend sessions. Migration CLI sessions have their separate bounded policy below.

Supported database overrides:

```text
DB_DRIVER, DB_HOST, DB_PORT, DB_NAME, DB_USERNAME, DB_PASSWORD, DB_SSL_MODE
DB_MAX_OPEN_CONNECTIONS, DB_MAX_IDLE_CONNECTIONS
DB_CONNECTION_MAX_LIFETIME_MINUTES, DB_CONNECTION_MAX_IDLE_TIME_MINUTES
DB_QUERY_TIMEOUT_SECONDS, DB_TRANSACTION_TIMEOUT_SECONDS
DB_SLOW_QUERY_THRESHOLD_MILLISECONDS, DB_DEBUG
```

The two `*_MINUTES` overrides convert to seconds; JSON counterparts use seconds. Explicit empty required environment values fail rather than falling back. `APP_PORT` is unsupported. Explicit backend identity/SSL/timeout values override pgx's corresponding `PG*` environment values. `PGSERVICE` is rejected to prevent a service file introducing a second database configuration. pgx's optional TLS certificate variables (`PGSSLROOTCERT`, `PGSSLCERT`, `PGSSLKEY`, etc.) and standard certificate files still follow pgx behavior; avoid unintended inherited TLS settings. Do not use production credentials for development.

`sslMode=disable` is an explicit local Docker choice, not a hardcoded runtime policy. Production deployment is outside M4A: choose and verify TLS for that deployment, normally `verify-full` with a trusted CA and matching hostname. `require` encrypts without the same hostname verification guarantees; `allow`/`prefer` permit plaintext fallback. No production certificate infrastructure is provisioned here.

Migration CLI now uses the same effective PostgreSQL connection settings, required JSON and validation as the backend. No separate migration credentials, host/port source or POSTGRES_* fallback is added. Its structured URL preserves special characters and IPv6 and rejects PGSERVICE consistently with M4A. The official golang-migrate PostgreSQL driver uses a CLI-owned pgx SQL pool and reserved connection, independent of the HTTP pool.

For migration sessions only, `queryTimeoutSeconds` bounds connection establishment, each complete migration-file execution through the driver context, and server statement/lock waits via `statement_timeout`/`lock_timeout`. Default is 5s; this is not a whole-run deadline. Server timeout parameters are set before driver metadata initialization. Migration source is `migrations/postgres`; the inspected empty MySQL directory is legacy only. Commands, metadata side effects, empty-source limits and recovery are in [CLI](cli.md#3-postgresql-migration-commands). Production TLS/certificate deployment remains outside this development milestone.

## 4. HTTP, CORS and metrics

HTTP defaults: read-header 5s, read 15s, write 30s, idle 60s, shutdown 15s; header limit 65536 bytes, request-body limit 1048576 bytes. Timeout values must be 1–300s and write timeout must exceed the 2-second readiness deadline. Header/body ranges and exact validation are in `config/config.go`.

| Area | Supported overrides |
|---|---|
| Environment | `APP_ENV` |
| HTTP | `HTTP_READ_HEADER_TIMEOUT_SECONDS`, `HTTP_READ_TIMEOUT_SECONDS`, `HTTP_WRITE_TIMEOUT_SECONDS`, `HTTP_IDLE_TIMEOUT_SECONDS`, `HTTP_SHUTDOWN_TIMEOUT_SECONDS`, `HTTP_MAX_HEADER_BYTES`, `HTTP_MAX_REQUEST_BODY_BYTES` |
| CORS | `CORS_ALLOWED_ORIGINS` (comma-separated), `CORS_ALLOW_CREDENTIALS` |
| Proxy trust | `TRUSTED_PROXIES` (comma-separated IP/CIDR) |
| Metrics | `METRICS_ENABLED`, `METRICS_PATH`, `METRICS_NAMESPACE` |

CORS defaults to no allowed browser origins, credentials false; requests without `Origin` are allowed. Trusted proxies default to none. Metrics default to disabled, `/metrics`, namespace `hris_core_service`. Enabled metrics must use a static root path that does not conflict with operational/reserved routes and a valid Prometheus namespace.

## 5. Retained inactive configuration

Redis is not initialized and is not a readiness dependency even if configured. Keep `redis.enabled=false` for this operational setup. Legacy JWT/cookie, refresh rate-limit and cache fields remain available to the loader without activating those features. Login rate-limit fields are active. Opaque authentication uses its separate lifetime below.

Other supported overrides (login controls are active; JWT, refresh and cache fields remain inactive):

```text
JWT_AUTH_SECRET_KEY, JWT_AUTH_EXPIRATION_TIME_IN_MINUTES
AUTH_HRIS_ADMIN_REFRESH_COOKIE_NAME, AUTH_HRIS_STUDENT_REFRESH_COOKIE_NAME
AUTH_REFRESH_COOKIE_SECURE, AUTH_REFRESH_COOKIE_SAMESITE, AUTH_REFRESH_COOKIE_PATH
AUTH_LOGIN_IP_RATE_LIMIT, AUTH_LOGIN_IP_RATE_WINDOW_SECONDS
AUTH_LOGIN_IDENTIFIER_FAILURE_LIMIT, AUTH_LOGIN_IDENTIFIER_FAILURE_WINDOW_SECONDS
AUTH_REFRESH_IP_RATE_LIMIT, AUTH_REFRESH_IP_RATE_WINDOW_SECONDS
AUTH_RATE_LIMIT_MAX_ENTRIES
REDIS_ENABLED, REDIS_ADDRESS, REDIS_USERNAME, REDIS_PASSWORD, REDIS_DATABASE
REDIS_KEY_PREFIX, REDIS_DIAL_TIMEOUT_MILLISECONDS
REDIS_READ_TIMEOUT_MILLISECONDS, REDIS_WRITE_TIMEOUT_MILLISECONDS
REDIS_USER_CACHE_TTL_SECONDS, REDIS_EXAM_SNAPSHOT_CACHE_TTL_SECONDS
REDIS_EXAM_EXECUTION_QUESTIONS_CACHE_TTL_SECONDS
```

There is no refresh-token lifetime environment override. Broad cleanup of these inactive settings requires a separate decision. Never commit secrets, log complete connection strings, or use production credentials for verification.

See [operations](operations.md), [CLI](cli.md), and [architecture](architecture.md).

## 6. API reference development environment

Scalar `/docs` and the existing `/swagger/doc.json` use `app.env` / `APP_ENV` and its existing precedence. No new documentation flag, JSON key or environment variable is added. Only `development` (case-insensitive, surrounding whitespace ignored) registers these routes; production, staging, unknown and empty values register neither. `GIN_MODE` does not control exposure. Do not mark a production deployment as development to expose docs.

Set `APP_ENV=development` for local use and restart the server. Docs are unauthenticated inside this gate and inherit global middleware. The server still validates configuration and connects to PostgreSQL before serving any route. Production secure-cookie validation is unchanged. Metrics paths `/docs` and `/docs/...` are now reserved alongside the existing `/swagger` namespace to reject route conflicts during validation, rather than panicking during registration.

The spec does not pin a host/port and retains root base path `/`; use the running backend origin. For port 8099 open `http://localhost:8099/docs`. The example configuration remains on8080. API generation does not modify local configuration; M4A updates only local database settings. Production files remain unchanged.

Scalar's CDN version is fixed in the small HTML template at `@scalar/api-reference@1.69.2/esm.js`; changing it is a source change requiring UI/security verification, not a runtime environment override. JavaScript and internet access to jsDelivr (including its versioned module chunks/styles) are needed to render the UI. JSON and probe endpoints work independently of CDN availability. Fonts use local system defaults. No Scalar Registry, account/API key, CLI, external request proxy or AI Agent is configured; telemetry and credential persistence are disabled. No analytics plugin is loaded.

See [operations](operations.md#7-scalar-api-reference-development-only) and [CLI](cli.md#5-api-specification-generation).

## 7. PostgreSQL Docker development (M4P)

Root `compose.yaml` provides one `postgres` service using the [official `postgres:17` image](https://hub.docker.com/_/postgres). Go runs locally and uses PostgreSQL through the backend configuration above. `POSTGRES_*` configures Docker only and is not consumed by the Go database loader; align the separate backend `DB_*` / JSON settings with this initialized database.

| Variable | Local example/default | Purpose |
|---|---|---|
| `POSTGRES_USER` | `mapatu` (example; required) | Initial PostgreSQL user |
| `POSTGRES_PASSWORD` | No default; required | Strong local password; generate with `openssl rand -hex 32` |
| `POSTGRES_DB` | `mapatu_dev` (example; required) | Initial database |
| `POSTGRES_PORT` | `5432` (default) | Host port bound only to `127.0.0.1`; container port stays 5432 |

Append only missing variables to root `.env`; preserve unrelated local settings and existing PostgreSQL initialization variables. If `.env` does not exist, create it locally using the example keys and a generated password. `.env` is gitignored; never commit it or copy real passwords into `.env.example`, logs or review records. The example password is a placeholder that must be replaced.

Compose interpolates `${...}` from the shell and root `.env` when run from this repository. **Shell environment values override `.env`** ([Docker interpolation reference](https://docs.docker.com/compose/how-tos/environment-variables/variable-interpolation/)). Check for exported `POSTGRES_*` overrides without printing passwords; unset unintended overrides before starting. `COMPOSE_FILE`, `COMPOSE_PROJECT_NAME`, `COMPOSE_ENV_FILES` and `--env-file`/`-p`/`-f` can also select different inputs or resources. Keep the same project identity across lifecycle commands. Required user/password/database values fail validation when empty or missing. Use `docker compose config -q`; plain `config` and `config --environment` can expose secrets.

The connection endpoint from the host is `127.0.0.1:${POSTGRES_PORT:-5432}`. If the host port is occupied, choose another free `POSTGRES_PORT` in `.env`; do not stop another project's database. A stopped container configured for the same port will conflict if started later.

Compose volume key `postgres_data` normally becomes `hris-core-service_postgres_data` under this repository's default project name. It mounts at `/var/lib/postgresql/data`, the PostgreSQL 17 image data location. Keep the same project name and volume to retain data across container recreation. Container stop, Docker Desktop closure and machine restart do not delete named data.

`POSTGRES_USER`, `POSTGRES_PASSWORD` and `POSTGRES_DB` initialize an **empty** data directory only. Changing `.env` after initialization does not update the existing database/user/password. Do not reset/delete the volume to apply credential changes; use a separately planned PostgreSQL administration procedure for an existing database. The container healthcheck checks readiness with `pg_isready`; it does not prove password authentication. Use the SQL and TCP authentication checks in [CLI](cli.md#8-docker--postgresql-development-commands).

## 8. Opaque authentication (M5B)

| JSON key | Default | Environment override | Bounds |
|---|---|---|---|
| `authentication.sessionLifetimeMinutes` | 15 | `AUTH_SESSION_LIFETIME_MINUTES` | Integer1–1440 minutes |

The existing source precedence and required JSON are unchanged. This setting controls the fixed expiry of newly issued opaque sessions; changing it does not rewrite existing expiry timestamps. Retained JWT durations/secret and refresh-cookie validation do not control or enable opaque tokens, JWT or cookie authentication. Password policy is reviewed code: Argon2id v19,64 MiB/3 passes/4 lanes,16-byte salt/32-byte output, max1024 password bytes, at most two concurrent verifications per service. These parameters are not configurable environment keys. Each auth lookup/transaction uses `database.queryTimeoutSeconds` plus caller cancellation; Argon2 work is bounded but not interruptible.

Local seeding and first-admin bootstrap are available through separate CLI commands. Login has process-local IP and identifier attempt limits; multi-instance and ingress protection require deployment configuration. See [authentication API](api/authentication.md).

## 9. Session lifecycle compatibility (M5C)

No new key, default or override is introduced. Logout/revocation does not extend expires_at; the fixed authentication.sessionLifetimeMinutes policy remains unchanged. Revocation transactions use database.queryTimeoutSeconds and caller cancellation, plus a fresh database timestamp after lock acquisition. Idle timeout, renewal/rotation and cleanup/retention are deferred; last_used_at remains optional and is not updated per request. See [authentication API](api/authentication.md).


## Local provisioning

Seeder runs and bootstrap use the same loader and PostgreSQL connector as the application. Explicit process environment wins over `.env`, which fills unset environment variables. Those environment values override `config.json`; optional keys may inherit defaults. Check the effective `APP_ENV`, `DB_HOST`, `DB_PORT`, `DB_NAME` and `DB_USERNAME` values before running a write command. `POSTGRES_*` selects Compose initialization settings.

Use exactly `development` in effective `APP_ENV` or `app.env`. The database host must be a numeric loopback address, such as `127.0.0.1` or `::1`. Set `--target` to the effective `host:port/database`, using brackets around an IPv6 address. The command also checks the actual peer endpoint, PostgreSQL database and connected role before writing. Confirm independently that the endpoint belongs to your local development cluster. A loopback connection cannot establish the provenance of a tunnel or proxy.

Keep database credentials in the existing protected configuration and enter administrator credentials at the interactive terminal. Disable PostgreSQL bind-parameter logging on the provisioning target to keep password hashes out of its logs. Bootstrap accepts no password flag, environment override, fallback password or redirected input. Both commands use existing query and transaction timeouts and require clean migration version 15 or later. See the [CLI](cli.md#4-development-seeding-and-administrator-bootstrap).

## Login abuse policy

These existing settings are now active for `POST /v1/auth/login`. Defaults are technical starting points, not approved business throughput targets. Restart after changes; integer limits must be 1–1000000 and windows 1–86400 seconds.

| JSON key under `rateLimiter` | Default | Environment override | Meaning |
|---|---|---|---|
| `authLoginIPLimit` | 10 | `AUTH_LOGIN_IP_RATE_LIMIT` | Attempts per effective client IP, including malformed requests reaching the handler |
| `authLoginIPWindowSeconds` | 60 | `AUTH_LOGIN_IP_RATE_WINDOW_SECONDS` | Fixed IP window starting at first admission |
| `authLoginIdentifierFailureLimit` | 5 | `AUTH_LOGIN_IDENTIFIER_FAILURE_LIMIT` | Legacy name retained; bounds **all admitted valid identifier attempts**, including success and operational/KDF-capacity failures |
| `authLoginIdentifierFailureWindowSeconds` | 300 | `AUTH_LOGIN_IDENTIFIER_FAILURE_WINDOW_SECONDS` | Fixed identifier window; rejection/success never resets or extends it |
| `maxEntries` | 100000 | `AUTH_RATE_LIMIT_MAX_ENTRIES` | Shared IP plus identifier record cap, range 2–1000000; small values impair availability |
| `trustedProxies` | `[]` | `TRUSTED_PROXIES` | Comma-separated IPs/CIDRs; empty trusts no forwarding source |

After IP admission, one bounded PostgreSQL `lower(btrim(...))` query uses the exact identity normalization and database locale. The limiter retains namespace-separated SHA-256 digests, never raw identifiers, passwords or tokens. The existing two-slot Argon2 bound remains separate (503 on saturation). Invalid/canceled controls and normalization failures fail closed with safe 500; missing limiter cannot disable protection. Policies count admissions atomically before password work, avoiding concurrent failure-registration overshoot.

At record capacity, live entries are never evicted. New keys get generic 429 until the earliest expiration; existing keys can use remaining budgets. Expired records are reclaimed on the next attempt using an expiry heap, without workers. Memory remains capped even when idle. Choose enough capacity for distinct IPs plus identifiers. `Retry-After` rounds the blocking window/earliest-expiration remainder up to whole seconds; it is guidance, not a reservation. IP and identifier budgets both apply and a retry can encounter the other budget.

All limits are **per process**; restart clears counters and multiple replicas each have independent budgets. This does not guarantee distributed brute-force/credential-stuffing protection. Public multi-instance deployment needs a reviewed shared/ingress abuse policy; Redis has not been introduced. Identifier abuse can temporarily deny a victim login within a fixed window; there is no persisted account lockout.

## HTTP security settings and deployment

| JSON key | Default | Allowed values | Environment override | Production impact |
|---|---|---|---|---|
| `cors.allowedOrigins` | `[]` | Exact HTTP(S) origins without path/query; optional sole `*` only without credentials | `CORS_ALLOWED_ORIGINS` | Configure actual frontend origins, e.g. `http://localhost:3000` only for that development frontend. Do not deploy a development origin by assumption. |
| `cors.allowCredentials` | false | Boolean; true requires explicit non-wildcard origins | `CORS_ALLOW_CREDENTIALS` | Opaque bearer API does not require browser cookies; retain false unless a real client need is reviewed. |
| `http.readHeaderTimeoutSeconds` | 5 | 1–300 | `HTTP_READ_HEADER_TIMEOUT_SECONDS` | Bounds slow header reads. |
| `http.readTimeoutSeconds` | 15 | 1–300 | `HTTP_READ_TIMEOUT_SECONDS` | Bounds complete request read, including slow JSON bodies. |
| `http.writeTimeoutSeconds` | 30 | 3–300 | `HTTP_WRITE_TIMEOUT_SECONDS` | Bounds response writes; no streaming/long-running API is currently registered. |
| `http.idleTimeoutSeconds` | 60 | 1–300 | `HTTP_IDLE_TIMEOUT_SECONDS` | Bounds idle keep-alive connections. |
| `http.shutdownTimeoutSeconds` | 15 | 1–300 | `HTTP_SHUTDOWN_TIMEOUT_SECONDS` | Drain deadline followed by forced close; readiness rejects while draining. |
| `http.maxHeaderBytes` | 65536 | 8192–1048576 | `HTTP_MAX_HEADER_BYTES` | Standard net/http header bound, with its documented parser read allowance. |
| `http.maxRequestBodyBytes` | 1048576 | 1024–52428800 | `HTTP_MAX_REQUEST_BODY_BYTES` | JSON boundary for body-bearing methods; login also has a fixed 16384-byte decoder bound. |

Login's effective body bound is the lower of the global and fixed login limits; no environment key changes the endpoint's 16 KiB bound. MaxBytesReader also bounds unknown-length bodies when consumed. Legacy multipart allowlist paths are not registered production endpoints; future upload handlers must define file limits before using that bypass.

Preflight requests with Origin require an allowed exact origin, an allowed requested method (`GET, POST, PUT, PATCH, DELETE, OPTIONS`) and only supported headers (`Authorization, Content-Type, X-LMS-Client, X-Request-ID, Accept`, case-insensitive). Success is 204. Invalid preflight method/headers return generic 403 `CORS_PREFLIGHT_NOT_ALLOWED`; absent Origin follows normal routing. Vary preserves other values and includes Origin for non-wildcard policies, plus preflight method/header selectors for OPTIONS. Error responses do not permit rejected browser access. CORS is not an authorization boundary.

Transport-independent headers remain nosniff, DENY framing and no-referrer. Authentication namespace responses, including early CORS/body errors, set no-store/no-cache. The Go listener serves HTTP; terminate HTTPS at the reviewed ingress, enforce HTTPS there, and configure HSTS there only for an HTTPS-only hostname and appropriate subdomain semantics. The app does not infer HTTPS from untrusted forwarded headers or emit HSTS on local HTTP. No blanket CSP/COEP/CORP was added that could break Scalar's pinned CDN or legitimate integrations. See [Go HTTP server semantics](https://pkg.go.dev/net/http#Server).
