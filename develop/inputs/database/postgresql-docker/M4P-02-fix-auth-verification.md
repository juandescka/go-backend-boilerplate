# M4P-02 — Fix PostgreSQL Password Verification

- **Feature:** `database/postgresql-docker`
- **Iteration:** `M4P-02`
- **Type:** Fix
- **Approval:** Covered by the user's autonomous M4P review/fix authorization.
- **Based on:** [M4P-01 review, F1](../../../reviews/database/postgresql-docker/M4P-01-review.md#f1--in-container-loopback-tcp-does-not-establish-password-verification)

## Objective and finding

Correct the misleading CLI authentication check: container loopback TCP may use `trust`, so supplying `PGPASSWORD` does not prove password verification. Retain the approved Compose baseline.

## Scope and files affected

Change only the authentication-check explanation and target in `documentation/cli.md`, using Compose hostname `postgres` for the non-loopback bridge path. Document a wrong-password negative control using a fixed non-secret sentinel. Preserve the M4P-01 review and all backend/production resources.

## Verification and acceptance

- Complete the original runtime plan after the image pull: healthy PostgreSQL 17, actual localhost port, named volume, read-only SQL/user/database/version, stop/start and unchanged database system identifier.
- Inspect safe `pg_hba_file_rules` fields; confirm service-hostname SQL succeeds with the configured password and fails with an intentionally wrong password. Do not log actual credentials or full inspect/config output.
- Run Compose validation, whitespace and backend/environment preservation checks; independently review this fix and the full M4P acceptance criteria in `M4P-02-review.md`.

## Documentation impact and constraints

CLI authentication instructions only; other M4P documentation already describes the correct separation of Compose and Go/MySQL. No source, dependency, schema, migration, image customization, credential reset or volume deletion.
