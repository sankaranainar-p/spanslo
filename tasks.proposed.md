I wrote 38 tasks to `/tmp/spanslo-tasks.md`, ready to be approved by you and copied into the spanslo repo. They pass `foundry.tasks.validate_tasks`, and every dependency points to an earlier task. I did not start development or approve anything.

- **M5 (T-001 to T-007):** I put RFC 8785 canonicalization (T-002) first because the spec flags it as the riskiest piece. Turn indexing follows it.
- **M6 (T-008 to T-018):** Recovery detection (T-011) now comes before episode grouping (T-012), because grouping needs it. T-017 (R-031 schema validation) is `blocked`.
- **M7 (T-019 to T-026):** Aggregation, ratio SLIs, percentiles, budgets, windows and burn rates, then the unit and property tests.
- **M8 (T-027 to T-034):** The adapter, `score`, `slo`, `report`, `gate`, the exit-code audit, golden traces, and the week-8 checks.
- **Blocked on fixtures (T-035 to T-038):** R-059 and the three external adapters wait for real captured output under `docs/g0/evidence/`.

I set `Status: blocked` on T-017 and T-035 to T-038 instead of `open`, so the orchestrator skips them. Change them to `open` if you want a different behaviour.

I added `Verify:` lines (go test, within the allowlist) and `Allowed models: Claude, Gemini Flash` on the pure test-and-fixture tasks only.

## Open questions
1. **JSON Schema validation (R-031, R-048):** `go.mod` will only have the YAML library. Go has no standard-library JSON Schema validator, and "Go types come from the schemas" implies a generator. Which dependency, if any, is allowed, or should schema checks live in a Python test over the output files? T-017 is blocked on this, and T-023 only parses and checks the SLO form until it is answered.
2. **Protected paths:** `docs/*`, `spec/*` and `go.mod` are protected. Tasks that need a change there cannot run, and none of mine touch them. You still add the YAML dependency by hand.
3. **`catalog/` and `spec/` are not in this checkout.** I couldn't check the fault types, so T-033 says "one per fault in `catalog/`". Hand-checked expected records are required, not generated from scorer output.
4. **Spec tag:** the plan says to tag a new spec release before week 5 code depends on `ended_at`, the new attributes, and mapping_version 6. That is not a task, so confirm it's done.

## Risks and gaps
- **R-018 warning:** the clamp warning can be tested in `core/`, but its stderr output can only be checked once `score` exists (T-028).
- **R-059:** it needs a trace-context or W3C-baggage example and a process-matching example. It stays blocked even after `v0.2.0` if the fixtures are missing.
- **Spec contradiction:** the spec header says version 0.6, but the plan cites 0.5. Check that the plan was written against the latest spec.
- **No test for the sidecar-before-input case:** R-016 and R-060 say `score` writes a sidecar "whenever it processed input". The spec does not say what happens if the output directory cannot be written. I left that to the exit-2 rule, so say if you want something different.