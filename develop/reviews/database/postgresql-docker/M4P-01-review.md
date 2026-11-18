# M4P-01 — Review

- **Feature:** `database/postgresql-docker`
- **Iteration:** `M4P-01`
- **Input:** [M4P-01 implementation](../../../inputs/database/postgresql-docker/M4P-01-implementation.md)
- **Date:** 13 November 2026 (Asia/Makassar)
- **Verdict:** **FAIL**
- **Review method:** Independent read-only review by `/root/m4p_review` in a separate context. Implementation files were not edited by the reviewer.

## Scope reviewed

Read `AGENTS.md`, `AGENTS.id.md`, `develop/README-develop.md`, the approved input, root `compose.yaml`, the complete tracked implementation diff, existing MySQL loader/connector/migration references and prior review conventions. Checked the official image entrypoint and PostgreSQL/Docker documentation for initialization, interpolation and authentication behavior.

This review evaluates the current implementation and documentation. Actual container startup, authentication and persistence checks are pending the implementer's image pull; no runtime PASS is inferred from static inspection.

## Actual changes

| File | Change |
|---|---|
| `compose.yaml` | Adds only official `postgres:17`, required interpolated credentials, localhost port, named data volume, restart policy and healthcheck |
| `.env.example` | Adds four safe PostgreSQL examples without replacing MySQL settings |
| `README.md`, `README.id.md` | Adds aligned English/Indonesian PostgreSQL quick starts and reference links |
| `documentation/configuration.md` | Documents Compose variables, precedence, first initialization, storage and backend boundary |
| `documentation/cli.md` | Documents validation, lifecycle, SQL and an incorrectly described TCP password check |
| `documentation/operations.md` | Documents independent PostgreSQL lifecycle and persistence |
| `documentation/architecture.md` | Documents local Go/MySQL and separate PostgreSQL development infrastructure |
| `develop/inputs/database/postgresql-docker/M4P-01-implementation.md` | Records approved scope, discovery and verification plan |
| This review | Records the independent assessment and required correction |

Ignored local `.env` contains each of the four PostgreSQL key names once. The reviewer read only key names, not values. Preservation of its pre-existing contents cannot be independently reconstructed from Git; the implementer's pre/post preservation evidence remains to be recorded in a later review.

## Verification evidence

| Check actually performed by reviewer | Result |
|---|---|
| `git status --short` and complete tracked diff inspection | Only the scoped example/docs changes and new Compose/input files were present before this review |
| `docker compose config -q` | Exit 0; no resolved configuration printed |
| `git diff --check` | Exit 0 |
| `git check-ignore .env` and tracked-file lookup | `.env` is ignored and untracked |
| Key-name-only local `.env` parser | All four expected `POSTGRES_*` key names present once; no values printed |
| `git diff --quiet HEAD -- main.go cmd config internal docs go.mod go.sum config.example.json compose.production.yml Dockerfile` | Exit 0; backend source, generated docs, dependencies, production Compose and backend Dockerfile preserved |
| Source inspection of loader/connector/migration references | Driver validation and active database/migration implementation remain MySQL |
| Official image entrypoint, PostgreSQL `initdb`/`pg_isready`, Docker interpolation references | Confirm F1 and static Compose semantics |
| Container startup, health, actual binding/mount inspection | **NOT VERIFIED** — image pull ongoing; reviewer ran no lifecycle operation |
| SQL, password authentication and stop/start persistence | **NOT VERIFIED** — runtime not yet available |
| Go build/tests, backend runtime, migrations | **NOT RUN** — no Go behavior/dependency change; migration execution is outside scope |

No secret, full resolved Compose model, full Docker inspect output or real credential was printed or included in this review. No destructive operation, volume reset, migration, business schema change, backend startup or unrelated Docker operation was performed by the reviewer.

## Findings

### F1 — In-container loopback TCP does not establish password verification

**Location:** `documentation/cli.md:171–174`.

The text calls its `PGPASSWORD=... psql -h 127.0.0.1` command a TCP password authentication test. In the baseline official image, this connection uses the container's own loopback address. PostgreSQL `initdb` defaults local TCP authentication rules to `trust`; the image invokes `initdb` without an authentication override, then appends its generic host password rule after those initial rules. Therefore this command can succeed without validating the configured password. Supplying `PGPASSWORD` alone does not prove that the server requested or checked it.

This is a verification/documentation defect, not a reason to change the approved Compose baseline. Replace the documented authentication target with the service hostname `postgres` so it resolves to the container's bridge address, and confirm the actual non-loopback rule and server address at runtime. Require both successful authentication with the configured password and rejection of an intentionally wrong non-secret password on the same path. Keep password values out of output. Record loopback behavior separately if tested; do not count loopback success as password evidence.

Primary sources: [PostgreSQL 17 initdb authentication defaults](https://www.postgresql.org/docs/17/app-initdb.html), [official PostgreSQL 17 image entrypoint](https://github.com/docker-library/postgres/blob/master/17/bookworm/docker-entrypoint.sh), [official image initialization/authentication notes](https://hub.docker.com/_/postgres). This finding is source-backed; runtime confirmation remains pending.

No other actionable static defect was identified in the approved scope.

## Ten review areas and acceptance assessment

| Area / acceptance criterion | Assessment | Evidence and limit |
|---|---|---|
| 1. Compose structure and development scope | **PASS (static)** | Exactly one `postgres` service; no backend, Redis, UI, custom build, init scripts or integration |
| 2. Official PostgreSQL major version | **PASS (static)** | Exact image reference `postgres:17`; actual pulled image/server version pending |
| 3. Environment and secret handling | **PASS (static)** | Required nonempty user/password/database guards, safe examples, ignored untracked `.env`, no secret output; actual credentials and pre-existing `.env` preservation not independently evaluated |
| 4. Localhost port binding | **PASS (static)** | `127.0.0.1:${POSTGRES_PORT:-5432}:5432`; actual published endpoint and host reachability pending |
| 5. Persistent storage | **PASS (static)** | Named key `postgres_data` mounted at PostgreSQL 17 data path `/var/lib/postgresql/data`; actual mount and continuity pending |
| 6. Healthcheck and readiness meaning | **PASS (static)** | `CMD-SHELL` uses escaped `$$POSTGRES_USER` / `$$POSTGRES_DB`; intervals/timeouts/retries provided; docs distinguish server readiness from password authentication; runtime health pending |
| 7. Restart and lifecycle safety | **PASS (static)** | `unless-stopped`; docs explain `up`, `start`, `stop`, `down`, destructive `down -v`, project identity and initial-only credentials; stop/start continuity pending |
| 8. Actual runtime operation and authentication | **NOT VERIFIED** | Image pull ongoing; no observed startup/SQL/authentication/persistence result yet; documented authentication method additionally has F1 |
| 9. Accurate, synchronized documentation | **FAIL** | Bilingual quick starts and architecture/configuration/lifecycle references align, but CLI password-verification instructions are misleading (F1) |
| 10. Backend and out-of-scope preservation | **PASS (static)** | Source/config/dependency/generated/production/Dockerfile preservation diff exit 0; active MySQL remains authoritative; no migration or integration added |

`PASS (static)` covers only the stated inspection. Final M4P acceptance still requires actual runtime evidence for the pending areas. F1 independently warrants **FAIL** rather than PASS or a runtime-only BLOCKED verdict.

## Next action

Preserve this FAIL record. Create `M4P-02` input referencing F1, correct only the authentication documentation, and complete the already authorized runtime checks without altering the supplied baseline. Verify healthy PostgreSQL 17, localhost-only publication, named mount, read-only SQL, configured user/database, non-loopback positive and negative password controls, and stop/start continuity using the same volume/database system identifier. Record backend/local environment preservation evidence without values. Request an independent `M4P-02` review and close M4P only after sufficient runtime evidence and a PASS verdict.
