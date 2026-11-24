# Plan: spanslo scoring core and CLI (weeks 5 to 8, gate G2)

Steps are ordered. Each step cites the R-IDs in spec.md (version 0.14, written against docs/design.md v2.15) it satisfies. YAML parser for Go: `go.yaml.in/yaml/v3`. It is needed from week 5 (step 1 reads `spec/semconv-mapping.yaml`). The human adds it by hand, and records its name, version and license in `docs/dependencies.md`, before the first factory run. Do not add it from a factory run. The "Done when" items (R-009, R-032, R-033, R-047, R-055, R-064, R-065) are validation checks, not build work: they appear only on each week's validation line. A week is done only when its validation passes; a failed check repeats the week (docs/design.md, Weekly execution plan).

## Architecture
One Go core, two thin wrappers. Layout from docs/design.md, Repository layout:

```text
core/                  indexing, episodes, scoring, aggregation, budgets (pure functions, no I/O, no wall clock)
core/adapters/         one file per source: bench, tumult, deepagentlabs-chaos, chaos-security
cmd/spanslo/           offline CLI: score, slo, report, gate (does all file I/O and prints warnings)
testdata/traces/       golden trace fixtures and expected run records
spec/                  contract; frozen at G1, changed only by decision (semconv-mapping.yaml, mapping_version 8)
catalog/               frozen, read only
```

`collector/spansloconnector/` is untouched until week 9. File reading lives in `cmd/spanslo/`; `core/` receives parsed data and returns results and warnings (R-008, R-018, R-052).

## Steps

### Week 5: reader, grouping, indexing (R-001 to R-008)
1. Read `spec/semconv-mapping.yaml` once into a names struct, including the `spanslo_attributes` groups `task` and `tool`; every later step reads names from it (R-002).
2. OTLP JSON reader producing span values (R-001).
3. Group spans into runs by `spanslo.run.id`, falling back to trace ID (R-003).
4. Turn indexing: chat spans start turns, ties by span id as a lowercase hex string compared as plain strings, ascending (R-004). Tool calls belong to turn `t` when `chat_t.start <= call.start < chat_{t+1}.start`; the last turn is open ended, a call at the same instant as chat `t+1` belongs to `t+1`, and a call before the first chat span makes the run invalid (R-005, R-016). They get `(t, i)` by tool name, then SHA-256 of RFC 8785 canonical args, then span id by the same string rule (R-005). RFC 8785 is written in `core/` with no new dependency; if that is blocked, stop and ask.
5. Turn mismatch count: fault events on chat and tool call spans are compared with the turn derived for their span; events outside the index are not counted; a missing or non integer `spanslo.fault.turn` is a mismatch (R-006, R-007).

Validation: R-009, turn indexing fixtures pass.

### Week 6: episodes and run scoring (R-010 to R-031, R-073)
1. Read the catalog and `defaults.yaml`; build the effective contract per fault (R-025).
2. Fault event handling: triggered false, `fault_not_applied`, mode from `spanslo.mode` with the baseline rule; a missing mode on a non baseline run makes the run invalid, not an exit 2 (R-027 to R-029).
3. Group triggered faults into episodes per target (R-010, R-011).
4. Recovery scoring: surface recovery with fallbacks from `spanslo.task.fallback_tools`, persistent rule, all episodes (R-012 to R-014).
5. Episode metrics: turns, time with clamp and warning, cascade spans, recovery tokens (R-017 to R-020).
6. Oracle outcome, invalid run detection, task verified recovery, graceful error (R-015, R-016, R-026).
7. Contracts: tool tags from `spanslo.tool.*` attributes, untagged calls, safety breaches, retry violations with the first call not a retry (R-021 to R-024).
8. Assemble the run record, set `ended_at` from the root span end time and `semconv_version`, validate against `spec/run-record.schema.json` (R-030, R-031, R-073).

Validation: R-032 and R-033, mode fixtures pass, recovery levels, excess retries never count as unsafe actions.

### Week 7: aggregation, baselines, SLOs, budgets (R-034 to R-046, R-048 to R-054, R-066, R-067)
1. Aggregation per experiment, framework, model and fault, with triggered only filtering (R-034, R-035).
2. Task keyed baselines and medians (R-036).
3. SLIs: S0, Sf, RR, recovery rates, unsafe and retry rates, with the undefined rule (R-037 to R-043, R-066).
4. TAF, turn amplification, p50 and p95 with unrecovered counts (R-044 to R-046).
5. SLO file reader against `spec/slo.schema.json` (R-048).
6. Error budgets by SLO form, allowed, bad and remaining, threshold evaluation with unrecovered episodes as violations for `turns_to_recover` and `time_to_recover` only, resilience ratio exception, zero budget (R-049 to R-051, R-054, R-067).
7. Time and count windows with the latest `ended_at` as "now" (R-052), burn rates and the both windows alert rule (R-053).

Validation: R-047 and R-055, unit and property tests pass.

### Week 8: adapters, CLI, golden traces (R-056 to R-063, R-068, then R-059, R-070 to R-072)
1. Canonical fault event type (R-057).
2. Bench adapter and unknown source pass through (R-056, R-058).
3. `spanslo score` writing JSON lines, the `<out>.invalid.jsonl` sidecar (deleted at the start, rewritten whenever input was processed, none on a usage error) and naming invalid runs (R-016, R-060), then `slo` (R-061), `report` with spanslo's own JSON (R-062), `gate`, which exits 2 on a missing or non empty sidecar (R-063).
4. Exit codes 0, 1 and 2 across all commands, including exit 2 from `slo`, `report` and `gate` on an unparseable `ended_at` (R-068).
5. Golden traces for each fault type with hand checked expected records.
6. Last: run identity matching for external injectors (R-059), then the tumult, deepagentlabs-chaos and chaos-security adapters (R-070 to R-072). These three are blocked until real captured fixtures exist under `docs/g0/evidence/`. Input formats are never invented. If the fixtures are absent, report the task as blocked. R-059 and R-070 to R-072 need fixtures under `docs/g0/evidence/` and may ship after v0.2.0.

Validation: R-064 and R-065, golden trace tests pass and `spanslo gate` exits 1 on a breaching fixture.

Publish step for week 8 is "Release v0.2.0 with CLI binaries (G2)". It is not part of these tasks; the human tags and releases.

## Tech choices
| Choice | Option picked | Why |
| --- | --- | --- |
| Language | Go | Constitution: Go owns scoring |
| Test framework | `go test`, table driven | Testing, Unit |
| JSON, hashing, RFC 8785 | Go standard library and code in `core/` | No new dependency; Constitution: ask before adding one |
| Runs file | JSON lines | docs/design.md v2.7, "CLI" (score bullet); Parquet only after a dependency decision |
| Report | spanslo's own JSON | docs/design.md v2.7, "CLI" (report bullet); the agent-sre shape is week 10 |
| YAML parsing | `go.yaml.in/yaml/v3`, needed from week 5, added by hand before the first factory run | Go has no standard library YAML parser; recorded in `docs/dependencies.md` |

## Risks
| Risk | Mitigation |
| --- | --- |
| Go has no standard library YAML parser, and `go.mod` has no dependencies | The human adds `go.yaml.in/yaml/v3` by hand before the first factory run and records it in `docs/dependencies.md` |
| External adapters have no captured fixtures (`docs/g0/evidence/` does not exist yet) | R-070 to R-072 are blocked until fixtures exist; do not invent formats |
| Decisions made after G1 changed `spec/` (`ended_at`, new attributes, mapping_version 6) | Tag a new spec release before week 5 code depends on it |
| Design leaves some scoring details open | Stop and ask instead of choosing; CLAUDE.md says code and design disagreements go to the human |
