# M9C-01 — Operational and Security Readiness

- **Date:** 2026-11-18 (Asia/Makassar)
- **Status:** Approved by M9 request; execution follows M9B PASS.
- **Objective:** Assess development operations/configuration/log privacy and deployment limitations; synchronize existing guides without changing runtime policy.

## Verified operations

Compose model validates and existing postgres:17 is healthy on loopback5433. No container restart/volume mutation needed. Current CLI help for server, migrate up/down/version, seed run and bootstrap super-admin exits0; `seed list` returns four registered names from `/private/tmp` with no local config/DB. M9B actual CLI tests verify version/up, named seed and bootstrap on disposable targets. Actual HTTP server verifies health/ready/docs/spec, timeout and graceful shutdown. Scalar browser CDN execution is not rerun; current route/spec and environment gate tests pass.

Loader remains process > .env > required JSON > defaults for supported overrides, with JSON-only HTTP port; Compose POSTGRES_* is separate. One shared runtime pool, bounded per-operation contexts, explicit migrations, active limiter and safe default CORS/proxy policy remain. Legacy JWT/cookie validators do not activate JWT/cookies. Database TLS is intentionally local-disable here; no production TLS certification.

## Risk register

| Finding | Severity | Current control | Impact / classification | Required before |
|---|---|---|---|---|
| Older M4B cleanup lacked live identity check | Medium | Owner/OID capture/recheck and removal verification now implemented/tested | RESOLVED | Completed in M9A/B |
| Stale security/package/spec documentation | Low | Existing guide corrections | RESOLVED when M9C docs reviewed | M9C acceptance |
| HTTPS/DB TLS/proxy provenance not deployed here | High if publicly exposed | Local loopback DB, no trusted proxies by default, explicit SSL setting | BLOCKING FOR PRODUCTION | Target ingress HTTPS/HSTS, exact trusted hops/header sanitation/direct-upstream restriction, verified DB TLS |
| Login counters independent per process/restart | Medium | Atomic bounded local counters and KDF cap | BLOCKING FOR PRODUCTION where shared/fleet abuse guarantees are needed | Reviewed shared ingress protection and capacity policy; no Redis added |
| Production admin provisioning/recovery undefined | High operational | Development-only hidden-input one-time bootstrap | BLOCKING FOR PRODUCTION | Approved provisioning/recovery procedure; never default administrator |
| Session retention/cleanup/renewal undecided | Medium operational | Fixed expiry (default15m), revocation, retained history, hash-only storage | DEFERRED NONBLOCKING for M10 | Production retention/capacity/privacy and access-recovery decisions |
| Longer organization/department cycles not blocked in SQL | Medium at future writer | Same-owner FKs and direct self-cycle CHECK; no public hierarchy writer | DEFERRED NONBLOCKING for M10 foundation | Transactional/concurrency-safe validation before any hierarchy writer |
| Business ownership resolution is fixture-tested only | High if omitted in new APIs | RequirePermission after bearer; real stored-location test | DEFERRED NONBLOCKING for foundation; mandatory M10 gate | Employee/Employment API owner lookup, scoped SQL, explicit approved grants, isolation tests |
| Legacy multipart/DTO/pagination helpers lack new business consumers | Low current | No registered upload/list routes; limits documented | DEFERRED NONBLOCKING | File-aware limits/DTO/pagination policy and tests before first consumer |

There is no unresolved BLOCKING FOR M10 defect. No hierarchy-management API exists; demo seeder inserts one root organization and cannot reparent it. Arbitrary DBA writes are outside the writer boundary.

## M10 considerations

Reuse global identity, owner-safe organizations/branches, explicit scoped authorization, migration runner, seed registry and disposable tests. M10 must define its own resource ownership/permission contract without assuming existing grants permit Employee operations. Preserve multiple employments per employee, independent employee activation, employment assignment records, manually supplied nullable employee number and tenant-scoped uniqueness. No Employee ERD/tables/endpoints or new grants are added here.

## Documentation work

Synchronize both README status/spec/package inventories, architecture package map, and add a compact existing-operations checklist/M10 boundary linked to one canonical final review. Correct identity security wording already done. Assess configuration/CLI/API/database/agent/workflow guides and leave accurate contracts unchanged. Evidence and actual verdict belong in matching review and canonical M9 final record.
