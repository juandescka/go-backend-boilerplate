# M7S Final Review

**M7S COMPLETE — PASS for local development provisioning.** The clean starting tree was at `10eb799`, which contains the prior M7 foundation and its PASS review.

| Stage | Verdict                                                                                                              |
| ----- | -------------------------------------------------------------------------------------------------------------------- |
| M7S-A | [PASS](M7S-A-01-review.md). Named Cobra CLI and static registry.                                                     |
| M7S-B | [PASS on re-review](M7S-B-02-review.md). The [first verification failure](M7S-B-01-review.md) remains in the record. |
| M7S-C | [PASS](M7S-C-01-review.md). Disposable security checks and full regressions.                                         |
| Final | PASS. Production bootstrap and recovery remain separate work.                                                        |

## Commands and datasets

Use the verified local development configuration and replace the example target with the effective endpoint.

```bash
go run main.go migrate up
go run main.go seed --help
go run main.go seed list
go run main.go seed run demo-tenant --target 127.0.0.1:5433/mapatu_dev
go run main.go bootstrap super-admin --target 127.0.0.1:5433/mapatu_dev
```

| Seeder              | Dataset                                         | Dependencies                  | Owned rerun |
| ------------------- | ----------------------------------------------- | ----------------------------- | ----------- |
| `dev-users`         | Inactive global demo user with no password      | None                          | No changes  |
| `demo-tenant`       | One demo tenant                                 | None                          | No changes  |
| `demo-organization` | One company in the demo tenant                  | `demo-tenant`                 | No changes  |
| `demo-memberships`  | One membership linking the demo user and tenant | `dev-users` and `demo-tenant` | No changes  |

An existing matching business key with a different reserved ID produces a conflict. The CLI reports a missing dependency without creating it. Migrations remain the source of the six fixed roles, eight permissions and 19 grants.

## Bootstrap security and verification

The trusted local operator confirms the configured target explicitly. Effective environment must be `development` and the host must be a numeric loopback address. The command pins and verifies the live peer endpoint, database and role, and requires a clean migration version of at least 15. The operator independently verifies local cluster provenance. The socket check cannot attest where a tunnel terminates.

The operator enters an email, display name and a confirmed password through a TTY. The implementation uses `golang.org/x/term` for hidden input and the existing Argon2id code for the hash. The password policy requires at least 5 characters and at most 1024 UTF-8 bytes. A single transaction serializes bootstrap, refuses an existing normalized email or prior platform SUPER_ADMIN assignment including an inactive one, and inserts one new user and one PLATFORM assignment with null tenant, organization and branch. The reserved bootstrap user protects against deletion of only the assignment. The guard relies on preserving either that user or the prior assignment. Direct DBA deletion of both falls outside the CLI guarantee. Production provisioning, access recovery and assignment management remain deferred.

The actual CLI passed list, named run, dependency, idempotency, environment and target guards. The pseudo-terminal verified hidden password input and restored terminal echo on interruption. PostgreSQL verified one winner under concurrent bootstrap, rollback on forced assignment failure and the existing-account and inactive-assignment refusals. Login for the new administrator succeeded with `platform.manage` and tenant access denied. Build, vet, default tests, M7 RBAC and authentication, session, migration CLI, health/readiness and Scalar checks passed. See the [execution evidence](M7S-C-01-review.md).

## Changed files and future work

- Source and dependency files include `cmd/seed.go`, `cmd/bootstrap.go`, `internal/infrastructure/seeding/{target,registry,datasets,bootstrap}.go`, `go.mod` and `go.sum`.
- Tests are `cmd/provisioning_integration_test.go` and `cmd/testdata/bootstrap_pty.py`.
- Updated documentation includes both READMEs, `documentation/{cli,operations,configuration,architecture}.md`, `documentation/api/authentication.md`, the identity, tenancy and RBAC schema references under `documentation/database`, and `develop/TEMPLATE_PROMPT.md`. `AGENTS.md` and `AGENTS.id.md` were assessed. The reusable seeder rule belongs in the template and the existing agent guidance already covers safe mutation, credential handling and verification. The API standard needs no new entry because M7S adds developer CLI commands.
- Development inputs and reviews are under `develop/inputs/seeder-bootstrap` and `develop/reviews/seeder-bootstrap`.

To add a dataset, choose a unique name and deliberate identity keys, implement a transaction-scoped dataset function, and register the name and description in the static map. Verify dependencies and owned no-op reruns using a disposable database. Update the existing [CLI authoring instructions and dataset table](../../../documentation/cli.md#add-a-named-seeder) and [operations guidance](../../../documentation/operations.md#local-seeding-and-bootstrap).

Suggested commit `feat(cli): add development seeders and local admin bootstrap`. Nothing was staged or committed. Prior migrations and M7 reviews were preserved. The development database was not seeded or bootstrapped.

M7S is ready for the separately scoped M8 Security Hardening and Rate Limiting work. Production availability remains outside this verdict.
