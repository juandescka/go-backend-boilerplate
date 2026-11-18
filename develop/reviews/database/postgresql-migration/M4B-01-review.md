# M4B-01 — Review

- **Feature:** `database/postgresql-migration`
- **Iteration:** `M4B-01`
- **Input:** [M4B-01 implementation](../../../inputs/database/postgresql-migration/M4B-01-implementation.md)
- **Dependencies:** [M4A-01 PASS](../postgresql-backend/M4A-01-review.md), [M4P-03 PASS](../postgresql-docker/M4P-03-review.md).
- **Date:** 16 November 2026 (Asia/Makassar)
- **Verdict:** **PASS for the approved M4B PostgreSQL Migration Foundation**
- **Review method:** Separate-context reviewer `/root/m4b_review` independently inspected guidance, input, implementation, tests, documentation, dependency source and preservation fingerprints. Runtime/build/test evidence below was executed by the implementing parent and is explicitly attributed. The reviewer edited only this review; no database operation, migration, build, test or Docker operation was repeated by the reviewer.

## Scope Reviewed and Actual Changes

The existing [Cobra migration commands](../../../../cmd/migrate.go) now load the existing M4A configuration and create the adapted [runner](../../../../internal/infrastructure/migration/runner.go). Command names remain `up`, `down`, `version` and `force`; `down` still calls `Steps(-1)` rather than rollback-all. No HTTP registry or application pool is initialized by these commands, and server startup never invokes migrations.

The runner uses official golang-migrate v4.18.3 `database/postgres.WithConnection`, supplied with one owned pgx SQL pool and reserved connection. The [configuration](../../../../internal/infrastructure/migration/config.go) constructs escaped PostgreSQL/file URLs from effective M4A settings and activates `migrations/postgres`. That directory contains only `.gitkeep`; inspected empty `migrations/mysql` remains present. No business SQL or permanent dummy migration is added.

Source opening precedes database initialization. Connection establishment and migration-file execution have configured context bounds; session `statement_timeout` and `lock_timeout` are applied before official metadata initialization. Library lock timeout is one second higher. Version uses the official driver's signed state, preserving the distinction between none (-1), actual zero, and dirty nil state. Errors expose fixed categories/validated SQLSTATE rather than SQL, URIs, credentials or raw driver details. Only the bare no-change sentinel becomes successful no-change output; a composite error containing an unlock failure remains an error.

Source, reserved connection and pool have explicit ownership and cleanup on initialization failures and completion. Repeated Close is safe. Only an already discarded connection's `sql.ErrConnDone` is benign; other cleanup failures are sanitized and affect command exit status. This matches official `WithConnection`, whose driver-owned DB is nil and whose Close otherwise only closes the supplied connection. No adapter, framework or new package layer is introduced.

Changes also include focused unit/opt-in disposable integration tests, scoped dependency cleanup, this input/review pair, bilingual README summaries and existing architecture/configuration/operations/CLI references. Unused MySQL SQL driver and its unique edwards25519 dependency are removed; `lib/pq` remains indirect through the official PostgreSQL migration driver. pgx, GORM, golang-migrate and Go versions remain unchanged.

## Independent Verification Evidence

| Check actually performed by reviewer | Result |
|---|---|
| Read bilingual AGENTS, actual `develop/README-develop.md`, approved input and latest M4A/M4P reviews | Scope and workflow preserved |
| Inspect final runner/config/commands, callers, unit/integration tests and current references | Implementation and documentation agree; no server migration call or active MySQL driver remains |
| Inspect exact cached migrate v4.18.3 postgres/file/iofs/migrate source and pgx v5.7.6 execution path | Confirmed metadata side effects, ownership/error behavior, background advisory lock, signed nil/dirty state, Steps(-1), Force >= -1, filename filtering, missing-direction behavior and compound SQL transport |
| Compare SHA-256 with `/private/tmp/m4b-before.json` | 83 baseline files: 71 identical; exactly 12 changed files match migration/code/dependency/documentation scope |
| Inspect protected baseline identities | Runtime connector, configuration/validation, registry/server, readiness/health/metrics/HTTP contracts, Docker/production files, generated Swagger, API standard and all prior history unchanged |
| Inspect ignored local file metadata and Git state | `.env` and `config.json` byte-identical, mode 0600, ignored and untracked |
| Inspect migration directories | PostgreSQL source contains only `.gitkeep`; legacy MySQL directory remains empty |
| `git diff --check` | Exit 0 |
| Search current README/reference prose for obsolete disabled-M4B claims | None found |

The integration harness was independently read, not executed by the reviewer. It creates a random uniquely named database only after checking creation privileges, records ownership only after successful creation, quotes the exact identifier and drops only that owned database. SQL fixtures live in temporary directories. Administrative/development database connections are not passed to the migration runner. CLI secrets are supplied through subprocess environment without being persisted in temporary JSON or printed.

## Actual Execution Evidence Supplied by Implementer

| Command/check | Observed result reported by implementer |
|---|---|
| Fresh M4A `TestPostgresSharedPoolIntegration` gate | PASS against existing PostgreSQL at host port 5433: shared GORM/pool identity, readiness, metrics and cleanup |
| `go build ./...`, `go vet ./...`, `go test ./...`, using `GOCACHE=/private/tmp/hris-m4b-go-cache` | All PASS |
| Five actual `go run main.go` help invocations: `migrate --help` and each of `migrate up/down/version/force --help` | All PASS; no database access needed |
| `M4B_INTEGRATION=1 M4B_CLI_BINARY=/private/tmp/hris-m4b-backend go test -race -count=1 -run TestPostgresMigrationIntegration -v ./internal/infrastructure/migration` | PASS, 5.25s; exercised current binary, disposable fixtures and concurrent migrators |
| Empty source/fresh state/source preflight | Ignored invalid filename; fresh version none; metadata initialized; empty Up/fresh Down fail truthfully; missing source fails before metadata/session creation |
| Ordered lifecycle and no-change | Two compound SQL versions apply in order; repeated Up returns no change; one Down removes only version 2; second Down reaches none |
| Missing source versions/directions | Forced version 42 absent from both directions fails Up/Down; one-sided source's metadata-only missing direction verified and documented, then explicitly reconciled in disposable DB |
| Dirty behavior and recovery | Controlled failing Up rolls back its compound SQL, persists dirty version 1 and blocks Up/Down; no automatic force; inspected manual recovery succeeds; failed first Down preserves -1/dirty and blocks Up |
| Real Cobra binary | Up/repeat-Up/one-step Down/version/force 0/force -- -1 all exercised; actual zero differs from none; dirty nil output retained; force changes metadata without applying SQL |
| Timeout/locking | Configured file-execution timeout; fresh runner observes persisted dirty state afterward; initialization advisory-lock wait bounded; two concurrent migrators yield one application and one no-change |
| Failure/cleanup | Wrong password, missing database, refusal and stalled handshake fail safely; success/error paths and repeated Close leave no migration sessions |
| Disposable ownership cleanup | Final `m4b_verify_f765c57af3876f95` removed; three earlier owned verification databases also removed; read-only follow-up finds none remaining |
| Limited HTTP preservation | Current binary with temporary config on port 54113 returns exact existing health/ready 200 envelopes; SIGTERM exits 0; no configured secrets in captured logs; temporary config removed |
| Existing development target follow-up | Read-only `mapatu_dev` check: no `schema_migrations`, zero business tables, zero migration connections |
| Secret/preservation follow-up | No actual configured secrets in cached/nonignored files; protected/local configuration fingerprints preserved |

An initial live verification run failed after a timeout because pgx had discarded the reserved connection: same-runner Version could not query it, and official postgres.Close formatted `sql.ErrConnDone` into an untyped error. Its owned database was removed. Before this review, cleanup was corrected to close directly owned resources, preserve genuine cleanup failures and accept only the typed already-closed sentinel; timeout-state verification now opens a fresh runner, matching separate CLI invocations. Final live/race checks passed after this correction and after guarding composite no-change/unlock errors. The initial attempt is not counted as PASS, and no historical FAIL review was overwritten.

The ordinary root HTTP port 8099 was occupied; the existing process was not touched. The limited regression used an isolated temporary port instead. No Docker mutation or lifecycle reset was needed for M4B verification.

## Findings

No remaining actionable finding within the approved scope. Empty-source failures and missing-direction metadata-only behavior are explicit golang-migrate v4.18.3 semantics, accurately documented and verified; they are not represented as a working business baseline schema. Complete up/down pairs remain a migration-review convention rather than an invented validator/framework.

## Acceptance Criteria Assessment

| Mandatory review area | Assessment | Evidence |
|---|---|---|
| 1. PostgreSQL migration driver | **PASS** | Official postgres driver with supplied pgx connection; real lifecycle and advisory-lock tests |
| 2. Configuration integration | **PASS** | Existing loader/validation and process > .env > JSON > defaults retained; same DB_* identity; no POSTGRES_* fallback |
| 3. DSN and secret security | **PASS** | Structured escaped URL, IPv6/special-character tests, explicit SSL/timeout, PGSERVICE rejection, safe errors and captured-output secret checks |
| 4. Migration directory | **PASS** | Active PostgreSQL `.gitkeep` only; empty MySQL directory preserved; no permanent fixture/business SQL |
| 5. Up behavior | **PASS** | Ordered pending-only compound fixtures, repeat no-change, dirty refusal and absent-version failure |
| 6. Down behavior | **PASS** | Exactly Steps(-1), two observed successive single rollbacks, fresh-state failure and dirty refusal |
| 7. Version behavior | **PASS** | Fresh none, current clean/dirty, actual zero and signed dirty nil preserved in runner and CLI |
| 8. Force behavior | **PASS** | Invalid/noninteger/overflow/<-1 rejected before configuration; 0/-1/manual recovery tested; metadata-only effect verified |
| 9. Dirty-state handling | **PASS** | Controlled Up/first-Down failures, persistent dirty, subsequent refusal and inspected explicit recovery; no automatic force |
| 10. Empty/malformed source | **PASS** | Source-first missing/duplicate failures, ignored unparseable filename, truthful empty Up/Down and documented one-sided-pair limitation |
| 11. Resource cleanup and locking | **PASS** | Explicit owned resources, repeated Close, timeout/discarded-connection handling, bounded initialization lock and concurrent serialization; no sessions left |
| 12. Dependency hygiene | **PASS** | Scoped MySQL removal/pq indirect dependency; existing runtime/version dependencies retained; build/vet/tests pass |
| 13. Documentation accuracy | **PASS** | Bilingual summaries/current references explain active commands, side effects, pairs, transactions, timeout, dirty/force and empty-source limits |
| 14. Verification evidence | **PASS** | Independent source/preservation review plus attributed full checks, actual CLI/disposable/race/negative/cleanup and limited HTTP evidence |
| 15. M4A runtime preservation | **PASS** | Runtime/config/HTTP files and local secrets identical; fresh shared-pool gate and isolated HTTP smoke; mapatu_dev remains without migration/business schema |

## Limits and Next Action

PASS covers the local development migration foundation, not business SQL, data conversion, production deployment or TLS/certificate operation. Live TLS modes were not exercised against a certificate-enabled server. No rollback/force or metadata initialization was run on `mapatu_dev`. Server timeout settings bound PostgreSQL work/lock waits, while callers still must account for transport failures; this is not a whole-run deadline or a general network-fault guarantee. A canceled runner may require disposal and reopening, as documented.

No business migration, seeder, migration pairing validator, automatic startup migration, production operation or Docker change is included. Existing baseline emptiness is intentional. No mandatory blocker remains and no fix iteration is required. Close **M4B-01 as PASS**; future reviewed business migration SQL needs its own approved scope.
