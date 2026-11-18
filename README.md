# Mapatu Backend

**A minimal, developer-friendly Go backend foundation for Mapatu HRIS.**

**Language:** English | [Bahasa Indonesia](README.id.md)

> **Project status:** Backend foundation covers operations, authentication, sessions, tenancy, RBAC, local provisioning and HTTP security. See the [canonical M9 readiness review](develop/reviews/foundation-readiness/M9-FINAL-review.md) for the verified scope and M10 gates. This is not a completed HRIS product or production deployment certification. Startup requires valid local configuration and a reachable development PostgreSQL instance.

## Overview

Mapatu favors pragmatic Clean Architecture, clear package responsibilities, idiomatic Go and minimal abstraction. Existing structure is preserved. M5B provides email/password login and opaque bearer current-user authentication. Tenant authorization follows RBAC Business Policy v1.0. Named seeders and first-admin bootstrap are available for local development.

## Technology stack

Go 1.24 (`go.mod`), Gin, Cobra, GORM + PostgreSQL, golang-migrate, Logrus and the existing optional Prometheus component.

## Project layout

```text
main.go                 CLI entrypoint
cmd/                    Server, migrations, named seeders and bootstrap
config/                 Configuration loading and validation
internal/
  registry/              Dependency initialization and cleanup
  routes/                Operational, authentication, docs and optional metrics routes
  handlers/              HTTP handlers
  usecases/health/       Readiness boundary and lifecycle state
  usecases/authentication/ Login, sessions and typed identity
  usecases/authorization/ Explicit scoped permission evaluation
  infrastructure/        PostgreSQL pool, stores, migrations and local provisioning
  shared/                Shared infrastructure utilities
documentation/           Technical reference
migrations/postgres/     Versions 000001–000015: identity, tenancy and RBAC policy
migrations/mysql/        Empty legacy directory, retained
develop/                 Development inputs and independent reviews
```

## Getting started

Use Go compatible with `go.mod` and the PostgreSQL Docker development database.

```bash
go version
go mod download
```

1. Create local `config.json` from `config.example.json` **only if no local config exists**. Preserve existing local files.
2. Start PostgreSQL using the Docker quick start below. Configure backend `database.driver=postgres`, host `127.0.0.1`, the published port (`5433` in the existing local M4P setup), and explicit `sslMode=disable` for local Docker. Set name/user/password to the initialized database using JSON or `DB_*`; `POSTGRES_*` is Docker-only and does not configure Go. Keep the HTTP application port in JSON.
3. The existing validator still requires a JWT secret for compatibility; opaque tokens do not use JWT signing. Generate a real local value (never commit it):

   ```bash
   export JWT_AUTH_SECRET_KEY="$(openssl rand -hex 32)"
   ```

4. Start the server from this repository root, then check `/health` and `/ready` using the configured HTTP port (example 8080; existing local setup 8099):

   ```bash
   go run main.go server
   ```

   In another terminal, use `curl -i http://localhost:8099/health` and `curl -i http://localhost:8099/ready` when the configured port is 8099.

`.env.example` lists supported overrides; retain your existing `.env`. Local `config.json` and `.env` are gitignored. The JSON example intentionally contains no JWT secret. See [configuration](documentation/configuration.md) for validation, precedence, pool settings and inactive options.

## PostgreSQL Docker quick start (M4P)

With Docker Desktop / Docker Engine and Compose available, append missing `POSTGRES_*` values from `.env.example` to your existing ignored `.env` and replace the password placeholder with a generated local password. Preserve existing settings. Run from this repository root:

```bash
docker compose up -d
docker compose ps
```

This starts only PostgreSQL 17 on `127.0.0.1:5432` (or `POSTGRES_PORT`), with persistent named storage and a healthcheck. The Go backend runs locally using this PostgreSQL database after matching its `DB_*` / JSON connection settings. See [configuration](documentation/configuration.md#7-postgresql-docker-development-m4p) for initialization/precedence and [Docker commands](documentation/cli.md#8-docker--postgresql-development-commands) for lifecycle and SQL checks.

## Quick command reference

| Command | Purpose |
|---|---|
| `go run main.go --help` | Show CLI help |
| `go run main.go server` | Verify DB connectivity, then serve HTTP |
| `go run main.go migrate --help` | Inspect implemented migration commands |
| `go build ./...` | Build all packages |
| `go test ./...` | Run focused tests |
| `go vet ./...` | Run static checks |

PostgreSQL migration CLI uses the same backend configuration and `migrations/postgres`. M5A supplies `users` and `auth_sessions` migrations; review the effective target before applying `go run main.go migrate up`. `version` reports applied state and may initialize metadata. Global users are separate from employee/tenant identity; login and bearer current-user authentication are implemented after explicit schema deployment. See the [identity/session schema](documentation/database/identity-session-schema.md). `down` rolls back one version; `force` is manual metadata recovery only. Server startup never runs migrations. Full command behavior is in [CLI reference](documentation/cli.md).

## Local seeding and administrator bootstrap

```bash
go run main.go seed list
go run main.go seed run demo-tenant --target 127.0.0.1:5433/mapatu_dev
go run main.go bootstrap super-admin --target 127.0.0.1:5433/mapatu_dev
```

Apply migrations before a write command and use the exact effective target. Writes require the development environment and a verified local PostgreSQL endpoint. Bootstrap requires an interactive terminal, a password of at least 15 characters (valid UTF-8, max 1024 bytes), and creates only the first platform administrator. Use a local cluster you control. See the [CLI and dependency order](documentation/cli.md#4-development-seeding-and-administrator-bootstrap) and [operational safeguards](documentation/operations.md#local-seeding-and-bootstrap).

## Tenancy & organization foundation

Module A adds SQL migrations000004–000009 for tenants, organizations, branches, departments, positions and work locations. Composite references enforce tenant/organization consistency; multiple head offices and organization-level work locations are supported. See the [schema contract](documentation/database/tenancy-organization-schema.md) for exact fields, scoped uniqueness and the deferred validation of longer hierarchy cycles. Membership and RBAC schema migrations000010–000014 add tenant associations, six fixed system roles and scope-safe assignments. Migration 000015 installs eight permission codes and 19 explicit role grants. The policy checks resource ownership and assignment containment. See the [RBAC contract](documentation/database/tenant-membership-rbac-schema.md). No organizational CRUD, employee assignment is implemented. Deploy migrations explicitly after reviewing the target; disposable tests do not migrate development data.

## Authentication foundation

`POST /v1/auth/login` verifies email/password and returns an opaque bearer token; `GET /v1/auth/me` validates that token, session expiry/revocation and user activation. See [authentication API](documentation/api/authentication.md) for inputs, errors, session lifetime and deployment requirements. M5C adds protected `POST /v1/auth/logout` and `/v1/auth/logout-all`, retaining revoked records. No registration, renewal/refresh, tenant permissions or default user is implemented. Logout-all is ordered against session creation; a fresh password login ordered afterward may create a new session. Login uses process-local IP and normalized-identifier attempt limits with HTTP 429 and `Retry-After`. Configure explicit CORS origins and trusted proxy IPs before deployment; multi-instance public exposure needs shared abuse protection. See [security configuration](documentation/configuration.md#login-abuse-policy) and [operations](documentation/operations.md#authentication-abuse-operations).

## Operational endpoints

| Method | Path | Behavior |
|---|---|---|
| `GET` | `/health` | HTTP 200 for liveness without DB checks |
| `GET` | `/ready` | HTTP 200 for successful DB ping; HTTP 503 on failure, timeout, unavailable checker or shutdown |

Probes use the root router and global middleware without authentication/login rate limiting. Readiness uses the shared SQL pool with a 2-second deadline and `SERVICE_NOT_READY` on failure. DB failure during startup prevents HTTP listening. SIGINT/SIGTERM mark shutdown, drain HTTP and close the pool once. See [operations](documentation/operations.md) for responses and middleware implications.

## API reference (development)

With `app.env` / `APP_ENV` set to `development`, open [http://localhost:8099/docs](http://localhost:8099/docs) when `app.port` is 8099. Otherwise use your configured port (the JSON example uses 8080). Restart the server after building changes.

Scalar replaces Swagger UI and reads the existing `/swagger/doc.json` specification. It documents health/readiness and all four authentication operations (login, current user, logout and logout-all), including readiness 503 and login 429. `/swagger/index.html` and Swagger UI assets are removed; consumers of the JSON URL keep the same path. Docs and JSON are unavailable outside development. Existing configuration/database startup requirements still apply.

Scalar v1.69.2 loads from pinned jsDelivr modules, so the UI needs internet/CDN access and JavaScript. No frontend server, npm build, Scalar account/key, Registry or Scalar CLI is needed. API requests go directly to your backend; AI Agent, telemetry and external proxy are disabled. System fonts avoid a separate font CDN.

Swaggo remains the only generator. Regenerate existing `docs/` artifacts after annotations change:

```bash
go generate .
```

This still invokes Swaggo v1.16.6 with the M3A command. See [CLI](documentation/cli.md#5-api-specification-generation).

## Documentation and workflow

- [API standards and HTTP foundation](documentation/api/api-standard.md): current response/error contracts, binding and pagination limits, routing and middleware conventions.
- [Architecture](documentation/architecture.md), [configuration](documentation/configuration.md), [operations](documentation/operations.md), [CLI](documentation/cli.md).
- [Development workflow](develop/README-develop.md): approved inputs in `develop/inputs/`, independent reviews in `develop/reviews/`.
- [Foundation readiness and M10 requirements](develop/reviews/foundation-readiness/M9-FINAL-review.md): current verification evidence, risks and development acceptance; [safe local checklist](documentation/operations.md#foundation-development-checklist).
- Tests, runtime evidence and limitations do not replace independent review.

## Development principles

Preserve structure, favor readable Go, add abstractions only when needed, verify according to risk, and synchronize documentation with implementation. Discuss structural or contract changes before applying them.
