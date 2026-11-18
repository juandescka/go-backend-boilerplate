# M3B-01 — Review

- **Feature:** `api/scalar`
- **Iteration:** `M3B-01`
- **Input:** [M3B-01 implementation](../../../inputs/api/scalar/M3B-01-implementation.md)
- **Prior milestone:** [M3A-01 PASS](../swagger/M3A-01-review.md)
- **Date:** 12 November 2026 (Asia/Makassar)
- **Verdict:** **PASS**
- **Review method:** Separate-context reviewer subagent `/root/review_scalar` inspected actual source/docs/dependency diff and ran focused tests, finding no actionable defect. Primary agent separately completed full checks, generator equality, isolated runtime and actual browser tests. Runtime/browser evidence is primary-agent verification, not a claim of independent subagent runtime execution.
- **Baseline:** clean `cf5303e`. Existing historical M1/M3A artifacts preserved.

## Scope and files inspected

Read both agent instruction files, actual workflow `develop/README-develop.md`, FIRST_PROMPT/template, current input and previous M3A PASS review. Inspected `main.go`, `cmd/server.go`, route files/tests, config loader/validator, registry, operational handlers/response/checker/lifecycle and global middleware, generated Go/JSON/YAML and module files. Evaluated both READMEs and all four existing technical references. Official Scalar integration/config/themes references in the input were consulted for current API and pinned asset behavior.

## Audit and actual changes

M3A's JSON URL was served by `gin-swagger`; deleting the wildcard/UI dependency alone would have broken spec access. The new direct JSON handler resolves that coupling while preserving one generated source. Runtime dependency audit found no other consumers of UI adapters. Existing tests/docs that referenced old UI were updated. UI replacement is approved; no business endpoint contract changed. Old UI bookmarks now receive404 instead of an invented redirect/second UI.

| File | Action | Reason |
|---|---|---|
| `internal/routes/swagger.go` | Modified | `/docs` Scalar HTML and direct Swaggo JSON with existing environment gate; old UI removed |
| `internal/routes/swagger_test.go` | Modified | Verify new presentation/privacy settings, retained spec, old UI removal and production absence |
| `cmd/server.go` | Modified | Wire renamed combined docs setup using existing App.Env |
| `config/config.go` | Modified | Reserve `/docs` namespace against metrics conflicts |
| `config/config_test.go` | Added | Prove collisions fail validation and normal metrics path stays valid |
| `go.mod`, `go.sum` | Modified | Remove unused gin-swagger/files UI dependencies and orphan sums; retain Swaggo and existing versions |
| `README.md`, `README.id.md` | Updated | Aligned new URL, spec compatibility, generation and CDN/privacy requirements |
| `documentation/architecture.md` | Updated | Existing annotations→Swaggo→Gin JSON→Scalar flow, same package layout and changed dependencies |
| `documentation/configuration.md` | Updated | Existing environment gate, docs metrics reservation, CDN version and no new flags |
| `documentation/operations.md` | Updated | Actual routes/statuses, old UI404, privacy/internet requirements, preserved M1 semantics |
| `documentation/cli.md` | Updated | Unchanged Swaggo command and no Scalar CLI/npm requirement |
| Current input and review | Added | Approved execution trace and final evidence |

The existing route filename/package was retained because it still serves the existing Swagger JSON URL; no package move/new abstraction is warranted for two simple responses. HTML uses official Scalar ESM1.69.2 with one versioned CDN reference and minimal system-font CSS. No custom bundling, account/key, Registry, new Go dependency or OpenAPI conversion pipeline. `main.go`, annotations and all three generated files are unchanged.

## Implementation findings

**No actionable remaining finding.**

- Development-only gate remains trimmed/case-insensitive `App.Env=development`. Production/blank/unknown modes register neither docs nor JSON; `GIN_MODE` does not override it. Global middleware still wraps both handlers.
- JSON uses `swag.ReadDoc()` from the same registered generated package, preserving `/swagger/doc.json` and response content type. Failure returns a generic existing envelope without raw errors; no second/manual spec is created.
- Scalar configuration points to the same-origin JSON. Proxy URL is empty; AI Agent is explicitly disabled, telemetry/auth persistence/default font CDN disabled. No analytics plugin, secret or user credential is embedded in constant HTML. No CSP, CORS or security middleware was relaxed.
- Standard Scalar presentation provides navigation/search/theme/schema display without another frontend project. Optional built-in promotion controls were not used; no MCP/Registry integration was implemented.
- Removing UI adapters is scoped to their actual consumers. Swaggo runtime/generator stays1.16.6. Metrics path validation now rejects the new docs namespace before registration; existing `/metrics` and Swagger reservation remain intact.
- No M1 handler/checker/lifecycle/pool/shared source change. Runtime startup still requires valid existing config and reachable MySQL.

## Verification evidence

Checks run from repository root with Go1.24.0. Use `GOCACHE=/private/tmp/hris-m1-01-go-cache` for writable cache. No migration, seed, production data mutation or stress test.

| Check | Result | Actual evidence |
|---|---|---|
| `go build ./...` | PASS | Exit0; runtime binary also built |
| `go vet ./...` | PASS | Exit0 |
| `go test ./...` | PASS | Exit0; config0.465s, routes3.002s, unchanged M1 checks retained |
| Independent focused tests | PASS | Reviewer ran `go test ./internal/routes ./config -count=1`, routes2.514s/config0.297s |
| `git diff --check` | PASS | Primary and reviewer exit0 |
| Existing `go generate .` | PASS | Swaggo1.16.6 directive unchanged; generated Go/JSON/YAML hashes identical before/after |
| Spec availability/accuracy | PASS | Typed spec tests; runtime JSON paths/definitions match committed generated JSON; only GET health/ready, existing envelopes and readiness503 |
| Scalar browser rendering | PASS | Actual CDN JS/styles loaded; Mapatu/title2operations/models displayed, search filtered readiness, light/dark toggle worked, Test Request returned real200 |
| Browser console | PASS | Warning/error inspection returned empty list on isolated and actual local page |
| Environment/security HTTP | PASS | Production docs/spec/oldUI404; development CORS rejection still403, global headers retained, metrics/docs conflict rejected |
| M1 dependency scenarios | PASS | Real DB stall readiness503 in2.01s with health/docs/spec200; recovery and actual outage checked; SIGTERM exits0 |
| Local requested runtime | PASS | `/docs`, `/swagger/doc.json`, `/health`, `/ready`200 at localhost8099; old `/swagger/index.html`404 |
| Preservation | PASS | Local `.env`/`config.json` hash comparison unchanged; no generated/main.go/M1/middleware/shared diffs |

### Runtime and browser details

`python3 -u /private/tmp/hris-m3b-scalar.UVBb45/smoke.py` exited0 against fresh isolated MySQL5.7.24 on33347 and app33348. It used generated genuine test secrets and example-derived temporary config. It covered development HTML/privacy config, retained generated JSON, oldUI404, production-mode404 with healthy probes, config metrics conflict, DB stall/recovery/outage and cleanup. Only the temporary DB was paused/stopped; all isolated processes were stopped afterward. Generated secrets were absent from application output.

Browser inspection showed Swagger2.0 read natively by Scalar, Mapatu identity, both operations, readiness503 and existing response model names. Search for readiness returned the actual operation. Theme controls changed between dark/light. The API client issued a local readiness test and displayed200 with `success:true`, `data.status:ready`; server completion log confirmed the browser's loopback `/ready` route, supporting the no-external-proxy configuration. No AI chat appeared; no Scalar account/key/Registry or optional promotion action was used. No fatal/load/console warning/error was observed. No unsupported claim of exhaustive third-party network auditing is made.

The user development server on8099 was preflighted, its old listener verified by PID/cwd, then gracefully replaced with the current source command `GOCACHE=/private/tmp/hris-m1-01-go-cache APP_ENV=development go run main.go server`. Local files remained unchanged. The new server intentionally remains running, and actual `/docs` browser rendering uses `http://localhost:8099` as backend origin. Temporary test DB was not used by this local server.

Screenshots (local evidence, not committed assets):
- `/private/tmp/hris-m3b-scalar.UVBb45/scalar-test.png`.
- `/private/tmp/hris-m3b-scalar.UVBb45/scalar-local-8099.png`.

CDN version1.69.2 availability was separately verified from jsDelivr package metadata; browser execution verified actual module compatibility. Official references: [HTML/JS](https://scalar.com/products/api-references/integrations/html-js), [configuration](https://scalar.com/products/api-references/configuration), [themes](https://scalar.com/products/api-references/themes).

## Documentation impact assessment

| Document | Assessment |
|---|---|
| README English/Indonesia | Updated consistently for `/docs`, retained JSON, old UI removal, CDN requirement and unchanged generation |
| Architecture | Updated only presentation/spec-serving/dependency responsibilities; existing folder/package layout retained |
| Configuration | Describes existing App.Env gate and precedence, docs reserved path, fixed CDN source and privacy flags; no invented env key |
| Operations | Replaces obsolete Swagger UI section; health/readiness contract sections preserved |
| CLI | Keeps exact single Swaggo command/version/output; explicitly no Scalar CLI/npm/build requirement |
| API-specific docs | Not added: existing six references cover this integration adequately |
| Workflow/examples/historical records | Unchanged: no process/config key change; M1/M3A history retained |

## Acceptance criteria

| Criterion | Assessment |
|---|---|
| Gin integration and development `/docs` | PASS |
| Actual Scalar render and spec load | PASS |
| Correct Mapatu metadata | PASS |
| Swaggo retained and existing generation works | PASS |
| Specification endpoint remains available | PASS |
| Old Swagger UI compatibility handled | PASS — JSON preserved; authorized old UI removal404, no second UI/extra alias |
| No excessive dependencies/structure changes | PASS — removed2Go UI dependencies, no new Go/frontend package/pipeline |
| M1 probes continue to work | PASS — source, tests and actual DB scenarios |
| Security middleware preserved | PASS — no relaxed CSP/CORS/headers; production gate and explicit privacy settings |
| Relevant docs synchronized | PASS |
| Actual verification and final review evidence | PASS |

## Limitations and next action

Scalar rendering needs JavaScript and internet/CDN access to jsDelivr including versioned lazy chunks/styles. JSON/probes remain independent of CDN availability. This is a pinned development baseline, not an offline asset deployment or production UI exposure. Runtime database tests used available MySQL5.7.24 locally, not every target deployment. The shared generic response schema/examples are deliberately preserved from M3A; no business annotation/spec overhaul was attempted. Standard Scalar promotion controls can exist, but no external account/Registry/AI/MCP feature was integrated or used.

**One implementation/review iteration, zero FIX iterations.** No verified blocker remains; close M3B for this scope. No fabricated FAIL or speculative M3B-02 artifact. No Git commit was performed.

Recommended Conventional Commit: `feat(api): replace Swagger UI with Scalar reference`.
