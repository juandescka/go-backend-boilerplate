# M3C-01 — Review

- **Feature:** `api/http-foundation`
- **Iteration:** `M3C-01`
- **Input:** [M3C-01 audit and standardization](../../../inputs/api/http-foundation/M3C-01-audit-and-standardization.md)
- **Date:** 12 November 2026 (Asia/Makassar)
- **Baseline:** clean commit `98b61d0`; M3B review is present and PASS.
- **Verdict:** **FAIL — documentation accuracy; no source fix required**
- **Review method:** Separate-context reviewer `/root/review_http_foundation` independently inspected actual source/dependency behavior, consumer searches and documentation. Primary agent also performed the read-only audit and documentation checks. No build/test/runtime evidence is claimed for this documentation-only work.

## Scope and files inspected

Both agent instructions, actual workflow `develop/README-develop.md`, FIRST_PROMPT/template and M3B PASS review. Current READMEs, architecture/configuration/operations/CLI, new `documentation/api/api-standard.md` and input. `cmd/server.go`, registry, all routes/handlers/tests, response/meta DTO, httphelper/validation/labels, observability/sanitization/logger, middleware, metrics and placeholder directory inventory. Installed Gin1.11.0 binding/json/query/default-validator code confirms decoder permissiveness and `binding` tag semantics; the local helper separately uses validator's default `validate` tag.

## HTTP foundation assessment

| Area | Decision | Evidence / conclusion |
|---|---|---|
| Response | KEEP / DOCUMENT | Existing success/error envelopes, null/empty/meta semantics and success helper200 are explicit; shared-to-handler DTO dependency has no active cycle |
| Errors | KEEP / DOCUMENT | Generic status mapping and active explicit codes distinguished; safe readiness/panic/spec responses preserved |
| Binding/validation | DOCUMENT / DEFER | Helpers have no endpoint callers; flat mapping limits, two validator engines and permissive JSON described without claiming active DTO endpoints |
| Pagination | DOCUMENT / DEFER | Builder only; no consumers/default/max/parser/total-pages; caller-supplied flags and arithmetic bounds described |
| Routing | KEEP / DOCUMENT | Active root probes/docs/spec/metrics and404/405; `/v1` conventions clearly future, not invented routes |
| Middleware | KEEP / DOCUMENT | Order/security/observability/CORS/body behavior assessed; empty-body explanation needs F1 correction |
| Legacy/placeholders | DEFER | No active errors package/domain/mapper implementation; label/header/upload residues not automatically removed |

No demonstrated active HTTP endpoint defect or necessary architecture change. Deferring unused nested validation and pagination hardening does not conceal a current route defect: repository searches find no callers of `BindAndValidate`, `BuildPaginationResponse`, or `IsRequestBodyTooLarge` outside their definitions. Future adoption is explicitly constrained by the standard. No new default, response/status contract, dependency, helper or business endpoint is imposed.

## Findings

### F1 — P2: qualify forwarding of empty bodies after the multipart gate

- **Location at review:** `documentation/api/api-standard.md:222`.
- **Problem:** the unconditional statement that empty/zero-length bodies pass through omits an earlier multipart decision.
- **Source:** `internal/middlewares/http_hardening.go:29–60`; after the method/body-nil guard, multipart processing at41–49 occurs before `ContentLength==0`. A write-method request with a non-nil empty body and multipart/form-data on a non-allowlisted path returns415.
- **Impact:** developers could incorrectly expect downstream404/405/validation instead of the existing415. Correct the description, not the middleware contract.

### F2 — P3: identify order_by as sorting metadata

- **Location at review:** `documentation/api/api-standard.md:51`.
- **Problem:** sentence groups `has_next` and `order_by` as pagination fields.
- **Source:** `internal/handlers/dtos/responses/meta/meta_response.go`: `has_next` is in PaginationResponse, `order_by` is in SortingResponse.
- **Impact:** ambiguous official guidance could put a sorting key in the pagination object. Correct the sentence while preserving existing keys and JSON structures.

A minor primary-agent follow-up is to spell out placeholder directory names with actual `internal/handlers/mappers` and separate domain/models/domain/repositories paths rather than shorthand. This is non-blocking reference clarity and can accompany the same small documentation correction; no directory is moved.

## Actual changes assessed

- Added one requested `documentation/api/api-standard.md` covering active/available/future distinctions, responses, errors, validation, pagination, routing, middleware and developer examples.
- Added discoverability links to both READMEs and architecture.
- Added approved input artifact.
- No source/test/config/dependency/generated changes. Configuration/operations/CLI were evaluated and remain accurate without changes because no runtime behavior or commands changed.

## Verification evidence

| Check | Status | Actual evidence |
|---|---|---|
| Source/consumer/dependency inspection | Completed | Commands reading current source, `rg` consumer search, installed Gin binding code, directory inventory |
| Five JSON examples | PASS | Python parsed all5; checked active health/ready/error bodies and illustrative validation/list envelope fields |
| Local links/anchors | PASS | Python checked target files and referenced Markdown headings, excluding inline/fenced code from link recognition |
| `git diff --check` | PASS | Primary and independent reviewer commands passed |
| Code/config/module/generated preservation | PASS | `git diff --quiet HEAD -- main.go cmd config internal docs go.mod go.sum config.example.json .env.example`, exit0 |
| Build / vet / Go tests | NOT RUN | No code changes; not required for prose under approved risk-based strategy |
| Runtime / database / Scalar smoke | NOT RUN | No runtime changes; repeated M1/M3B smoke explicitly unnecessary |

An initial link-check regex incorrectly interpreted `BindAndValidate[T](...)` inside inline code as a Markdown link. The checker was corrected to exclude code; this was not a document or implementation defect. No tests, migrations, seed, dependency update, generator run, server restart or Git commit were performed.

## Documentation consistency and remaining issues

The overall standard matches source and clearly labels examples as illustrative and pagination defaults as unestablished. F1/F2 prevent passing mandatory accuracy until corrected. Operations/configuration/CLI remain unchanged for the reasons above; no duplication of workflow or extra technical documents is needed. Existing M1/M3A/M3B history is preserved.

Other limitations are documented and non-blocking for the active scope: response/handler DTO coupling, nested formatter key collisions, non-string min/max wording, pagination policy/overflow obligations, strict JSON/201/204 decisions and LMS/SPMB residues. No approval is being requested for deferred changes because none is executed.

## Acceptance assessment

Audit coverage, response/error analysis, binding/pagination evaluation, route conventions, middleware responsibility, preservation/scope and proportional checks are complete. New standard exists and required input/review traceability is present. Documentation accuracy is **FAIL** for F1/F2; final review therefore is **FAIL**, not an active code/runtime regression.

## Next action

Create M3C-02 input referencing F1/F2; correct only prose, verify affected examples/links/source preservation and request focused separate-context re-review. Preserve this initial FAIL. No source implementation or repeated Go/runtime tests are justified by these findings.
