# M6A-01 — Tenant & Organization Implementation Input

- **Feature:** `tenancy-organization`
- **Date:** 2026-11-17 (Asia/Makassar)
- **Type:** Approved autonomous implementation, beginning with audit
- **Authorization:** User-supplied “MASTER PROMPT — M6: Tenancy & Organization — Complete Autonomous Implementation” in this chat.
- **Execution status:** Audit performed; implementation BLOCKED by missing approved schema.
- **Review:** [M6A-01](../../reviews/tenancy-organization/M6A-01-review.md)

## Objective and constraints

Execute M6A → review/PASS → M6B → review/PASS → M6C → final review, with routine in-scope fixes. Create only the six approved Module A tables through the existing PostgreSQL migrations. Preserve authentication, configuration, migration semantics and existing work. No staging, commits, seeding, administrative APIs, RBAC or speculative repositories. Destructive verification is authorized only on newly created disposable databases with verified targets.

The user explicitly requires: “jangan mengarang aturan bisnis atau tipe kolom yang belum ditetapkan” and “Untuk perbedaan material yang dapat memengaruhi kompatibilitas jangka panjang, tandai NEEDS DECISION.” Missing ERD alone is not the blocker; the absence of sufficient explicit replacement decisions is.

## Pre-execution Git baseline

Executed in the backend repository before edits:

```bash
git status --short
git diff --stat
git diff --cached --stat
git diff --check
git ls-files --others --exclude-standard
```

All completed successfully with empty output. Tracked, staged and untracked baselines were clean. Earlier milestone changes were committed baseline content; there were no existing user changes to integrate.

## Audit results and reusable conventions

| Area | Observed source / decision |
|---|---|
| Approved Module A ERD | Not found in workspace inventory, documentation, development inputs/reviews, migrations or domain models |
| Available ERD | `documentation/database/identity-session-schema.md` defines Module B users/sessions only |
| Migration source | `migrations/postgres`; existing complete pairs 000001 users, 000002 auth_sessions, 000003 session token-hash index; next available version is 000004 |
| UUID | Existing IDs are UUID with `DEFAULT gen_random_uuid()` |
| Audit | Existing `created_at`/`updated_at` are non-null TIMESTAMPTZ with CURRENT_TIMESTAMP defaults; nullable UUID `created_by`/`updated_by` reference users; writers maintain updated_at |
| Foreign keys | Existing immediate, nondeferrable NO ACTION update/delete behavior |
| Configuration | Required config.json, process > .env > JSON > defaults for supported overrides; POSTGRES_* configures Compose, DB_* configures Go |
| Startup | Registry/connector creates the shared PostgreSQL/GORM pool; no AutoMigrate or startup schema creation |
| Persistence | No current Module A consumer; no demonstrated need for Go models/repositories |
| Seeder | NOT IMPLEMENTED; Cobra registers server and migrate commands |

Historical PASS reviews were found for M4A, M4B, M4C, M5A, M5B, M5C and LOG-01. They were inspected as prior evidence, not rerun or promoted to current runtime verification.

## NEEDS DECISION

| Contract | Explicitly available | Missing decision |
|---|---|---|
| Tenant identity | UUID convention; is_active; nullable deactivation_reason TEXT and deactivated_at TIMESTAMPTZ; no duplicate status/subscription fields | Identity column names/types/requiredness and uniqueness (for example whether name/code/slug exists); activation default if any |
| Organization | Belongs to a tenant; multiple organizations per tenant | Identity fields/types/nullability/uniqueness; parent relationship yes/no and exact contract; approved master fields |
| Branch | Organization relationship; nullable description TEXT; is_head_office; multiple head offices allowed | Exact identity contract, tenancy representation and uniqueness scope |
| Department | Department/unit master; no employee assignments | Organization/branch scope, branch optionality, parent relationship and uniqueness |
| Position | Position master, separate from employee assignment | Ownership scope, unique identity, optional approved hierarchy/level |
| Work location | Future location-based attendance consumer, no attendance service | Ownership and branch optionality; approved descriptive/location fields, types and nullable/range rules |

Do not add name/code/slug, parent IDs, branch scope, coordinate/radius columns or tenant_id duplication by assumption. The required next input is an approved ERD or an explicit schema decision covering these contracts. A proposal, if requested, must remain PROPOSED until accepted.

## Implementation and verification plan after decision

1. Re-audit Git and the supplied design; preserve this blocked review history.
2. Create the next M6A input/review pair and new ordered tenant/organization migration pairs using the next versions then available.
3. Verify M6A columns, keys, constraints and up/down/reapply through the actual CLI on an owned disposable PostgreSQL 17 database; review before M6B.
4. Implement M6B only after M6A PASS, enforcing the approved scopes with minimal appropriate references/indexes; test cross-tenant invalid inserts/updates and multiple head offices.
5. After M6B PASS, verify all six tables, rollback boundaries preserving Module B/unrelated objects, reapply and proportional Go/runtime regressions; finish final review.

## Documentation impact

Current execution changes only development audit/review artifacts. No implemented schema/API/configuration contract changed, so README, technical documentation and generated API specs remain accurate and unchanged. After design resolution, add one canonical tenancy/organization schema document and synchronize affected architecture, operations, CLI and bilingual summaries with implemented behavior.

## Acceptance and stop condition

Audit and baseline recording are complete. Approved tenant/organization schema, migration implementation and runtime acceptance criteria are **NOT VERIFIED / BLOCKED**. Stop before M6A SQL; M6B/M6C depend on its PASS. No database commands, build, vet, tests, Docker lifecycle or seed operations were run for this checkpoint.
