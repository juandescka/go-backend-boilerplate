# M4P-02 — Review

- **Feature:** `database/postgresql-docker`
- **Iteration:** `M4P-02`
- **Input:** [M4P-02 fix auth verification](../../../inputs/database/postgresql-docker/M4P-02-fix-auth-verification.md)
- **Original scope:** [M4P-01 implementation](../../../inputs/database/postgresql-docker/M4P-01-implementation.md)
- **Previous review:** [M4P-01 FAIL](M4P-01-review.md)
- **Date:** 13 November 2026 (Asia/Makassar)
- **Verdict:** **FAIL**
- **Review method:** Independent separate-context reviewer `/root/m4p_review`; current input, CLI changes and unchanged Compose baseline inspected directly. Runtime failure below is actual implementer-supplied command evidence, not a command executed by this reviewer.

## Scope and actual changes

The M4P-02 fix changes only the PostgreSQL authentication explanation/commands in `documentation/cli.md`, adds the corresponding fix input and this review, and preserves the M4P-01 FAIL record. The original M4P Compose/example/README/reference-document changes remain in scope and unchanged by this fix. No implementation file was edited by the reviewer.

The CLI now explains that socket checks, `pg_isready` and container loopback TCP do not establish password verification. Both authentication controls use service hostname `postgres`, container port 5432 and identical user/database arguments. The positive control passes the configured password through the container environment without printing it; the negative control uses a public non-secret sentinel and explicitly requires a password error and nonzero exit.

## Findings

### F1 — Corrected statically; live authentication controls pending

The misleading loopback test in M4P-01 is corrected at `documentation/cli.md:171–180`. Using the service DNS name selects the container bridge address rather than its loopback path. A live successful positive control, failed wrong-password control and safe authentication-rule inspection remain required; the corrected command text alone is not runtime authentication evidence.

### F2 — Actual startup fails because host port 5432 is occupied

The implementer ran `docker compose up -d`. Docker pulled `postgres:17`, created network `hris-core-service_default`, volume `hris-core-service_postgres_data` and container `hris-core-service-postgres-1`, then failed to start the service with a port-binding error: `127.0.0.1:5432` was already in use. The initial sandboxed listener check had produced no output, which did not establish that Docker could bind the port. No healthy service, SQL result or persistence continuity has yet been demonstrated.

This is an observed runtime acceptance failure. The approved baseline already supports an alternative host port through ignored local `.env`. After a host-context free-port probe, change only local `POSTGRES_PORT` to the verified free port; retain Compose's default 5432 and its fixed localhost binding. Do not stop unrelated projects, change existing credentials, remove volumes or modify the image/backend. Reuse the already-created M4P project and volume, then perform the full runtime acceptance checks.

## Verification evidence

| Check | Result / provenance |
|---|---|
| Read M4P-02 input and current CLI diff | Reviewer confirmed the fix matches F1 and adds the required negative control |
| Read root `compose.yaml` | Original baseline preserved, including default host port 5432 and environment override |
| `docker compose config -q` | Reviewer ran; exit 0, no resolved model printed |
| `git diff --check` | Reviewer ran; exit 0 |
| `git diff --quiet HEAD -- main.go cmd config internal docs go.mod go.sum config.example.json compose.production.yml Dockerfile` | Reviewer ran; exit 0 |
| `docker compose up -d` | Implementer supplied actual failure: image pulled and resources created, then host port binding failed (F2) |
| Host-context listener/free-port audit | In progress at review time; no free alternative port is asserted here |
| Healthy state, actual published binding/mount, SQL and version | **NOT VERIFIED** — startup failed |
| Safe `pg_hba_file_rules` inspection and positive/negative password controls | **NOT VERIFIED** — startup failed |
| Stop/start and same volume/database system identifier | **NOT VERIFIED** — startup failed |
| Local `.env` previous-content preservation | No values read by reviewer; final implementer preservation evidence pending |
| Go build/tests, migrations, backend runtime | **NOT RUN** — no Go change; migration/backend integration outside scope |

No secret, full Compose resolution or full Docker inspect output was displayed or recorded by the reviewer. The reviewer ran no Docker mutation or lifecycle command.

## Full M4P acceptance assessment

| Area / criterion | Assessment | Evidence and limit |
|---|---|---|
| 1. Single-service development Compose | **PASS (static)** | Only `postgres`; no backend/UI/Redis/custom image or integration |
| 2. Official PostgreSQL 17 image | **PASS (static/pull)** | `postgres:17` preserved; image pulled according to actual startup evidence; SQL version pending |
| 3. Environment and secrets | **PASS (static)** | Required nonempty credentials and ignored `.env` approach retained; no credential values in review; final preservation evidence pending |
| 4. Localhost port and usable endpoint | **FAIL (runtime)** | Static binding correct; Docker cannot bind current host port 5432 (F2) |
| 5. Named persistent storage | **PASS (static); runtime continuity NOT VERIFIED** | Correct mount declaration; project volume created; live mount and continuity pending |
| 6. Escaped healthcheck / actual healthy status | **PASS (static); runtime NOT VERIFIED** | Baseline escaped container variables retained; service never reached healthy |
| 7. Restart and lifecycle | **PASS (static); runtime NOT VERIFIED** | `unless-stopped` and safe lifecycle docs retained; actual stop/start pending |
| 8. SQL and password authentication | **NOT VERIFIED** | F1 command fixed; no successful runtime controls because of F2 |
| 9. Accurate, aligned documentation | **PASS (static)** | F1 corrected and negative control documented; bilingual READMEs and other M4P references remain aligned |
| 10. Backend/out-of-scope preservation | **PASS (static)** | Source/config/dependency/generated/production/Dockerfile preservation diff exit 0 |

M4P-02's documentation correction is statically accepted, but its required original runtime plan is not satisfied. The observed startup failure F2 warrants **FAIL**. No PASS result is inferred for pending runtime criteria.

## Next action

Preserve both FAIL reviews. Create M4P-03 input referencing F2, complete a host-context free-port probe, change only ignored local `POSTGRES_PORT` and reconcile this same project. Verify actual healthy PostgreSQL 17, localhost-only published endpoint, named mount, read-only SQL/user/database/version, safe host authentication rules, correct-password success, wrong-password rejection and stop/start persistence with unchanged volume/database system identifier. Record environment/backend preservation safely, then request an independent M4P-03 review. No additional feature or baseline change is required.
