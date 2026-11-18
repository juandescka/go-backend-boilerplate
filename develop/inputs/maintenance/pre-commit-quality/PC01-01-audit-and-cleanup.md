# PC01-01 — Pre-Commit Quality Audit and Cleanup

- **Feature:** `maintenance/pre-commit-quality`
- **Iteration:** `PC01-01`
- **Date:** 17 November 2026 (Asia/Makassar)
- **Approval:** User's PRE-COMMIT-01 prompt authorizes audit, targeted cleanup, rule alignment and autonomous review/fix. No staging or commit.
- **Prerequisite:** [M5B-01 review](../../../reviews/identity/authentication/M5B-01-review.md) records PASS; compare actual working source rather than treating that history as fresh verification.

## Objective and initial Git state

Preserve the uncommitted authentication foundation while making a small evidence-based test cleanup, clarifying important comments, and updating permanent development rules. No runtime logic, schema, API, config key/default/precedence, dependency version or package change.

Before editing, ran status (including individual untracked files), tracked/staged diff stats/checks and untracked inventory. HEAD is `5a8c056` (M5A schema); there are **40 existing uncommitted M5B files: 20 tracked modifications and 20 untracked files; nothing staged**. Neither diff check reported whitespace errors. No changes from earlier milestones are automatically attributed to this task. `/private/tmp/pc01-before.json` records current file hashes, HEAD, index fingerprint and status; selected cleanup targets are copied without secrets to `/private/tmp/pc01-before-files` for exact relative comparison.

| Area | Existing changes | Risk | Action |
|---|---|---|---|
| Authentication/source | 13 Go files for models, handler/DTOs, usecase/token, store, middleware, wiring | Security and API contracts | KEEP behavior; comment-only clarification in selected files |
| Tests | 7 M5B-related changed/new test files; repository has 16 test files / 28 top-level tests plus 1 benchmark | Security, operational, concurrency and database regression | REVIEW all relevant boundaries; SIMPLIFY one repeated fake-denial block |
| Configuration | Loader plus JSON/environment examples (3 files) | Compatibility/precedence | KEEP values/keys; clarify retained JWT override comment only |
| Database | 000003 index up/down pair | Session integrity | KEEP SQL; original M5A SQL also unchanged |
| Generated files | Existing Swaggo Go/JSON/YAML (3 files) | Spec/runtime consistency | KEEP; no annotation/generator change or regeneration needed |
| Dependencies | go.mod promotes pinned x/crypto to direct | Dependency scope | KEEP; no upgrade/tidy |
| Technical documentation | 9 README/reference files | Accurate current contracts | KEEP; no product-contract change from this cleanup |
| Development history | M5B input/review pair | Attribution and history | KEEP byte-identical; add separate PC01 input/review |

## Test value assessment

Counts use top-level `Test*` functions, not table rows or a reduction target. All 28 functions stay; 27 KEEP unchanged and 1 SIMPLIFY, 0 MERGE, 0 REMOVE. Benchmark is separately retained. All 16 files stay.

| Test/group | Behavior/risk | Assessment/action |
|---|---|---|
| `TestBearerMiddlewareFailuresAndContextIsolation` | Header validation, storage denial→401, safe identity/context, operational500 | SIMPLIFY five differently named subtests that all assign the same `ErrUnauthenticated`, issue the same request and check the same401. Keep one explicit generic storage-denial assertion; do not pretend the fake models expiry/revocation/activation |
| `TestAuthenticationPostgresIntegration` | Actual SQL expired/revoked/inactive checks, login→me, hash storage, transaction rollback/timeouts | KEEP; directly protects the real states that the repeated unit labels do not model |
| Password's 2 tests + benchmark | Argon2 correctness, malformed PHC/resource bounds, local latency | KEEP; private decoder check targets a security/resource boundary, not arbitrary internals; merging into costly KDF checks adds no clarity |
| Auth usecase's 3 tests | Distinct credential outcomes, token format/hash/context, concurrent capacity/cancellation | KEEP; storage fake and blocking fake cover different boundaries, not duplicate mocking |
| Other auth HTTP test | JSON/field/size limits, identical credential denials, safe responses/logs | KEEP; HTTP contract complements usecase decisions and SQL integration |
| Config's 3 tests | Loader precedence/lifetime validation/metrics route conflicts | KEEP; startup safety, not framework-only checks |
| Existing operational, DB connector/readiness, migration runner/CLI, Scalar tests | Public health/ready/spec contracts, timeout/cleanup/escaping/dirty state and migration integrity | KEEP; unit and integration scopes are complementary |
| Repeated disposable harness setup across opt-in integration files | Different lifecycle/ownership/rollback requirements | DEFER shared-helper extraction; no defect or benefit sufficient to justify a generic test framework |

The only unit fixture removal is the unused `creates` counter/increment in the HTTP fake (no assertion reads it); it is not coverage. Keep failed-login/session-persistence assertions in usecase/integration tests. A single explicit storage-denial assertion preserves the prior unit path and checks the full401 challenge/envelope; real DB validity cases remain intact. No new mock, framework, test function or coverage threshold.

## Comments and naming assessment

Retain meaningful comments on dummy KDF work, verification capacity, encoded token hashing, transactional state/hash recheck, DB timeouts, configuration precedence, pool ownership and migrator cleanup. Add concise English reasons for rejecting ambiguous duplicate credentials and resolving current user/session state per request. Clarify the retained JWT override comment (it does not set opaque lifetime) and replace a misleading Authorization section label with operational/authentication wording. No annotations/generated comments or change-history markers.

Go identifiers follow PascalCase/camelCase with existing idiomatic initialisms. New configuration uses camelCase; public auth DTOs and SQL use snake_case; environment names use UPPER_SNAKE_CASE. Existing legacy keys such as `security.JWTAuthKey`, response-helper compatibility keys and approved `created_by`/`updated_by` FK columns stay unchanged. No style-driven renaming is justified.

## Rules/documentation impact

Update AGENTS EN/ID in matching locations with selective why-comments, meaningful existing-comment preservation, risk-based observable testing, evidence-required test removal, simple fixtures/no duplicate tests/no coverage threshold, and naming with compatibility exceptions. Update TEMPLATE_PROMPT to carry these policies into approved implementation/review/fix workflows, autonomous in-scope review/fix, documentation synchronization, no automatic staging/commit and final model/effort plus actual-scope Conventional Commit recommendations. Fix its/AGENTS' workflow links to the actual `README-develop.md`; do not broaden into unrelated technical-document rewrites.

## Verification and acceptance

Run build, vet, fresh full Go tests, tracked and staged diff checks; focused routes race tests cover the modified HTTP regression. Compare token streams before/after the four comment-only production files to establish no executable change. Compare every baseline file/initial index/HEAD; all existing M5B files outside the five authorized source/test cleanup targets must remain byte-identical. Check bilingual policy alignment and real Markdown links. Record that database integrations are not rerun because neither runtime tokens nor their test code changes; M5B's recent executed disposable evidence is retained, not represented as fresh PC01 runtime execution. No database mutation, Docker lifecycle or generation is needed.

Use an available separate-context reviewer for the 13 requested quality/preservation areas, actual diff and evidence. Preserve any formal FAIL and iterate only within scope; otherwise close on honest PASS. Recommend separating the existing M5B feature commit from this maintenance scope, with a combined commit option if needed; never stage or commit for the user.
