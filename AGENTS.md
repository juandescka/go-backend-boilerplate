# Mapatu Backend — Agent Instructions

[English](AGENTS.md) | [Bahasa Indonesia](AGENTS.id.md)

> Scope: `backend/hris-core-service`. These instructions guide coding agents working in this repository. They do not authorize changes by themselves: follow the active user request and approved development input.

## Project context

Mapatu is a Go backend foundation for a multi-tenant HRIS. The design preference is **pragmatic Clean Architecture**, readable idiomatic Go, minimal abstraction, and preserving the existing repository layout. The M0 audit identified incomplete operational wiring; do not confuse a declared component with a working one.

Start from these sources:

- [`README.md`](README.md): overview and quick start.
- [`documentation/architecture.md`](documentation/architecture.md): current package map and architectural conventions.
- [`documentation/configuration.md`](documentation/configuration.md): current configuration behavior.
- [`documentation/operations.md`](documentation/operations.md): health/readiness and lifecycle.
- [`documentation/cli.md`](documentation/cli.md): CLI commands and verification status.
- [`develop/README-develop.md`](develop/README-develop.md): input → execution → review → fix workflow.

**The actual source code is authoritative for observed behavior.** Documentation can be incomplete or outdated; report discrepancies rather than silently claiming unimplemented behavior exists. Proposals are not implemented features.

## Before making changes

1. Read the relevant files, nearby callers, and the active approved input in `develop/inputs/`, if one exists.
2. State the expected scope and identify files affected.
3. Preserve current directory/package structure, naming, API contracts, and existing behavior where possible.
4. **Propose structural changes before applying them.** Moving/renaming packages, adding layers, changing public API contracts, swapping core dependencies, or broad cleanup requires explicit approval.
5. For analysis/planning-only requests: **do not edit files or run build, tests, migrations, or dependency installation** unless separately requested.
6. For cleanup, record tracked, staged and untracked changes before editing; preserve prior work and distinguish it from the new task. **Never stage or commit automatically.**

## Implementation principles

- Prefer the smallest coherent change that solves a demonstrated problem.
- Keep routing in `internal/routes/`, HTTP translation in `internal/handlers/`, and dependency setup/cleanup in `internal/registry/`, following the actual codebase.
- Use `internal/usecases/` and `internal/infrastructure/` where responsibilities justify them; do not create a use case, repository, interface, factory, or extra layer for a trivial endpoint.
- Introduce interfaces when an actual consumer boundary or alternate implementation warrants one, not by default.
- Avoid copying complex LMS/SPMB modules wholesale. Treat those projects as references, never as permission to modify them.
- Do not create placeholder packages, speculative features, or unused generic helpers.
- Keep config, logging, error responses, and naming consistent with the existing project; avoid exposing secrets or internal errors in HTTP responses.
- Do not implement unrelated features or refactor broad areas as side effects of a narrowly scoped task.

### Code comments and naming

- Explain important or non-obvious decisions, especially security, transactions, concurrency, compatibility and error handling. Explain why; do not narrate obvious operations or add comments to every line.
- Keep comments concise, natural and in English. Preserve meaningful existing comments and update them when behavior changes. Follow GoDoc where documentation is useful; avoid AI markers, milestone annotations and source change history.
- Use PascalCase for exported Go identifiers and camelCase for locals/unexported identifiers, retaining idiomatic initialisms. Configuration JSON uses camelCase, public API JSON and PostgreSQL names use snake_case, and environment variables use UPPER_SNAKE_CASE.
- Preserve existing keys/contracts and approved exceptions (including legacy JWT keys and `created_by`/`updated_by` audit FKs). Do not rename a public/config/schema identifier merely for style; propose breaking changes first.

### Logging policy

- Reuse the existing structured logger and middleware. Use concise human-readable native messages and stable `event_code` values; do not add a duplicate message field or an unnecessary logging abstraction/dependency.
- Include the actual application/API `error_code` when reliably available through request-local metadata. Never infer it from HTTP status, buffer/parse response bodies for logging, or change the response contract for telemetry.
- Preserve request IDs across responses, completion logs and related diagnostics. Emit one HTTP completion event; keep a separate security/operational event only when it serves a distinct purpose, without repeated per-layer logging.
- Choose levels by operational significance: routine authentication denials can be INFO, policy/capacity rejections WARN, internal failures ERROR. Preserve conservative fallbacks for unclassified outcomes; do not treat every 4xx as a security incident.
- Never log passwords/hashes, raw tokens/token hashes, Authorization/cookies, credentials, secret configuration or body/header dumps. Preserve bounded, single-line sanitization of untrusted metadata; omit sensitive query values.
- Use focused risk-based checks for error-code accuracy, correlation/isolation, level policy, duplication, privacy and unchanged HTTP/auth behavior. See [operations](documentation/operations.md) for the implemented format and boundaries.

### Security preservation

- Preserve authoritative database identifier normalization, bounded login admission before password work, explicit trusted-proxy boundaries and generic authentication/throttle errors. Never evict active limits to hide saturation or claim process-local controls are distributed. Keep health/readiness outside login limits.
- Never display local bootstrap secret files; verify ignore/history/permissions through metadata only. Keep the interactive 15-character minimum and existing local-only first-admin guard.

## Verification and safety

- Choose checks proportional to changed behavior and risk; do not require a test file for every source file or an arbitrary coverage target.
- Test observable behavior and critical contracts, prioritizing security/authentication/authorization, database integrity, concurrency and business rules. Do not create tests merely to fill a template.
- Prefer simple fixtures and small consumer-boundary fakes; avoid excessive mocks, duplicate setup and tests that only repeat framework behavior. Unit and integration tests can protect different risks even when they call the same function.
- Merge/simplify only when clarity improves without losing regression protection. Remove a test only with evidence that its relevant coverage is obsolete or remains equivalently protected; record the before/after behavior and run the relevant checks. Test/file counts are not success targets.
- Distinguish static inspection, build, unit/integration/API checks, and manual verification. Record only commands actually run and their actual outcomes.
- Never claim runtime PASS if the program could not be run or necessary dependencies were unavailable.
- Do not run destructive migration/down/reset, production data mutations, or broad cleanup without explicit permission and a verified target environment.
- Do not overwrite local secrets, `.env`, or runtime config. Use examples/placeholders without real credentials when documentation is requested.
- Avoid unrelated changes to `go.mod`/`go.sum` or generated files; dependency changes should be scoped and justified.

## Development artifacts and review

For work using the tracked workflow, store **approved execution inputs** under `develop/inputs/<domain>/<feature>/` and **corresponding actual-execution reviews** under `develop/reviews/<domain>/<feature>/`. Use matching IDs, such as `M1-01-implementation.md` ↔ `M1-01-review.md`; fixes use the next ID and reference the preceding review. Do not overwrite earlier FAIL reviews or fabricate historical PASS results.

Review verdicts are **PASS**, **FAIL**, and **BLOCKED**. A review must identify changes, evidence, acceptance-criteria status, remaining issues, and next action. See [`develop/README-develop.md`](develop/README-develop.md) for the detailed workflow. Create or update such artifacts only within the approved task scope.

When the approved task includes autonomous review–fix, continue routine in-scope fixes with matching new inputs/reviews until PASS or a genuine blocker. Keep implementation and review phases distinct; prefer a separate-context reviewer when available, otherwise disclose structured self-review. Never call self-review independent or overwrite a FAIL review. Changes beyond approval still require a decision.

## Documentation changes

- Keep `README.md` and `README.id.md` aligned when changing shared instructions.
- Keep reference documents accurate for the behavior implemented, clearly marking pending or proposed features.
- Put command summaries in README and detailed supported CLI usage in `documentation/cli.md`.
- Keep process history in `develop/`, not in `documentation/`.
- Do not add new documentation folders or standards without a real need.

## Completion report

When work is complete, summarize: **scope**, **files changed**, **behavior**, **verification actually performed**, **limitations**, and **remaining decisions**. For a planning task, provide findings and a proposed change matrix, then stop for approval.
