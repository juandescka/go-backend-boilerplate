# M7S-B-02 — Verify the production guard with valid configuration

Address the [first verification failure](../../reviews/seeder-bootstrap/M7S-B-01-review.md). Set the secure cookie setting for the intentionally production-classified test invocation so it reaches the provisioning environment guard. Rerun the live test using a fresh disposable database. Record any further independent review findings before closing this checkpoint.

The separate-context review also found that an interrupt during hidden terminal input could skip terminal restoration. Restore the terminal before exiting on SIGINT or SIGTERM and verify interruption through the pseudo-terminal. Isolate the inactive-assignment check from the reserved bootstrap-user check. Correct the transaction-timeout documentation.
