# M3A-01 — Review

- **Feature:** `api/swagger`
- **Iteration:** `M3A-01`
- **Input:** [M3A-01 implementation](../../../inputs/api/swagger/M3A-01-implementation.md)
- **Date:** 12 November 2026 (Asia/Makassar)
- **Verdict:** **PASS**
- **Review method:** Reviewer subagent `/root/review_swagger` used a separate context, inspected actual source/generated artifacts/docs and ran focused tests, recommending PASS with no actionable finding. Primary agent separately completed full checks, generator reproducibility, actual MySQL/browser scenarios and the requested local URL verification. Runtime is primary-agent evidence, not represented as independently rerun by the subagent.
- **Baseline:** clean working tree at `625005a`; existing committed `docs/` artifacts were included in inspection. M1 input/reviews remain unchanged.

## Files inspected

`AGENTS.md`, actual `develop/README-develop.md`, `develop/FIRST_PROMPT.md`, `develop/TEMPLATE_PROMPT.md`, current input, both READMEs, all four technical documents; `main.go`, `cmd/server.go`, all route files/tests, operational handler/response types, health checker/lifecycle, registry/connector, global HTTP/observability middleware, configuration environment loader and metrics reserved-path validation; `go.mod`, `go.sum`, all three generated docs artifacts, and relevant selected Swaggo integration source. Local runtime config was only used for the existing development server; secret contents were not displayed or committed.

## Root cause assessment

| Question | Verified baseline |
|---|---|
| Swagger route registered? | No; server/routes only set up operational probes/optional metrics, so Gin's 404 fallback handled Swagger |
| UI dependencies available in module? | No Swaggo runtime/UI dependencies in go.mod |
| Specification generated? | Yes: committed `docs/docs.go`, `swagger.json`, `swagger.yaml`; generation alone did not expose HTTP |
| Generated package connected to runtime? | No importing runtime package, so no registered spec |
| Gin path mismatch? | No alternate Swagger prefix was implemented; the route was entirely absent |
| Global middleware blocking request? | No; no-Origin 404 was missing-route behavior, not authentication/CORS rejection |
| Existing environment flag disabling docs? | None; current changes use existing app environment without adding a flag |
| Annotation accuracy? | Health/ready paths and response types existed; global metadata still identified LMS and inactive Auth/Users/BearerAuth |

Baseline `go build ./...` also failed with the missing `github.com/swaggo/swag` import from the committed generated package. These are verified wiring/dependency/metadata defects, not missing business endpoints.

## Actual changes

| File | Action | Reason |
|---|---|---|
| `internal/routes/swagger.go` | Added | Same-origin Swagger UI/spec/assets registration, development-only gate, generated-package import |
| `internal/routes/swagger_test.go` | Added | Validate UI/assets/spec identity/envelopes and fail-closed environment behavior |
| `cmd/server.go` | Modified | Call Swagger setup with existing `App.Env` after global middleware/root probe setup |
| `main.go` | Modified | Mapatu metadata, Operations tag, root base path, one pinned generator directive; remove inactive auth metadata |
| `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml` | Regenerated | Synchronize existing artifacts with active API metadata |
| `go.mod`, `go.sum` | Modified | Required Swaggo adapters/runtime and their transitive modules; existing versions retained |
| `README.md`, `README.id.md` | Updated | Aligned local URL, environment gate, restart and regeneration instructions |
| `documentation/architecture.md` | Updated | Existing generated package responsibility/import and route integration |
| `documentation/operations.md` | Updated | UI/spec URLs, middleware, environment responses and unchanged M1 requirements |
| `documentation/configuration.md` | Updated | Existing APP_ENV gate, precedence, production absence, current-origin port |
| `documentation/cli.md` | Updated | Exact pinned generator command/version and committed outputs |
| Current input and this review | Added | Approved scope, actual evidence and independent review assessment |

The existing `docs/` generated Go package remains in place because Swaggo needs its runtime registration. No separate documentation framework, custom generator, package move or architecture layer is introduced. Manually maintained technical documentation stays in `documentation/`.

## Implementation review and findings

**No actionable remaining finding.**

- `internal/routes/swagger.go:14–19` registers the entire wildcard only for trimmed/case-insensitive development. Production, blank, staging/test and unknown environments expose neither UI, spec nor assets. This uses existing loader precedence. `GIN_MODE` does not enable docs.
- The route imports existing generated registration, serves embedded UI assets via standard adapters and uses relative `doc.json`; no CDN/static-directory/port dependency was added. Spec host is unset and base path is `/`.
- `main.go:9–22` has exactly one pinned generation path, Mapatu identity and active Operations metadata. No inactive bearer scheme/business routes are advertised.
- Spec paths are exactly GET `/health` and GET `/ready`. Both reference existing success envelopes; readiness 503 references the existing error envelope and names `SERVICE_NOT_READY`. Shared optional envelope definitions remain generated from source; they do not advertise extra endpoints.
- Global middleware and config loading were preserved. No changes to probe handlers/checker, DB pool/registry, lifecycle/timeout, response envelopes or fail-fast startup. Metrics configuration already reserves the Swagger namespace.
- Local development config and credentials were preserved. No production configuration/DB or migration was changed. Required new dependencies are scoped to existing Swaggo artifacts; no upgrade of unrelated modules.

## Verification evidence

Go 1.24.0; commands run from repository root. Go checks used `GOCACHE=/private/tmp/hris-m1-01-go-cache` due to filesystem restrictions. No destructive operations or business migrations/seeds were run.

| Check | Result | Actual evidence |
|---|---|---|
| `go build ./...` | PASS | Exit 0; current runtime binary also built successfully |
| `go vet ./...` | PASS | Exit 0 |
| `go test ./...` | PASS | Exit 0; existing M1 plus new Swagger tests passed |
| Independent `go test ./internal/routes -run Swagger -count=1` | PASS | Reviewer exit 0, 0.814s |
| `git diff --check` | PASS | Exit 0 |
| Pinned `go generate .` | PASS | v1.16.6 via directive; repeat generation left all three generated hashes identical |
| HTTP development smoke | PASS | UI/index, doc.json, initializer, CSS/JS assets 200; global request IDs and CORS retained |
| HTTP production smoke | PASS | UI/spec/assets all existing `ROUTE_NOT_FOUND` 404; health/ready still 200 |
| Browser render and spec loading | PASS | Actual UI displayed Mapatu and both GET operations; no load or console error; both Try-it-out operations returned 200 |
| M1 DB failure behavior | PASS | Real isolated MySQL stall: readiness 503 in 2.01s, health/Swagger 200; recovery verified; actual outage preserved statuses |
| Requested local URL after restart | PASS | `http://localhost:8099/swagger/index.html` browser loaded spec; UI, JSON, health and ready all HTTP 200 |
| Preservation checks | PASS | Local config hash comparison unchanged; git diff confirmed all M1 handlers/checker/infra/registry/lifecycle/middleware/config loader/shared code untouched |

### Runtime details

`python3 -u /private/tmp/hris-m3a-swagger.grywSe/smoke.py` exited 0 against fresh isolated MySQL 5.7.24 on 33337 and application on 33338. Test config derived from the example, with generated actual local DB/JWT secrets via supported overrides. It verified development assets/spec, production gate, healthy probes, DB stall/recovery/outage, safe errors/logs and SIGTERM cleanup. Only the temporary DB process was paused/stopped; all isolated processes were subsequently stopped.

Browser validation used the actual rendered Swagger UI, not HTML 200 alone. It displayed the Mapatu title, root base URL, Operations and both probes; readiness documentation included 503. Executing GET health and ready through the UI returned their real JSON envelopes on the same origin. Console warning/error inspection returned an empty list.

For the exact user URL, an existing development process on8099 was verified to belong to this repository. A safe startup preflight verified local config/DB initialization before stopping it. That exact old process was gracefully replaced with `GOCACHE=/private/tmp/hris-m1-01-go-cache APP_ENV=development go run main.go server`. The new development server is intentionally left running. No local secrets/configuration were overwritten, and no production process was restarted.

Screenshots (local runtime artifacts, not committed source):
- `/private/tmp/hris-m3a-swagger.grywSe/swagger-ui.png` — isolated UI executing readiness.
- `/private/tmp/hris-m3a-swagger.grywSe/swagger-local-8099.png` — restored UI at requested development URL.

Generation used the local cached-file module proxy during verification because `GOPROXY=off` prevents Go's version/deprecation lookup for a pinned `go run @version`; default online module access or equivalent cached proxy is documented. A temporary focused-test compile error and sandbox cache error were corrected before initial review; final checks above are actual successful runs, not assumed passes.

## Documentation assessment

All six existing references were evaluated and updated. READMEs align in both languages. Architecture distinguishes generated `docs/` from manual `documentation/`; operations covers URLs/normal responses/middleware and M1 dependency behavior; configuration describes only existing app environment/port mechanisms; CLI provides exact tool versions and regeneration outputs. No new technical reference file was necessary. Example files and workflow need no change because no configuration key or development-process rule was added. Existing M1 historical inputs/reviews remain unedited.

## Acceptance criteria

| Criterion | Assessment |
|---|---|
| Root cause 404 identified | PASS — absent route/deps/import plus stale metadata, existing spec recognized |
| Swagger route and UI load | PASS — live isolated and requested8099 browser |
| Valid spec successfully loaded | PASS — parsed typed Swagger, runtime/stored paths/definitions match, UI renders/executes operations |
| Mapatu metadata | PASS — generated identity, root path, Operations only, no inactive auth metadata |
| Operational endpoints preserved | PASS — unchanged source, existing tests, healthy/outage/timeout live checks |
| Minimal architecture | PASS — existing package layout/Swaggo artifacts retained |
| Relevant documentation synchronized | PASS — all six references updated |
| Actual verification evidence | PASS — commands/live/browser observations above |
| Final separate-context review | PASS — reviewer recommendation plus completed full/runtime evidence |

## Limits and next action

No verified blocking issue remains. Runtime coverage used MySQL 5.7.24 locally; production non-exposure was exercised using a temporary application in production mode, not by modifying a production deployment. Initial app configuration and MySQL startup requirements remain, including retained JWT validation. Generator first use needs module access or a suitable cache. The generated spec intentionally covers operational endpoints rather than metrics/assets/business modules. No deployment or new business capability is claimed.

**One implementation/review iteration; zero fix iterations.** Close M3A for this scope. No M3A-02 input or historical FAIL was fabricated. No Git commit was performed. Recommended Conventional Commit: `feat(api): enable development Swagger UI and specification`.
