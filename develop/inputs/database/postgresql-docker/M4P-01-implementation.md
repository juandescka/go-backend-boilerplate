# M4P-01 — PostgreSQL Docker Development Setup

- **Feature:** `database/postgresql-docker`
- **Iteration:** `M4P-01`
- **Type:** Implementation
- **Approval:** Approved by the user's M4P master prompt, 2026-11-13.

## Objective and scope

Provide one official PostgreSQL 17 development service in root `compose.yaml`, with environment-based credentials, localhost access, a named volume and healthcheck. Keep Go/Gin running locally and preserve its existing MySQL implementation until M4A. No migrations, schemas, backend integration, dependency changes, or production changes.

## Existing environment findings

- Working tree was clean before implementation. No development Compose configuration exists. `compose.production.yml` is empty; an existing backend `Dockerfile` will be preserved.
- Root `.env` exists with MySQL and other application settings, but no `POSTGRES_*` variables. It is ignored by `.gitignore` and not tracked. `.env.example` exists.
- Docker Engine 27.4.0 and Compose v2.31.0-desktop.2 are available after accessing the daemon outside the filesystem sandbox.
- No shell `POSTGRES_*` or Compose control-variable overrides were set during audit. No host listener on 5432 was found.
- Another active Compose project publishes PostgreSQL on 127.0.0.1:15432. An unrelated stopped legacy container reserves 5432 in its configuration; do not start or change it while this service uses 5432. No `hris-core-service` Compose project or matching named volume exists.
- Required repository guidance and technical docs were read. Workflow lives at `develop/README-develop.md`; `develop/README.md` does not exist despite older links.
- `config/config.go` validates MySQL only; the connector and migration runner use MySQL. They remain unchanged.

## Implementation plan and files affected

1. Add the supplied baseline to `compose.yaml` without backend services or custom images.
2. Append only missing PostgreSQL variables to ignored `.env`, generating a strong random local password without printing it. Append safe examples to `.env.example`; retain every existing local setting.
3. Add short bilingual README quick starts and update existing configuration, CLI, operations, and architecture references.
4. Run static and actual Docker checks; record evidence in the paired review. Request a separate-context reviewer after implementation, fixing verified in-scope findings in new iterations if needed.

Expected tracked files: `compose.yaml`, `.env.example`, `README.md`, `README.id.md`, `documentation/{configuration,cli,operations,architecture}.md`, this input and the paired review. `.gitignore` already protects `.env` and needs no change.

## Security and persistence

Use `${VAR:?…}` for required credentials, `127.0.0.1:${POSTGRES_PORT:-5432}:5432`, `postgres_data:/var/lib/postgresql/data`, `restart: unless-stopped`, and `pg_isready` with container environment expansion. Do not log resolved configuration or credentials. Preserve all other Docker resources. Never delete/reset volumes. Document first-initialization credential behavior and shell-over-`.env` precedence.

## Verification plan and acceptance

- `docker compose config -q`, focused parsed-model checks without printing secrets, `git diff --check`, `.env` ignore/untracked and preservation checks.
- Recheck project/port isolation; `docker compose up -d`; wait for health; inspect safe status, binding and volume fields.
- Read-only SQL: `SELECT 1`, actual database/user/version, and TCP password authentication. No business tables or migrations.
- Stop/start only the newly created project service; require healthy again and the same volume/database system identifier.
- Verify Go/config/dependencies/generated Swagger, production Compose and backend Dockerfile remain unchanged.
- Independent review must assess Compose, image, environment security, port, persistence, healthcheck, lifecycle, runtime evidence, docs and preservation. PASS requires actual runtime evidence; unavailable runtime means BLOCKED.

## Documentation impact

Keep full commands in `documentation/cli.md`, Compose settings and credential semantics in `configuration.md`, lifecycle in `operations.md`, and the local-Go/separate-PostgreSQL boundary in `architecture.md`. README quick starts link to these references. Existing MySQL, Swagger, migration, health and readiness documentation stays valid.
