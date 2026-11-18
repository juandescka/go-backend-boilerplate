# M7S-A-01 — Seeder CLI and registry review

**PASS for the command foundation.** The [stage input](../../inputs/seeder-bootstrap/M7S-A-01-implementation.md) authorizes one named run with explicit target confirmation. The Cobra help and metadata-only list worked. An unknown name returned nonzero before any database access. The existing command tests passed. The code reuses the existing connector and verifies the confirmed loopback peer, database and role, then checks migration state in the write transaction. Registration and live target and mutation tests follow in stages B and C.
