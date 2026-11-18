# M4C-01 — PostgreSQL Database Foundation Final Verification

- **Feature:** `database/foundation-verification`
- **Iteration:** `M4C-01`
- **Mode:** Audit + integration verification + targeted fix only if warranted
- **Approval:** User's M4C master prompt and instruction to continue, 16 November 2026.

## Objective and scope

Verify the combined local PostgreSQL foundation before M5A without creating business schema or changing working architecture. PASS may require no source or reference-document edits. Add this input and an independent paired review; preserve all current M4B work, including uncommitted files.

## Dependency review

| Milestone | Gate evidence | Assessment |
|---|---|---|
| M4P | [M4P-03 PASS](../../../reviews/database/postgresql-docker/M4P-03-review.md); root Compose image postgres:17, localhost port, named volume, healthcheck | VERIFY current daemon/model; reuse unchanged lifecycle evidence |
| M4A | [M4A-01 PASS](../../../reviews/database/postgresql-backend/M4A-01-review.md); actual GORM postgres/pgx pool, loader, startup, readiness, metrics, close ownership | VERIFY current integration; preserve contracts |
| M4B | [M4B-01 PASS](../../../reviews/database/postgresql-migration/M4B-01-review.md); active official postgres migrator, temporary fixtures/disposable lifecycle, correct signed state and one-step Down | VERIFY current integration; preserve existing implementation |

M4B is present in the working tree but not committed; commit absence does not invalidate its review. `/private/tmp/m4c-before.json` records every current cached/nonignored/untracked file and ignored local `.env`/`config.json` by hash. The initial Git status is also recorded separately. Historical reviews are evidence for their then-current code, not substitutes for current source inspection.

## Existing implementation assessment

| Component | Actual implementation | Classification |
|---|---|---|
| Docker PostgreSQL | Official 17; POSTGRES_* interpolation; localhost-only; named data volume | KEEP / VERIFY |
| Configuration | Existing godotenv, required JSON, process > .env > JSON > defaults; explicit DB_*; no POSTGRES_* fallback | KEEP / VERIFY actual equality |
| Runtime | One pgx SQL pool attached to GORM postgres, startup PingContext, existing pool limits | KEEP / VERIFY |
| Readiness/metrics/shutdown | Same runtime pool; existing 2-second request deadline, stats metrics and close-once registry | KEEP / VERIFY |
| Migration CLI | Official golang-migrate postgres.WithConnection using a separate CLI-owned pgx pool/connection; source-first validation | KEEP / VERIFY |
| Migration state | Steps(-1), metadata-only explicit force >= -1, signed none/zero/dirty nil, safe errors | KEEP / VERIFY |
| Production source | migrations/postgres contains only .gitkeep; legacy mysql directory empty and retained | KEEP; business migrations DEFER |
| Security/docs | Escaped URLs, fixed diagnostics, ignored local files, explicit development TLS choice, current bilingual/reference docs | VERIFY; DOCUMENT/FIX only demonstrated discrepancies |

Repository guidance, current examples/references/API standard, source and previous reviews are read together with the unchanged context from M4P/M4A/M4B. The actual workflow file is `develop/README-develop.md`. No new framework, dependency upgrade, Go test or business feature is planned.

## Risk matrix and resource safety

| Risk | Control |
|---|---|
| Mutating mapatu_dev or another project | Read-only checks only; no migration CLI state command on that database, including version |
| Destructive migration fixtures | Reuse explicit opt-in test that creates a random owned disposable database and drops only that exact successfully created name |
| Interrupting an existing backend/DB | Never stop PostgreSQL or port8099 process; use isolated temporary config/module workspace and free HTTP port |
| Credential leakage | Capture config/model/logs without printing full content; compare secrets in memory; temporary configs mode0600 with secrets passed through environment where needed |
| Resource leaks | Own and terminate only verification processes/proxy; assert migrator sessions and owned disposable DB cleanup; retain named volume |
| Unrelated/uncommitted changes lost | Preserve complete pre-M4C hashes and Git baseline, including current M4B artifacts |
| Unnecessary verification/edits | Reuse prior valid Docker lifecycle, race/Scalar/production-limit evidence; no source/reference changes without a verified issue |

## Verification strategy

1. Audit dependency reviews/current source and configuration; compare actual effective backend/migrator properties with resolved Compose in memory, without displaying credentials.
2. Run `docker compose config -q`, safe daemon status/binding/mount checks and read-only PostgreSQL SQL. Do not repeat stop/start or recreate containers.
3. Run build, vet, fresh full Go tests and existing read-only M4A integration. No new race run unless a concurrency-relevant implementation changes; prior M4B race evidence remains applicable to unchanged code.
4. Verify actual `go run main.go server` through an isolated temporary module/config workspace if the configured HTTP port is occupied. Confirm exact health/ready envelopes, metrics, graceful shutdown and secret-safe output. Verify controlled fail-fast without stopping DB.
5. Use a verification-only TCP proxy to interrupt/stall only the test application's DB path: readiness 200 → 503 → 200, health stays 200, bounded timeout and recovery. If unnecessary/unsafe, reuse current applicable M1/M4A tests and explicitly classify reused evidence.
6. Build a current CLI binary and rerun the existing disposable migration integration test for ordered two-version up, one-step down, version/force, dirty/empty-source/locking/cleanup. Help commands need no DB; never run migration commands against mapatu_dev.
7. Recheck schema/metadata, exact owned resource cleanup, all baseline fingerprints, secret safety, dependencies and documentation links; obtain separate-context final review.

## Documentation impact

Review README EN/ID and existing architecture/configuration/operations/CLI/API-standard references against current behavior. They already describe PostgreSQL, separate runtime/CLI pools, configurable5433/default5432, precedence, explicit SSL, no startup migrations, metadata side effects, empty-source limitations, one-step down and manual force. Change none if accurate. Keep evidence/history in develop rather than duplicate technical documentation.

## Acceptance criteria

- Prior milestone PASS gates and actual integrated runtime/migration/configuration verified.
- Docker remains healthy/local-only with the same named volume; no unrelated resource changes.
- Startup success/fail-fast, exact health/readiness200/503, pool/metrics/cleanup and safe cancellation evidence sufficient and honestly attributed.
- Migration lifecycle/dirty/force verified only on disposable owned DB; no permanent fixture or automatic migration.
- No secret exposure, configuration overwrite, business schema, reset, production operation, M5 work or unnecessary abstraction.
- All relevant checks pass, affected references accurate, pre-existing work/history preserved, and final review PASS; use a separate-context reviewer if available, otherwise explicitly record structured self-review as permitted by the user.

## Review availability update

The separate-context reviewer completed an initial source/prerequisite audit without an actionable finding, then its turn failed because of a usage limit before final review. Do not claim a completed independent M4C review. Continue the authorized checks and use the user's explicit structured self-review fallback; keep independent initial observations distinct from the final self-review and actual execution evidence.
