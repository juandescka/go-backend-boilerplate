# M6A-02 — Contract Recovery and Implementation

- **Status:** APPROVED by the user's recovered Module A contract in this chat; execution begins with contract recording/validation.
- **Based on:** [M6A-01 BLOCKED](../../reviews/tenancy-organization/M6A-01-review.md), findings F1/F2.
- **Scope:** Establish the [official six-table schema contract](../../../documentation/database/tenancy-organization-schema.md), then implement only tenants/organizations in versions 000004/000005. Preserve M6A-01 and the previous overall blocked checkpoint unchanged.
- **Baseline:** Three untracked prior M6 audit Markdown files; no staged or tracked changes, git diff --check clean. These are earlier work from this chat, not new implementation changes.

## Decisions and acceptance

Recovered design supplies all six tables' exact fields and scopes, resolving F1/F2. Reuse existing UUID/audit/FK behavior; raw business code/slug uniqueness, no speculative activation/coordinate coupling or recursive triggers. Longer hierarchy cycles remain a forbidden domain state requiring future transactional write validation; document the database enforcement boundary honestly.

Record and obtain a separate-context contract check before writing migration SQL. Verify tenants/organizations on newly created, ownership-checked PostgreSQL 17 databases through the actual migration CLI: exact types/null/defaults, unique scopes, parent tenant consistency/self-parent, actor FKs, conservative deletion, one-step down/reapply. Review M6A before beginning M6B.

## Documentation impact

Create one canonical Module A schema reference now. Align architecture/CLI/operations and bilingual READMEs with implemented SQL as the stages complete. Do not alter config/API specs/authentication contracts. No repositories or business APIs are needed without a consumer. Continue automatically through M6B/M6C/final only after preceding PASS.
