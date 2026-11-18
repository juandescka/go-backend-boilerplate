# M8A-01 — Login abuse protection

Approved by the user-provided M8 master prompt on 2026-11-18. Scope: audit, implement, verify and separately review login throttling; then continue M8B/M8C. Never stage/commit, read local secret contents or mutate development data.

## Initial audit
- Starting tracked/staged/untracked tree clean; diff checks empty. M7-FINAL-02 and M7S-FINAL record PASS. Historical M7S accepted a five-character bootstrap minimum; M8 explicitly requires fifteen, so M8B will tighten new bootstrap input without changing login compatibility or historical reviews.
- Retain opaque random bearer tokens/hash-only persistence, fixed expiry, revocation, inactive-user rejection, dummy Argon2 verification, two-slot KDF bound, tenant membership and explicit scoped RBAC, static seeders and guarded local CLI bootstrap.
- Rate-limit config/events exist but no runtime limiter implementation exists; registry wiring is commented out. Implement one bounded fixed-window limiter in the existing shared/security package, no Redis/workers. IP first; PostgreSQL-normalized identifier digest second.
- Existing proxy config defaults empty; Gin uses validated proxies. Existing security headers, timeouts, body limits, safe recovery and error envelopes work and will be preserved. CORS currently accepts invalid preflights and sets Vary only on allowed origins. Auth early middleware errors lack no-store. M8B will fix these narrowly.
- Potential behavior changes: login 429 with Retry-After; configured identifier failure-named settings conservatively count all admitted attempts including success, avoiding failure-registration races and account-state-dependent resets. Fixed windows do not extend on rejection. Bootstrap minimum becomes fifteen. Invalid CORS preflights become 403.
- Actual local file .bootstrap-admin-password is regular, mode 0600, ignored by Git and excluded by Docker. Neither actual nor reported typo filename appears in accessible Git history. Contents never opened; no deletion authorized.

## Implementation and verification plan
Use existing handler/service/store/registry. Database normalization after IP admission adds one bounded query and fails closed. Shared map plus expiry heap bounds records and reclaims expired entries without background goroutines or active-key eviction. Full capacity rejects new keys until earliest expiry; existing keys retain their budgets. Deterministic clock tests cover windows, capacity, cancellation and concurrency; actual Gin tests cover 429, logs and spoofed IP. Document process-local limits and saturation availability tradeoff. Review with a separate-context reviewer, preserve FAIL history and use incremented fix iterations if necessary.
