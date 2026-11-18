# Tasks

The Planner writes this file and the human approves it. The orchestrator only changes the Status line.
Prerequisite (human, not a task): `go.yaml.in/yaml/v3` added by hand to go.mod and recorded in docs/dependencies.md. go.mod and docs/* are protected paths.

## T-001: Names struct from semconv-mapping.yaml
Satisfies: R-002
Inputs: spec/semconv-mapping.yaml, plan.md Week 5 step 1
Outputs: core/names.go loading every gen_ai.*, error.type, operation name value and spanslo_attributes (task, tool) name; test that scans non test Go files for gen_ai literals
Done when: the scan test finds no `gen_ai.` string literal outside _test.go files and names load from the YAML in a unit test
Allowed models: Claude
Depends on: none
Milestone: M5
Verify: go test ./core/...
Status: done

## T-002: RFC 8785 canonical JSON and argument hash
Satisfies: R-005
Inputs: RFC 8785, plan.md Week 5 step 4
Outputs: core/canon.go with Canonicalize and ArgsHash (SHA-256 hex or bytes), stdlib only
Done when: the RFC 8785 test vectors (key order, number serialization, string escaping, unicode) pass; if exact number formatting cannot be done with stdlib the task stops and asks
Allowed models: Claude
Depends on: none
Milestone: M5
Verify: go test ./core/...
Status: done

## T-003: OTLP JSON reader
Satisfies: R-001
Inputs: raw file exporter OTLP JSON samples, T-001 names
Outputs: core span types and a parser from bytes to spans (attributes, events, status, start and end times); no file I/O in core/
Done when: a raw OTLP JSON fixture parses to the expected spans, attributes, events and times, and malformed JSON returns an error
Allowed models: Claude
Depends on: T-001
Milestone: M5
Verify: go test ./core/...
Status: done

## T-004: Group spans into runs
Satisfies: R-003
Inputs: T-003 span types
Outputs: GroupRuns in core/: by spanslo.run.id, else trace ID
Done when: a table test shows spans with and without spanslo.run.id land in the right runs
Allowed models: Claude
Depends on: T-003
Milestone: M5
Verify: go test ./core/...
Status: done

## T-005: Turn and step indexing
Satisfies: R-004, R-005, R-006
Inputs: T-002 ArgsHash, T-004 runs, design.md v2.7 "Step indexing"
Outputs: core/index.go assigning turn t to chat spans (start time, then span id bytes) and (t, i) to tool calls (tool name, then args hash)
Done when: a test shows identical (t, i) for shuffled span order and for identical start times, including parallel calls with the same tool and arguments
Allowed models: Claude
Depends on: T-002, T-004
Milestone: M5
Verify: go test ./core/...
Status: open

## T-006: Turn mismatch count
Satisfies: R-007
Inputs: T-005 indexing, fault events with spanslo.fault.turn
Outputs: turn_mismatches computed in core/, turns taken from chat spans only
Done when: a test with one matching and one mismatching fault event gives turn_mismatches 1
Allowed models: Claude
Depends on: T-005
Milestone: M5
Verify: go test ./core/...
Status: open

## T-007: Week 5 fixtures and purity check
Satisfies: R-008, R-009
Inputs: T-001 to T-006
Outputs: testdata fixtures for parallel calls with identical start times and reasoning only turns; test that core/ imports no os, net, io/ioutil or time.Now
Done when: the turn indexing fixtures pass and the import check passes
Allowed models: Claude, Gemini Flash
Depends on: T-006
Milestone: M5
Verify: go test ./core/...
Status: open

## T-008: Effective contract from catalog
Satisfies: R-025
Inputs: catalog/defaults.yaml, a fault file, spec/fault.schema.json
Outputs: core/contract.go merging fault contract keys over defaults (must_not_call_tags, max_retries_same_call, max_retries_same_tool)
Done when: a test shows a key set in the fault file overrides the default and omitted keys keep it
Allowed models: Claude
Depends on: T-001
Milestone: M6
Verify: go test ./core/...
Status: open

## T-009: Fault events, untriggered, not applied, mode
Satisfies: R-027, R-028, R-029
Inputs: T-004 runs, design.md v2.7 "Run identity" and "Invalid runs"
Outputs: fault event extraction, fault_not_applied, untriggered handling, mode from spanslo.mode with the baseline rule; absent mode on a non baseline run returns an invalid run reason
Done when: tests cover baseline with no mode (none), non baseline with no faults (fault_not_applied), all untriggered faults (empty episodes), and absent mode on a non baseline run (invalid, no exit)
Allowed models: Claude
Depends on: T-005
Milestone: M6
Verify: go test ./core/...
Status: open

## T-010: Outcome, invalid runs, ended_at
Satisfies: R-016, R-073
Inputs: spanslo.task_success evaluation event, root invoke_agent span
Outputs: success parse; invalid reasons for zero, many or bad score events and for a missing or unfinished root span; ended_at as RFC 3339 UTC with Z using a real parse
Done when: tests cover missing, duplicate and 0.5 score events, a missing root span, an unfinished root span, and ended_at 2026-02-30T00:00:00Z (invalid)
Allowed models: Claude
Depends on: T-004
Milestone: M6
Verify: go test ./core/...
Status: open

## T-011: Recovery detection
Satisfies: R-012, R-013, R-014
Inputs: spanslo.task.fallback_tools on the root span, execute_tool span status
Outputs: qualifying recovery call finder, persistent rule, all episodes recovered rollup
Done when: tests cover recovery on the target, recovery on a fallback, a later error on a fallback (not recovered), and persistent mode where only a fallback recovers
Allowed models: Claude
Depends on: T-009
Milestone: M6
Verify: go test ./core/...
Status: open

## T-012: Episode grouping
Satisfies: R-010, R-011
Inputs: T-009 triggered faults, T-005 indexing
Outputs: episodes per target with target, first and last fault turn, fault_count; a triggered fault after a recovery starts a new episode
Done when: a test with two faults before a recovery and one after gives two episodes with the right counts
Allowed models: Claude
Depends on: T-009, T-011
Milestone: M6
Verify: go test ./core/...
Status: open

## T-013: Episode metrics
Satisfies: R-017, R-018, R-019, R-020
Inputs: T-012 episodes, T-011 qualifying call
Outputs: turns_to_recover, time_to_recover_ms with clamp to 0 and a returned warning, cascade_spans, recovery_tokens
Done when: tests cover a recovered and an unrecovered episode (nulls, end of run interval) and a negative time that clamps to 0 and returns a warning
Allowed models: Claude
Depends on: T-012, T-011
Milestone: M6
Verify: go test ./core/...
Status: open

## T-014: Task verified recovery and graceful error
Satisfies: R-015, R-026
Inputs: T-010 success, T-011 recovery, spanslo.oracle.graceful_error
Outputs: task_verified_recovery (true, false, null) and graceful_error set only with an unrecovered episode
Done when: a table test covers baseline, no triggered fault, all recovered with pass, recovered with fail, and unrecovered
Allowed models: Claude
Depends on: T-010, T-011
Milestone: M6
Verify: go test ./core/...
Status: open

## T-015: Safety and efficiency contracts
Satisfies: R-021, R-022, R-023, R-024
Inputs: T-008 contract, T-005 call order, spanslo.tool.destructive_hint and open_world_hint
Outputs: unsafe_actions, untagged_tool_calls, retry_budget_violations (one per maximal run of consecutive calls to one tool that exceeds any limit; first call is not a retry; failed calls count)
Done when: tests show 4 identical calls allowed and 5 a breach with defaults, a different tool between calls breaks the run, two limits exceeded in one run count once, untagged calls are never unsafe, and retries never add to unsafe_actions
Allowed models: Claude
Depends on: T-005, T-008
Milestone: M6
Verify: go test ./core/...
Status: open

## T-016: Assemble the run record
Satisfies: R-030
Inputs: T-006, T-009 to T-015, spec/run-record.schema.json
Outputs: ScoreRun in core/ returning a run record, warnings and an invalid reason; semconv_version from semconv.commit in spec/semconv-mapping.yaml; the Go record type is hand written with JSON tags mirroring spec/run-record.schema.json; a Python test (the schema validation task, T-028) catches drift
Done when: scoring a fixture returns a record with every required field and semconv_version equal to the pinned SHA
Allowed models: Claude
Depends on: T-006, T-013, T-014, T-015
Milestone: M6
Verify: go test ./core/...
Status: open

## T-017: Week 6 validation fixtures
Satisfies: R-032, R-033
Inputs: T-016
Outputs: fixtures for single, burst, intermittent and persistent modes; a run that calls an unrelated tool after a fault then fails the oracle; a run with excess retries
Done when: the unrelated tool run scores surface_recovered false and task_verified_recovery false, all four mode fixtures pass, and excess retries give unsafe_actions 0
Allowed models: Claude, Gemini Flash
Depends on: T-016
Milestone: M6
Verify: go test ./core/...
Status: open

## T-018: Aggregation and baselines
Satisfies: R-034, R-035, R-036
Inputs: run records, design.md "Safeguards"
Outputs: core/aggregate.go rolling up by experiment_id, framework, model and fault; triggered only filter; baseline sets keyed by task_id, framework, model with medians
Done when: a test shows fault_not_applied and untriggered runs excluded from fault SLIs and baseline medians taken over the matching key only
Allowed models: Claude
Depends on: T-016
Milestone: M7
Verify: go test ./core/...
Status: open

## T-019: Ratio SLIs and the undefined rule
Satisfies: R-037, R-038, R-039, R-040, R-041, R-042, R-043, R-066
Inputs: T-018
Outputs: S0, Sf, RR and the four rates; an undefined result (not 0 or 1) when a denominator is zero
Done when: tests cover no baseline runs, no triggered runs, S0 equal to 0 and a normal case for every SLI
Allowed models: Claude
Depends on: T-018
Milestone: M7
Verify: go test ./core/...
Status: open

## T-020: TAF and turn amplification
Satisfies: R-044, R-045, R-066
Inputs: T-018 baseline medians
Outputs: per run taf and turn_amplification, undefined when task_id is null, no matching baseline, or median is 0
Done when: tests cover each undefined case and a computed case
Allowed models: Claude
Depends on: T-018
Milestone: M7
Verify: go test ./core/...
Status: open

## T-021: Percentiles with unrecovered counts
Satisfies: R-046
Inputs: T-013 episode values, T-020
Outputs: p50 and p95 for the four episode SLIs, taf and turn_amplification, excluding unrecovered episodes, reported with the unrecovered count
Done when: a test shows unrecovered episodes absent from percentiles and counted in the reported unrecovered count
Allowed models: Claude
Depends on: T-019, T-020
Milestone: M7
Verify: go test ./core/...
Status: open

## T-022: SLO file reader
Satisfies: R-048
Inputs: spec/slo.schema.json, design.md "SLOs and error budgets"
Outputs: SLO file parser (slos list, windows map, the three forms, burn default true); rejects files that are not exactly one form
Done when: a valid file parses and files with two forms, no form or a bad window are rejected
Allowed models: Claude
Depends on: T-001
Milestone: M7
Verify: go test ./core/...
Status: open

## T-023: Error budgets and SLO evaluation
Satisfies: R-049, R-050, R-051, R-054, R-067
Inputs: T-019, T-021, T-022
Outputs: budget per form, allowed, bad, remaining, exhausted when bad is greater than allowed; threshold_max evaluation with unrecovered episodes as violations for turns_to_recover and time_to_recover only; resilience SLO compared with its target only; no burn for burn false; zero budget handling
Done when: tests cover each of the three forms, the unrecovered rule per SLI, an RR SLO with no budget, and a zero budget giving +Inf burn for bad above 0 and 0 otherwise, never NaN
Allowed models: Claude
Depends on: T-021, T-022
Milestone: M7
Verify: go test ./core/...
Status: open

## T-024: Windows and burn rate alerts
Satisfies: R-052, R-053
Inputs: T-023, ended_at on records
Outputs: "now" as latest ended_at, time windows, count windows ordered by ended_at then run_id, burn rate per long and short window, alert only when both reach factor
Done when: tests show a run on the time window edge, count window tie ordering by run_id, and no alert when only the long window reaches factor
Allowed models: Claude
Depends on: T-023
Milestone: M7
Verify: go test ./core/...
Status: open

## T-025: Week 7 unit and property tests
Satisfies: R-047, R-055
Inputs: T-019 to T-024
Outputs: unit tests for zero baselines and untriggered faults; property tests (stdlib testing/quick) for RR 1.0 when faulted equals baseline, taf 1.0 for identical runs, remaining at least 0 while bad is at most allowed
Done when: unit tests give the R-066 and R-028 results and the property tests pass
Allowed models: Claude, Gemini Flash
Depends on: T-024
Milestone: M7
Verify: go test ./core/...
Status: open

## T-026: Canonical fault event, bench adapter, passthrough
Satisfies: R-056, R-057, R-058
Inputs: design.md "Canonical fault event", spanslo.fault.* names
Outputs: core/adapters/bench.go converting bench fault records to canonical events; unknown sources pass through untouched
Done when: a bench record converts with all R-057 attributes and an unknown source record is neither changed nor dropped
Allowed models: Claude
Depends on: T-003
Milestone: M8
Verify: go test ./core/...
Status: open

## T-027: spanslo score command
Satisfies: R-060, R-016, R-018
Inputs: T-016, T-026
Outputs: cmd/spanslo score: reads traces, runs adapters, writes JSON lines, deletes the old sidecar first, writes <out>.invalid.jsonl (run_id, reason), prints each run status and names invalid runs, prints core warnings to stderr; exits 0 with invalid runs, 2 with no sidecar on unreadable input or usage error
Done when: tests show a mixed input exits 0 with a sidecar naming the invalid run, a stale sidecar is replaced, and unparseable input exits 2 and leaves no sidecar
Allowed models: Claude
Depends on: T-016, T-026
Milestone: M8
Verify: go test ./cmd/...
Status: open

## T-028: Run records validate against the schema
Satisfies: R-031
Inputs: spec/run-record.schema.json, T-027 score command, testdata fixtures
Outputs: bench/tests/test_run_records.py, a Python test that runs the spanslo score command on the fixtures and validates every emitted record against spec/run-record.schema.json with jsonschema
Done when: every record emitted from the fixtures validates, and a record with a missing required field fails the test
Allowed models: Claude
Depends on: T-017, T-027
Milestone: M8
Verify: uv run --project bench pytest bench/tests/test_run_records.py
Status: open

## T-029: spanslo slo command
Satisfies: R-061
Inputs: T-024, T-022
Outputs: cmd/spanslo slo: reads JSON lines runs and the SLO file, prints met or not, budget remaining and each burn alert
Done when: a test prints the expected lines for a met SLO, a breached SLO and a firing alert, and exits 0 for all
Allowed models: Claude
Depends on: T-024, T-027
Milestone: M8
Verify: go test ./cmd/...
Status: open

## T-030: spanslo report command
Satisfies: R-062
Inputs: T-021, T-024
Outputs: cmd/spanslo report writing spanslo's own JSON: SLI values, percentiles with unrecovered counts, SLO results
Done when: a test compares the report on a fixture with a hand checked expected JSON and exits 0 when SLOs are breached
Allowed models: Claude
Depends on: T-029
Milestone: M8
Verify: go test ./cmd/...
Status: open

## T-031: spanslo gate command
Satisfies: R-063
Inputs: T-029, T-027 sidecar
Outputs: cmd/spanslo gate: checks the sidecar first (missing or non empty gives exit 2), then exits 1 on an unmet SLO or firing alert, else 0
Done when: tests show exit 2 for a missing sidecar and a non empty sidecar, exit 1 for a breaching fixture, exit 0 for a clean fixture, and exit 2 for empty input
Allowed models: Claude
Depends on: T-029
Milestone: M8
Verify: go test ./cmd/...
Status: open

## T-032: Exit code audit across commands
Satisfies: R-068
Inputs: T-027, T-029 to T-031
Outputs: table test for every command: unknown flag, missing file, unparseable runs or SLO file and unparseable ended_at all exit 2; slo and report exit 0 on breach
Done when: the table test passes for score, slo, report and gate
Allowed models: Claude, Gemini Flash
Depends on: T-031
Milestone: M8
Verify: go test ./cmd/...
Status: open

## T-033: Golden traces, model faults
Satisfies: R-064
Inputs: catalog/ (model-malformed-tool-args, model-rate-limit-429, model-server-error-5xx, model-timeout, model-truncated-stream), T-027
Outputs: testdata/traces/<fault>.json for the 5 model-* faults with hand checked expected run records and a golden test; expected records are written by hand, not generated from scorer output
Done when: one golden trace exists for each of the 5 model-* faults and the golden test passes
Allowed models: Claude
Depends on: T-027
Milestone: M8
Verify: go test ./...
Status: open

## T-034: Golden traces, tool faults and memory fault
Satisfies: R-064
Inputs: catalog/ (the 6 tool-* faults and memory-irrelevant-retrieval), T-027
Outputs: testdata/traces/<fault>.json for the 6 tool-* faults and memory-irrelevant-retrieval (7) with hand checked expected run records and a golden test; expected records are written by hand, not generated from scorer output
Done when: one golden trace exists for each of those 7 faults and the golden test passes
Allowed models: Claude
Depends on: T-027
Milestone: M8
Verify: go test ./...
Status: open

## T-035: Week 8 validation checks
Satisfies: R-065
Inputs: T-031, T-033, T-034
Outputs: end to end test of the three gate and score cases
Done when: gate exits 1 on a breaching fixture, exits 2 with a missing and with a non empty sidecar, and score removes a stale sidecar and writes none on exit 2
Allowed models: Claude, Gemini Flash
Depends on: T-033, T-034
Milestone: M8
Verify: go test ./...
Status: open

## T-036: Run identity matching for external injectors
Satisfies: R-059
Inputs: real captured injector output under docs/g0/evidence/ (absent)
Outputs: trace context and baggage matching, then fallback on process, SPANSLO_RUN_ID and time window
Done when: blocked until fixtures exist; then a test matches an event by trace context, and by the fallback when context is missing
Allowed models: Claude
Depends on: T-026
Milestone: M8
Status: blocked

## T-037: Tumult adapter
Satisfies: R-070
Inputs: captured Tumult output under docs/g0/evidence/ (absent)
Outputs: core/adapters/tumult.go with source tumult
Done when: blocked until fixtures exist; then each fixture converts to canonical events
Allowed models: Claude
Depends on: T-026
Milestone: M8
Status: blocked

## T-038: DeepAgentLabs chaos adapter
Satisfies: R-071
Inputs: captured output under docs/g0/evidence/ (absent)
Outputs: core/adapters/deepagentlabs_chaos.go with source deepagentlabs-chaos
Done when: blocked until fixtures exist; then each fixture converts to canonical events
Allowed models: Claude
Depends on: T-026
Milestone: M8
Status: blocked

## T-039: Chaos security adapter
Satisfies: R-072
Inputs: captured output under docs/g0/evidence/ (absent)
Outputs: core/adapters/chaos_security.go with source chaos-security
Done when: blocked until fixtures exist; then each fixture converts to canonical events
Allowed models: Claude
Depends on: T-026
Milestone: M8
Status: blocked
