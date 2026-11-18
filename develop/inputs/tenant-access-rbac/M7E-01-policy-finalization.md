# M7E-01 — Finalize RBAC Business Policy v1.0

Approved by the M7E master prompt. Resolve B1 and B2 from the [original M7 review](../../reviews/tenant-access-rbac/M7-FINAL-review.md) using the newly approved eight permissions, six-role matrix and scope containment policy. B3 remains explicitly denied and deferred for any future assignment-management API.

Keep migrations through 000014 and the existing role identities. Add a deterministic reference-data migration with rollback limited to its own grants and permissions. Preserve one parameterized authorization query, existing authentication and typed request scope. Verify target ownership and active states before applying assignment containment. Cover the seeded matrix, containment boundaries, stored resource ownership through a test route, authentication and disposable migration lifecycle. Update canonical documentation and issue new C, D and final reviews while preserving prior reviews.
