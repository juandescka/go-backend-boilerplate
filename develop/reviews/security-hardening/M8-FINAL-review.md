# M8 Final Security Review and Consolidated Report

## A. Status

**M8 COMPLETE — PASS**, 2026-11-18 (Asia/Makassar), for requested backend foundation security hardening. No unresolved high/medium security defect found. This verdict does not certify a production deployment. Starting tree was clean, changes remain unstaged/uncommitted.

## B. Milestones and review method

| Stage | Status | Review |
|---|---|---|
| M8A | PASS | [M8A-01](M8A-01-review.md) |
| M8B | PASS | [M8B-01](M8B-01-review.md) |
| M8C | PASS | [M8C-01](M8C-01-review.md) |
| M8 Final | PASS | This review |

Separate-context reviewer `/root/security_review` reviewed tracked/new source, tests, generated artifacts, docs and secret metadata, independently ran focused M8A/M8B suites, and returned PASS for each stage and final scope. Primary agent ran full/default/race/opt-in PostgreSQL suites; the reviewer reviewed reported execution evidence rather than independently rerunning those DB suites. The primary agent then consolidated these findings. No review is misrepresented as wholly independent execution. No formal FAIL review was created or overwritten.

## C. Security findings

Retained working controls: Argon2id with bounded dummy verification, two simultaneous KDF slots, opaque random tokens/hash-only persistence, fixed session expiry and revocation, generic account-state denials, strict bearer parsing, active membership and explicit scoped grants, deny by default, guarded local first-admin bootstrap, safe structured logging/recovery, body readers/server timeouts and Scalar development gate.

Corrected weaknesses: rate-limit fields existed without runtime implementation; shared live limit eviction is forbidden and capacity is bounded; a one-record configuration is now rejected because it cannot hold a full IP/identifier pair. CORS accepted invalid preflights and failed to vary denied/no-Origin decisions. Authentication errors raised before handlers could be cached. Historical bootstrap accepted five characters, contrary to this request's fifteen-character policy. These are addressed without changing token architecture, permission catalog, role matrix, route surface, migrations or dependencies.

Added controls: atomic process-local fixed windows in existing shared/security; IP before decoder and authoritative PostgreSQL identifier normalization before KDF; generic429/Retry-After and existing security-event integration; exact preflight method/header validation, merged Vary, early namespace no-store, fifteen-character bootstrap input and both secret-name ignore variants.

Nonblocking deferred controls/risks: no distributed limiter, fleet-wide audit platform, WAF/Redis, new session renewal/retention policy or production bootstrap. Identifier and record-capacity pressure can temporarily deny legitimate login; IP NATs share budgets. Fixed windows can permit bursts across boundaries. Record-capacity rejection uses the admission-stage reason, not a distinct saturation event/metric. Correct HTTPS/proxy/shared ingress deployment remains necessary. Future organization hierarchy writers must enforce longer-cycle validation already deferred by M6.

## D. Rate limiting

| Aspect | Implemented behavior |
|---|---|
| Route | Only `POST /v1/auth/login`; probes, bearer checks and logout stay outside login throttling |
| Default IP policy | 10 admitted attempts per60 seconds; malformed requests reaching the login handler also count |
| Default identifier policy | 5 admitted valid-identifier attempts per300 seconds, including successful login and later operational/KDF failures |
| Compatibility | Existing failure-named JSON/env keys retained; all-attempt semantics explicitly documented, no failure-registration race or account-state reset |
| Keys | Namespace-separated SHA-256 of canonical client IP / DB-normalized identifier; fixed-size digests only, no passwords/tokens/raw account keys retained |
| Normalization | Existing PostgreSQL locale and `lower(btrim(...))`, exact same identity expression as lookup; one bounded query after IP admission |
| Client IP | Existing Gin trusted-proxy validation/canonical helper; empty default trusts no proxy and ignores spoofed forwarded headers |
| Storage | Registry-scoped mutex/map plus expiry min-heap; default100000 combined IP/identifier records, configured range2–1000000 |
| Expiration | Fixed deadline from first admission; no extension/reset on denial or success; synchronous lazy reclaim, no background goroutine |
| Rejection | 429 `RATE_LIMIT_EXCEEDED`, generic message, `Retry-After` rounds blocking remainder up to seconds; earliest expiry for new-key capacity rejection |
| Failure | Canceled/missing limiter/normalization failures fail closed with safe500, not unlimited admission; existing KDF saturation remains503 |
| Capacity | Never evict live limits; new keys wait until earliest expiry, known keys may use remaining budgets; retries are guidance, not reserved admission |
| Multiple instances | Each process/restart has independent/fresh counters; deployment needs shared abuse controls for fleet-wide guarantees |

Defaults are technical starting points rather than previously approved business throughput policy. Existing config/env sources and precedence remain unchanged. See [configuration](../../../documentation/configuration.md#login-abuse-policy). PostgreSQL normalization follows [official string-function semantics](https://www.postgresql.org/docs/17/functions-string.html); proxy handling follows [Gin trusted-proxy documentation](https://gin-gonic.com/en/docs/server-config/trusted-proxies/).

## E. HTTP security

- CORS retains explicit exact origins, default empty origins and credentials false, Authorization support and exposed Retry-After/request ID. Wildcard requires explicit noncredentialed config. Valid preflight204; invalid requested method/header403 `CORS_PREFLIGHT_NOT_ALLOWED`; rejected origins retain403 `CORS_ORIGIN_NOT_ALLOWED`. Vary merges Origin and OPTIONS selectors with existing values.
- Trusted proxies remain existing validated IP/CIDR config, empty by default. Production must specify real proxy hops and sanitize forwarding headers/restrict direct upstream access. No blanket trust added.
- Retain nosniff, DENY framing, no-referrer; auth namespace adds no-store/no-cache before early errors. No new CSP/COEP/CORP that could break Scalar. HTTP listener does not emit HSTS based on untrusted headers; verified HTTPS ingress owns HTTPS enforcement/HSTS.
- Global body default1MiB (configurable1024–52428800 bytes), body-bearing JSON methods, known and unknown lengths bounded; login decoder fixed16KiB effective minimum with global bound. Future upload handlers must establish their own file limits before any legacy multipart bypass is used.
- Retained header/read/write/idle/drain defaults5/15/30/60/15 seconds, configurable existing bounds; default header size65536 bytes. Actual incomplete-header timeout and graceful shutdown tested. See [Go server semantics](https://pkg.go.dev/net/http#Server).
- Generic safe error envelopes preserve400/401/403/413/415/422/429/500/503 distinctions. No SQL, private error, stack, hash, password or token is returned. One completion event per request; one authentication decision event when applicable.

## F. Authentication and RBAC verification

Tested valid and invalid/unknown/inactive/no-password login; equivalent identifiers and successful-attempt throttling; session creation/token-hash uniqueness/expiry; logout/current/all/revoked sessions; concurrent lock orderings, stale user/hash rechecks, atomic rollback and canceled waits. Existing fixed lifetime and bearer parsing preserved.

Existing PostgreSQL full RBAC suite PASS: active membership, explicit grant and assignment required; matching/wrong tenant, organization and branch; active owners; no child-organization traversal; no implicit platform tenant bypass; removed/revoked grants observed. Exact six roles, eight permission codes and nineteen role-permission pairs remain unchanged. No production business or privileged HTTP route added.

## G. Seeder and bootstrap security

Actual CLI/PTY and PostgreSQL tests PASS: named seeders, list/dependencies/idempotency/preserved edits/no accidental updates/no duplicate reference data/no seeder privilege; development/numeric-loopback/live-target/migration guards; existing-account refusal; one-time bootstrap including inactive prior grant/reserved user; atomic rollback/concurrent single winner; PLATFORM-only assignment without tenant access.

New bootstrap requires at least15 Unicode characters, valid UTF-8, maximum1024 bytes; shared Argon2id retained. Historical M7S review said five; historical reviews were preserved, implementation/current docs now follow requested fifteen. Existing login password compatibility stays unchanged. There is no default password, automatic first-user elevation, password file/environment flag or public bootstrap endpoint. Production bootstrap remains refused.

Metadata-only secret audit found actual `.bootstrap-admin-password` (reported typo `.bootstra-admin-password` absent), regular operator-owned file, mode0600. Both spellings are now excluded by Git and Docker. `git log --all` found no locally accessible history for either exact path. This is not proof about remote/deleted history or other copies. No secret contents were opened/printed/copied/uploaded/deleted or included in tests/reviews. Example JSON secrets are empty and example environment password is a placeholder.

Operator actions: retain a necessary credential in a trusted password manager, then explicitly remove an unnecessary local password copy. No automatic deletion. Rotate credentials if any actual exposure is known; no history rewrite was performed.

## H. Verification and data safety

Full commands, durations and cleanup evidence are in [M8C-01](M8C-01-review.md). All commands used a writable temporary Go cache.

| Check | Status | Evidence |
|---|---|---|
| Build | PASS | `go build ./...` and current CLI binary build |
| Vet | PASS | `go vet ./...` |
| Default tests | PASS | `go test -count=1 ./...`, repeated after final test changes |
| Rate limiter | PASS | deterministic clock, threshold/concurrency/churn/capacity/cancel tests; bounded heap/map |
| HTTP integration | PASS | actual Gin negative cases and actual CLI server requests/timeout/shutdown |
| Session regression | PASS | existing M5B disposable suite plus M5C lifecycle/locks, final7.85s run |
| Tenancy/RBAC | PASS | M6 full/M7 full PostgreSQL17 suites |
| Seeder/bootstrap | PASS | M7S actual command/PTY/PG suite including short-password refusal |
| Race | PASS | security/routes/middlewares/observability packages |
| Performance sample | PASS | known-IP concurrent326.3ns/op and saturated-new-key176.6ns/op, both0alloc; Apple M1 Pro observation, no SLA |
| Security config | PASS | invalid CORS/proxy/capacity/timeout guards and documented defaults/overrides |
| Documentation/OpenAPI | PASS | synchronized docs and pinned `go generate .`; live spec exposes429; Scalar routes remain development-only |
| Git/migration safety | PASS | whitespace clean, index empty; no migrations000001–000015 or go.mod/go.sum diff |

All fixture writes/migrations/down operations stayed in ownership/OID-checked disposable databases, each removed. No migration/provisioning operation targeted mapatu_dev; configured development catalog checks stayed read-only and unchanged. No persistent development test users, real administrator mutations, Docker volume deletion, local secret deletion or production credential change. Actual browser JavaScript rendering of Scalar was not rerun; source/gate/HTML/spec regression passed and existing CDN/CSP behavior preserved.

Initial default Go-cache sandbox denial and generator DNS denial were environment restrictions, resolved using writable temporary cache and permitted pinned generator network access. Required checks subsequently passed.

## I. Files changed

Production source:
- `cmd/bootstrap.go`
- `internal/handlers/authentication_handler.go`
- `internal/infrastructure/authentication/store.go`
- `internal/middlewares/http_hardening.go`
- `internal/registry/registry.go`
- `internal/shared/security/login_limiter.go` (new), `security_event.go`
- `internal/usecases/authentication/authentication.go`

Tests:
- `cmd/provisioning_integration_test.go`, `cmd/security_boundary_test.go` (new), `cmd/testdata/bootstrap_pty.py`
- `config/config_test.go`
- `internal/registry/authentication_integration_test.go`, `internal/registry/security_integration_test.go` (new)
- `internal/routes/authentication_test.go`, `internal/routes/login_abuse_test.go` (new)
- `internal/shared/observability/http_test.go`
- `internal/shared/security/login_limiter_test.go` (new, includes benchmarks)
- `internal/usecases/authentication/authentication_test.go`

Configuration/build exclusions: `config/config.go` (capacity minimum), `.gitignore`, `.dockerignore`. No local configuration or credentials changed, no new keys/dependencies.

Generated artifacts: `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`.

Documentation/developer guidance: every file listed in section J.

Development artifacts: three required inputs under `develop/inputs/security-hardening` and four required reviews under `develop/reviews/security-hardening`, including this consolidated report.

## J. Documentation changes and assessed unchanged files

Updated every following file:
1. `README.md` — active abuse controls, security links and bootstrap minimum.
2. `README.id.md` — synchronized Indonesian guidance.
3. `documentation/architecture.md` — actual boundaries, limiter placement/state/query and middleware order.
4. `documentation/configuration.md` — existing active settings/defaults/allowed ranges/env/production effects, HTTP/CORS/proxy/transport policy and limitations.
5. `documentation/operations.md` —429 troubleshooting, capacity/proxy/fleet assumptions and explicit safe operator secret handling.
6. `documentation/cli.md` — new fifteen-character requirement, same commands/flags.
7. `documentation/api/api-standard.md` — active429, new preflight403, Vary/no-store and stable auth/authz errors.
8. `documentation/api/authentication.md` — accurate deployed abuse controls, process locality and enumeration-resistant contract.
9. `AGENTS.md` — reusable preservation rules for normalized admission/proxy/bounds/secret metadata.
10. `AGENTS.id.md` — matching Indonesian rules.
11. `develop/TEMPLATE_PROMPT.md` — concise security-scope preservation instruction.

Generated docs listed in I follow annotations via existing generator; no manual generator edits. M8 inputs/reviews record execution rather than duplicate a new security guide. Existing bootstrap guide was updated, not duplicated.

Assessed and unchanged: `develop/FIRST_PROMPT.md` remains orientation-only; `develop/README-develop.md` retains workflow; existing M7/M7S reviews preserve historical evidence; identity/tenancy/RBAC schema documents require no change because schema, session format, scope containment and role/grant catalog are unchanged. `config.example.json` and `.env.example` already contain existing settings and safe placeholders; no new key/flag source requires examples. Session retention/production provisioning are still deferred, not silently activated.

## K. Recommended Git Conventional Commit

```text
feat(security): harden login abuse protection and HTTP boundaries

Add bounded process-local login admission limits with safe Retry-After responses.
Validate CORS preflights, prevent authentication response caching, and enforce
fifteen-character bootstrap passwords. Verify existing session/RBAC/provisioning
behavior with disposable PostgreSQL tests and synchronize security guidance.
```

Recommendation only: nothing staged or committed.

## L. Next milestone

Ready for **M9 — Foundation Integration & Final Readiness Review**, with explicit attention to deployment HTTPS/proxy provenance, shared multi-instance abuse protection, production provisioning/recovery and existing retention/hierarchy policy boundaries. M9 was not implemented. Suggested available model for that review: `gpt-6.2-sol`, high reasoning, because it spans multiple security boundaries and runtime evidence.
