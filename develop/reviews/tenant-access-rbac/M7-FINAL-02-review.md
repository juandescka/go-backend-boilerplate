# M7 Final Review, Revision 02

**M7 PASS for the approved RBAC authorization foundation.** M7E resolves B1 and B2 with the newly adopted RBAC Business Policy v1.0. B3 is explicitly deferred and denied. The [original BLOCKED review](M7-FINAL-review.md) records the state before policy approval.

| Checkpoint | Result |
|---|---|
| M7A membership schema | PASS from [M7A-01](M7A-01-review.md) |
| M7B fixed roles and RBAC schema | PASS from [M7B-01](M7B-01-review.md) |
| M7C authorization | PASS after containment and stored-resource checks in [M7C-02](M7C-02-review.md) |
| M7D integration and security | PASS after disposable regressions in [M7D-02](M7D-02-review.md) |
| M7E policy finalization | PASS in [M7E-01](M7E-01-review.md) |

## Approved permission catalog and matrix

The complete eight-code catalog is `platform.manage`, `tenant.read`, `membership.read`, `organization.read`, `branch.read`, `department.read`, `position.read` and `work_location.read`.

| Fixed role | Grants |
|---|---|
| SUPER_ADMIN | `platform.manage` |
| TENANT_ADMIN | `tenant.read`, `membership.read`, `organization.read`, `branch.read`, `department.read`, `position.read`, `work_location.read` |
| HR_ADMIN | `organization.read`, `branch.read`, `department.read`, `position.read`, `work_location.read` |
| MANAGER | `organization.read`, `branch.read`, `department.read`, `position.read` |
| SUPERVISOR | `branch.read`, `department.read` |
| EMPLOYEE | None |

## Authorization and migration

Every decision requires an explicit grant and a valid, active assignment. Tenant and organization assignments can cover contained resources. Organization containment stops at directly owned branches and never traverses child organizations. Branch access requires the stored branch owner to match. Platform grants apply only on the platform path. Authentication, active owners, active tenant membership and the validated resource scope are checked before protected work. The [canonical policy](../../../documentation/database/tenant-membership-rbac-schema.md) explains the owner contract and denial rules.

Migration [000015 up](../../../migrations/postgres/000015_seed_rbac_policy_v1.up.sql) installs the eight permissions and 19 grant pairs. It uses deterministic permission IDs and rejects pre-existing code or ID collisions. [000015 down](../../../migrations/postgres/000015_seed_rbac_policy_v1.down.sql) removes the seeded pairs and matching permission identities. An unrelated mapping that depends on a seeded permission prevents rollback. Prior migrations through 000014 were hash-verified. The suite applied SQL to disposable PostgreSQL 17 databases, and the authentication regression observed the development catalog without migrating it.

## Evidence and follow-up

Build, vet, default Go tests and Git whitespace checks passed. The final RBAC suite passed authorization and rollback/reapply checks. The actual CLI and session regression applied through version 15 and preserved existing authentication behavior. Both disposable databases passed ownership checks and were removed. The separate-context reviewer inspected source and M7E execution evidence. Positive business authorization is verified through stored work-location fixtures. Production business routes and assignment management remain deferred. Future assignment APIs need actor authority, delegation bounds, self-assignment prevention and escalation safeguards. No default user or automatic assignment is provisioned.

Recommended M7E commit after recording the existing M7 foundation separately.

```text
feat(authorization): finalize RBAC business policy v1.0
```
