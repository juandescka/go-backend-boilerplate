# M7S-C-01 — Integration, security and documentation verification

After [stage B passed](../../reviews/seeder-bootstrap/M7S-B-02-review.md), run the full Go build, vet and test suite. Verify current CLI authentication and session lifecycle and the M7 RBAC database suite through version 15. Keep writes in owned disposable PostgreSQL 17 databases.

Check the README pair, CLI documentation and dependency order, environment configuration and precedence, operational recovery, authorization separation and the future seeder authoring entry. Preserve all prior reviews, migrations and local configuration. Close the final review only after checking the actual results.
