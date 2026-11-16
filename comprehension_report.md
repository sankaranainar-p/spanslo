# Stage 6: Comprehension Test of spec.md (lines 1 to 97)

Reviewer: egm-comprehension. The reviewer read spec.md alone, with no design.md and no other context.

## What this proposes
Build, in Go, an offline scoring core plus a CLI (`spanslo score`, `slo`, `report`, `gate`) for weeks 5 to 8. It reads recorded OTLP JSON traces and groups spans into runs. It indexes turns and steps, groups injected faults into episodes, and checks recovery and safety contracts. It then emits run records, computes SLIs and baselines, evaluates SLOs with error budgets and burn rates, and writes an "agent-sre" report. Adapters turn different fault-injector outputs into one canonical fault event. Collector, Prometheus, Grafana and Python harness work are excluded. The spec is a numbered requirements table (R-001 to R-065) with per-week "done" criteria.

## How the existing system works, per this document
The document says very little about the existing system. It implies that a fault injector or harness records faults as events (`spanslo.fault.*`) on OpenTelemetry traces. The traces also carry `spanslo.run.id`, `spanslo.mode`, an evaluation-result event `spanslo.task_success`, and oracle attributes. It implies that `docs/design.md` defines the data model, SLI table, fault catalog and run record. It also implies that `spec/semconv-mapping.yaml`, `spec/run-record.schema.json`, `spec/slo.schema.json` and `catalog/defaults.yaml` already exist. How an SLO file looks and how the fault catalog is structured is not shown here. The reader cannot reconstruct the system from this file alone.

## Findings

| # | Severity | Scope | Finding |
| --- | --- | --- | --- |
| 1 | Blocker | Whole file, every Source column, R-007 to R-064 | The spec's meaning depends on `docs/design.md`, which is cited by section name but never summarised or linked. Episode, run record, SLI, SLO, fault catalog, contract, baseline, oracle, mode, surface, target and fallback are undefined here. Add a glossary or inline definitions, and link design.md. |
| 2 | Blocker | R-048, R-049 to R-054 | R-048 points to a section that lives in design.md. The keys `target`, `target_max`, `threshold_max`, `objective`, `burn` and `windows` have no schema or example here. "Long and short window pair" (R-053) and "bullet 1/2/3/4" are undefined. R-051 cannot be understood without the file shape. |
| 3 | Major | R-010, R-012, R-013, R-017, R-018 | "Qualifying recovery", "declared fallback" and "persistent mode" are undefined. The modes are never listed. The spec does not say where a fallback is declared. R-012 reads circularly. |
| 4 | Major | R-021 to R-024, R-042 | `must_not_call_tags`, the retry limits, "policy deny verdict", `destructiveHint` and `openWorldHint` are unexplained, as is how a tag relates to an annotation. R-023 is ambiguous about whether the same tool limit counts consecutive calls. R-042 introduces "policy deny verdict" with no source. |
| 5 | Major | R-036, R-039, R-044, R-045, R-055 | Sf, S0, RR and TAF are used without expansion, and Sf and S0 are not tied to `success_under_fault` and `baseline_success`. "Task identity is known" is undefined, as is the behavior when it is unknown. R-036 says "baseline median" while R-037 defines a rate. |
| 6 | Major | R-015, R-016, R-026 | The "oracle" is never defined, and its relation to the `task_success` event is unclear. R-026 does not say what `graceful_error` is set to. |
| 7 | Major | R-056 to R-059 | The four adapter sources are unexplained and unlinked. The canonical fault event fields have no meanings. R-059 does not name the environment variable or the matching rule. "Pass through untouched" does not say to where. |
| 8 | Major | R-062 | The "agent-sre shaped report" has no definition, link or schema, so it cannot be built or tested from this file. |
| 9 | Major | R-005, R-006, R-009 | Handling of reasoning only turns, and of tool calls with no chat span, is not stated. "Canonicalized JSON arguments" has no rule. |
| 10 | Major | R-017 to R-020 | "Run end" is undefined when no recovery occurs, and the values for an unrecovered episode are not given. The behavior for virtual duration faults is not stated. |
| 11 | Nit | R-002, R-030 | "Week 3 mapping" and "commit pinned in the mapping" are not explained. |
| 12 | Nit | Header, Goal | Gate G2, CLAUDE.md language boundaries and who approves the draft are not explained. "Weeks 5 to 8" has no calendar anchor. |
| 13 | Nit | R-047, R-055 | Test requirements with undefined thresholds. "Budgets never go negative before exhaustion" is unclear. |
| 14 | Nit | R-001, R-003, R-060 | The relation between "raw" and "normalized" traces and the adapters is never explained. |

## Elephant note (not part of the reviewer's report)
This reviewer had no access to `docs/design.md` by design. Findings 1 and 2 follow from spec.md citing design.md by section name instead of restating it. The user asked for citations, not restatements, so these are a scope call for the human. Findings 4, 6 and 10 match design questions already raised in the hand off notes (policy deny verdict, unrecovered episodes).
