# M7S-B-01 — Demo datasets and first-platform-admin bootstrap

After [stage A](../../reviews/seeder-bootstrap/M7S-A-01-review.md), register the four approved datasets. Reserve deterministic IDs and check them against database business keys so collisions cannot claim existing rows. Use an inactive passwordless demo identity. Accept only existing seed-owned dependencies and preserve edits and activation state on rerun.

Create a separate local administrator CLI. Read identity and password interactively and use the existing password hashing policy. Serialize bootstrap across transactions and refuse any prior platform assignment or existing email. Create a new user and a platform assignment atomically. Preserve the reference data owned by migrations.
