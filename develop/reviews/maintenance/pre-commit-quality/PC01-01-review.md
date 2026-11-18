# PC01-01 — Pre-Commit Quality Review

- **Feature:** `maintenance/pre-commit-quality`
- **Iteration:** `PC01-01`
- **Input:** [PC01-01 audit and cleanup](../../../inputs/maintenance/pre-commit-quality/PC01-01-audit-and-cleanup.md)
- **Date:** 17 November 2026 (Asia/Makassar)
- **Verdict:** **PASS for the targeted pre-commit audit and cleanup**
- **Review method:** Separate-context reviewer `/root/m5a_review` independently read the approved request, input, exact pre-PC comparisons, current tests/policies and preservation state. Build/test/token-stream results below were executed by the primary implementer and are attributed accordingly. The reviewer ran only static/read-only checks and edited only this review; no database operation, build, test, staging or commit was repeated by the reviewer.

## Scope and attribution

The initial working tree already contained **40 uncommitted M5B files**: 20 tracked modifications and 20 untracked files, with nothing staged, at HEAD `5a8c056`. The prior [M5B-01 PASS](../../identity/authentication/M5B-01-review.md) is milestone history, not fresh PC01 runtime verification. Authentication, its index migration, configuration, generated specification and technical reference changes must not be attributed to this cleanup.

PC01 changes exactly eight baseline files, plus its new input/review pair:

| Files | PC01-only change |
|---|---|
| `AGENTS.md`, `AGENTS.id.md` | Aligned selective comments, risk-based testing, naming/compatibility, preservation and approved review–fix rules; actual workflow links |
| `develop/TEMPLATE_PROMPT.md` | Carry those policies into prompts, accurate evidence/docs, no automatic staging/commit and actual-scope model/commit recommendations |
| `config/config.go` | Clarify retained JWT duration versus opaque session lifetime; comment only |
| `internal/middlewares/authentication.go` | Explain rejection of ambiguous Authorization credentials; comment only |
| `internal/infrastructure/authentication/store.go` | Explain resolving current session/user state together; comment only |
| `internal/registry/registry.go` | Replace misleading Authorization banner with accurate operational/global-authentication scope; comment only |
| `internal/routes/authentication_test.go` | Simplify repeated fake denial assertions and remove an unread fake counter |

No runtime logic, configuration value/key/precedence, SQL, dependency, API/annotation, package structure or generated artifact changes are part of PC01. Existing meaningful security/transaction/concurrency/pool comments are retained.

## Test value and coverage

The reviewer compared the modified test directly with its pre-PC copy. Five differently named subtests all performed the same `ErrUnauthenticated` assignment, identical HTTP request and status assertion; they did not model five distinct database states. One explicit storage-denial contract preserves that middleware path and strengthens it with `WWW-Authenticate: Bearer` and `UNAUTHORIZED` checks. The removed `creates` counter was not read by any assertion.

Actual SQL expiry, revocation and inactive-user checks remain unchanged in the disposable authentication integration. Credential failures, hash-only persistence, transaction rollback, token/header parsing, context isolation, operational failures, password bounds, concurrency/cancellation, database/migration and health/readiness/Scalar tests remain protected at their appropriate boundaries. Separate unit, HTTP and database tests are complementary, not duplicate merely because they reach the same function.

Inventory is unchanged: **16 test files, 28 top-level Test functions and one benchmark**. Classification: **27 KEEP, 1 SIMPLIFY, 0 MERGE, 0 REMOVE functions**. No new mock/framework/fixture infrastructure or artificial coverage target was introduced. Shared disposable-harness extraction is reasonably deferred because ownership/lifecycle contracts differ and no defect justifies that abstraction.

## Verification evidence

| Evidence | Actual result and provenance |
|---|---|
| Exact cleanup diff versus `/private/tmp/pc01-before-files` | Independently inspected; only the eight authorized files differ. `/private/tmp/pc01-cleanup.diff` agrees; the whole M5B Git diff is not mistaken for PC01 |
| Baseline `/private/tmp/pc01-before.json` | Reviewer independently compared all 118 files: none missing; exactly eight cleanup targets changed. Of the original 40 M5B files, 35 remain byte-identical and five have the documented comment/test cleanup |
| Git preservation | Reviewer independently confirmed HEAD and SHA-256 of `git ls-files --stage -z` match baseline; cached diff remains empty. This is staged-entry/content preservation, not a claim about raw index stat-cache bytes |
| Policies, naming and links | Independently checked EN/ID rule alignment and compatibility exceptions; corrected actual workflow links and PC01 local Markdown paths resolve |
| `git diff --check`, `git diff --cached --check` | Independently executed: PASS |
| `go build ./...`, `go vet ./...`, `go test -count=1 ./...` | Primary executed with writable temporary Go cache: PASS; all ordinary packages pass, opt-in DB tests skip as designed |
| `go test -race -count=1 ./internal/routes` | Primary executed: PASS, 7.320s; covers the modified HTTP test plus operational/Scalar contracts |
| Four production Go token-stream comparisons | Primary's Go scanner comparison: PASS; executable tokens match pre-PC copies, establishing comment-only source changes |
| Test inventory comparison | Independently counted current 16/28/1 and inspected retained definitions; primary's pre/post name/path inventory also matches |

The reviewer read the secret-free primary execution summary `/private/tmp/pc01-verification-evidence.md`. PostgreSQL disposable tests, migrations, Docker lifecycle, live-server smoke and Swaggo generation were **not rerun for PC01**: executable source, integration tests and annotations are unchanged. Recent M5B/M5A/shared-pool database results remain historical/reused evidence, not fresh PC01 execution. No PC01 database/Docker operation occurred.

## Thirteen-area assessment

| Required review area | Assessment |
|---|---|
| 1. Existing user changes preserved | PASS — baseline, exact task-relative comparison and staged-entry/HEAD checks distinguish prior M5B work |
| 2. No accidental behavior changes | PASS — production diffs are comments; executable token comparison and Go checks pass |
| 3. Risk-based test selection | PASS — one demonstrated duplicate fake path simplified; security/database/concurrency/operational coverage retained |
| 4. No critical coverage removed | PASS — generic denial path remains stronger; real validity cases and all 28 test functions remain |
| 5. No unnecessary mocks/fixtures | PASS — unread counter removed; no new testing abstraction |
| 6. Meaningful code comments | PASS — concise reasons for credential ambiguity, state resolution, compatibility and composition boundary |
| 7. No excessive comments | PASS — selective changes; no obvious-operation narration, AI marker or new source history |
| 8. Naming conventions | PASS — Go initialisms, camelCase config, snake_case API/SQL and uppercase environment conventions documented with existing-contract exceptions |
| 9. No unapproved config contract change | PASS — config edit is commentary; keys/defaults/validation/precedence and schema names unchanged |
| 10. Updated AGENTS/TEMPLATE | PASS — required policies aligned in EN/ID; explicit approval still bounds autonomous review–fix |
| 11. Documentation consistency | PASS — actual workflow links fixed; no unrelated technical-document rewrite or false runtime claim |
| 12. Verification evidence | PASS — current build/vet/full suite/focused race plus static preservation; not-run database scope clearly separated |
| 13. Git diff scope | PASS — eight existing cleanup targets plus paired PC01 artifacts; no staging, commit, dependency/schema or broad refactor |

## Findings and completion

No remaining actionable finding. The reviewer caught one stale TEMPLATE workflow link target during pre-review inspection; it was corrected and rechecked before this formal verdict. One formal iteration: **PC01-01 PASS**; no FAIL review was overwritten or invented and no post-review fix iteration is needed.

This verdict covers cleanup/preservation, not a new production-security certification. M5B's documented limitations, including the requirement for login-abuse controls before public exposure, remain unchanged. Broad refactoring, test-harness unification and other historical cleanup stay deferred.

For manual commits, distinguish the existing feature from maintenance: `feat(auth): add opaque bearer authentication foundation`, then `chore(quality): align development rules and simplify auth tests`, using deliberate hunk separation for shared files. A combined feature commit must also describe the index/config/docs/tests and development-rule cleanup. **No staging or commit was performed.**
