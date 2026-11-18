# M4P-03 — Fix Local PostgreSQL Port Conflict

- **Feature:** `database/postgresql-docker`
- **Iteration:** `M4P-03`
- **Type:** Fix
- **Approval:** Covered by the user's autonomous M4P review/fix authorization.
- **Based on:** [M4P-02 review, F2](../../../reviews/database/postgresql-docker/M4P-02-review.md#f2--actual-startup-fails-because-host-port-5432-is-occupied)

## Objective and evidence

Resolve Docker's actual bind failure on host port 5432 without touching another service. A host-context Python socket probe confirmed 5432 fails with address already in use and 5433 can bind. Sandboxed probes cannot establish availability: listener enumeration returned no output and socket bind was denied. A combined elevated audit's automatic approval timed out; a narrower read-only Docker audit and minimal host socket probe succeeded afterward. No user approval blocker remains.

## Scope and files affected

Change only the newly added `POSTGRES_PORT` in ignored root `.env` from 5432 to 5433. Keep all original environment bytes, database credentials, baseline `compose.yaml` default 5432, project identity, previously created named volume and unrelated resources. Add this input and a paired independent review; retain earlier FAIL reviews.

## Verification and acceptance

Reconcile the same project with `docker compose up -d`, wait for healthy, verify actual published `127.0.0.1:5433 -> 5432/tcp`, PostgreSQL 17, `SELECT 1`, configured database/user, named mount and initialized cluster identifier. Verify bridge-hostname correct-password success and wrong-password failure plus safe authentication rules. Stop/start only this project's service, wait for healthy and confirm the same volume and cluster identifier. Confirm host endpoint reachability and Docker Desktop Compose grouping through daemon metadata. Repeat targeted Compose, whitespace, environment preservation/secret checks and backend fingerprints. Require independent final PASS, otherwise record actual findings or blocker.

## Documentation impact and constraints

Existing docs already distinguish default 5432 from configurable `POSTGRES_PORT`, explain free-port selection and preserve MySQL until M4A. No tracked default/example needs a machine-specific change. Record actual local port 5433 in the review and final report. No migration, business SQL, volume reset/deletion, production change or Go source/dependency change.
