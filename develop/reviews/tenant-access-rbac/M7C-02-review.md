# M7C-02 — Authorization re-review after M7E

**PASS for the approved v1.0 foundation.** This review supersedes the acceptance status in [M7C-01](M7C-01-review.md). The [M7E policy input](../../inputs/tenant-access-rbac/M7E-01-policy-finalization.md) resolves the grant and containment decisions.

A single parameterized SQL decision checks active user, assignment, explicit mapping, compatible role scope, and the active and correctly related owners. Tenant access requires active membership. The verified resource tuple remains available to the handler on success. Tenant grants stay in their tenant. Organization grants reach directly owned branches and their resources. Branch grants require the same branch. Platform grants apply only on the isolated platform path.

The final PostgreSQL 17 rerun passed in owned database `m7_verify_4e6a2bb001814d86bf1dbc53af3078c6`, which was removed. It verified positive containment with stored work locations, denial for resource and tenant ID substitution, denial for sibling branches and child organizations, and empty EMPLOYEE grants. Inactive owners, user, membership and assignment denied access. Unknown or removed permissions denied the next request. Expired and revoked tokens returned 401. Invalid ownership returned 403, and backend lookup failure stopped the request safely. The default middleware tests also passed.

The separate-context review checked query and migration source and the primary agent's M7E execution evidence. Business route ownership is verified through an integration fixture. Future production routes must supply a resolver and constrain resource access to the verified scope.
