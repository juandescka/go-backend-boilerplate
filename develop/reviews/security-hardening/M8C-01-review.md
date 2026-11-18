# M8C-01 Integration Review

- Input: [M8C-01](../../inputs/security-hardening/M8C-01-integration.md).
- Scope: authentication/session lifecycle, limiter/HTTP composition, PostgreSQL normalization, tenancy/RBAC, provisioning/terminal safety, concurrency and final tooling.
- Status: **PASS**. Separate-context reviewer `/root/security_review` inspected complete production/test/documentation/generated diffs and independently ran focused M8A/M8B tests. Full opt-in PostgreSQL execution was performed by the primary agent and its evidence reviewed by the separate reviewer; no claim that the reviewer independently reran the database suites.
- Findings/security severity: no unresolved high/medium defect. A one-record unusable limiter was caught and corrected during M8A implementation, before its formal PASS. Real-clock HTTP assertions use bounded positive Retry-After, and the live expiry fixture uses a three-second window/six-second recovery deadline. No historical FAIL review was overwritten.
- Files changed in this stage: authentication_integration_test.go fixture budgets/call; new registry/security_integration_test.go reuses owned DB; limiter benchmark added to login_limiter_test.go; M8C input/reviews. Existing default limits and regression negatives remain unchanged.

## Executed evidence

All Go commands below used `GOCACHE=/private/tmp/hris-m8-go-cache`; the current binary is `/private/tmp/hris-m8-backend`.

| Check | Actual command | Result |
|---|---|---|
| Build | `go build ./...`; `go build -o /private/tmp/hris-m8-backend .` | PASS |
| Vet | `go vet ./...` | PASS |
| Default suite | `go test -count=1 ./...` | PASS, repeated after final test changes |
| Race | `go test -race -count=1 ./internal/shared/security ./internal/routes ./internal/middlewares ./internal/shared/observability` | PASS |
| Authentication/session + security HTTP | `M5B_INTEGRATION=1 M5B_CLI_BINARY=/private/tmp/hris-m8-backend go test -count=1 -run '^TestAuthenticationPostgresIntegration$' -v ./internal/registry` | PASS initially 6.74s, final affected rerun 7.85s |
| Tenancy | `M6_INTEGRATION=1 M6_STAGE=full M6_CLI_BINARY=/private/tmp/hris-m8-backend go test -count=1 -run '^TestTenancyOrganizationSchemaIntegration$' -v ./internal/infrastructure/migration` | PASS 1.39s |
| Scoped RBAC | `M7_INTEGRATION=1 M7_STAGE=full go test -count=1 -run '^TestTenantRBACPostgresIntegration$' -v ./internal/infrastructure/migration` | PASS 1.09s |
| Seed/bootstrap/PTY | `M7S_INTEGRATION=1 M7S_CLI_BINARY=/private/tmp/hris-m8-backend go test -count=1 -run '^TestProvisioningPostgresIntegration$' -v ./cmd` | PASS 2.31s |
| Performance sample | `go test -run '^$' -bench '^BenchmarkLoginLimiter$' -benchtime=1s -benchmem ./internal/shared/security` | PASS; known-IP concurrent 326.3ns/op, 0B/0alloc; saturated-new-key 176.6ns/op, 0B/0alloc |
| Generated spec | `go generate .` | PASS pinned workflow; live server exposes429 |
| Whitespace/staging/migrations | `git diff --check`; `git diff --cached --stat`; `git diff --name-only -- migrations/postgres go.mod go.sum` | PASS; no staged, migration or dependency diff |

Tests use independent ownership/OID-checked PostgreSQL 17 databases. Removed: `m5b_verify_b99cc2adf73db1ca`, `m5b_verify_3ca4267cac9e08f3`, `m6_verify_9b79afff65fc3bbf`, `m7_verify_5153fad3c6594ef184f9c4e0b5126ff9`, `m7s_verify_3470c705be0505e389912777ee77ea48`. Development public catalog guards PASS; no migrator or provisioning command targeted mapatu_dev. No Docker volume operations.

## Acceptance evidence

- Authentication: successful login/session creation, unknown/wrong/inactive/no-password generic401, hash-only token storage, active-user/password write recheck, persistence rollback, cancellation, expiry/revocation/current/all logout and concurrent lock orderings PASS.
- Limiter: atomic concurrent threshold, fixed reset, distinct keys, bounded churn/heap, capacity rejection/recovery, cancellation, trusted/untrusted IPs, no credential/identifier logging and single decision/completion events PASS. Actual PostgreSQL case/space/non-ASCII equivalence throttles identically; successful login consumes the budget.
- Actual CLI/server HTTP: allowed/invalid preflight, auth no-store, generated429, Scalar route, real incomplete-header timeout, login/429/expiry recovery, repeated health and graceful signal shutdown PASS. Default middleware tests additionally cover exact/disallowed origins, known/chunked oversized input, invalid JSON/media and safe panic/error envelopes. Scalar JavaScript browser rendering was preserved by source/routes, not re-certified by a new browser session.
- Authorization: no/disabled membership, explicit grants, matching/wrong tenant/org/branch, no child traversal, no PLATFORM tenant bypass, active owners/assignments and immediate grant changes PASS in existing full RBAC suite. Six roles/eight permissions/nineteen mappings unchanged.
- Seed/bootstrap: list/named datasets/dependencies/collisions/idempotency/no unintended updates/no grants, environment/live-target/one-time/new-account/atomic/concurrent guards, hidden terminal and interruption restoration, actual fourteen-character rejection, Argon2/PLATFORM-only grant and no tenant bypass PASS. Boundary unit tests cover UTF-8 and maximum bytes.

## Documentation, risks and next action

All changed settings/contracts documented; README languages synchronized. Benchmarks are observations on this Apple M1 Pro, not SLAs or end-to-end latency guarantees. One additional DB normalization query is intentional; expired-record heap cleanup is synchronous and bounded, no workers are introduced. Counter state remains process-local, saturation/identifier pressure can temporarily impair login, and capacity telemetry shares the IP/identifier admission-stage reason rather than a distinct capacity metric. Those nonblocking limits are documented. HTTPS/proxy/fleet controls, production provisioning, session retention and future hierarchy writers remain deployment/follow-up work.

Build-cache sandbox failures and initial generator DNS denial were environment issues: checks were rerun with a writable temporary cache and approved network access. No product test failure was hidden.

Next: record final M8 review; ready for M9 review only, no M9 implementation.
