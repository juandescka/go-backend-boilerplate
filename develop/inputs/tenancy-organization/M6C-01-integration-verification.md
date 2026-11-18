# M6C-01 — Integration and Integrity Verification

- **Status:** APPROVED under original/recovered autonomous scope.
- **Dependencies:** [M6A-03 PASS](../../reviews/tenancy-organization/M6A-03-review.md), [M6B-02 PASS](../../reviews/tenancy-organization/M6B-02-review.md).
- **Contract:** [Official Module A schema](../../../documentation/database/tenancy-organization-schema.md).

## Scope and acceptance

Review all six tables together; complete full-stage exact index/constraint/lifecycle assertions and execute on a new owned disposable PostgreSQL17 database. Check each one-step rollback removes only its table, preserves Module B/unrelated objects and reapply returns clean empty Module A.

Adapt the existing authentication integration's version3 assumptions to the expanded migration source without weakening its index rollback/session/auth/security/health assertions. Exercise authentication/session lifecycle and startup-without-schema behavior through its existing owned disposable fixture. No new runtime service/repository/model/API/permission is warranted.

Run go build ./..., go vet ./..., go test ./..., migration CLI help and focused live integration. Verify Git changes and historical preservation. Separate-context review after evidence; fix routinely in scope until PASS, then final integration review. Do not mutate mapatu_dev except read-only catalog observations; all migration up/down/version commands target disposable DBs.

## Documentation impact

Document opt-in verification commands and updated auth-test scope. Canonical schema and technical summaries already aligned at M6B; update only actual verification/usage details as needed. Preserve earlier blocked overall report; final completion must use a new numbered overall review instead of overwriting history.
