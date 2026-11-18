# M4A-01 — Review

- **Feature:** `database/postgresql-backend`
- **Iteration:** `M4A-01`
- **Input:** [M4A-01 implementation](../../../inputs/database/postgresql-backend/M4A-01-implementation.md)
- **Date:** 13 November 2026 (Asia/Makassar)
- **Verdict:** **PASS for the approved M4A PostgreSQL backend integration**
- **Review method:** Separate-context reviewer `/root/m4a_review` independently inspected the input, final implementation/tests, complete tracked diff, relevant callers, local dependency source and preservation fingerprints. Build/test/runtime results below were executed by the implementing parent and are explicitly attributed. The reviewer changed only this review artifact and repeated no runtime mutation or migration.

## Scope Reviewed and Actual Changes

Runtime MySQL is replaced with official `gorm.io/driver/postgres` v1.6.0 and pgx/v5 v5.7.6, retaining Go 1.24.0 and GORM v1.31.2. [ConnectDB](../../../../internal/infrastructure/database.go) opens one pgx `database/sql` pool, applies existing pool settings, performs a bounded startup ping, and attaches GORM to that same pool with automatic ping disabled. Startup errors use fixed, secret-safe diagnostic categories.

[Configuration](../../../../config/config.go) retains godotenv and required JSON, adds `DB_DRIVER`, `DB_HOST`, `DB_PORT`, `DB_SSL_MODE`, and validates PostgreSQL identity/password/port and explicit SSL modes. URL construction escapes credentials/database names and supports IPv6. `PGSERVICE` is rejected; explicit connection fields override corresponding pgx environment inputs. Optional pgx TLS certificate behavior is documented.

[Migration CLI](../../../../cmd/migrate.go) now fails before configuration or runner creation until M4B. The deferred MySQL runner/source remains unchanged. GORM MySQL is removed; `go-sql-driver/mysql` remains indirect through golang-migrate. Metrics change only HELP text. Focused configuration, escaping, diagnostics, guard and opt-in live pool tests are added; the existing readiness fixture uses pgx. Both READMEs, examples and existing architecture/configuration/operations/CLI references are synchronized.

Ignored local database settings are aligned with initialized PostgreSQL at `127.0.0.1:5433`; HTTP remains 8099. Unrelated local settings and all `POSTGRES_*` entries are preserved. No package restructuring, business schema, migration conversion, generated API change, Docker/production change or unrelated feature was introduced.

## Independent Verification Evidence

| Reviewer check actually performed | Result |
|---|---|
| Read AGENTS English/Indonesian, actual workflow `develop/README-develop.md`, input, final source/tests and affected references | Implementation and documentation match approved scope |
| Inspect registry/server/readiness/handler/metrics callers and local official PostgreSQL dialector source | Supplied `Conn` becomes GORM's pool directly; no second production pool or hidden dialector connection is created |
| Search production pool creation, automatic migration and migration runner calls | One `stdlib.OpenDB`; no `AutoMigrate` or active `migration.NewRunner` call; CLI guard precedes loader/network access |
| `git diff --check` | Exit 0 |
| Compare SHA-256 against `/private/tmp/m4a-before.json` | 60 of 75 baseline files unchanged; 15 changes exactly match runtime/config/dependency/docs scope |
| Protected-file comparison | Compose, Dockerfile, production Compose, generated Swagger, API standard, routes/handlers/middleware/health, registry/server, legacy migrator and prior history unchanged |
| Recompute unrelated local env/JSON hashes using parent-supplied exclusion definitions | Both match pre-M4A baseline; no values printed; all `POSTGRES_*` and unrelated secrets/settings preserved |
| Check local file permissions and Git metadata | `.env` and `config.json` are mode 0600, ignored and untracked |
| Read secret-free runtime results and verification harness | Assertions cover actual server, negative startup, isolated outage/recovery, timeout and migration guard scenarios |

The reviewer did not repeat builds/tests already completed successfully by the parent. No Docker mutation, schema query/write or migration was run by the reviewer.

## Actual Execution Evidence Supplied by Implementer

| Command/check | Observed result |
|---|---|
| `go build ./...`, `go vet ./...`, `go test ./...` with isolated `GOCACHE` | All PASS |
| `M4A_INTEGRATION=1 go test -count=1 -run TestPostgresSharedPoolIntegration -v ./internal/registry` | PASS: PostgreSQL dialector, GORM/registry pool pointer equality, configured limit, read-only `SELECT 1`, matching pool metric, canceled readiness, repeated close, zero remaining pool connections and closed-pool failure |
| Focused race checks with live integration enabled for infrastructure, registry and routes | PASS |
| Actual root `go run main.go server` | Connected to PostgreSQL on 5433; served HTTP on 8099 |
| Live `/health`, `/ready`, metrics, `/docs`, `/swagger/doc.json` | Exact existing 200 probe envelopes and request IDs; existing pool metrics; Scalar HTML/spec HTTP 200 with health/readiness paths |
| SIGTERM and captured output checks | `application.server.shutdown_completed`; configured PostgreSQL password/JWT absent from captured logs |
| Unused port, wrong password, nonexistent database, invalid port and invalid SSL | Nonzero startup failure before server-start event; safe refusal/authentication/database/validation diagnostics |
| Isolated application TCP proxy outage/recovery | Readiness 200 → existing 503 `SERVICE_NOT_READY` → 200; health remained 200; PostgreSQL never stopped |
| Isolated stalled transport | Readiness 503 in 2.00s; startup with 1s configured timeout failed safely in 1.04s |
| All four valid legacy migration commands against stalled DB | Blocked before DB access, including `force 1` |
| Read-only database follow-up | Business-table count 0; no `schema_migrations`; no remaining test application connections |
| Secret/preservation checks | Configured password/JWT absent from tracked/nonignored files; PostgreSQL initialization values and unrelated settings retained |

Runtime assertions/results are recorded in secret-free `/private/tmp/m4a-runtime-results.json`; their implementation was inspected in `/private/tmp/m4a-runtime-check.py`. These are local verification evidence, not new application tooling or committed secrets.

## Findings

No remaining actionable finding within the approved scope. No fix iteration is required. The preserved MySQL migration dependency is intentional deferred infrastructure, not an active runtime connector.

## Acceptance Criteria Assessment

| Mandatory review area | Assessment | Evidence |
|---|---|---|
| 1. PostgreSQL driver correctness | **PASS** | Official dialector/pgx, actual PostgreSQL startup and GORM read-only query |
| 2. Configuration precedence | **PASS** | Existing process > `.env` > JSON > defaults sequence retained; focused precedence/validation tests; no `POSTGRES_*` fallback |
| 3. DSN security | **PASS** | Structured URL/userinfo/path/query and IPv6 encoding; special-character tests; sanitized parse/startup errors; secret checks |
| 4. Connection pool preservation | **PASS** | One production pool; pointer equality, limits, shared metrics/readiness and cleanup verified |
| 5. Startup connectivity | **PASS** | Actual success plus refusal/authentication/missing-database/validation/timeout failures before HTTP startup |
| 6. Health/Readiness | **PASS** | Exact existing contracts; isolated outage/recovery and 2-second deadline; health independent of DB |
| 7. Metrics | **PASS** | Existing six names/labels preserved; HELP text adapted; same-pool metric test and live scrape |
| 8. Graceful shutdown | **PASS** | Existing lifecycle/ownership unchanged; SIGTERM completion, double-close and connection cleanup evidence |
| 9. Migration isolation | **PASS** | Guard before loader/runner; all four valid commands blocked; no schema/version table created |
| 10. Dependency hygiene | **PASS** | Scoped official driver/pgx additions, unused GORM MySQL removal, necessary deferred MySQL SQL dependency retained; build/vet/tests pass |
| 11. Documentation accuracy | **PASS** | Bilingual quick starts/examples/references align on ports, precedence, TLS, pool, query-context limitation and M4B migration boundary |
| 12. Verification evidence | **PASS** | Static and attributed executed evidence cover positive, negative, timeout, lifecycle, preservation and security behavior without schema mutation |

## Limits and Next Action

PASS applies to local development M4A, not production deployment, business-query correctness or M4B migrations. Live TLS/certificate deployment was not tested; local Docker explicitly uses `sslMode=disable`. Future queries still require caller contexts: no global socket read/write deadline or statement timeout is introduced. Scalar HTML/spec availability was checked; browser rendering was not repeated because UI code and generated metadata are unchanged. The exact historical MySQL error cannot be reconstructed from its deliberately generic diagnostic. Scoped secret and cleanup checks demonstrate these exercised paths, not an exhaustive absence-of-leaks proof.

Close M4A-01 as **PASS**. Preserve historical reviews and defer PostgreSQL migration implementation to a separately approved M4B input.
