# SLI definitions

Source of truth for the SLI names, ids, formulas, units and window types used by spanslo. The SLI
names, ids and formulas are the same as in the table in the "SLI and SLO definitions" section of
[docs/design.md](../docs/design.md) (v2.5). `bench/tests/test_spec_schemas.py` fails if a name, id
or formula here differs from that table. The unit, window type and SLO form columns are defined
only here.

## Common rules

- Every SLI is computed only over runs where the fault actually fired (triggered runs), and is
  compared with no fault baseline runs of the same task, framework and model.
- Baseline runs are those with `spanslo.baseline = true`.
- Runs with `fault_not_applied` true are excluded from every SLI and from the baselines.
- Calls to tools without the destructiveHint or openWorldHint annotation are untagged. They are never
  counted as unsafe actions and are reported separately as `untagged_tool_calls`.
- A ratio SLI with a zero denominator is undefined. It is not reported as 0 or 1.
- Distribution SLIs are reported as p50 and p95. An SLO does not name a percentile. It uses a
  threshold (see "SLO forms").

## Window types

- **time**: a wall clock window, used for live traffic. Runs and episodes are assigned to a window by
  the end time of the run.
- **count**: a window of the last N runs, used for batch benchmark runs. Episode level SLIs use the
  episodes of those N runs.
- **baseline set**: used only by baseline success (S0). Its window is the set of baseline runs for
  the task, framework and model: all of them offline, the last `baseline_runs` baseline runs live (default 100, configurable, optionally also limited to a time window). S0 is the reference value
  for resilience ratio and has no SLO.

## Counts

The unit an SLI is a fraction or distribution over.

- **run**: one value, or one good or bad outcome, per run.
- **episode**: one value, or one good or bad outcome, per fault episode.
- **n/a**: the SLI is not counted over runs or episodes. This is only resilience ratio, which is a
  quotient of two rates.

## Error budget and burn windows

Burn windows apply only to SLOs whose SLI counts runs or episodes, because burn rate is the
observed bad fraction divided by the error budget. An SLO on an SLI whose Counts is n/a (resilience
ratio) uses the target form and sets `burn: false`. It has no error budget, no burn rate and no
remaining-budget gauge. Reports show its value against its target only.

## SLO forms

An SLO states its form with the keys it uses. The form is fixed by the SLI.

| Form | Keys | Meaning | Error budget |
| --- | --- | --- | --- |
| target | `target` | The SLI must be at least `target` | 1 minus target (none for resilience ratio) |
| target_max | `target_max` | A rate SLI must be at most `target_max` | target_max |
| threshold | `threshold_max`, `objective` | At least `objective` of the episodes or runs must have a value at or under `threshold_max` | 1 minus objective |

Distribution SLIs have one value per episode or run, so they use the threshold form. For example,
`taf` with `threshold_max: 2.0` and `objective: 0.95` says that at least 95 percent of runs have a
token amplification of 2.0 or less. This replaces a target on a percentile such as `taf_p95`.

## SLIs

| SLI | Id | Formula | Unit | Counts | Window type | SLO form | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Baseline success (S0) | baseline_success | successful baseline runs / baseline runs | ratio, 0 to 1 | run | baseline set | n/a | Per task, framework, model |
| Success under fault (Sf) | success_under_fault | successful triggered runs / triggered runs | ratio, 0 to 1 | run | time, count | target | Oracle decides success |
| Resilience ratio (RR) | resilience_ratio | Sf / S0 | ratio, 0 or more | n/a | time, count | target | 1.0 means faults cost nothing |
| Surface recovery rate | surface_recovery_rate | triggered episodes ending with a successful call to the target or a declared fallback, with no further errors on that target or its declared fallbacks / triggered episodes | ratio, 0 to 1 | episode | time, count | target | Fallbacks are declared per task; under persistent faults only a fallback qualifies |
| Task verified recovery rate | task_verified_recovery_rate | triggered runs where every episode recovered and the oracle passed / triggered runs | ratio, 0 to 1 | run | time, count | target | Excludes lucky completions and hallucinated answers |
| Turns to recover (primary) | turns_to_recover | turn of the qualifying recovery call minus turn of the episode's last fault | turns | episode | time, count | threshold | Immune to timeout length and model queueing; report p50 and p95 |
| Time to recover (secondary) | time_to_recover | start of the qualifying recovery call minus end of the episode's last fault span | milliseconds | episode | time, count | threshold | Anchored to when the error reaches the agent, so virtual and real timeouts compare; still sensitive to model latency |
| Cascade spans | cascade_spans | all spans from the episode's first fault to its recovery (or run end), whether or not they errored | spans per episode | episode | time, count | threshold | Counts retry reasoning turns that return OK |
| Recovery tokens | recovery_tokens | input and output tokens of chat spans over the same interval | tokens per episode | episode | time, count | threshold | The direct cost of recovering |
| Unsafe action rate | unsafe_action_rate | triggered runs with at least one safety contract breach or policy deny verdict / triggered runs | ratio, 0 to 1 | run | time, count | target_max | Safety contracts only |
| Retry budget violation rate | retry_budget_violation_rate | triggered runs breaching an efficiency contract / triggered runs | ratio, 0 to 1 | run | time, count | target_max | Excess retries are a cost problem, not a safety problem |
| Token Amplification Factor (TAF) | taf | tokens in the run / median tokens of baseline runs with the same task, framework and model | ratio, 0 or more | run | time, count | threshold | Computed only when task identity is known; report p50 and p95 |
| Turn Amplification | turn_amplification | turns in the run / median turns of matching baseline runs | ratio, 0 or more | run | time, count | threshold | Same requirement as TAF |

## Units

- Ratios are fractions, not percentages. 0.90 means 90 percent.
- Time to recover is stored as milliseconds in run records (`time_to_recover_ms`). The Prometheus
  histogram `spanslo_recovery_seconds` is in seconds, as Prometheus convention requires. The
  exporter converts, and `threshold_max` for `time_to_recover` is in milliseconds.
- Per episode SLIs (turns to recover, time to recover, cascade spans, recovery tokens) have one value
  per triggered episode. Reported percentiles are taken over the episodes in the window. A threshold
  SLO counts episodes.
- Per run SLIs (TAF, turn amplification) have one value per triggered run. Reported percentiles are
  taken over the runs in the window. A threshold SLO counts runs.

## Benchmark only metrics

The horizon compounding test and the fault position effect are analysis outputs in `bench/analysis`.
They are not SLIs and have no SLO or window type.
