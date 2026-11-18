# LOG-01-01 — Structured Logging & Error Context

- **Feature:** `observability/logging-enhancement`
- **Iteration:** `LOG-01-01`
- **Date:** 17 November 2026 (Asia/Makassar)
- **Approval:** User's LOG-01 prompt authorizes targeted logging, focused tests, documentation/rule alignment and autonomous review/fix. No staging/commit or CORS/configuration change.
- **Baseline:** Initial tree clean at `acaf779` (M5C committed). Tracked/staged stats and diff checks empty/clean. `/private/tmp/log01-before.json` records123 file hashes, HEAD, staged entries and status, including ignored local-file hashes without values. M5C/PC01 prior PASS records read; historical verification is not new LOG-01 runtime evidence.

## Objective and audit

Expose reliable API error codes in the existing one-per-request completion event and make native messages readable, without changing response contracts, framework or data collection.

| Area | Current behavior | Decision |
|---|---|---|
| Logger | Logrus; stdout; exact development uses TextFormatter, other environments JSONFormatter; production INFO, otherwise DEBUG | KEEP initialization, framework, formatter/level configuration and dependencies |
| Native message | Logrus msg contains event-code strings on completion, auth, readiness and server lifecycle | IMPROVE concise English native message; no custom message field |
| Event code/category | Stable fields for HTTP, security decisions and operations | KEEP all existing code values/categories; no new event per error |
| API error code | Error helper knows actual code but completion sees only status | ADD small request-local typed metadata/accessor in existing response package; no body buffering/parsing |
| Request ID | Validated/generated once, same response header/Gin/standard context | KEEP generator/propagation; verify correlation/isolation |
| Levels | HTTP success INFO, all4xx WARN,5xx ERROR; security success INFO/denials and operational failures WARN | REFINE routine sourced401 UNAUTHORIZED completion/security denial to INFO; actual auth operation failure ERROR, resource-capacity rejection WARN; preserve conservative fallback for unclassified HTTP4xx/5xx |
| Readiness/server | Stable operational codes/reasons and lifecycle fields; code-like native messages | IMPROVE messages only, preserve levels/DB/HTTP lifecycle behavior |
| Privacy | Completion excludes body/query/Auth/cookies; bounded path/agent and canonical IP; panic omits value/raw errors | KEEP; only bounded trusted-style error code added, never error message/details/input or Origin |
| Duplicate logs | One completion; separate security decision and operational/panic diagnostics | KEEP distinct purposes, add no new emission or per-layer error log |

## Error propagation and level design

The response package will store only code/status in private Gin metadata when it emits the first error response. Accessor returns a code only for matching final error status and bounded existing UPPER_SNAKE_CASE format (max64 bytes), omitting unreliable/unsafe/unclassified values without altering API serialization. Already committed responses must not gain a misleading code from a later helper call. Success logs never carry error_code. No package cycle, global error state, new dependency, public error wrapper or response dump.

Completion retains event_code=http.request.completed and existing fields, choosing static native messages for completed/rejected/failed outcomes. A sourced401 UNAUTHORIZED is routine rejection, not ERROR; unknown401 remains conservatively WARN and internal5xx remains ERROR. Do not infer INVALID_CREDENTIALS or another application code from status. Existing errorCodeFromStatus mapping remains solely the actual API helper's behavior.

Security events retain decision code/outcome/reason and request correlation independently from HTTP timing/status. Active auth denial messages become human-readable at INFO, operational auth failures ERROR and local verification capacity WARN. Existing success/unused-event behavior is preserved outside the targeted paths. Readiness still has its existing operational reason/dependency record, and panic retains diagnostic type/stack separately; completion is still emitted exactly once. No new audit trail, tracing, metrics, CORS or database work.

## Actual before/after and verification

Capture controlled HTTP output before editing production code using the actual logger, operational/auth handlers, Gin routes and CORS middleware, with isolated fixture configuration. Cases: health success, rejected CORS, invalid credentials and simulated operational auth error. Correlation IDs are fixed synthetic IDs; no real credentials used. Native JSON msg/level/event/error/status/request ID will be compared against captured after output, not invented examples. A temporary capture-only test is removed from final source after both captures; focused regression tests remain in one grouped observability file.

Tests cover exact wire bodies/statuses, helper-source error propagation, success/unclassified/unsafe-code omission, committed-response ownership, abort/panic/readiness behavior, context-aware levels, separate security-vs-completion counts, sensitive data exclusion and concurrent request-ID/error isolation. Use actual handlers and a small existing consumer-boundary storage fake; no DB integration test or test framework. Test both existing text/JSON formatter behavior as useful without replacing initialization.

Run build/vet/fresh suite, focused race checks and tracked/staged diff checks. Compare protected hashes, HEAD/staged entries and actual final scope. No DB migrations/queries, Docker changes, generation, dependency changes, local config writes or CORS fix is needed. Preserve authentication/session/source behavior and current metadata sanitization. An available separate-context reviewer assesses all16 requested areas and attributed evidence, with paired fix history only for formal FAIL.

## Documentation impact

Update AGENTS EN/ID, TEMPLATE_PROMPT, existing operations logging reference and API standard's error-code/completion relationship. No duplicate logging guide. README/config/generated API need no edits because no commands, settings, endpoints or wire contracts change. Document native msg vs stable event_code vs optional actual error_code, level/duplicate-event policy, request correlation and privacy restrictions. Every fix reassesses these documents. Final commit recommendation covers actual LOG-01 changes only unless new user work appears; never stage/commit.
