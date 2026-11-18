# M1-01 — Review

- **Feature:** `operational/health-readiness`
- **Iteration:** `M1-01`
- **Input:** [M1-01 implementation](../../../inputs/operational/health-readiness/M1-01-implementation.md)
- **Date:** 12 November 2026 (Asia/Makassar)
- **Verdict:** **FAIL — documentation accuracy only**
- **Review method:** Separate-context reviewer subagent `/root/review_m1` independently inspected current source/docs and ran focused race tests. The primary agent, which previously implemented M1, also inspected source and executed the full checks/live smoke test below. Runtime evidence is primary-agent verification; it is not represented as an independently rerun runtime test.
- **Git baseline:** `5cea08bef138c93561c9851f43719140a1347282`, with existing uncommitted implementation and pre-existing user edits. Review assessed actual working files, including untracked files, not HEAD alone.

## Scope and files inspected

- Instructions/context: `AGENTS.md`, actual workflow `develop/README-develop.md`, `develop/FIRST_PROMPT.md`, `develop/TEMPLATE_PROMPT.md`, M1-01 input and current autonomous-review request. `develop/README.md` is absent; no duplicate or rename was created.
- Startup/CLI/config: `main.go`, `cmd/root.go`, `cmd/server.go`, migration command/runner declarations relevant to CLI documentation, `config/config.go`, `config.example.json`, `.env.example`, `go.mod`, `go.sum`.
- Database/registry: `internal/infrastructure/database.go`, `database_readiness.go`, `database_readiness_test.go`, `internal/registry/registry.go`.
- HTTP/lifecycle: `internal/routes/routes.go`, `routes_test.go`, `internal/handlers/operational_handler.go`, `internal/usecases/health/readiness.go`, `lifecycle.go`, `internal/middlewares/http_hardening.go`, `metrics_middleware.go`.
- Shared dependencies: `internal/shared/response/response.go`, `observability/http.go`, `logger/logger.go`, `metrics/component.go`, `metrics/contracts.go`.
- Documentation: both READMEs and `documentation/architecture.md`, `configuration.md`, `operations.md`, `cli.md`.
- Selected dependency implementation: `go-sql-driver/mysql@v1.8.1/connector.go` and `connection.go` to confirm context cancellation and physical-connection session initialization.

## Implementation assessment

No verified M1 source defect was found. The connector returns GORM and the same SQL pool, applies existing pool limits, explicitly pings with a bounded context, closes on failure, suppresses raw connection details, and does not run migrations or business SQL. GORM automatic ping/version query are disabled. Driver context watching is installed before handshake, so the deadline also applies when acquiring a new physical connection.

Registry setup order and existing metrics wiring are consistent. Its pool is shared by GORM, readiness and DB stats. Metrics registration failure closes the pool; `Registry.Close()` retains `sync.Once` for repeated/concurrent calls. No Redis client is initialized. Repeated-close safety was inspected; the race test does not invent registry coverage where no test exists.

Routes register GET probes on the public root router with existing global middleware. Health does not call the checker. Readiness uses the request context plus a 2-second deadline, existing error envelope/code and lifecycle flag. The server checks DB before opening HTTP, logs started after binding, marks shutdown before HTTP drain, forces close on drain timeout and closes registry resources. Live tests also cover listener failure and connection cleanup.

## Findings

### F1 — P3: documentation overstates absence of SQL during connection initialization

- **Locations:** `documentation/configuration.md:46` (startup described as “query-free”); related wording in `documentation/architecture.md:122` (startup/readiness described as only pinging).
- **Source evidence:** `internal/infrastructure/database.go:32` sets `driverConfig.Params` to `charset=utf8mb4`. The selected driver's `connection.go:55–75` implements this with `SET NAMES utf8mb4` when each physical connection is opened. `connector.go` invokes that initialization before returning the connection. An existing idle connection only requires a ping, but a ping that opens/reopens a connection also performs session setup.
- **Impact:** operators/reviewers are given an inaccurate description of SQL issued at startup or reconnect. The implementation correctly avoids business/schema/version queries; it does not avoid necessary driver session SQL.
- **Required fix:** clarify both existing documents, retaining connector behavior, API/configuration contracts and package structure. No source refactor is justified.
- **Verdict rationale:** operational severity is low and there is no demonstrated source blocker. However, documentation synchronization is an explicit mandatory acceptance criterion; retain this FAIL record and resolve the factual mismatch in the next narrow iteration rather than silently amending this review or forcing a source change.

## Verification evidence — performed on 12 November 2026

Commands below were executed from repository root with `GOCACHE=/private/tmp/hris-m1-01-go-cache` where applicable. No dependencies were updated during review.

| Check | Result | Evidence |
|---|---|---|
| `go build ./...` | PASS | Exit 0 |
| `go vet ./...` | PASS | Exit 0 |
| `go test ./...` | PASS | Exit 0; routes 2.461s, infrastructure 0.566s; other packages report no tests |
| `git diff --check` | PASS | Exit 0 |
| Independent focused race check | PASS | Reviewer ran `go test -race ./internal/routes ./internal/infrastructure ./internal/usecases/health ./internal/registry`; routes/infrastructure passed, health/registry have no test files |
| CLI help | PASS | Built current source to temporary binary; root/server/migrate help all exit 0 |
| Runtime smoke | PASS | `python3 -u /private/tmp/hris-m1-review-20261112.4jk1Ei/smoke-review.py`, exit 0; fresh real MySQL 5.7.24 and app on test ports 33327/33328 |

### Runtime observations

The test created a fresh data directory under `/private/tmp/hris-m1-review-20261112.4jk1Ei/data`, an empty dedicated database and a dedicated user with a generated password. The app used a temporary config derived from the JSON example plus generated JWT/DB secrets through supported overrides. No existing local configuration, shared DB, production environment, migration or seed was used.

- Root probes returned 200 with correct envelopes, request IDs/security headers and no authentication. POST returned 405 with `METHOD_NOT_ALLOWED`; `/v1/health` returned 404. Disabled metrics returned 404; disallowed Origin returned 403 through existing CORS.
- Occupied-port startup failed without a false started event; MySQL process-list connection count returned to the existing instance count, demonstrating failure cleanup.
- SIGINT gracefully exited 0 and application DB connections reached zero.
- Enabled existing metrics returned 200 with max-open gauge 25 and ready route observations.
- Paused only the test MySQL process: readiness returned safe 503 in **2.00s**, health stayed 200; resuming MySQL restored readiness 200.
- SIGTERM during an in-flight real DB probe drained its 503 response in **2.14s**, exited 0 and left zero application DB connections.
- With shutdown timeout set to 1s in temporary config, SIGTERM during a blocked probe forced HTTP close in **1.02s**, logged `application.server.shutdown_forced`, returned nonzero and left zero application DB connections. This is the intended timeout error path, not a test failure.
- After stopping only the isolated MySQL process, health remained 200 and readiness returned safe `SERVICE_NOT_READY` 503. A fresh startup failed before HTTP listening in **0.01s**.
- No business tables were created. Generated password/JWT values were absent from app logs/error output. All temporary server/DB processes were stopped.

## Documentation consistency

Both READMEs, examples, operations and CLI match the implemented operational behavior and correctly retain inactive auth/Redis/seed limitations. The configuration/architecture mismatch in F1 remains. The fix only needs these two existing documents. The other references, README setup and workflow do not need a behavior change.

## Acceptance criteria assessment

| Criterion | Assessment | Evidence |
|---|---|---|
| Existing-config MySQL connector | PASS | `database.go`, live example-derived config |
| One managed pool and configured limits | PASS | Source ownership/settings; live DB stats and cleanup observations |
| Context-aware shared readiness ping | PASS | Checker tests; driver inspection; live 2s timeout/recovery |
| Consistent registry/dependency initialization | PASS | Source and successful startup |
| Existing metrics compilation/wiring | PASS | Build, focused test, live metrics |
| Public operational routes | PASS | Source, route tests, live unauthenticated GET/405/root paths |
| Health independent of DB | PASS | Test and real DB stall/outage |
| Readiness healthy/failure/unavailable/timeout/shutdown | PASS | Focused tests and real DB scenarios |
| Safe startup errors and failure cleanup | PASS | Fixed messages; unavailable DB and occupied-port checks |
| Graceful/forced shutdown and pool cleanup | PASS | SIGINT/SIGTERM/in-flight/forced live scenarios; once-only close inspected |
| Existing architecture/scope preserved | PASS | No package move, new layer, schema/business feature or configuration-policy change |
| Configuration examples | PASS | Actual local values supplied; no shared fake secret used; local files preserved |
| Affected documentation synchronized | FAIL | F1 |
| Proportional verification | PASS | Actual checks above with limits below |
| Input traceability | PASS | Existing approved M1-01 input plus this review |

## Remaining issues and limits

F1 is the only verified finding requiring a fix. Tests are scoped; no arbitrary coverage target is claimed. Runtime used MySQL 5.7.24, not all supported deployment environments/versions. Pool lifetime/idle-expiry settings and startup ORM/metrics-registration failure branches were inspected rather than individually fault-injected. No business migration/seed execution was tested or required. Existing legacy configuration validators and stale workflow-name references are preserved outside this narrow operational correctness finding.

## Next action

Create `develop/inputs/operational/health-readiness/M1-02-fix-connection-docs.md` referencing F1, correct only the two documentation statements, validate their consistency and obtain a separate-context re-review. Save the next verdict in `M1-02-review.md`; do not overwrite this FAIL.
