# M1-02 — Fix Connection Initialization Documentation

- **Feature:** `operational/health-readiness`
- **Iteration:** `M1-02`
- **Type:** Fix
- **Based on:** [M1-01 review, F1](../../../reviews/operational/health-readiness/M1-01-review.md#f1--p3-documentation-overstates-absence-of-sql-during-connection-initialization)
- **Approval:** Approved by the user's 12 November 2026 autonomous review/fix request, which permits relevant documentation corrections for verified M1 findings.
- **Status:** Executed; separate-context [M1-02 review](../../../reviews/operational/health-readiness/M1-02-review.md) is PASS

## Objective and findings to address

Resolve F1 without changing runtime behavior. The connector configures `charset=utf8mb4`; the selected MySQL driver issues `SET NAMES utf8mb4` while opening physical connections. Startup/reconnection therefore is not completely free of SQL, although application probes use only `PingContext` and do not query business tables/schema/version.

## In scope

- Correct `documentation/configuration.md` to distinguish application ping from driver session initialization.
- Clarify the corresponding statement in `documentation/architecture.md`.
- Recheck documentation consistency; preserve current source, examples, contracts, package structure and historical input/review.
- Record actual fix verification and separate-context reviewer assessment in M1-02 review.

## Out of scope and constraints

No connector refactor, new tests mirroring unchanged code, interface/layer/package change, dependency update, configuration-policy change, auth/Redis feature, business schema/migration/seed, or production access. README/operations/CLI/workflow changes only if the correction demonstrates an actual inconsistency there; otherwise record why they are unchanged. Do not commit automatically.

## Acceptance criteria

- [x] No claim that opening a MySQL connection is query-free.
- [x] Documents explain `SET NAMES utf8mb4` is driver session setup on new physical connections.
- [x] Application checks remain described as shared-pool `PingContext`, without business/schema/version queries or migrations.
- [x] No source/API/configuration/architecture behavior change.
- [x] Separate-context re-review confirms F1 resolved and no remaining blocking M1 finding.

## Verification guidance

Inspect the final wording against `database.go` and the selected driver implementation. Run `git diff --check` and confirm source/example/module fingerprints are unchanged. Reuse the successful same-session build/vet/test/race/runtime evidence from M1-01 because this fix only changes prose; do not rerun expensive unchanged checks without a new concern. Inspect the two READMEs, operations, CLI and actual workflow for related impact. Retain M1-01 FAIL and create M1-02 review with actual results and limits.
