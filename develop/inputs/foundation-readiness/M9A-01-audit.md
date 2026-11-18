# M9A-01 — Foundation Contract Audit

- **Date:** 2026-11-18 (Asia/Makassar)
- **Status:** Executed; approved by the user's M9 master request.
- **Scope:** Audit current runtime, identity/session, tenancy, authorization, provisioning and security contracts before implementation. Preserve migrations, privileges, local data, secrets and Git index.

## Baseline and reading

Initial `git status --short`, `git diff --stat`, `git diff --cached --stat` and `git diff --check` were empty/clean. No pre-existing changes require separation.

Read AGENTS.md/AGENTS.id.md, FIRST_PROMPT/template/workflow, both READMEs, architecture/configuration/operations/CLI/API guides, three database contracts, current source and existing integration suites. Historical anchors: M1-02; M3B-01 and M3C-02; M4C-01 and M4P-03; M5A/B/C; M6-FINAL-02 (supersedes blocked final); M7-FINAL-02/M7E (approved policy supersedes blocked final); M7S-FINAL; M8-FINAL. Historical MySQL, missing seeders and five-character bootstrap descriptions reflect older milestones, not current contracts.

## Integration contract inventory

| Contract | Producer | Consumer | Current status | Evidence | Risk |
|---|---|---|---|---|---|
| One pool, explicit migration lifecycle | PostgreSQL connector/registry | readiness, metrics, auth and authorization | Consistent in source; runtime pending M9B | ConnectDB, Registry.Close, server lifecycle | Connectivity readiness does not certify schema |
| Global typed identity | bearer middleware/auth store | auth handlers, RequirePermission | Consistent | private identity context, active user/session join | Authentication alone grants nothing |
| Email/password/token storage | PostgreSQL lower(btrim), Argon2id, token generator | login/session lookup | Consistent | normalized email index, hash-only session writes, unique token index | TLS/client logging deployment-dependent |
| Session lifecycle | auth transactions | bearer requests/logout | Consistent | user→session locks, fresh clock after waits | Already authenticated requests may finish |
| Tenant and organization ownership | SQL versions 4–9 composite FKs | authorization owner tuple/future writers | Consistent | schema suites; multiple head offices valid | Longer cycles require future writer validation |
| Active membership and explicit grants | SQL versions 10–15, access store | RequirePermission | Consistent | six roles/eight permissions/19 pairs; direct containment only | Resolver must fetch stored resource ownership |
| Platform separation | fixed SUPER_ADMIN PLATFORM grant | platform authorization/bootstrap | Consistent | platform.manage only; no tenant bypass | Production provisioning/recovery deferred |
| Named fixtures vs privilege bootstrap | seed registry and bootstrap transaction | explicit Cobra commands | Consistent | local target guard, atomic writes, one-time lock | Loopback alone cannot prove tunnel provenance |
| Login admission | trusted Gin IP and DB-normalized identifier | bounded process-local limiter then KDF | Consistent | IP before decoder, identifier before password work | Counters independent per replica/restart |
| HTTP boundary/errors/logging | request ID/recovery/headers/CORS/body middleware | routes, completion metadata | Consistent | early auth no-store, safe envelopes | Ingress HTTPS/forwarding requires deployment evidence |
| Disposable migration cleanup | M4B test CREATE/name flag | cleanup DROP | **Mismatch F1** | No live owner/OID recheck unlike later suites | Cannot run this destructive fixture under M9 policy until corrected |
| Active abuse-control documentation | M8 implementation | identity/session schema readers | **Mismatch F2** | Deferred-work paragraph still lists login-abuse controls | Misleads developers about current protection |

## Findings and acceptance

F1: add same-name guard, record created database OID/owner, recheck immediately before exact-name DROP, verify removal. Test-only correction; no migrator/runtime change.

F2: remove obsolete deferred label and point to active process-local protection and remaining shared-ingress requirement.

No demonstrated production-code defect or new privilege is required. Runtime verification remains pending; historical PASS is not current execution evidence. Run existing disposable suites in M9B, augment only actual gaps. Review before proceeding.
