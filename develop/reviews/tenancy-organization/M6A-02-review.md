# M6A-02 — Review

- **Input:** [M6A-02](../../inputs/tenancy-organization/M6A-02-contract-recovery-implementation.md)
- **Verdict:** **FAIL — verification fixture collision**
- **Method:** Separate-context contract-plan approval followed by primary-agent structured review of actual SQL and failed execution; this runtime finding is not an independent full SQL review.

## Scope and changes

Recorded canonical six-table contract before SQL; added versions 000004/000005 and one opt-in CLI-backed disposable integration test. Prior blocked artifacts preserved.

## Evidence and finding F3

`go build -o /private/tmp/hris-m6-backend .` passed after permitted sandbox escalation for Go cache access. Docker read-only status showed existing healthy PostgreSQL 17 on localhost:5433. `M6_INTEGRATION=1 M6_STAGE=a M6_CLI_BINARY=/private/tmp/hris-m6-backend go test -count=1 -run '^TestTenancyOrganizationSchemaIntegration$' -v ./internal/infrastructure/migration` applied to clean version5 and repeated up successfully, then failed in the organization ownership-update assertion.

**Severity:** Blocking test correctness, narrow fixture fix required. The root organization and destination-tenant organization deliberately share code SHARED. Updating only tenant_id can violate both scoped code uniqueness and parent-reference integrity; expecting SQLSTATE23503 specifically does not isolate the intended FK behavior. Change the attempted code to a nonconflicting value in that assertion so the test checks ownership references deterministically. No schema weakening is warranted.

The test-owned database `m6_verify_b07d51759019a05b` was removed after owner/OID checks; the development public catalog remained unchanged. The only development connection used catalog SELECT in a read-only transaction; no migrator opened there. Full lifecycle and remaining assertions did not run, so M6A acceptance is FAIL, not PASS.

## Documentation impact and next action

The recovered contract resolves prior design findings F1/F2. No schema-document change is needed for this fixture correction. Create M6A-03 fix input, correct only the conflicting ownership-update fixture and rerun stage A. M6B remains unstarted until M6A PASS.
