# M7S-B-02 — Dataset and bootstrap re-review

**PASS.** The production fixture now reaches the environment guard with valid configuration. The separate-context reviewer also found a terminal restoration hazard on interruption. The final password reader does not change terminal mode independently. Normal and signal cleanup restore the same state. The reviewer rechecked those changes and confirmed the source verdict.

The final built CLI passed the disposable PostgreSQL 17 and pseudo-terminal integration in 2.61 seconds. It verified all four datasets, owned no-op reruns, dependency and collision refusals, production and wrong-target rejection, noninteractive refusal, hidden password prompts and echo restoration after SIGINT. Bootstrap rejected an existing normalized email and a pre-existing inactive platform assignment, rolled back an induced assignment failure, and allowed exactly one of two concurrent invocations. The created user signed in with the generated test password and had only the existing platform grant. The test removed its owned database `m7s_verify_14ce2566f20a8546189c2d88ce5fce2a`.

See the [fix input](../../inputs/seeder-bootstrap/M7S-B-02-fix-verification.md) and [CLI contract](../../../documentation/cli.md#4-development-seeding-and-administrator-bootstrap).
