# M7S-A-01 — Named seeder CLI and target verification

The M7S master prompt authorizes a metadata-only list and explicit named runs. M5 identity/session, M6 tenancy and the [M7 PASS review](../../reviews/tenant-access-rbac/M7-FINAL-02-review.md) establish the schema prerequisites. Migration 000015 owns the fixed role and permission policy.

Use Cobra and the existing PostgreSQL connector. Require the effective development environment, an explicit numeric loopback endpoint and database name, a matching live connection and a clean migration version of at least 15. Reject unknown names before configuration or database access. No automatic run-all path. Register the initial datasets in stage B.
