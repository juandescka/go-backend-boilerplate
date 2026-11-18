# M9B-01 — Integration Review

- **Input/commands:** [M9B-01 integration](../../inputs/foundation-readiness/M9B-01-integration.md)
- **Date:** 2026-11-18 (Asia/Makassar)
- **Verdict:** **PASS**.
- **Method:** Primary executed all PostgreSQL suites; separate-context `/root/foundation_review` reviewed their supplied output/source and independently ran all eight `TestPermissionBoundary` cases (0.224s), formatting and whitespace. DB execution is not attributed to reviewer.

## Actual verification

All commands in the input were executed with `GOCACHE=/private/tmp/hris-m9-go-cache`. Build/vet/default suite exit0. Current CLI built from unchanged production source. Initial sandbox M7 connection attempt failed before database creation; local network escalation allowed the successful run below. This was an environment restriction, not an approval-review rejection or hidden database failure.

| Existing suite | Test duration | Owned disposable database, removed | Coverage |
|---|---:|---|---|
| M4B migrator lifecycle | 5.24s | `m4b_verify_1dcd91433e22db28` | empty/source/dirty state, actual Cobra commands, one-step rollback, timeout/advisory locks/concurrency; corrected OID cleanup |
| M5A identity/session catalog | 0.76s | `m5a_verify_ed84abb4ea095808` | exact columns/defaults, five FKs, normalized uniqueness, audit, rollback/reapply/unrelated data |
| M6 full tenancy catalog | 0.98s | `m6_verify_d765f258005ede8d` | six tables, 24 FKs/19 indexes, cross-owner rejection, optional branch, multiple heads, down9→3/reapply |
| M7 full RBAC | 1.22s | `m7_verify_90873c2192bb4464b553a91c537e6f1f` | five tables, 20 FKs/9 indexes, exact 6/8/19 catalog/matrix, scoped isolation, stored-resource HTTP fixture, rollback15→9/reapply |
| M5B auth + M5C/M8 helpers | 9.58s | `m5b_verify_2533430017106750` | actual registry/current migrations1–15, login/me/logout/all, locking/cancellation/rollback, real CLI HTTP security and shutdown |
| M7S provisioning CLI/PTY | 2.18s | `m7s_verify_1a9dc10f770f467cdf6ccb6a7ae21889` | named registry, dependencies, reruns/conflicts/preserved edits, no seed privilege, hidden/15-character passwords, bootstrap atomicity/concurrency |

All six cleanup paths verified created identity/owner/OID before exact-name DROP. M5A/M6/M5B reported unchanged development catalogs from read-only snapshots. No migrator/provisioning command targeted mapatu_dev. M4B synthetic fixture exercises explicit force semantics after inspecting synthetic state; no repository migration error was bypassed with force.

## Security acceptance

- Login known/wrong/unknown/inactive/passwordless, normalized identifiers, hash-only unique tokens, expired/revoked/inactive401, malformed bearer and context cancellation pass.
- Membership cannot imply permission; PLATFORM never bypasses tenant. Active owners/membership/assignment, exact grant, tenant/organization/direct branch containment, no child inheritance, unrelated branch, removed mapping and unknown permission pass.
- Stored work-location fixture looks up resource ID and owners; forged tenant/resource combinations deny. Real Employee/Employment APIs do not exist and are not certified.
- Existing M8/default tests cover allowed/denied origins, valid/invalid preflight, spoofed forwarding headers, body limits, Retry-After and no-store. Actual server helper covers login/throttle/window recovery, header-read timeout, health/ready/docs/spec and graceful interrupt shutdown.
- Existing lifecycle suite covers cross-user logout isolation, both creation/logout-all lock orders, expiry/revocation/activation rechecks and atomic rollback.
- Provisioning confirms PLATFORM-only administrator without tenant authority and unchanged 6/8/19 reference counts; no real credentials used.

## Actual changes, findings and documentation

Extended the existing middleware permission table with global header/request ID/error-envelope/completion correlation assertions. All eight cases pass; no new success-path framework or business route. Production source and applied SQL unchanged. No unresolved integration defect; severity: none blocking M10. Runtime evidence is bounded to local PostgreSQL17 and current foundation routes/fixtures.

## Next action

Continue M9C operational/security assessment and synchronize current documentation. Full default suite will run again after the final test edit; no DB rerun is needed for prose-only changes.
