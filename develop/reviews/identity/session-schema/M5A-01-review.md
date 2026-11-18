# M5A-01 — Identity & Session Schema Review

- **Feature:** `identity/session-schema`
- **Iteration:** `M5A-01`
- **Input:** [M5A-01 implementation](../../../inputs/identity/session-schema/M5A-01-implementation.md)
- **Date:** 16 November 2026 (Asia/Makassar)
- **Verdict:** **PASS for M5A — Identity & Session Schema; ready for separately scoped M5B work**
- **Review method:** Separate-context reviewer `/root/m5a_review` independently inspected instructions, input, actual SQL, test harness, changed references, unchanged migration/runtime callers and preservation fingerprints. Build, PostgreSQL, CLI and integration results were executed by the primary implementer and are explicitly attributed below; the reviewer did not repeat migrations, Docker operations, builds or tests. The reviewer edited only this review.

## Scope and actual changes

Four SQL files supply two complete six-digit sequential migration pairs in the existing PostgreSQL source: `000001_create_users` and `000002_create_auth_sessions`. The existing `.gitkeep` and migration infrastructure are retained. Users are created first, with optional self-referencing audit actors; sessions follow, with a required user and optional audit actors. Each down file drops only its paired table, with no CASCADE or schema reset.

`users` has exactly the 11 approved columns, UUID primary key with built-in `gen_random_uuid()` default, nullable password hash, explicit required activation value and all agreed TIMESTAMPTZ audit/lifecycle fields. One unique expression index enforces global `lower(btrim(email))` identity. There is no tenant/company/organization/branch key, employee identity, status or deleted-at column.

`auth_sessions` has exactly the 13 approved columns, UUID primary key/default, required user/token hash/expiration, optional INET device metadata and complete audit fields. A nonunique user index supports several sessions per user. There is no session is_active, refresh-token table, token generator or lifecycle implementation.

All five foreign keys reference users, are immediate/nondeferrable, and use NO ACTION for update/delete. Actor fields remain nullable for bootstrap/system writes; deletion cannot silently cascade into session history. Initial created_at/updated_at values default to CURRENT_TIMESTAMP; subsequent updated_at changes remain the writer's responsibility. No trigger, extension or Go model without a consumer is added.

The new opt-in integration test uses actual repository migrations and the existing Cobra binary, a random freshly created database, effective existing configuration and exact-name ownership/OID-checked cleanup. Development-target access is limited to catalog SELECTs within a read-only transaction. One official schema guide and the existing architecture/CLI/operations/bilingual README references document current behavior and deferred decisions. The paired implementation input records the audit, implementation decisions and verification strategy.

## Independent static evidence

| Reviewer check | Result |
|---|---|
| Compare exact SQL columns, nullability, defaults, PK/FK/index definitions with approved prompt/input | PASS; exact users 11 / sessions 13, no extra business schema |
| Inspect normalized-email index and documented lookup contract | PASS; one global lower/btrim expression; ordinary edge-space and locale behavior stated explicitly |
| Inspect migration pairs and unchanged runner/commands | PASS; users before sessions; down uses existing Steps(-1); each down owns one table; no automatic force/startup migration |
| Inspect disposable test target construction and cleanup ordering | PASS; CREATE must succeed; random name never reused; DB_NAME overridden in isolated CLI environment; owner/OID checked before exact-name DROP; no force-drop of active connections |
| Read schema guide and changed references | PASS; no login/seeder/tenancy/RBAC availability claim; Swaggo/Scalar/Docker instructions preserved; timestamp/retention/index limits explicit |
| Independently recompute SHA-256 against `/private/tmp/m5a-before.json` | 90 baseline files checked; only README EN/ID, architecture, CLI and operations differ; none missing |
| Protected-file preservation | Runtime, configuration, dependencies, Docker, local `.env`/`config.json`, generated API files and previous history remain byte-identical |
| `git diff --check` | Exit 0 |
| New input/schema Markdown link check | All existing targets resolve; the pending review target is fulfilled by this artifact |

The reviewer read the primary's secret-free actual-execution summary at `/private/tmp/m5a-verification-evidence.md`. This is attributed verification evidence, not a claim that the reviewer executed its commands independently.

## Actual verification supplied by implementer

| Check | Actual result and evidence |
|---|---|
| `go build ./...` | PASS, exit 0 with `GOCACHE=/private/tmp/hris-go-cache` |
| `go vet ./...` | PASS, exit 0 |
| `go test -count=1 ./...` | PASS, all packages; opt-in database checks run separately below |
| `docker compose ps` | Existing postgres:17 healthy, localhost publication 5433→5432; no container/volume lifecycle mutation |
| Effective loader target | Existing loader resolves 127.0.0.1:5433, mapatu_dev/mapatu, SSL disable, 5-second query timeout; credentials withheld; no hardcoded test port |
| Current CLI | Backend built at `/private/tmp/hris-m5a-backend`; migrate help and force help exit 0 without DB access |
| `M5A_INTEGRATION=1 M5A_CLI_BINARY=/private/tmp/hris-m5a-backend go test -count=1 -run '^TestIdentitySessionSchemaIntegration$' -v ./internal/infrastructure/migration` | PASS; test 0.66s, package 1.207s |
| Actual disposable CLI lifecycle | Fresh version none → up → clean version 2 → repeat up no-change → one down to 1 retaining users/data → second down to none → reapply clean 2; no force |
| Catalog contracts | Exact 11/13 columns, sizes/nullability, UUID PK/defaults, explicit activation, TIMESTAMPTZ, INET, five NO ACTION FKs and four total indexes verified |
| Integrity positive/negative checks | Duplicate normalized email INSERT/UPDATE and duplicate PKs rejected with 23505; all required fields reject NULL with 23502; all five invalid user/actor FKs rejected with 23503; valid references accepted |
| Optional values and session relation | Nullable password/audit/lifecycle/device fields accepted; IPv4/IPv6 accepted; multiple sessions per user accepted |
| Retention and audit behavior | User deletion rejected while referenced; sessions preserved; updated_at remains writer-managed |
| Rollback/reapply isolation | Unrelated sentinel survives both down steps and reapply; no extra business tables, seed rows or unnecessary extensions; no migration connections remain |
| Owned database cleanup | Final `m5a_verify_47b55539ac6a7c95` ownership/OID checked and dropped; absence verified |
| Development preservation | Read-only before/after public catalog matches; no migrator opened there; final read-only SQL on mapatu_dev/mapatu reports public table count 0 and remaining M5A database count 0 |
| `M4A_INTEGRATION=1 go test -count=1 -run '^TestPostgresSharedPoolIntegration$' -v ./internal/registry` | PASS; test 0.02s, package 0.469s; shared-pool/readiness/cleanup regression uses read-only SQL |
| Secrets and preservation | Configured password/JWT absent from scanned nonignored content; baseline protected files unchanged; updated/new Markdown targets checked |

The first default-cache build was prevented by the filesystem sandbox; the writable temporary Go cache resolved it without a source/dependency change. Two early integration attempts found verifier SQL errors: internal pg_class.relkind needed a text cast before any disposable database was created, and an index assertion contained an extra parenthesis. The latter attempt's owned `m5a_verify_9c1032d3d5443ec7` was safely removed. Both assertions were corrected before the final complete successful run; SQL migration files required no correction. These attempts are not counted as successful verification. No historical FAIL review was overwritten and no formal review/fix iteration is invented for these pre-review harness corrections.

## Acceptance assessment

| Required review area | Assessment |
|---|---|
| M4C prerequisite and preservation | PASS — prior PASS/source inspected; protected files unchanged; fresh Go and read-only shared-pool checks pass |
| Approved Module B compliance/global identity | PASS — supplied approved baseline used; no separate final ERD found; users remain distinct from employees/membership |
| Users schema | PASS — exact columns/types/required and nullable fields, UUID and explicit activation |
| Auth sessions schema | PASS — exact columns/types, required hash/expiration, optional metadata, no is_active |
| Email uniqueness | PASS — single global normalized unique expression index, insert/update failures observed |
| Password/token security | PASS for schema scope — hash-only storage contracts, nullable password not authorization, no generator/default account/plaintext fixture |
| Audit columns and timestamp responsibility | PASS — complete agreed columns, nullable actor FKs, TIMESTAMPTZ, documented writer responsibility |
| Foreign keys and retention | PASS — valid/invalid actor/user checks and conservative deletion observed; no cascading history deletion |
| Indexes | PASS — two PKs, normalized-email uniqueness and required per-user lookup; future workload indexes explicitly deferred |
| Migration up/down/reapply | PASS — actual current CLI against fresh owned disposable, exact one-step semantics and unrelated-object preservation |
| No unintended seeder or scope expansion | PASS — no auth API, tenancy/employee/RBAC/provider schema, model scaffolding or business seeds |
| Documentation accuracy | PASS — single official schema guide, accurate CLI/architecture/operations and aligned README EN/ID |
| Actual verification and cleanup | PASS — successful full integration, ownership-checked cleanup, final development catalog and no remaining M5A databases |
| No breaking changes outside scope | PASS — no runtime/package/configuration/CLI/dependency/Docker change |
| Final review | PASS — independent static review plus explicitly attributed executed evidence |

## Findings, limitations and next action

No remaining actionable finding within the approved M5A scope. One formal iteration: **M5A-01 PASS**, no post-review fix iteration needed.

Email normalization intentionally covers ordinary U+0020 edge spaces and database-locale lowercasing; full syntax, other whitespace, Unicode/IDNA and provider policy require M5B decisions. Hash columns cannot prove that writers supplied cryptographic hashes. Token-hash lookup/uniqueness, expiry/revocation and nullable actor indexes remain deferred to real query contracts. Hard-deletion, anonymization and retention policy remain undecided; NO ACTION preserves references meanwhile. These are documented design boundaries, not unimplemented M5A acceptance criteria.

No production TLS/deployment, new authentication flow, full HTTP outage exercise, race run, Docker restart or dependency regeneration was repeated because this change adds SQL/documentation and an opt-in test without altering runtime code. Existing M4C evidence remains historical; fresh build/tests/shared-pool checks and independent fingerprints provide current preservation evidence.

Migrations were applied **only to disposable verification databases**. `mapatu_dev` still has no public tables; this review does not claim users/auth_sessions were deployed there. Schema is ready for **M5B — Authentication Foundation**, subject to its own scope and deliberate migration application to the correct target. **Seeder CLI: NOT IMPLEMENTED.** No automatic Git commit or M5B implementation was performed.
