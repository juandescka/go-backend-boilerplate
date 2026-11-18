# M9B-01 — Backend Integration Verification

- **Date:** 2026-11-18 (Asia/Makassar)
- **Status:** Approved by M9 request; execution evidence in matching review.
- **Prerequisite:** [M9A PASS](../../reviews/foundation-readiness/M9A-02-review.md).

## Scope and acceptance

Reuse M4B migrator lifecycle, M5A exact identity catalog, M6 full ownership schema, M7 full policy/ownership/bearer fixture, M5B authentication/session/M8 real-server security, and M7S actual CLI/PTY provisioning suites. Fresh PostgreSQL17 databases only; all destructive cleanup verifies generated identity/owner/OID. Do not target mapatu_dev with migrations or fixture writes. Preserve reference catalog (6/8/19), no Employee feature or privilege expansion.

Existing tests already cover stored-resource resolution and identity→RBAC handoff. Extend the existing permission-boundary table only to verify global headers/request ID, safe error envelopes and completion-log error correlation across authorization denial/failure. No duplicate fixture framework or production endpoint.

## Commands

Use `GOCACHE=/private/tmp/hris-m9-go-cache` and current binary `/private/tmp/hris-m9-backend`. Existing opt-in flags (not new configuration):

```bash
go build ./...
go vet ./...
go test -count=1 ./...
M4B_INTEGRATION=1 M4B_CLI_BINARY=/private/tmp/hris-m9-backend go test -count=1 -run '^TestPostgresMigrationIntegration$' -v ./internal/infrastructure/migration
M5A_INTEGRATION=1 M5A_CLI_BINARY=/private/tmp/hris-m9-backend go test -count=1 -run '^TestIdentitySessionSchemaIntegration$' -v ./internal/infrastructure/migration
M6_INTEGRATION=1 M6_STAGE=full M6_CLI_BINARY=/private/tmp/hris-m9-backend go test -count=1 -run '^TestTenancyOrganizationSchemaIntegration$' -v ./internal/infrastructure/migration
M7_INTEGRATION=1 M7_STAGE=full go test -count=1 -run '^TestTenantRBACPostgresIntegration$' -v ./internal/infrastructure/migration
M5B_INTEGRATION=1 M5B_CLI_BINARY=/private/tmp/hris-m9-backend go test -count=1 -run '^TestAuthenticationPostgresIntegration$' -v ./internal/registry
M7S_INTEGRATION=1 M7S_CLI_BINARY=/private/tmp/hris-m9-backend go test -count=1 -run '^TestProvisioningPostgresIntegration$' -v ./cmd
git diff --check
```

M5C lifecycle/M8 security helpers execute inside M5B; no invented M5C flag. M4B deliberately tests explicit dirty-state recovery with synthetic SQL only; force is never used to bypass an actual repository migration error. Runtime restrictions must be reported, never converted to PASS from historical evidence.
