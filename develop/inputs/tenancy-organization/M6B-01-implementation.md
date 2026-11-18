# M6B-01 — Organizational Structure Implementation

- **Status:** APPROVED under the original autonomous request and recovered contract.
- **Dependency:** [M6A-03 PASS](../../reviews/tenancy-organization/M6A-03-review.md).
- **Reference:** [Official schema contract](../../../documentation/database/tenancy-organization-schema.md).

## Scope and acceptance

Implement branches/departments/positions/work_locations in versions000006–000009 only. Use exact approved types/defaults/nullability and business scopes. Composite organization, department-parent and branch FKs must reject inconsistent tenant/organization on inserts/updates. Preserve multiple head offices, optional location branch, nullable independent coordinates, hierarchy>=1 and positive radius/range checks. Do not invent branch-scoped departments/position assignments or recursive triggers.

Extend the same disposable actual-CLI integration suite to stage b, verifying six-table contracts and critical integrity risks, Module A-only reverse rollback and reapply. Obtain separate-context review before M6C. No changes to applied migrations, configuration, auth/HTTP behavior, API spec, seeding or repositories.

## Documentation impact

Synchronize the canonical schema reference's migration source/status, architecture boundaries, migration CLI/operations and bilingual README schema summaries with actual implementation. Preserve all previous inputs/reviews and the earlier blocked checkpoint. Future longer-cycle validation is documented, not claimed implemented.
