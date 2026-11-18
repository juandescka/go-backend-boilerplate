# M9A-02 — Correct Cleanup Guard and Stale Security Wording

- **Status:** Approved by autonomous in-scope fix authorization; executed before M9B.
- **Based on:** [M9A-01 FAIL](../../reviews/foundation-readiness/M9A-01-review.md)
- **Scope:** F1 live OID/owner validation and name collision guard in the existing M4B disposable fixture; F2 current login-abuse wording in the identity schema guide.
- **Constraints:** No runtime feature, migration, privilege, dependency or local configuration change.
- **Acceptance:** Fail closed on identity/ownership uncertainty, exact-name cleanup only, verify removal, retain active limiter/deployment distinction. Review diff, run build/default tests and execute corrected fixture during M9B with its existing opt-in flags.
