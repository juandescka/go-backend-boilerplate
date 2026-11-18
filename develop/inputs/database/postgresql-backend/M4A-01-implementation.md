# M4A-01 — PostgreSQL Backend Integration

- **Feature:** `database/postgresql-backend`
- **Iteration:** `M4A-01`
- **Type:** Audit first + implementation
- **Approval:** User's M4A master prompt, 13 November 2026; autonomous review/fix authorized.
- **Baseline:** [M4P-03 PASS](../../../reviews/database/postgresql-docker/M4P-03-review.md).

## Objective and scope

Replace only the runtime MySQL connector/configuration with PostgreSQL 17 through the official GORM PostgreSQL dialector and pgx database/sql adapter. Preserve package layout, registry ownership, one shared pool, startup/readiness bounds, metrics names, public HTTP responses and graceful shutdown. No schema, migration, production, Docker or business feature work.

## Existing architecture assessment

| Component | Current implementation | Decision / required change |
|---|---|---|
| Configuration | godotenv.Load → defaults → required config.json → supported environment overrides → validation; process > .env > JSON > defaults | ADAPT: retain loader; add DB_DRIVER/HOST/PORT/SSL_MODE, PostgreSQL-only validation and explicit SSL mode |
| Database driver | go-sql-driver/mysql + GORM MySQL | ADAPT: pgx stdlib.OpenDB + GORM postgres.Config{Conn: pool} |
| DSN | Structured MySQL driver.Config | ADAPT: safely escaped PostgreSQL URL; no raw connection/parse error output |
| Registry | Creates connector once, shares SQL pool, cleans up on metrics registration failure | KEEP |
| Readiness | Same pool PingContext with existing handler's 2-second request deadline | KEEP; adapt driver-specific test fixture |
| Metrics | sql.DB.Stats collectors on same pool; HELP text says MySQL | KEEP names/labels; ADAPT HELP text only |
| Shutdown | Registry.Close sync.Once; server signal handling and bounded HTTP shutdown | KEEP |
| Migration dependency | Existing runner imports golang-migrate MySQL, source migrations/mysql | DEFER TO M4B; retain compilation dependencies; fail closed at CLI runner boundary before loading/configuring any DB |
| GORM MySQL dialector | Used only by runtime connector | REMOVE IF UNUSED after replacing connector |

Required guidance, references, examples and latest M1/M4P reviews were read. Workflow is actually `develop/README-develop.md`; older `develop/README.md` references are stale. Working tree was clean. PostgreSQL container is currently healthy at host port 5433. Docker's combined audit approval timed out; a narrower `docker compose ps` retry succeeded, leaving no blocker.

## Root cause analysis

The existing local JSON selects MySQL at 127.0.0.1:8889 and HTTP port 8099. Existing `.env` supplies DB_* overrides; backend already auto-loads it. POSTGRES_* affects Compose only and is ignored by the loader. Validator accepts only mysql, and ConnectDB always builds a MySQL connection. Thus a healthy M4P PostgreSQL service at 5433 cannot satisfy this runtime's startup ping. The old generic error intentionally discards driver details, so source alone cannot reconstruct whether a historical MySQL failure was refusal, credentials, missing database or timeout; do not invent that detail.

## Implementation decisions and files affected

- Retain JSON plus DB_* conventions without a silent POSTGRES_* fallback. Update only database fields in ignored local config.json and existing DB_NAME/USERNAME/PASSWORD values to match the already initialized Compose database; preserve other settings/secrets and all POSTGRES_* values, including port 5433.
- Safely encode user/password/database/IPv6 through net/url and net.JoinHostPort. Require valid PostgreSQL driver, host, port, database, user, password and explicit SSL whitelist. Local sslMode=disable is explicit; deployment TLS remains configurable.
- Use one pgx stdlib pool, existing pool settings, bounded startup PingContext, then attach GORM with automatic ping disabled. Keep driver errors internal; expose fixed diagnostic categories for refusal, authentication, missing database, timeout/cancellation and invalid configuration.
- Add a fixed migration-unavailable guard before loader/runner creation; preserve command names/help/argument structure and legacy runner/source. No MySQL or PostgreSQL migrations will execute.
- Target `config/config.go`, focused config tests, `internal/infrastructure/database.go` and tests, existing readiness test, `cmd/migrate.go` and guard test, metrics HELP text, go.mod/go.sum, config.example.json/.env.example and affected existing docs. Add integration tests only for meaningful connection/ownership/runtime risks, gated by an explicit opt-in.
- Add official gorm.io/driver/postgres v1.6.0 and a Go-1.24-compatible pgx/v5 release; retain current GORM and unrelated dependencies. Remove GORM MySQL only once unused; retain go-sql-driver/mysql transitively for the deferred migrator.

## Migration boundary

No migration SQL/runner conversion, new migration directory, schema_migrations, AutoMigrate, seeder or business tables. Existing migration runner and MySQL directory remain historical/deferred infrastructure; commands must clearly fail before database access until M4B implements PostgreSQL migration support.

## Verification plan and acceptance

Run build, vet, full tests and focused race checks as appropriate. Test escaping, config validation/precedence and secret-safe errors; guard migration before any loader/network work. Actual PostgreSQL connection must succeed on configurable host port 5433. Verify one SQL pool shared with GORM/readiness/metrics, cleanup, canceled/deadline pings and repeated close. Run actual server with local config, inspect existing health/ready envelopes, metrics and Scalar/spec routes, then SIGTERM and verify owned pool/process cleanup.

For runtime readiness failure, use an isolated TCP proxy controlled by the verification harness, so only the test application's database transport is interrupted; leave PostgreSQL and other developers' services running. Verify fail-fast with isolated wrong credentials/nonexistent database/unused port and timeout/cancellation, without printing secrets or raw driver errors. Do not mutate schema or repeat volume lifecycle tests.

## Documentation impact

Synchronize both READMEs, architecture, configuration, operations, CLI and configuration examples for PostgreSQL runtime, deterministic precedence, explicit TLS, local5433/default5432, safe troubleshooting, shared pool and blocked migration commands. Inspect API standard without changing it because HTTP contracts remain the same. Preserve all historical input/review records and generated Swagger. Obtain separate-context review covering the user's twelve areas; fix verified in-scope findings in subsequent matching input/review pairs until PASS or a genuine blocker.
