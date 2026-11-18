# M3C-02 — Fix API Standard Accuracy

- **Feature:** `api/http-foundation`
- **Iteration:** `M3C-02`
- **Type:** Documentation Fix
- **Based on:** [M3C-01 review](../../../reviews/api/http-foundation/M3C-01-review.md), F1/F2
- **Approval:** active user autonomous M3C review/fix request authorizes routine documentation corrections within scope.
- **Status:** Executed; separate-context [M3C-02 review](../../../reviews/api/http-foundation/M3C-02-review.md) is PASS

## Objective and findings

Resolve verified standard inaccuracies without changing runtime behavior:

1. F1: qualify empty/zero-length forwarding as occurring after the existing multipart gate. The write-method/body-nil checks still precede this gate; non-allowlisted multipart with a non-nil empty body can return415.
2. F2: assign `has_next` to pagination and `order_by` to sorting; preserve `isActive` as the existing filter key.
3. Non-blocking reference clarification: spell out actual placeholder directories rather than singular/compressed shorthand. Do not move/create source packages.

## Scope and constraints

Only `documentation/api/api-standard.md` needs content correction. Current README/architecture links remain valid; config/operations/CLI/generation have no behavior change and need no further rewrite. Preserve M3C-01 input/FAIL, active probe/docs contracts, source/tests/config/modules/generated artifacts and all earlier milestones. No new helper, test framework or business route. No Go tests/build/runtime/DB work merely for these sentences.

## Acceptance criteria

- [x] Empty-body statement matches middleware branch order, including multipart exception.
- [x] Metadata field ownership is accurate.
- [x] Placeholder paths match actual directories.
- [x] JSON examples and local references remain valid.
- [x] Code/config/generated/module diff remains empty.
- [x] Separate-context re-review confirms no blocking documentation/source finding.

## Verification

Read the corrected passages against middleware/meta DTO/directory inventory. Repeat lightweight JSON/link/anchor and `git diff --check` checks; confirm source preservation. Save actual M3C-02 review with the final verdict and explain no-impact documents and checks deliberately not run. No automatic commit.
