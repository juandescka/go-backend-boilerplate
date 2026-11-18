# M6A-03 — Isolate Ownership FK Verification

- **Status:** APPROVED under autonomous routine-fix authorization.
- **Based on:** [M6A-02 F3](../../reviews/tenancy-organization/M6A-02-review.md).
- **Scope:** Change the attempted root-organization tenant move to use a nonconflicting code, isolating parent-reference FK rejection from scoped-code uniqueness. Preserve SQL and prior reviews.
- **Acceptance:** Actual CLI stage-A disposable integration passes all schema, integrity and rollback/reapply assertions; obtain separate-context SQL/test review before M6B.
- **Documentation impact:** None to technical contract; record actual verification in the next review.
