# M7B-01 — RBAC schema and fixed catalog

Approved scope: recovered user contract and [canonical schema](../../../documentation/database/tenant-membership-rbac-schema.md), separately reviewed before SQL. M7A [PASS](../../reviews/tenant-access-rbac/M7A-01-review.md).

Implement migration pairs000011–000014 in dependency order, exact twelve assignment columns, fixed immutable role identity/scope tuples, six deterministic catalog rows, global permissions/mappings and scope-safe assignments. No assigned_at/expires_at, extra assignment column, implicit grants or privileged bootstrap. Expand existing disposable M7 fixture for FK/unique/shape/scope mutation rejection, empty catalogs and lifecycle. Tighten repeat-up assertion per M7A review. Keep business matrix and inheritance OPEN; schema PASS does not claim full RBAC acceptance. Review separately and continue safe foundation work under the user's explicit recovery instruction even while business authorization acceptance remains BLOCKED.
