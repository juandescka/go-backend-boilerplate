# M7D-01 — Integration review

- Status: **BLOCKED for full business authorization acceptance; integrated foundation checks PASS**.
- Input: [M7D-01](../../inputs/tenant-access-rbac/M7D-01-integration-verification.md).
- Scope: five M7 tables, live authorization mechanics and M5/M6/HTTP operational regressions. Final separate-context integrated source/evidence review found no new runtime/schema defect (foundation PASS, overall business acceptance BLOCKED); primary agent ran all runtime commands. The [final report](M7-FINAL-review.md) records attribution and remaining decisions.

## Evidence

- `GOCACHE=/private/tmp/hris-m7-gocache go build ./...`: PASS; current executable built at /private/tmp/hris-m7-backend.
- `GOCACHE=/private/tmp/hris-m7-gocache go vet ./...`: PASS.
- `GOCACHE=/private/tmp/hris-m7-gocache go test -count=1 ./...`: PASS. Opt-in live tests skip in this run and were executed separately.
- `GOCACHE=/private/tmp/hris-m7-gocache M7_INTEGRATION=1 M7_STAGE=full go test -count=1 -run '^TestTenantRBACPostgresIntegration$' -v ./internal/infrastructure/migration`: PASS, test0.67s/package1.037s. Database m7_verify_87566fc387774346a84e03a4dc3f03d2 cleaned up after exact schema/defaults,20FK/9index/no-trigger checks, integrity/authorization, rollback14→9/reapply14 and M5/M6 preservation. Earlier A/B/C runs also PASS and cleaned up.
- `GOCACHE=/private/tmp/hris-m7-gocache M5B_INTEGRATION=1 M5B_CLI_BINARY=/private/tmp/hris-m7-backend go test -count=1 -run '^TestAuthenticationPostgresIntegration$' -v ./internal/registry`: PASS, test4.47s/package5.014s. Database m5b_verify_d37cb32a3463ad2d cleaned up. Actual current CLI applies1–14; login/me/logout/all, expiration/revocation, ownership, both concurrency orderings, lock-wait revalidation, rollback/cancellation, safe errors/logs, health/readiness, startup-without-schema and Scalar spec preserved. Existing original3→2 token-index rollback retained after later-module down; reapply14 preserves identity rows. Development catalog unchanged through read-only checks; no migrator targeted it.
- Whitespace, gofmt, migration pairs and local Markdown links: PASS. Initial link-check regex misread an inline Go generic signature as a Markdown link; corrected checker excludes code spans, no product defect.

## Security assessment

No observed runtime bypass: exact tenant/organization/branch matching, independent membership, active states, explicit mapping, platform separation, no role-name inference, no hierarchy inheritance and immediate next-query revocation. Parameterized SQL, safe401/403/500, success-only owner context and middleware aborts. Synthetic positive permission data never survives disposable cleanup. Existing migrations/config/secrets/dependencies/generated OpenAPI unchanged; no production management endpoint or default privilege.

## Findings and next action

B1/B2 remain material acceptance blockers: business permission matrix and required inheritance semantics have no approved contract. Fixture mechanics do not substitute for positive approved business-policy tests. Grant administration remains unexposed pending authority rules. Documentation synchronized; no configuration/API change. Proceed to final separate-context review and consolidated **M7 BLOCKED** report; obtain the missing contracts before full acceptance. Do not implement M8 or migrate development automatically.
