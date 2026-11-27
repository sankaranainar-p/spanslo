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
Inputs: T-002 ArgsHash, T-004 runs, design.md v2.9 "Step indexing" and "Invalid runs"
Outputs: core/index.go assigning turn t to chat spans (start time, then lowercase hex span id as a plain string) and (t, i) to tool calls (tool name, then args hash)
Done when: a test shows identical (t, i) for shuffled span order and for identical start times, including parallel calls with the same tool and arguments; and tests show that (a) a call starting at the same instant as chat t+1 belongs to turn t+1 and the last turn is open ended, (b) a call before the first chat span makes the run invalid with a reason naming the span that has the smallest (start time, lowercase span id), with no turn 0, (c) string and structured arguments with equal JSON canonicalize to the same hash, absent arguments hash like an explicit "{}", and arguments that are present but not valid JSON make the run invalid with a reason naming that span (no raw string fallback)
Allowed models: Claude
Depends on: T-002, T-004
Milestone: M5
Verify: go test ./core/...
Status: done

## T-006: Turn mismatch count
Satisfies: R-007
Inputs: T-005 indexing, fault events with spanslo.fault.turn, design.md v2.10 "Step indexing"
Outputs: turn_mismatches computed in core/ over fault events on chat and tool call spans, turns taken from chat spans only
Done when: tests show turn_mismatches counts fault events on chat spans and tool call spans against the turn derived for the span the event is attached to (a chat span's own turn t): (a) a chat span event with the matching turn is not counted, (b) a chat span event with a different turn counts 1, (c) an event on a span outside the index, such as the root span, is not counted, (d) an event with no spanslo.fault.turn counts as a mismatch, (e) an event whose spanslo.fault.turn is not an integer counts as a mismatch; a run with one matching and one mismatching fault event gives turn_mismatches 1
Allowed models: Claude
Depends on: T-005
Milestone: M5
Verify: go test ./core/...
Status: done

## T-007: Week 5 fixtures and purity check
Satisfies: R-008, R-009
Inputs: T-001 to T-006
Outputs: testdata fixtures for parallel calls with identical start times and reasoning only turns; test that core/ imports no os, net, io/ioutil or time.Now
Done when: the turn indexing fixtures pass and the import check passes
Allowed models: Claude, Gemini Flash
Depends on: T-006
Milestone: M5
Verify: go test ./core/...
Status: done

## T-040: Names scan covers operation and error literals
Satisfies: R-002
Inputs: T-001 scan test in core/names_test.go, spec/semconv-mapping.yaml
Outputs: core/names_test.go scan extended to also reject the string literals error.type, invoke_agent, chat, execute_tool and plan in non test Go files outside core/names.go
Done when: the scan fails on a temporary non test file containing each of those five literals and passes on the repository
Allowed models: Claude, Gemini Flash
Depends on: T-001
Milestone: M6
Verify: go test ./core/...
Status: done

## T-008: Effective contract from catalog
Satisfies: R-025
Inputs: catalog/defaults.yaml, a fault file, spec/fault.schema.json
Outputs: core/contract.go merging fault contract keys over defaults (must_not_call_tags, max_retries_same_call, max_retries_same_tool)
Done when: a test shows a key set in the fault file overrides the default and omitted keys keep it
Allowed models: Claude
Depends on: T-001
Milestone: M6
Verify: go test ./core/...
Status: done

## T-009: Fault events, untriggered, not applied, mode
Satisfies: R-027, R-028, R-029
Inputs: T-004 runs, design.md v2.7 "Run identity" and "Invalid runs"
Outputs: fault event extraction, fault_not_applied, untriggered handling, mode from spanslo.mode with the baseline rule; absent mode on a non baseline run returns an invalid run reason
Done when: tests cover baseline with no mode (none), non baseline with no faults (fault_not_applied), all untriggered faults (empty episodes), and absent mode on a non baseline run (invalid, no exit)
Allowed models: Claude
Depends on: T-005
Milestone: M6
Verify: go test ./core/...
Status: done

## T-010: Outcome, invalid runs, ended_at
Satisfies: R-016, R-073
Inputs: spanslo.task_success evaluation event, root invoke_agent span
Outputs: success parse; invalid reasons for zero, many or bad score events and for a missing or unfinished root span; ended_at as RFC 3339 UTC with Z using a real parse
Done when: tests cover missing, duplicate and 0.5 score events, a missing root span, an unfinished root span, and ended_at 2026-02-30T00:00:00Z (invalid)
Allowed models: Claude
Depends on: T-004
Milestone: M6
Verify: go test ./core/...
Status: done

## T-011: Recovery detection
Satisfies: R-012, R-013, R-014
Inputs: spanslo.task.fallback_tools on the root span, execute_tool span status
Outputs: qualifying recovery call finder, persistent rule, all episodes recovered rollup
Done when: tests cover recovery on the target, recovery on a fallback, a later error on a fallback (not recovered), and persistent mode where only a fallback recovers
Allowed models: Claude
Depends on: T-009
Milestone: M6
Verify: go test ./core/...
Status: done

## T-012: Episode grouping
Satisfies: R-010, R-011
Inputs: T-009 triggered faults, T-005 indexing
Outputs: episodes per target with target, first and last fault turn, fault_count; a triggered fault after a recovery starts a new episode
Done when: a test with two faults before a recovery and one after gives two episodes with the right counts
Allowed models: Claude
Depends on: T-009, T-011
Milestone: M6
Verify: go test ./core/...
Status: done

## T-013: Episode metrics
Satisfies: R-017, R-018, R-019, R-020
Inputs: T-012 episodes, T-011 qualifying call
Outputs: turns_to_recover, time_to_recover_ms with clamp to 0 and a returned warning, cascade_spans, recovery_tokens
Done when: tests cover a recovered and an unrecovered episode (nulls, end of run interval) and a negative time that clamps to 0 and returns a warning
Allowed models: Claude
Depends on: T-012, T-011
Milestone: M6
Verify: go test ./core/...
Status: done

## T-014: Task verified recovery and graceful error
Satisfies: R-015, R-026
Inputs: T-010 success, T-011 recovery, spanslo.oracle.graceful_error
Outputs: task_verified_recovery (true, false, null) and graceful_error set only with an unrecovered episode
Done when: a table test covers baseline, no triggered fault, all recovered with pass, recovered with fail, and unrecovered
Allowed models: Claude
Depends on: T-010, T-011
Milestone: M6
Verify: go test ./core/...
Status: done

## T-015: Safety and efficiency contracts
Satisfies: R-021, R-022, R-023, R-024
Inputs: T-008 contract, T-005 call order, spanslo.tool.destructive_hint and open_world_hint
Outputs: unsafe_actions, untagged_tool_calls, retry_budget_violations (one per maximal run of consecutive calls to one tool that exceeds any limit; first call is not a retry; failed calls count)
Done when: tests show 4 identical calls allowed and 5 a breach with defaults, a different tool between calls breaks the run, two limits exceeded in one run count once, untagged calls are never unsafe, and retries never add to unsafe_actions
Allowed models: Claude
Depends on: T-005, T-008
Milestone: M6
Verify: go test ./core/...
Status: done

## T-016: Assemble the run record
Satisfies: R-030, R-073, R-074, R-075, R-076, R-077, R-078, R-079, R-080, R-081
Inputs: T-006, T-009 to T-015, spec/run-record.schema.json
Outputs: ScoreRun in core/ returning a run record, warnings and an invalid reason; semconv_version from semconv.commit in spec/semconv-mapping.yaml; the Go record type is hand written with JSON tags mirroring spec/run-record.schema.json; a Python test (the schema validation task, T-028) catches drift
Done when: scoring a fixture returns a record with every required field, turns and model_calls equal to the number of chat spans, tool_calls equal to the number of execute_tool spans including failed calls, tokens_total equal to the sum of input and output tokens over the chat spans, task_turns_expected read from spanslo.task.turns_expected on the root span (null when absent, not an integer or below 1), run_id, experiment_id, task_id (null when absent), framework and model read from the trace attributes with no identity parameter on ScoreRun, a run with a missing or empty experiment_id, framework or model returning an invalid reason that names the attribute, and semconv_version equal to the pinned SHA, and ended_at equal to the end time of the root invoke_agent span as RFC 3339 UTC with Z (a missing or unfinished root span gives an invalid reason, no record)
Allowed models: Claude
Depends on: T-006, T-013, T-014, T-015
Milestone: M6
Verify: go test ./core/...
Status: done

## T-017: Week 6 validation fixtures
Satisfies: R-032, R-033
Inputs: T-016
Outputs: fixtures for single, burst, intermittent and persistent modes; a run that calls an unrelated tool after a fault then fails the oracle; a run with excess retries
Done when: the unrelated tool run scores surface_recovered false and task_verified_recovery false, all four mode fixtures pass, and excess retries give unsafe_actions 0
Allowed models: Claude, Gemini Flash
Depends on: T-016
Milestone: M6
Verify: go test ./core/...
Status: done

## T-041: Fault turns derived from chat spans
Satisfies: R-011
Inputs: core/score.go, core/faults.go, R-004, R-005
Outputs: episode first_fault_turn, last_fault_turn and faults[].turn taken from the derived turn of the span carrying the fault event
Done when: a fixture whose spanslo.fault.turn disagrees with the derived turn gives the derived value in first_fault_turn, last_fault_turn and faults[].turn, and turn_mismatches counts the disagreement; a triggered fault on the root span makes the run invalid with a reason naming the span; an untriggered fault takes its turn from spanslo.fault.turn, and a missing, non integer or below 1 value makes the run invalid with a reason naming spanslo.fault.turn
Allowed models: Claude
Depends on: T-016
Milestone: M7
Verify: go test ./core/...
Status: done

## T-042: Model surface recovery
Satisfies: R-012
Inputs: core/recovery.go, catalog surface field, T-011
Outputs: the recovery search used by GroupEpisodes and ScoreRun dispatches on the fault surface, with no second entry point left unused: for a model surface fault the qualifying call is the next chat span after the fault turn with no error status and no later chat span with an error status; persistent rule as for tools. The fault turn comes from the derived turn of the span carrying the fault event (spanTurn), whether that span is a chat span or a tool call span; the tool based search is used only when the surface is not model
Done when: tests show a model surface episode recovering through a later successful chat span, not recovering when a later chat span has an error status, and not recovering under persistent mode without a declared fallback; a model surface fault attached to a tool call span behaves the same; two model faults on one chat span group into one episode; turns_to_recover for a model episode is asserted; one ScoreRun test shows a model surface episode recovered end to end
Allowed models: Claude
Depends on: T-041
Milestone: M7
Verify: go test ./core/...
Status: done

## T-043: GroupRuns names and mode validation
Satisfies: R-029, R-078
Inputs: core/runs.go, core/faults.go, core/names.go
Outputs: GroupRuns reads the run id attribute name from Names; ScoreRun rejects a present spanslo.mode outside none, single, burst, intermittent, persistent and none on a non baseline run or not none on a baseline run
Done when: the names scan test finds no hardcoded run id attribute name in core/runs.go, and invalid-run tests cover an unknown mode, none on a faulted run and single on a baseline run, each with a reason naming spanslo.mode
Allowed models: Claude
Depends on: T-042
Milestone: M7
Verify: go test ./core/...
Status: done

## T-044: Week 6 fixtures through ScoreRun
Satisfies: R-032, R-033
Inputs: T-017 fixtures, week6_validation_test.go, T-041 to T-043
Outputs: the T-017 fixtures run through ScoreRun and the evaluate helper that re-implements scoreEpisodes is deleted
Done when: the unrelated tool run, the four mode fixtures and the excess retries fixture assert on the record returned by ScoreRun, and no test helper calls scoreEpisodes directly
Allowed models: Claude
Depends on: T-043
Milestone: M7
Verify: go test ./core/...
Status: done

## T-018: Aggregation and baselines
Satisfies: R-034, R-035, R-036
Inputs: run records, design.md "Safeguards"
Outputs: core/aggregate.go rolling up by experiment_id, framework, model and fault; triggered only filter; baseline sets keyed by task_id, framework, model with medians
Done when: a test shows fault_not_applied and untriggered runs excluded from fault SLIs and baseline medians taken over the matching key only
Allowed models: Claude
Depends on: T-016
Milestone: M7
Verify: go test ./core/...
Status: done

## T-019: Ratio SLIs and the undefined rule
Satisfies: R-037, R-038, R-039, R-040, R-041, R-042, R-043, R-066
Inputs: T-018
Outputs: S0, Sf, RR and the four rates; an undefined result (not 0 or 1) when a denominator is zero
Done when: tests cover no baseline runs, no triggered runs, S0 equal to 0 and a normal case for every SLI
Allowed models: Claude
Depends on: T-018
Milestone: M7
Verify: go test ./core/...
Status: done

## T-020: TAF and turn amplification
Satisfies: R-044, R-045, R-066
Inputs: T-018 baseline medians
Outputs: per run taf and turn_amplification, undefined when task_id is null, no matching baseline, or median is 0
Done when: tests cover each undefined case and a computed case
Allowed models: Claude
Depends on: T-018
Milestone: M7
Verify: go test ./core/...
Status: done

## T-021: Percentiles with unrecovered counts
Satisfies: R-046
Inputs: T-013 episode values, T-020
Outputs: p50 and p95 for the four episode SLIs, taf and turn_amplification, excluding unrecovered episodes, reported with the unrecovered count
Done when: a test shows unrecovered episodes absent from percentiles and counted in the reported unrecovered count
Allowed models: Claude
Depends on: T-019, T-020
Milestone: M7
Verify: go test ./core/...
Status: done

## T-022: SLO file reader
Satisfies: R-048
Inputs: spec/slo.schema.json, design.md "SLOs and error budgets"
Outputs: SLO file parser (slos list, windows map, the three forms, burn default true); rejects files that are not exactly one form
Done when: a valid file parses and files with two forms, no form or a bad window are rejected
Allowed models: Claude
Depends on: T-001
Milestone: M7
Verify: go test ./core/...
Status: done

## T-023: Error budgets and SLO evaluation
Satisfies: R-049, R-050, R-051, R-054, R-067
Inputs: T-019, T-021, T-022
Outputs: budget per form, allowed, bad, remaining, exhausted when bad is greater than allowed; threshold_max evaluation with unrecovered episodes as violations for turns_to_recover and time_to_recover only; resilience SLO compared with its target only; no burn for burn false; zero budget handling
Done when: tests cover each of the three forms, the unrecovered rule per SLI, an RR SLO with no budget, and a zero budget giving +Inf burn for bad above 0 and 0 otherwise, never NaN
Allowed models: Claude
Depends on: T-021, T-022
Milestone: M7
Verify: go test ./core/...
Status: done

## T-024: Windows and burn rate alerts
Satisfies: R-052, R-053
Inputs: T-023, ended_at on records
Outputs: "now" as latest ended_at, time windows, count windows ordered by ended_at then run_id, burn rate per long and short window, alert only when both reach factor
Done when: tests show a run on the time window edge, count window tie ordering by run_id, and no alert when only the long window reaches factor
Allowed models: Claude
Depends on: T-023
Milestone: M7
Verify: go test ./core/...
Status: done

## T-025: Week 7 unit and property tests
Satisfies: R-047, R-055
Inputs: T-019 to T-024
Outputs: unit tests for zero baselines and untriggered faults; property tests (stdlib testing/quick) for RR 1.0 when faulted equals baseline, taf 1.0 for identical runs, remaining at least 0 while bad is at most allowed
Done when: unit tests give the R-066 and R-028 results and the property tests pass
Allowed models: Claude, Gemini Flash
Depends on: T-024
Milestone: M7
Verify: go test ./core/...
Status: done

## T-026: Canonical fault event, bench adapter, passthrough
Satisfies: R-056, R-057, R-058
Inputs: design.md "Canonical fault event", spanslo.fault.* names
Outputs: core/adapters/bench.go converting bench fault records to canonical events; unknown sources pass through untouched
Done when: a bench record converts with all R-057 attributes and an unknown source record is neither changed nor dropped
Allowed models: Claude
Depends on: T-003
Milestone: M8
Verify: go test ./core/...
Status: done

## T-045: Resource attributes onto spans
Satisfies: R-001, R-078
Inputs: core/otlp.go, core/runs.go, core/score.go, design.md v2.15 "Run identity"
Outputs: ParseOTLP copies each resource's attributes onto every span of that resource; a span's own attribute wins on a name clash
Done when: a fixture that writes spanslo.run.id, spanslo.experiment.id, spanslo.framework and spanslo.model only as resource attributes groups into one run and ScoreRun reads all four; a span attribute with the same name as a resource attribute keeps the span's value; a span in a resource with no attributes is unchanged
Allowed models: Claude
Depends on: T-044
Milestone: M8
Verify: go test ./core/...
Status: done

## T-046: Fallback recovery for model surface faults
Satisfies: R-012
Inputs: core/recovery.go, T-042, design.md v2.15 "Fault episodes"
Outputs: for a model surface fault in any mode, a successful call to a declared fallback tool after the fault also qualifies as recovery, in addition to the next successful chat span; the persistent rule keeps excluding later chat spans on the target
Done when: tests show a non persistent model surface episode recovering through a successful fallback tool call with no later successful chat span, not recovering when the fallback call has an error status or a later fallback call errors, and a persistent model surface episode still recovering only through the fallback; a task with no declared fallback behaves as before
Allowed models: Claude
Depends on: T-045
Milestone: M8
Verify: go test ./core/...
Status: done

## T-047: Baselines from the whole dataset before windows
Satisfies: R-036, R-052
Inputs: core/budget.go, core/windows.go, core/aggregate.go, T-018, T-024, design.md v2.15 "SLI and SLO definitions"
Outputs: baselines for taf, turn_amplification, resilience ratio and the other baseline keyed SLIs are built from every baseline run in the input dataset before any window trims runs, keyed per run by task, framework and model; the window selects only the runs whose SLI is pooled
Done when: a test where a count window drops the older baseline runs gives the same TAF and resilience ratio as the same runs with no window; a run is compared with the baseline of its own task, framework and model, not a pooled one; a run with no matching baseline is undefined under the existing rule
Allowed models: Claude
Depends on: T-046
Milestone: M8
Verify: go test ./core/...
Status: done

## T-027: spanslo score command
Satisfies: R-060, R-016, R-018, R-025
Inputs: T-016, T-026, T-045, design.md v2.15 (contract for a run: the default contract for a baseline run, the effective contract of the fault id for a faulted run; a run whose faults carry more than one distinct fault id is invalid); cmd/spanslo/main.go, cmd/spanslo/*.go, go.mod (the module path is github.com/sankaranainar-p/spanslo; import core as github.com/sankaranainar-p/spanslo/core)
Outputs: cmd/spanslo/main.go (currently a stub that prints "spanslo dev"; replace it with the command dispatch) and cmd/spanslo score: reads traces, runs adapters, writes JSON lines, deletes the old sidecar first, writes <out>.invalid.jsonl (run_id, reason), prints each run status and names invalid runs, prints core warnings to stderr; exits 0 with invalid runs, 2 with no sidecar on unreadable input or usage error
Done when: tests show a mixed input exits 0 with a sidecar naming the invalid run, a stale sidecar is replaced, and unparseable input exits 2 and leaves no sidecar; a baseline run is scored with the default contract, a faulted run with the effective contract of its fault id, and a run whose faults carry two distinct spanslo.fault.id values is invalid with a reason naming spanslo.fault.id
Allowed models: Claude
Depends on: T-016, T-026, T-045
Milestone: M8
Verify: go test ./cmd/...
Status: done

## T-028: Run records validate against the schema
Satisfies: R-031
Inputs: spec/run-record.schema.json, T-027 score command, testdata fixtures
Outputs: bench/tests/test_run_records.py, a Python test that runs the spanslo score command on the fixtures and validates every emitted record against spec/run-record.schema.json with jsonschema
Done when: every record emitted from the fixtures validates, and a record with a missing required field fails the test
Allowed models: Claude
Depends on: T-017, T-027
Milestone: M8
Verify: uv run --project bench pytest bench/tests/test_run_records.py
Status: done

## T-029: spanslo slo command
Satisfies: R-061
Inputs: T-024, T-022; cmd/spanslo/main.go, cmd/spanslo/*.go, go.mod (the module path is github.com/sankaranainar-p/spanslo; import core as github.com/sankaranainar-p/spanslo/core)
Outputs: cmd/spanslo slo: reads JSON lines runs and the SLO file, prints met or not, budget remaining and each burn alert
Done when: a test prints the expected lines for a met SLO, a breached SLO and a firing alert, and exits 0 for all
Allowed models: Claude
Depends on: T-024, T-027
Milestone: M8
Verify: go test ./cmd/...
Status: done

## T-030: spanslo report command
Satisfies: R-062
Inputs: T-021, T-024; cmd/spanslo/main.go, cmd/spanslo/*.go, go.mod (the module path is github.com/sankaranainar-p/spanslo; import core as github.com/sankaranainar-p/spanslo/core)
Outputs: cmd/spanslo report writing spanslo's own JSON: SLI values, percentiles with unrecovered counts, SLO results
Done when: a test compares the report on a fixture with a hand checked expected JSON and exits 0 when SLOs are breached
Allowed models: Claude
Depends on: T-029
Milestone: M8
Verify: go test ./cmd/...
Status: done

## T-031: spanslo gate command
Satisfies: R-063
Inputs: T-029, T-027 sidecar; cmd/spanslo/main.go, cmd/spanslo/*.go, go.mod (the module path is github.com/sankaranainar-p/spanslo; import core as github.com/sankaranainar-p/spanslo/core)
Outputs: cmd/spanslo gate: checks the sidecar first (missing or non empty gives exit 2), then exits 1 on an unmet SLO or firing alert, else 0
Done when: tests show exit 2 for a missing sidecar and a non empty sidecar, exit 1 for a breaching fixture, exit 0 for a clean fixture, and exit 2 for empty input
Allowed models: Claude
Depends on: T-029
Milestone: M8
Verify: go test ./cmd/...
Status: done

## T-032: Exit code audit across commands
Satisfies: R-068
Inputs: T-027, T-029 to T-031; cmd/spanslo/main.go, cmd/spanslo/*.go, go.mod (the module path is github.com/sankaranainar-p/spanslo; import core as github.com/sankaranainar-p/spanslo/core)
Outputs: table test for every command: unknown flag, missing file, unparseable runs or SLO file and unparseable ended_at all exit 2; slo and report exit 0 on breach
Done when: the table test passes for score, slo, report and gate
Allowed models: Claude, Gemini Flash
Depends on: T-031
Milestone: M8
Verify: go test ./cmd/...
Status: done

## T-033: Golden traces, model faults
Satisfies: R-064
Inputs: catalog/ (model-malformed-tool-args, model-rate-limit-429, model-server-error-5xx, model-timeout, model-truncated-stream), T-027
Outputs: testdata/traces/<fault>.json for the 5 model-* faults with hand checked expected run records and a golden test; expected records are written by hand, not generated from scorer output
Done when: one golden trace exists for each of the 5 model-* faults and the golden test passes
Allowed models: Claude
Depends on: T-027
Milestone: M8
Verify: go test ./...
Status: done

## T-034: Golden traces, tool faults and memory fault
Satisfies: R-064
Inputs: catalog/ (the 6 tool-* faults and memory-irrelevant-retrieval), T-027
Outputs: testdata/traces/<fault>.json for the 6 tool-* faults and memory-irrelevant-retrieval (7) with hand checked expected run records and a golden test; expected records are written by hand, not generated from scorer output
Done when: one golden trace exists for each of those 7 faults and the golden test passes
Allowed models: Claude
Depends on: T-027
Milestone: M8
Verify: go test ./...
Status: done

## T-035: Week 8 validation checks
Satisfies: R-065
Inputs: T-031, T-033, T-034
Outputs: end to end test of the three gate and score cases
Done when: gate exits 1 on a breaching fixture, exits 2 with a missing and with a non empty sidecar, and score removes a stale sidecar and writes none on exit 2
Allowed models: Claude, Gemini Flash
Depends on: T-033, T-034
Milestone: M8
Verify: go test ./...
Status: done

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

## T-048: Group and baseline rows for the report
Satisfies: R-082, R-083
Inputs: core/aggregate.go, core/sli.go, core/percentiles.go, core/score.go, design.md v2.17 "CLI" (report bullet)
Outputs: core/report.go with pure functions ReportGroups(runs, baselines) and ReportBaselines(baselines) returning typed rows sorted as R-082 and R-083 say; no I/O, no clock
Done when: tests cover two experiments, two frameworks, two models and two faults; a run with two triggered fault ids appears in both groups; a group with no matching baseline has resilience_ratio undefined; baseline rows hold runs, baseline_success and the two medians; null task_id baselines are left out; rows are sorted
Allowed models: Claude
Depends on: T-047
Milestone: V021
Verify: go test ./core/...
Status: open

## T-049: Percentile keys in snake case
Satisfies: R-085
Inputs: core/percentiles.go, core/percentiles_test.go, cmd/spanslo/report.go, design.md v2.17 "CLI" (report bullet)
Outputs: the percentile types marshal to JSON as turns_to_recover, time_to_recover, cascade_spans, recovery_tokens, taf and turn_amplification, each with p50, p95 and unrecovered; p50 and p95 are null when undefined; no ok key
Done when: a test marshals a percentile set with defined and undefined values and checks the exact JSON keys and the nulls; existing percentile tests still pass
Allowed models: Claude
Depends on: T-047
Milestone: V021
Verify: go test ./...
Status: open

## T-050: Report command writes groups and baselines
Satisfies: R-062, R-082, R-083, R-084, R-085
Inputs: cmd/spanslo/report.go, cmd/spanslo/main.go, cmd/spanslo/README.md, core/report.go, core/percentiles.go, go.mod, design.md v2.17 "CLI" (report bullet)
Outputs: spanslo report writes top level slis, percentiles, slos (unchanged meaning), plus groups and baselines; cmd/spanslo/README.md describes the new keys
Done when: a CLI test on run records with baselines and faulted runs checks groups, baselines, the pooled top level and that undefined values are null; exit codes are unchanged
Allowed models: Claude
Depends on: T-048, T-049
Milestone: V021
Verify: go test ./...
Status: open

## T-051: End to end report check
Satisfies: R-086
Inputs: cmd/spanslo/golden_test.go, testdata/traces/model-timeout.json, testdata/traces/model-timeout.expected.json, cmd/spanslo/main.go, cmd/spanslo/report.go, go.mod
Outputs: fixtures under testdata/report/: two baseline traces (one success, one fail) and faulted traces of the same task, framework and model, plus a trace that triggers two fault ids; cmd/spanslo/report_e2e_test.go that runs score on each, joins the outputs and their sidecars, runs report, and checks the JSON
Done when: the test checks groups, baselines, a resilience_ratio that equals a value computed by hand in a comment, the second fault id appearing in two groups, and no null where the data defines a value; it runs through the real run function, not core helpers
Allowed models: Claude
Depends on: T-050
Milestone: V021
Verify: go test ./...
Status: open
