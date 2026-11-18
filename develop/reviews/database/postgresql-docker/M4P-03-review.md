# M4P-03 — Review

- **Feature:** `database/postgresql-docker`
- **Iteration:** `M4P-03`
- **Input:** [M4P-03 fix local port](../../../inputs/database/postgresql-docker/M4P-03-fix-local-port.md)
- **Original scope:** [M4P-01 implementation](../../../inputs/database/postgresql-docker/M4P-01-implementation.md)
- **Previous reviews:** [M4P-01 FAIL](M4P-01-review.md), [M4P-02 FAIL](M4P-02-review.md)
- **Date:** 13 November 2026 (Asia/Makassar)
- **Verdict:** **PASS for M4P PostgreSQL Docker development setup**
- **Review method:** `/root/m4p_review` independently reviewed the final input, implementation diff, Compose, documentation and preservation metadata in a separate context. Actual Docker/SQL/lifecycle evidence below was supplied by the implementing parent agent and is explicitly attributed; the reviewer did not repeat Docker mutations or lifecycle checks.

## Scope and actual changes

Root `compose.yaml` provides exactly one official `postgres:17` development service with required environment credentials, localhost publication, a named data volume, `unless-stopped` and an escaped container-variable healthcheck. Go remains local with its existing MySQL implementation until M4A.

The complete tracked scope is `compose.yaml`, `.env.example`, bilingual READMEs, existing configuration/CLI/operations/architecture references, three matching inputs and three reviews. Ignored local `.env` adds PostgreSQL settings while preserving its original contents; M4P-03 changes only the newly added host port to 5433. Compose and `.env.example` retain default 5432. Earlier FAIL reviews remain intact.

No backend service, UI, Redis, custom image, dependency, generated file, migration, business schema, production configuration or backend Dockerfile change was introduced. The reviewer edited only review artifacts.

## Findings resolved

| Finding | Resolution and evidence |
|---|---|
| F1: loopback TCP mislabeled as password verification | CLI now uses service hostname `postgres`, explains loopback trust and requires a wrong-password negative control. Parent observed bridge-address correct-password success and wrong-password rejection; authentication rules show loopback trust and generic host SCRAM |
| F2: host port 5432 occupied | Parent's host-context socket probe observed address-in-use on 5432 and successful bind on 5433. Only ignored local `POSTGRES_PORT` changed. Actual healthy publication is `127.0.0.1:5433 -> 5432/tcp`; no unrelated service was stopped |

No remaining actionable finding was identified within the approved scope.

## Independent static verification

| Check performed by reviewer | Actual result |
|---|---|
| Read final M4P-03 input, complete final tracked diff and root Compose | Scope matches authorization; fixed default, localhost binding, named volume, healthcheck and restart baseline retained |
| `docker compose config -q` | Exit 0; no resolved configuration printed |
| `git diff --check` | Exit 0 |
| Baseline SHA-256 comparison using `/private/tmp/m4p-before.json` | All 41 protected files matched: main, CLI/backend/config packages, generated docs, dependencies, Dockerfile, production Compose, ignore rules and example JSON |
| Local port/metadata-only checks | `POSTGRES_PORT` is 5433; `.env` mode is 0600 and it is absent from Git's cached/nonignored file set; no credential value printed |
| README/reference review | English/Indonesian quick starts aligned; docs distinguish default 5432 from configurable host port, first initialization, shell precedence, persistent storage and unchanged backend MySQL/readiness |

The reviewer requested no Docker escalation or runtime operation during this final review. No interrupted approval or uncompleted command is used as evidence.

## Actual runtime evidence supplied by implementer

| Command/check | Observed result |
|---|---|
| Host-context socket bind probe | 5432 failed with Errno 48/address in use; 5433 bind succeeded |
| `docker compose up -d --wait --wait-timeout 90` | Exit 0; service healthy |
| `docker compose ps` | `hris-core-service-postgres-1`, image `postgres:17`, running/healthy, only `127.0.0.1:5433->5432/tcp` published |
| `docker compose ls` | `hris-core-service` running (1), using this root `compose.yaml`; Compose/Desktop daemon grouping confirmed |
| Focused safe container inspection | Restart `unless-stopped`; named mount `hris-core-service_postgres_data` at `/var/lib/postgresql/data`; actual healthcheck uses single `$POSTGRES_USER` and `$POSTGRES_DB` |
| Read-only SQL | `SELECT 1` returned 1; database `mapatu_dev`, user `mapatu`, PostgreSQL `17.12 (Debian 17.12-1.pgdg13+1)` |
| Safe authentication-rule inspection | Loopback host rules use `trust`; generic host rule uses `scram-sha-256` |
| Correct-password control using `psql -h postgres` | Returned 1; server/client bridge address `172.23.0.2` |
| Wrong-password control on identical path | `FATAL: password authentication failed`, exit 2; actual password not printed |
| `docker compose stop` | Service stopped, `Exited (0)` |
| Restart and bounded health polling | Plain `docker compose start` succeeded; container returned healthy |
| Post-start persistence SQL and focused mount check | `SELECT 1`, configured user/database succeeded; same named volume and system identifier `7707569386355830822` before/after stop/start; business-table count 0 |
| Final host endpoint handshake | `127.0.0.1:5433` responded with PostgreSQL SASL authentication code 10; no password sent |
| Unrelated resource observation | Existing `merajutcerita` remained healthy on port 15432; unrelated resources untouched |
| Environment/backend preservation checks | Original `.env` byte-prefix hash matched baseline; ignored/untracked, mode 0600; generated password absent from cached/nonignored files; protected source/dependency/generated/production files unchanged |
| Final `docker compose config -q` / `git diff --check` | Both passed |

One verification attempt used unsupported `docker compose start --wait` with Compose 2.31, so it did not restart the stopped service; an initial host handshake consequently received connection refused. The implementer immediately corrected the command to plain `docker compose start`, polled safely until healthy and completed the successful post-start SQL/persistence/host checks above. The failed attempt is not counted as successful evidence, and the published CLI correctly documents plain `start`.

An earlier combined host-audit approval timed out. Narrower Docker inspection and a minimal host probe subsequently succeeded; the timeout is not used as evidence and leaves no unresolved blocker.

## Full acceptance assessment

| Review area / acceptance criterion | Assessment | Evidence |
|---|---|---|
| 1. Compose structure and scope | **PASS** | One PostgreSQL service, no backend integration/custom image/additional service |
| 2. Official PostgreSQL 17 | **PASS** | `postgres:17` declared/pulled/running; SQL reports 17.12 |
| 3. Environment and secret security | **PASS** | Required nonempty credential interpolation; ignored local `.env`; original prefix preserved; safe examples; no secret output or tracked-password match |
| 4. Localhost-only configurable host port | **PASS** | Default 5432 retained; local override 5433 resolves actual conflict; publication and host PostgreSQL handshake verified |
| 5. Named persistent data | **PASS** | Correct named mount/data path; same volume and cluster identifier after stop/start |
| 6. Container-variable healthcheck | **PASS** | Escaping correct in source and actual container command; healthy before/after lifecycle |
| 7. Restart/lifecycle semantics | **PASS** | Actual `unless-stopped`; stop/plain-start works; docs accurately describe `up`, `start`, `stop`, `down`, destructive `down -v`, project identity and first-only credentials |
| 8. Actual SQL/authentication operation | **PASS** | Read-only SQL/user/database/version; bridge positive/negative password controls; host SASL response |
| 9. Accurate synchronized docs | **PASS** | Bilingual quick starts, configurable port, Compose/Go separation, precedence/storage/credential/lifecycle/authentication instructions align with implementation |
| 10. Backend/out-of-scope preservation | **PASS** | Independent protected fingerprints and parent preservation evidence; no source/dependency/generated/migration/production/business change |

## Limits and next action

Docker Desktop grouping is verified through daemon/Compose metadata; UI button clicks were not performed. Restart policy is verified from configuration and live metadata; an actual machine/daemon restart was not performed. Non-destructive stop/start proves continuity of the existing cluster; destructive removal/reset or container recreation was deliberately not used as a test. These limits do not leave an unmet development-setup criterion.

No Go build/tests were run because no Go/config/dependency behavior changed. No migration, business data write, volume deletion/reset, backend integration or production operation was performed. A healthy PostgreSQL container does not alter backend `/ready`, which still depends on MySQL.

Close M4P for the approved development scope with history preserved: **M4P-01 FAIL → M4P-02 FAIL → M4P-03 PASS**. Local PostgreSQL is available at `127.0.0.1:5433`; credentials remain only in ignored `.env`. Defer PostgreSQL backend integration to M4A. No further M4P fix input is warranted.
