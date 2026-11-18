# M4B-01 — PostgreSQL Migration Foundation

- **Feature:** `database/postgresql-migration`
- **Iteration:** `M4B-01`
- **Mode:** Audit first + implementation + independent review
- **Approval:** User's M4B master prompt, 16 November 2026; autonomous fix loop authorized.
- **Dependencies:** [M4P-03 PASS](../../../reviews/database/postgresql-docker/M4P-03-review.md), [M4A-01 PASS](../../../reviews/database/postgresql-backend/M4A-01-review.md).

## Objective and scope

Activate PostgreSQL migrations through the existing Cobra/golang-migrate runner. Preserve package layout, command names, one-step rollback, backend configuration and HTTP lifecycle. No business schema, production operation, MySQL data migration, AutoMigrate, automatic startup migrations or Docker changes.

## M4A dependency verification

The working tree was clean at baseline (`c433cd4`). Actual source retains the official GORM PostgreSQL/pgx connector, one shared SQL pool, bounded startup/readiness, unchanged metrics/shutdown and process > `.env` > required JSON > defaults precedence. Latest M4A independent review is PASS. Fresh opt-in `TestPostgresSharedPoolIntegration` passed against PostgreSQL 17.12 on 127.0.0.1:5433, verifying GORM/pool identity, readiness, metrics and cleanup. Container remains healthy. Read-only baseline of `mapatu_dev`: zero business tables and no `public.schema_migrations`. Local credentials were not displayed.

Required repository guidance, current references/examples/source, M4P/M4A review history and exact installed golang-migrate v4.18.3 source were inspected. Workflow actually lives at `develop/README-develop.md`. PostgreSQL official migration/driver references were checked at tag v4.18.3.

## Existing migration assessment

| Component | Existing behavior | Action |
|---|---|---|
| Cobra commands | Existing up/down/version/force; M4A guard blocks runner before configuration | ADAPT: load existing configuration and enable runner; retain command forms |
| Migration driver | Registered golang-migrate MySQL; currently unreachable | ADAPT: official PostgreSQL driver with owned pgx SQL pool/reserved connection |
| Database configuration | Required JSON and DB_* overrides already PostgreSQL; POSTGRES_* only Docker | KEEP; build escaped PostgreSQL URL from the same effective settings |
| Migration source | Filesystem `migrations/mysql`, inspected and empty | ADAPT: active `migrations/postgres/.gitkeep`; preserve legacy directory |
| Up | Existing Up; swallows ErrNoChange and always prints applied | KEEP library Up; ADAPT truthful no-change output and safe errors |
| Down | Existing Steps(-1), not Down-all | KEEP one step; expose fresh-state/source errors accurately |
| Version | Maps nil to zero, conflating real version 0 | FIX: distinguish -1/none, 0, and dirty nil state |
| Force | Atoi; library accepts >= -1; errors may echo input | ADAPT: validate before dialing; retain metadata-only semantics, no automatic force |
| Error handling | Raw wrapped library errors can contain URL, SQL and server detail | FIX: safe operation/category/SQLSTATE diagnostics and dirty version; no raw error/SQL output |
| Lifecycle | migrate.New can leave opened resources on initialization errors; CLI ignores Close errors | FIX: explicit source/connection/pool ownership and cleanup on every path; propagate close errors |
| MySQL dependencies | SQL driver used only by deferred migrator; GORM MySQL already removed | REMOVE when no consumer remains; retain golang-migrate version and legacy SQL directory |

## Implementation decisions and CLI contract

- Use official `database/postgres.WithConnection` with a single owned pgx stdlib SQL pool and one reserved `*sql.Conn`. This supports all M4A SSL modes, avoids default libpq Open/WithInstance failure-cleanup gaps and preserves configured identity. No HTTP server/pool is initialized by migration commands.
- Open filesystem source before connecting; missing/duplicate source errors therefore cannot create metadata. The PostgreSQL driver initializes `schema_migrations` and acquires advisory locks, including for `version`; document that metadata initialization is not read-only.
- Apply existing queryTimeoutSeconds to connection establishment and migration-session statement/lock timeouts, including initialization. Set library lock timeout above the server lock deadline to avoid a live background lock operation when closing. No retry/locking framework.
- Keep `up` ordered/pending-only, `down` exactly Steps(-1), `version` reporting dirty accurately, and `force` accepting integer >= -1 only. `force -- -1` represents library nil version; zero is a distinct valid metadata version. No automatic repair/force.
- Preserve nil dirty state using official driver.Version: migrate.Migrate.Version discards dirty for version -1 after a failed first rollback. Do not label this clean.
- Preserve actual empty-source semantics from v4.18.3: fresh Up cannot locate a first migration, fresh Steps(-1) has no applied version, while Version can report none after metadata initialization. Do not manufacture SQL or conflate missing scripts with no pending work.
- Use six-digit sequential positive versions beginning 000001, with matching PostgreSQL `.up.sql`/`.down.sql` pairs. Library filename filtering and transaction behavior will be documented based on actual tests/source.

## Files affected and dependencies

Target `cmd/migrate.go` and its test, `internal/infrastructure/migration/{config,runner}.go` and focused tests, `migrations/postgres/.gitkeep`, go.mod/go.sum only as needed for driver cleanup, bilingual READMEs and existing architecture/configuration/operations/CLI references. Add this input and paired review. Preserve runtime connector/registry/config/HTTP contracts, generated Swagger, Docker files, local `.env`/`config.json` and previous history.

## Migration safety and testing strategy

All SQL application/rollback/dirty/force tests use newly created uniquely named disposable databases owned by the verification process, never `mapatu_dev`. Create/drop only those exact test-owned databases; temporary fixtures live under test temp directories. No permanent dummy migrations. Inspect permissions before mutation and fail safely if isolation cannot be established. PostgreSQL container/volume and other projects remain untouched.

Run build, vet, full tests, CLI help and focused source/escaping/security tests. Opt-in live tests cover empty/fresh state, two sequential fixture migrations, repeat Up/no change, one-step Down, rollback to none, force 0/-1, controlled SQL failure/dirty refusal/recovery, failed first Down dirty nil, invalid/duplicate/missing source, connection failures/timeouts, supported locking and cleanup. Verify local database state and backend preservation afterward; limited HTTP regression only, not a full M4A/M1 retest.

## Documentation impact

Update existing CLI reference with active commands, empty-source behavior, metadata side effects, safe rollback/force/recovery and transactional limitations. Align configuration/architecture/operations and README EN/ID without duplicating command detail or adding a migration guide unless needed. Preserve historical PASS records. Obtain separate-context review of all fifteen requested areas; fix verified in-scope findings using new paired iteration records until PASS or genuine blocker.
