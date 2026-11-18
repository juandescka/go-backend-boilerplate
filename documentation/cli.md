# CLI Command Reference

Commands are registered by Cobra in `cmd/`. Run the following commands from the `hris-core-service` repository root unless stated otherwise. Operational wiring is implemented; live startup requires valid local configuration and reachable PostgreSQL. See [Configuration Guide](configuration.md).

## 1. Quick reference

| Task                             | Command                          | Notes                                                |
| -------------------------------- | -------------------------------- | ---------------------------------------------------- |
| Show CLI help                    | `go run main.go --help`          | No server startup                                    |
| Show server help                 | `go run main.go server --help`   | No server startup                                    |
| Start the backend                | `go run main.go server`          | Requires valid configuration and PostgreSQL          |
| Show migration help              | `go run main.go migrate --help`  | Inspect commands before using them                   |
| Apply migrations                 | `go run main.go migrate up`      | Apply pending PostgreSQL migrations in order         |
| Inspect migration version        | `go run main.go migrate version` | Reports version/dirty state; may initialize metadata |
| **Regenerate API specification** | **`go generate .`**              | **Recommended; pins Swaggo v1.16.6**                 |
| Build Go packages                | `go build ./...`                 | Compile all packages                                 |
| Run Go tests                     | `go test ./...`                  | Automated tests                                      |
| Run static checks                | `go vet ./...`                   | Go vet                                               |

Local provisioning commands are in [section 4](#4-development-seeding-and-administrator-bootstrap). The Scalar UI is a renderer, not a replacement for the Swaggo generator.

## 2. Server and help

```bash
go run main.go --help
go run main.go server --help
go run main.go server
```

`server` initializes dependencies, verifies PostgreSQL connectivity, then binds HTTP. DB failure is fail-fast. SIGINT/SIGTERM trigger graceful shutdown and shared-pool cleanup. No application-specific server flags are defined; configure host/port and timeouts through supported configuration keys.

## 3. PostgreSQL migration commands

Run from the repository root. Migration CLI uses the same required `config.json`, automatic `.env` loading and `DB_*` overrides as the backend (process > `.env` > JSON > defaults). It does not start HTTP or reuse/create the backend registry. Verify the effective database target before any command; local M4P uses host port 5433. `POSTGRES_*` still configures Docker only.

| Command                                  | Actual behavior                                                                                        |
| ---------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| `go run main.go migrate --help`          | Lists existing subcommands; no database access                                                         |
| `go run main.go migrate up`              | Applies pending versions in increasing order; `No pending migrations` is a successful no-change result |
| `go run main.go migrate down`            | Rolls back **exactly one** applied version using `Steps(-1)`; no rollback-all command                  |
| `go run main.go migrate version`         | Reports version and dirty flag; fresh/fully rolled-back state prints `Migration version: none`         |
| `go run main.go migrate force <version>` | Sets metadata/clears dirty without applying or undoing SQL; integer must be >= -1                      |

`force 0` represents an actual version zero, displayed as `0 dirty: false`, rather than none. Use `force -- -1` for golang-migrate's nil version (Cobra requires `--` before the negative positional argument). Values below -1, nonintegers and overflow fail before configuration/database access. These are recovery semantics, **not setup instructions**. Never force a version without checking which SQL actually succeeded. No force runs automatically.

### Files and empty-source behavior

Active filesystem source: `migrations/postgres`, resolved from the working directory; missing/unreadable/duplicate sources fail before database access. The legacy `migrations/mysql` directory was inspected as empty and is retained. Use positive six-digit sequential versions beginning `000001`, each with matching `description.up.sql` and `description.down.sql`, for example `000001_description.up.sql` / `000001_description.down.sql`. Do not mix timestamp and sequential numbering. M5A supplies complete pairs `000001_create_users` and `000002_create_auth_sessions`; M5B adds `000003_index_session_token_hash`; `.gitkeep` is retained. See the [identity/session schema](database/identity-session-schema.md) for columns, constraints and deferred authentication decisions.

With current source, `up` creates global users, sessions, their token-hash index, then the six Module A tables through version9 and five membership/RBAC tables through version 14. Version 15 adds the v1.0 catalog and role grants. Versions000004–000009 and their scope/integrity rules are documented in the [tenancy/organization schema](database/tenancy-organization-schema.md). From version 15, one down removes only the seeded policy data and refuses rollback if later mappings depend on it. From version 14, five one-step downs remove role_assignments, role_permissions, permissions, roles and tenant_memberships, returning to9 and preserving M5/M6 rows. See the [RBAC contract](database/tenant-membership-rbac-schema.md). From version9, one `down` drops only work_locations and returns to8. Six one-step downs remove Module A in reverse dependency order and stop at3, preserving Module B. From version3, one `down` drops only the token-hash index and preserves both identity tables/data at version2. For the original M5A source, `up` creates global users then sessions. From clean version 2, one `down` drops sessions only (version 1); a second drops users (none). These operations delete the corresponding table data. Verify the target and rollback plan first; never demonstrate down/force on existing development data. Test lifecycle on a newly created disposable database.

The following empty-source behavior applies to an empty source (including test fixtures), rather than the current populated source, and is verified against **golang-migrate v4.18.3**:

- An empty source is valid to open. `version` reports none after driver metadata initialization. It does not create business tables.
- `up` on a fresh database with an empty source exits nonzero: `migration source contains no versions`. This differs from no pending migrations after an applied version.
- `down` with no applied version exits nonzero: `no applied migration to roll back`.
- Unparseable filenames are ignored by the file driver; an invalid SQL body in a correctly named file fails execution. Duplicate version/direction files fail source opening. A forced/current version absent from both directions in the source is an error, not no-change success.
- Complete up/down pairs are a repository convention checked during migration review. The library does **not** enforce pairs: if only one direction exists, the missing direction can run as a metadata-only migration, leaving actual schema unchanged. Do not omit a rollback file expecting the runner to reject it; this library limitation was verified with a disposable fixture.

### Metadata, locking, transactions and recovery

The official PostgreSQL driver checks/creates `schema_migrations` during initialization and uses PostgreSQL advisory locking. **Even `version` may create metadata and acquire a lock**, so it is not guaranteed read-only. Initialization errors close the source/reserved connection/pool. Each CLI invocation owns one migration SQL pool/connection, closes it on success/failure and reports cleanup failures through a nonzero exit status. HTTP startup never invokes this runner.

Migrations use pgx under the official PostgreSQL migration driver. `database.queryTimeoutSeconds` / `DB_QUERY_TIMEOUT_SECONDS` bounds connection establishment, each complete migration-file execution through the driver context, and server-side statement/lock waits; default 5s. This is not a deadline for the entire multi-version run. Library lock timeout is set one second above this bound. These limits apply to CLI sessions only, not backend query policy. Raise the configured value intentionally for reviewed long-running migrations; there is no retry loop.

Cancellation/timeout may discard the reserved pgx connection. Close that runner and inspect persisted state through a new CLI invocation (`version`); do not assume the same in-process runner remains usable after a canceled operation. An already discarded connection is treated as closed during cleanup, while other cleanup failures remain errors.

The driver sends each file as SQL without naive semicolon splitting (`MultiStatementEnabled=false`). A multi-statement file executes as one PostgreSQL query/implicit transaction. Single-statement operations such as `CREATE INDEX CONCURRENTLY` need their own file outside an explicit transaction. Do not assume the entire multi-version run is one transaction: completed earlier versions remain applied if a later version fails. Review SQL/transaction boundaries per migration ([versioned driver reference](https://github.com/golang-migrate/migrate/blob/v4.18.3/database/postgres/README.md)).

A failed SQL migration leaves dirty metadata and blocks later up/down. Error output uses safe categories and SQLSTATE where available, withholding raw SQL, driver details, passwords and URIs. Inspect the corresponding migration file and actual schema before deciding recovery. Repair or manually reconcile partial effects only under a separately verified recovery plan, then explicitly force **the version matching the inspected state** and retry the intended operation. Do not delete metadata, reset a database or use automatic force to hide a failure. A failed first rollback can leave version -1 dirty; `version` explicitly prints `none dirty: true`, and up/down remain blocked.

Lifecycle/dirty/force tests are opt-in and use a newly created disposable database, temporary SQL and real CLI binary; never run destructive fixture tests on `mapatu_dev` or an existing business database. Migrations and seeding are separate explicit CLI operations.

## 4. Development seeding and administrator bootstrap

`seed --help` and `seed list` read command metadata and work without local configuration or a PostgreSQL connection. `seed run <name> --target <host:port/database>` runs one dataset. An unknown name exits nonzero before loading configuration. Apply migrations to the intended local development database first.

```bash
go run main.go migrate up
go run main.go seed --help
go run main.go seed list
go run main.go seed run dev-users --target 127.0.0.1:5433/mapatu_dev
go run main.go seed run demo-tenant --target 127.0.0.1:5433/mapatu_dev
go run main.go seed run demo-organization --target 127.0.0.1:5433/mapatu_dev
go run main.go seed run demo-memberships --target 127.0.0.1:5433/mapatu_dev
```

Replace the example target with the effective local development endpoint and database. All writes require the development environment, a confirmed numeric loopback endpoint, matching live database and role, and a clean migration version of at least 15. See [configuration](configuration.md#local-provisioning) and [operations](operations.md#local-seeding-and-bootstrap).

| Seeder              | Dataset                                                                                     | Dependencies               |
| ------------------- | ------------------------------------------------------------------------------------------- | -------------------------- |
| `dev-users`         | One inactive global demo user with no password                                              | None                       |
| `demo-tenant`       | One tenant using code `MAPATU_DEMO` and slug `mapatu-demo`                                  | None                       |
| `demo-organization` | One company using code `MAPATU_DEMO_ORG` in the demo tenant                                 | `demo-tenant`              |
| `demo-memberships`  | One active membership linking the demo user and tenant. The user retains its global status. | `dev-users`, `demo-tenant` |

A successful first run reports `Created: 1`. A recognized rerun reports `Already exists: 1. Created: 0` and preserves edits. Business-key collisions with unowned rows and missing dependencies return nonzero. Run each missing dependency explicitly. The seeders create zero role assignments.

### Initial platform administrator

```bash
go run main.go bootstrap super-admin --target 127.0.0.1:5433/mapatu_dev
```

Use a trusted operator terminal on the verified local development target. Enter a new email, a display name and a password twice when prompted. Password input is hidden. Use at least 15 characters and at most 1024 bytes of valid UTF-8. Terminal input is mandatory. The command uses the existing Argon2id implementation and commits one new active user and its PLATFORM assignment together. An existing email, previous platform assignment including an inactive one, unsafe target, missing grant, invalid input or database error produces nonzero exit status. Only the fixed SUPER_ADMIN role and its existing `platform.manage` grant are used. Production initialization and assignment administration require separate operational approval.

### Add a named seeder

1. Choose a unique name and one narrowly defined development dataset. Record its explicit dependencies and reserved identity keys.
2. Implement a dataset function in `internal/infrastructure/seeding`. It receives a context and transaction and returns whether it created data. Verify dependency ownership and distinguish an owned no-op from a conflicting record.
3. Register the function and description in the registry map. The command runner picks up its name and description from that map.
4. Test the dataset's risky paths on a disposable database, including idempotency and dependency or ownership failures. Update this table and the [operations guide](operations.md#local-seeding-and-bootstrap).

## 5. API specification generation — Swaggo + Scalar

The recommended command is:

```bash
go generate .
```

The single `go:generate` directive in `main.go` invokes the following **pinned** generator command, which can also be run directly:

```bash
go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g main.go -o docs --parseInternal
```

### What about `swag init`?

If **Swaggo CLI v1.16.6** has already been installed on your machine, the equivalent CLI command is:

```bash
swag init -g main.go -o docs --parseInternal
```

Check the installed binary first:

```bash
swag --version
```

Do **not** use plain `swag init` as the documented Mapatu command: it does not explicitly preserve the project's entrypoint, output directory, and internal-package parsing options. `swag init -g main.go --parseInternal` may happen to produce the same output when Swaggo's defaults point to `docs/`, but Mapatu should use the full explicit command above or, preferably, `go generate .`.

`go generate .` is preferred because it does not depend on an independently installed global `swag` binary. On the first execution it may require access to the Go module proxy/checksum service, unless the needed modules are cached.

After API metadata or annotation changes, regenerate and commit these existing generated files together:

- `docs/docs.go`
- `docs/swagger.json`
- `docs/swagger.yaml`

Do not edit generated files manually. The output is Swagger 2.0; `--parseInternal` includes types in existing internal packages. Keep the established `docs/` location. No new generator, output folder, or conversion pipeline is required.

**Scalar displays the generated specification; it does not generate it.** Gin serves `swag.ReadDoc()` at `/swagger/doc.json`, without `gin-swagger` or reading the generated files from disk. Scalar replaces the old Swagger UI. There is no required Scalar CLI, Node/npm command, OpenAPI conversion, or second generator.

## 6. Local API documentation and operational checks

With valid local configuration and a running PostgreSQL instance, start the backend:

```bash
go run main.go server
```

For the documented development instance at `http://localhost:8099`, use:

| Path                | Purpose                                                                             |
| ------------------- | ----------------------------------------------------------------------------------- |
| `/docs`             | Scalar API Reference; development only                                              |
| `/swagger/doc.json` | Generated Swagger specification; check route availability in the active environment |
| `/health`           | Liveness                                                                            |
| `/ready`            | Readiness                                                                           |

Example requests:

```bash
curl -i http://localhost:8099/health
curl -i http://localhost:8099/ready
curl -i http://localhost:8099/swagger/doc.json
```

Open `http://localhost:8099/docs` in a browser with `APP_ENV=development`. The actual host/port follow local configuration. The Scalar page imports Scalar v1.69.2 from jsDelivr, so the **browser** needs CDN access; the backend itself does not download Scalar assets and does not require Node/npm. Loading the HTML page alone does not prove Scalar has loaded the specification: verify that its endpoint list renders in the browser.

Generating the API specification does **not** launch the backend, connect to PostgreSQL, modify local configuration, or run migrations. See [Operations Guide](operations.md#7-scalar-api-reference-development-only).

## 7. Go development tools

```bash
go version
go mod download
go build ./...
go vet ./...
go test ./...
```

Optional, when relevant:

```bash
# Detect data races in code exercised by tests
go test -race ./...

# Run matching Go tests in all packages
go test ./... -run 'TestOperational'

# Reconcile dependencies only when dependency changes are intended
go mod tidy
```

Focused tests cover operational route/response semantics, readiness deadline/cancellation, missing/closed pools, and the existing metrics route. Tests are separate from live PostgreSQL/server smoke checks. `go mod tidy` can modify `go.mod` and `go.sum`; review its diff before committing.

Implementation evidence and reviewer verdicts belong in `develop/`, not in this technical command reference.

## 8. Docker & PostgreSQL Development Commands

Run from this repository root with Docker Engine running and the [local PostgreSQL variables](configuration.md#7-postgresql-docker-development-m4p) configured. Only PostgreSQL runs in Compose; Go runs locally against it using separately aligned backend `DB_*` / JSON settings.

| Task                                      | Command                           | Behavior                                                                  |
| ----------------------------------------- | --------------------------------- | ------------------------------------------------------------------------- |
| Validate Compose                          | `docker compose config -q`        | Validates interpolation/model without printing credentials                |
| Initial startup / reconcile configuration | `docker compose up -d`            | Creates project network, volume and container as needed                   |
| Start existing containers                 | `docker compose start`            | Starts stopped containers; does not apply changed `.env`/Compose settings |
| Stop containers                           | `docker compose stop`             | Retains containers, network and named volume                              |
| Check status                              | `docker compose ps`               | Inspect running/healthy state                                             |
| Follow PostgreSQL logs                    | `docker compose logs -f postgres` | Ctrl-C stops following, not the database                                  |
| Remove project containers/network         | `docker compose down`             | Retains the named volume; use `up -d` to recreate                         |

Wait for `healthy` before SQL checks. **`docker compose down -v` deletes the named volume and its database data; never use it as routine shutdown.** Do not prune/reset volumes. Keep the same Compose project identity to reuse the same data.

Read-only SQL using the configured container user/database:

```bash
docker compose exec -T postgres sh -c 'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT 1;"'
docker compose exec -T postgres sh -c 'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT current_database(), current_user;"'
```

The socket commands above and `pg_isready` do not verify the password. Container loopback TCP can also use `trust`. Use service hostname `postgres` to connect through the container's bridge address and verify host password authentication without printing the password:

```bash
docker compose exec -T postgres sh -c 'PGPASSWORD="$POSTGRES_PASSWORD" psql -w -h postgres -p 5432 -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT 1;"'
```

An intentionally wrong, non-secret password on the same path must fail with a password authentication error and a nonzero exit status:

```bash
docker compose exec -T postgres sh -c 'PGPASSWORD=m4p_intentionally_wrong_password psql -w -h postgres -p 5432 -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT 1;"'
```

For interactive `psql` from a terminal (exit with `\q`):

```bash
docker compose exec postgres sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
```

Docker Desktop recognizes the Compose project after `up -d` (default name `hris-core-service`). Its Containers view can stop/start the project or service; terminal commands work independently of the UI. Desktop start, like `compose start`, reuses the existing configuration. Apply changes with `compose up -d`; changing initialization credentials does not alter existing volume contents. A healthy PostgreSQL container alone does not prove backend readiness: `/ready` checks the backend shared PostgreSQL pool using its own connection settings. No migrations run through this Compose file.

## References

- [Swaggo generator, v1.16.6](https://github.com/swaggo/swag/tree/v1.16.6)
- [Scalar HTML/JS integration](https://scalar.com/products/api-references/integrations/html-js)
- [Scalar configuration](https://scalar.com/products/api-references/configuration)

## Authentication verification (M5B)

M5C adds HTTP logout and logout-all. Authentication needs the M5A/M5B schema. The migrate up command applies all pending migrations. Start the backend with the existing `go run main.go server` after explicit schema deployment; health/readiness remain available before business schema is installed. See [authentication API](api/authentication.md).

```bash
go build -o /private/tmp/hris-m5b-backend .
M5B_INTEGRATION=1 M5B_CLI_BINARY=/private/tmp/hris-m5b-backend go test -count=1 -run '^TestAuthenticationPostgresIntegration$' -v ./internal/registry
```

This opt-in integration creates/removes its own uniquely named PostgreSQL database using verified ownership/OID; all migration and fixture writes stay there. Existing development database is only inspected via read-only catalog queries. The test needs PostgreSQL17, create-database permission and the current built CLI. Continue using `go generate .` for the pinned Swaggo/Scalar specification.

The existing disposable `TestAuthenticationPostgresIntegration` also exercises M5C lifecycle and deterministic PostgreSQL lock orderings through the actual routes/store; its M5B opt-in variable and fixture ownership checks are retained. It tests the full current migration source, then rolls back later module versions before exercising the original token-index-only rollback and full reapply, preserving identity rows. No separate lifecycle CLI, worker or test framework is added. Do not run migration version/up on an unverified target, or down/force on existing development data for demonstration.

## Tenancy & organization verification

The [schema integration test](../internal/infrastructure/migration/tenancy_organization_schema_integration_test.go) uses the actual CLI on a fresh, owned PostgreSQL17 database. It needs the current built binary and create-database permission; the configured development database is inspected only in read-only catalog transactions. M6_STAGE=a limits source to versions1–5; b verifies versions1–9; full adds exact integrated catalog/index counts and per-step table preservation. Each run checks ownership/OID before exact-name cleanup.

```bash
go build -o /private/tmp/hris-m6-backend .
M6_INTEGRATION=1 M6_STAGE=full M6_CLI_BINARY=/private/tmp/hris-m6-backend go test -count=1 -run '^TestTenancyOrganizationSchemaIntegration$' -v ./internal/infrastructure/migration
```

Verification includes cross-tenant/organization rejection, nullable parent/branch scope, multiple head offices, field bounds, audit conventions, repeat-up and down9→3/reapply9 while preserving Module B/unrelated objects. Longer hierarchy cycles are deliberately demonstrated as an unenforced database boundary and immediately restored in the disposable fixture. Future write operations must validate acyclicity; this test does not certify a future authorization or hierarchy service. See the [canonical contract](database/tenancy-organization-schema.md).

## Membership & RBAC verification

Migration 000010 creates memberships. Versions 000011–000014 create the fixed global roles, permission and grant tables, and scoped assignments. Version 000015 supplies the eight v1.0 permissions and 19 explicit grants. Accounts, memberships and assignments are provisioned separately. See the [contract](database/tenant-membership-rbac-schema.md).

Opt-in verification uses a new owned disposable PostgreSQL17 database, never the configured development database, with exact-name/OID/owner cleanup. It uses the existing migration runner on copied stage sources:

```bash
M7_INTEGRATION=1 M7_STAGE=a go test -count=1 -run '^TestTenantRBACPostgresIntegration$' -v ./internal/infrastructure/migration
M7_INTEGRATION=1 M7_STAGE=b go test -count=1 -run '^TestTenantRBACPostgresIntegration$' -v ./internal/infrastructure/migration
M7_INTEGRATION=1 M7_STAGE=full go test -count=1 -run '^TestTenantRBACPostgresIntegration$' -v ./internal/infrastructure/migration
```

Stage a verifies memberships through version 10. Stage b verifies all five tables through version 14. Stage full applies version 15 and verifies the exact matrix, database-backed containment, bearer authentication through a stored-resource fixture, and rollback isolation.

## Local provisioning verification

Build the current binary, then run the opt-in checks with PostgreSQL 17, a local role allowed to create a disposable database, and Python 3 for the pseudo-terminal fixture.

```bash
go build -o /tmp/mapatu-m7s-backend .
M7S_INTEGRATION=1 M7S_CLI_BINARY=/tmp/mapatu-m7s-backend go test -count=1 -run '^TestProvisioningPostgresIntegration$' -v ./cmd
```

The suite verifies database ownership before cleanup. It tests the actual named commands, hidden terminal input, bootstrap interruptions, concurrent creation and rollback. The test provisions only its disposable database.
