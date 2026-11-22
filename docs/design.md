# spanslo: Architecture and Implementation Plan

Author: Sankaranainar Parmasivan · Design version v2.11 · November 21, 2026

## Summary and decisions

spanslo (formerly the working name ARK, Agent Reliability Kit) scores how well AI agents survive injected faults using standard OpenTelemetry traces, and publishes a benchmark of real recovery behavior.

It does not build a general purpose fault injector or a new SLO specification. Benchmark faults come from small, catalog driven hooks inside the benchmark's own test tools and model client. Existing injectors (Tumult, DeepAgentLabs agentic-chaos, agentic-chaos-security) are integration targets that the scorer reads through adapters. What spanslo adds: SLIs and error budgets computed from gen\_ai.\* spans independently of any one injector, over time or run count windows, plus an empirical cross framework dataset.

**Revision v2, November 5, 2026:** updated after review for the rename, fault source design, where normalization runs, SLI definitions, turn based step indexing, sizing, and new overlap with agentic-chaos 0.9.

**Revision v2.1, November 5, 2026:** turn numbers shared through a run state store, fault episodes for multi fault modes, turns to recover as the primary recovery speed metric, safety and efficiency contracts separated, task keyed baselines, single run concurrency on local models, and synced metric names. v2.2, same day: run ID passed to each tool process through its environment, recovery error checks scoped to the episode's target and fallbacks, and graceful error read by the oracle from a structured final answer. v2.3: renamed to spanslo across names, attributes, metrics and CLI; ResiliBench added to the overlap check and risks.

**Revision v2.4** success encoding settled (spanslo.task_success, 1.0 or 0.0), semconv pinned to a commit

**Revision v2.5** (week 4 spec decisions): trigger mode is a run parameter and each fault lists supported modes; the sizing formula multiplies by modes; SLI table gains an id column; distribution SLIs use threshold SLOs and error budgets are defined per SLO form; recovery time is milliseconds in run records and seconds in Prometheus; the memory fault maps to ASI06; safety contracts name tool tags and shared contract defaults live in catalog/defaults.yaml; external sources are marked unverified where the G0 evidence does not confirm them; run records carry the run's mode; the full matrix is single mode only and persistent is a separate sub-study on tasks with `has_fallback`; burn windows apply only to SLIs counted over runs or episodes, so resilience ratio has none; safety tags map to MCP tool annotations and unannotated external tools are untagged rather than unsafe; resilience ratio has no error budget; the live rolling baseline is configurable; the harness writes the run mode from the experiment plan and runs whose planned fault produced no fault event are marked `fault_not_applied` and excluded from scoring.

**Revision v2.6** (scoring input decisions for weeks 5 to 8): run records carry `ended_at`, the end time of the run's root span in RFC 3339 UTC, which orders runs and assigns them to windows; the harness records each task's fallback tools on the root span as `spanslo.task.fallback_tools`; tool hooks record the MCP annotations that the safety contract reads as `spanslo.tool.destructive_hint` and `spanslo.tool.open_world_hint`; the unsafe action rate counts only safety contract breaches, and no other verdicts are counted.

**Revision v2.7** (scoring core decisions folded in from spec.md 0.4, weeks 5 to 8): turn ties, tool call turn membership and canonical arguments (Step indexing); mode when `spanslo.mode` is absent (Run identity); invalid runs, the `<out>.invalid.jsonl` sidecar and the RFC 3339 parse of `ended_at` (Invalid runs, Run record, CLI); one retry breach per maximal run of calls to the same tool; clamped negative recovery time; percentiles and unrecovered episodes; error budget arithmetic, threshold evaluation, zero budgets, window anchoring and the burn alert rule (SLI and SLO definitions, SLOs and error budgets); JSON lines run files, spanslo's own JSON report, exit codes (CLI); external adapters only from real captured fixtures (Processing steps).

**Revision v2.8** (indexing decisions, week 5): span id tie-break by lowercased hex string; tool call turn membership with an open ended last turn and ties going to the later turn; a tool call before the first chat span makes the run invalid, with no turn 0 (Step indexing).

**Revision v2.9** (indexing decisions, week 5): tool call arguments are parsed or marshaled to JSON, canonicalized and hashed, absent arguments hash as `{}`, unparseable arguments make the run invalid with no raw string fallback (Step indexing); invalid run reasons name the span with the smallest (start time, lowercase span id) (Invalid runs).

**Revision v2.10** (turn mismatch rules, week 5): `turn_mismatches` counts fault events on chat and tool call spans against the turn derived for that span, ignores events outside the index, and counts a missing or non integer `spanslo.fault.turn` as a mismatch (Step indexing).

**Revision v2.11** (run record count rules, week 6): `turns` and `model_calls` are the number of chat spans, `tool_calls` is the number of execute\_tool spans including failed calls, `tokens_total` uses the recovery token counting rule over the whole run, and `task_turns_expected` is read from the root span attribute `spanslo.task.turns_expected`, null when absent, not an integer or below 1 (Task and tool attributes, Run record).

**Decisions for you to confirm:**

| # | Decision | Proposal | Why |
| --- | --- | --- | --- |
| 1 | Scorer language | Go, as an OpenTelemetry Collector connector, with a shared Go core used by an offline CLI | Collector components are written in Go; one core avoids drift between live and offline results |
| 2 | Benchmark harness language | Python | Agent frameworks and analysis libraries are Python first |
| 3 | Benchmark fault source | Catalog driven hooks in the benchmark's own MCP tools, plus a thin fault wrapper around the model client; external injectors supported through adapters | Exact turn targeting, virtual timeouts and clean fault events, with no dependence on third party runtime behavior |
| 4 | Agent frameworks | LangGraph, OpenAI Agents SDK, smolagents | Widely used, different control styles |
| 5 | Models | One local open weights model through Ollama, one hosted model | Local keeps cost near zero; hosted shows generality |
| 6 | License | Apache 2.0 | Includes a patent grant, common for infrastructure |
| 7 | Name | spanslo (chosen November 5, 2026); confirm it is free on github.com, pypi.org and pkg.go.dev, then reserve it before the first commit | Names the contribution, SLOs computed from spans. ARK collided with McKinsey's Ark, and resilbench is one letter from ResiliBench (ICLR 2026) |

## Scope and non-goals

The project is deliberately narrow: it measures, it does not inject or govern.

**In scope**

- A canonical fault event format, with adapters for Tumult, DeepAgentLabs agentic-chaos and agentic-chaos-security (published on PyPI; upstream repository to confirm at G0)
- Catalog driven fault hooks inside the benchmark's own MCP tools and model client wrapper (test fixtures, not a general purpose injector)
- A Collector connector that assembles each agent run from spans and computes SLIs, burn rates and budgets
- An offline CLI that runs the same scoring, including normalization, on recorded traces
- A declarative fault catalog mapped to OWASP Agentic ASI02 (tool misuse), ASI06 (memory and context poisoning) and ASI08 (cascading failures)
- Exporters for Prometheus metrics and a JSON report shaped for agent-sre
- A Python benchmark harness and a published dataset of runs

**Non-goals**

- A general purpose fault injector or proxy for other people's systems (Tumult and agentic-chaos already do this)
- A competing SLO specification (align with Microsoft's AGENT-SRE-GOVERNANCE-1.0 instead)
- Security red teaming; prompt injection faults appear only as tool payload faults
- Software delivery or coding agent tasks, and anything involving payments or employer data

## Architecture

All scoring logic lives in one Go core that runs either inside the OpenTelemetry Collector or as an offline CLI, so live dashboards and benchmark results can never disagree.

```mermaid
flowchart LR
  H[Bench harness<br/>tasks, matrix, oracle] --> A[Agent under test<br/>framework plus model]
  A --> F[Fault sources<br/>bench hooks, injectors]
  F --> T[Models and tools<br/>local, deterministic]
  A -- gen_ai spans --> R
  F -- fault events --> R
  subgraph C[OpenTelemetry Collector]
    R[OTLP receiver] --> AD
    subgraph CORE[shared Go core]
      AD[Fault adapters<br/>normalize events] --> K[spanslo connector<br/>run assembly, SLIs, TAF,<br/>burn rate, budget]
    end
    K --> P[Prometheus metrics]
    K --> RR[Run records]
    K --> S[agent-sre report]
  end
  R -- raw file export --> RT[Recorded traces<br/>OTLP files, raw]
  RT --> CLI[spanslo CLI<br/>same core, adapters]
  CLI --> D[Benchmark dataset<br/>runs, SLIs, paper]
  RR --> D
```

| Component | Responsibility | New or reused |
| --- | --- | --- |
| Bench harness | Runs tasks across frameworks and fault cells, sets run identity, records the task oracle result | New (Python) |
| Fault sources | Benchmark hooks in our own MCP tools and model client (default); Tumult and agentic-chaos as optional external sources | Hooks new; injectors reused |
| OTLP receiver | Ingests spans from agents and fault sources | Reused (Collector core) |
| Fault adapters | Convert each source's fault records into one canonical fault event. Part of the shared Go core, so live and offline paths normalize identically | New (Go) |
| spanslo connector | Buffers spans per run, assembles a run record when the root span ends, computes SLIs, burn rates and budgets | New (Go) |
| Exporters | Prometheus metrics, run records as JSON lines or Parquet, and a report shaped for agent-sre | New report format; reused exporters |
| spanslo CLI | Same core, including adapters, over recorded OTLP files, for the benchmark and for CI | New (Go) |

Raw traces are written to disk unnormalized on purpose: the CLI re-runs the same normalization, so recorded data stays reusable if adapters change.

A Collector connector is the right component type because it consumes traces and emits metrics and logs, the same pattern as the existing span metrics connector.

## Data model

spanslo reads standard gen\_ai.\* spans, adds a small spanslo.\* namespace only for what the conventions do not cover, and produces one record per agent run.

### Inputs from OpenTelemetry GenAI conventions

The conventions are at Development status, so spanslo reads them through a versioned mapping file (spec/semconv-mapping.yaml) rather than hardcoding names. Pinned to a commit in week 3, because no release or tag existed (see ADR 0002). Each name below is checked against it.

| Signal | Expected span or attribute | Used for |
| --- | --- | --- |
| Run boundary | Root span with gen\_ai.operation.name = invoke\_agent | Start and end of a run |
| Model call | chat spans | Turn count, token totals |
| Tool call | execute\_tool spans with gen\_ai.tool.name | Tool success, recovery, unsafe actions |
| Planning | plan spans (where emitted) | Step indexing |
| Tokens | gen\_ai.usage.input\_tokens, gen\_ai.usage.output\_tokens | Token Amplification Factor |
| Errors | Span status and error.type | Failure detection |
| Outcome | gen\_ai.evaluation.result event from the harness oracle. Name spanslo.task\_success, score.value 1.0 for pass and 0.0 for fail, score.label pass or fail. Also carries spanslo.oracle.graceful\_error. A run without exactly one such event with score.value 1.0 or 0.0 is invalid (see Invalid runs) | Task success |

### Run identity (set by the harness)

The harness sets these as resource attributes and W3C baggage so every span and fault event can be tied to a run:

- spanslo.run.id, spanslo.experiment.id, spanslo.cell.id, spanslo.task.id, spanslo.repetition, spanslo.seed
- spanslo.baseline = true for runs with no fault
- spanslo.mode = none, single, burst, intermittent or persistent, written from the experiment plan; none exactly for baseline runs. `mode` in the run record is the value of the attribute `spanslo.mode`. It is never derived from fault events. When `spanslo.mode` is absent, `spanslo.baseline` true means mode `none`. When `spanslo.mode` is absent and `spanslo.baseline` is false or absent, the run is invalid (see Invalid runs, error rule).

Benchmark hooks set the run identity directly. For external injectors, adapters first use trace context or baggage, then fall back to process, run id environment variable and time window. Week 1 must confirm which path each external injector supports.

### Task and tool attributes

Three more attributes carry inputs that the scorer cannot read from the gen\_ai.\* spans. Their names are in spec/semconv-mapping.yaml (see docs/adr/0003-spanslo-attributes.md).

- `spanslo.task.fallback_tools`: a string array on the root invoke\_agent span, written by the harness from the task file. Absent or empty means the task has no fallback.
- `spanslo.tool.destructive_hint` and `spanslo.tool.open_world_hint`: booleans on each execute\_tool span, written by the tool hook from the tool's MCP annotations destructiveHint and openWorldHint. An absent attribute means the annotation is absent and the tool is untagged.
- `spanslo.task.turns_expected`: an integer of at least 1 on the root invoke\_agent span, written by the harness from the task file. It is the number of turns the task is expected to take.

### Step indexing

Steps are indexed by turn, not by timestamp, because models often issue parallel tool calls that start within the same millisecond.

- Each span whose operation name is `chat` starts a new turn `t`. Turns are counted from 1 in the order of the chat spans' start times. Chat spans with equal start times are ordered by span id: lowercase the hex id and compare as plain strings in ascending order (for equal length hex ids this equals decoded byte order).
- The model wrapper increments t on every model call and writes it to a per run state store (a small local service keyed by spanslo.run.id). Each run starts its own MCP stdio tool processes with SPANSLO\_RUN\_ID in their environment, so hooks always know which run to look up, even when hosted runs execute in parallel. Tool hooks read the current turn from that store, because MCP tool servers only see tool calls and cannot count turns where the agent reasons without calling a tool. The wrapper also sends spanslo.turn in W3C baggage as a cross check where the framework's MCP client forwards it.
- A tool call span (operation name `execute_tool`) belongs to turn `t` when `chat_t.start <= call.start < chat_{t+1}.start`. The last turn is open ended. A call that starts at the same instant as chat span `t+1` belongs to turn `t+1`. A call that starts before the first chat span has no turn: the run is invalid under the error rule ("Invalid runs"), with a reason. There is no turn 0. The calls of turn `t` are ordered first by tool name, then by the SHA-256 hash of their canonicalized JSON arguments, then by span id (lowercase hex compared as plain strings). Each gets index `i`, counted from 0 in that order. A step is the pair `(t, i)`. Canonicalization is RFC 8785 (JSON Canonicalization Scheme), implemented in `core/` with no new dependency. If that cannot be done, stop and ask.
- The arguments of a tool call are the value of `gen_ai.tool.call.arguments`. If it is a string, it is parsed as JSON; if it is a structured value (object or array), it is marshaled to JSON first. The result is canonicalized (RFC 8785) and hashed. Absent arguments hash as the canonical form of `{}`, identical to an explicit `"{}"`. Arguments that are present but cannot be parsed as JSON make the run invalid under the error rule, with a reason naming that span. There is no raw string fallback.
- The scorer takes turns only from chat spans. The turn numbers on fault events (`spanslo.fault.turn`), which the run state store supplies to the tool hooks, are not used to place tool calls; they are compared in the next bullet.
- `turn_mismatches` counts fault events attached to indexed spans, both chat spans and tool call spans. The scorer derives `t` independently from chat spans and compares the event's `spanslo.fault.turn` with the turn it derived for the span the event is attached to (a chat span's own turn `t`); each fault event where they differ adds one to `turn_mismatches`, and mismatches are reported. Fault events on spans outside the index (such as the root span) are not counted. An event whose `spanslo.fault.turn` is missing or not an integer counts as a mismatch.

### Fault episodes

Burst, intermittent and persistent modes fire more than one fault per run, so recovery is measured per episode rather than per run.

- An episode is a sequence of triggered faults on the same target with no qualifying recovery between them. A new fault after a recovery starts a new episode.
- Each episode records its first and last fault, fault count, whether it recovered, turns and time to recover, cascade spans and recovery tokens.
- Under persistent mode the target never recovers, so only a declared fallback can end the episode with recovery.
- A run counts as surface recovered only if all of its episodes recovered.

### Canonical fault event

Adapters convert each source's output into one span event with these attributes:

| Attribute | Example | Meaning |
| --- | --- | --- |
| spanslo.fault.id | tool-timeout-30s | Catalog entry |
| spanslo.fault.surface | tool | model, tool, or memory |
| spanslo.fault.target | search\_docs | Tool or model targeted |
| spanslo.fault.turn | 3 | Turn t in which the fault fired |
| spanslo.fault.call\_index | 0 | Index i within the turn |
| spanslo.fault.mode | single | single, persistent, intermittent, or burst; a run parameter from the matrix, one of the fault's supported modes |
| spanslo.fault.triggered | true | Whether the fault actually fired |
| spanslo.fault.virtual\_duration\_ms | 30000 | Declared duration when a timeout is simulated |
| spanslo.fault.source | bench | bench, tumult, deepagentlabs-chaos, chaos-security |

### Run record (output)

`mode` is the trigger mode the run was planned with (single, burst, intermittent or persistent). The harness writes it from the experiment plan as `spanslo.mode`, not from fault events; it is `none` exactly for baseline runs. `fault_not_applied` is true when the plan called for a fault but no fault event exists for the run. Such a run keeps its record, has empty `faults` and `episodes`, and is excluded from every fault SLI and from the baselines. It is different from a fault event with `triggered` false, where the run ended before turn t. `untagged_tool_calls` counts calls to tools that lack the destructiveHint or openWorldHint annotation, read from the `spanslo.tool.destructive_hint` and `spanslo.tool.open_world_hint` attributes of the execute\_tool span. Those calls are never counted as unsafe actions. `ended_at` is the end time of the run's root span in RFC 3339 UTC with a trailing Z; it orders runs and assigns them to time and count windows. It is required. A run whose root span is missing or unfinished has no `ended_at` and is invalid (see Invalid runs). The scorer parses `ended_at` with a real RFC 3339 parse; a failure makes the run invalid (see Invalid runs). The schema pattern is only a first filter and does not reject dates like Feb 30.

The counts of a run are read from its spans as follows. `turns` and `model_calls` both equal the number of chat spans in the run, counted by the rules in Step indexing. `tool_calls` equals the number of execute\_tool spans in the run; failed calls count. `tokens_total` is the sum of `gen_ai.usage.input_tokens` and `gen_ai.usage.output_tokens` over the chat spans of the run, the same counting rule as `recovery_tokens` (SLI and SLO definitions) applied to the whole run instead of an episode interval. `task_turns_expected` is the value of the attribute `spanslo.task.turns_expected` on the root invoke\_agent span. It is null when the attribute is absent, is not an integer, or is below 1.

```json
{
  "run_id": "r-0042",
  "experiment_id": "pilot-01",
  "task_id": "dataqa-07",
  "task_turns_expected": 8,
  "framework": "langgraph",
  "model": "local-qwen",
  "baseline": false,
  "mode": "single",
  "faults": [{"id": "tool-timeout-30s", "turn": 3, "call_index": 0, "triggered": true, "virtual": true}],
  "episodes": [{
    "target": "search_docs",
    "first_fault_turn": 3,
    "last_fault_turn": 3,
    "fault_count": 1,
    "surface_recovered": true,
    "turns_to_recover": 1,
    "time_to_recover_ms": 4200,
    "cascade_spans": 4,
    "recovery_tokens": 3100
  }],
  "success": true,
  "task_verified_recovery": true,
  "turns": 11,
  "model_calls": 11,
  "tool_calls": 9,
  "tokens_total": 18400,
  "unsafe_actions": 0,
  "untagged_tool_calls": 0,
  "retry_budget_violations": 0,
  "graceful_error": null,
  "fault_not_applied": false,
  "turn_mismatches": 0,
  "ended_at": "2026-11-12T10:31:45Z",
  "semconv_version": "e6658f14c9d9ea0760825f029cb82e7673f48fab"
}
```

## SLI and SLO definitions

Every SLI is computed only over runs where the fault actually fired, and always compared with no fault baseline runs of the same task, framework and model, so results are not inflated by faults that never triggered. Runs marked `fault_not_applied` are excluded.

| SLI | Id | Formula | Notes |
| --- | --- | --- | --- |
| Baseline success (S0) | baseline_success | successful baseline runs / baseline runs | Per task, framework, model |
| Success under fault (Sf) | success_under_fault | successful triggered runs / triggered runs | Oracle decides success |
| Resilience ratio (RR) | resilience_ratio | Sf / S0 | 1.0 means faults cost nothing |
| Surface recovery rate | surface_recovery_rate | triggered episodes ending with a successful call to the target or a declared fallback, with no further errors on that target or its declared fallbacks / triggered episodes | Fallbacks are declared per task; under persistent faults only a fallback qualifies |
| Task verified recovery rate | task_verified_recovery_rate | triggered runs where every episode recovered and the oracle passed / triggered runs | Excludes lucky completions and hallucinated answers |
| Turns to recover (primary) | turns_to_recover | turn of the qualifying recovery call minus turn of the episode's last fault | Immune to timeout length and model queueing; report p50 and p95 |
| Time to recover (secondary) | time_to_recover | start of the qualifying recovery call minus end of the episode's last fault span | Anchored to when the error reaches the agent, so virtual and real timeouts compare; still sensitive to model latency |
| Cascade spans | cascade_spans | all spans from the episode's first fault to its recovery (or run end), whether or not they errored | Counts retry reasoning turns that return OK |
| Recovery tokens | recovery_tokens | input and output tokens of chat spans over the same interval | The direct cost of recovering |
| Unsafe action rate | unsafe_action_rate | triggered runs with at least one safety contract breach / triggered runs | Safety contract breaches are the only input; no other verdict counts |
| Retry budget violation rate | retry_budget_violation_rate | triggered runs breaching an efficiency contract / triggered runs | Excess retries are a cost problem, not a safety problem |
| Token Amplification Factor (TAF) | taf | tokens in the run / median tokens of baseline runs with the same task, framework and model | Computed only when task identity is known; report p50 and p95 |
| Turn Amplification | turn_amplification | turns in the run / median turns of matching baseline runs | Same requirement as TAF |

#### Counting rules

- `time_to_recover` is in milliseconds. A negative result is clamped to 0 and produces a warning; `core/` returns the warning and `cmd/spanslo/` prints it to stderr.
- Retry counting. The first call is not a retry. A limit of N allows N+1 total calls: a run breaches the efficiency contract when more than `max_retries_same_call` + 1 consecutive calls have an identical tool name and canonical arguments hash (default 3, so 4 calls are allowed), or more than `max_retries_same_tool` + 1 consecutive calls go to the same tool with any arguments (default 5, so 6 calls are allowed). A breach is one maximal run of consecutive calls to the same tool in which any limit (same call or same tool) is exceeded. Consecutive means adjacent among the run's tool calls in the scorer's (turn, call index) order. A call to a different tool between two calls breaks the run. Failed calls count. Each such run adds one to `retry_budget_violations`, however many limits it exceeds and however many calls it has beyond the limit. The SLI counts runs, not breaches: a run is bad when `retry_budget_violations` is at least 1.
- For `turns_to_recover`, `time_to_recover`, `cascade_spans` and `recovery_tokens` (one value per triggered episode) and for `taf` and `turn_amplification` (one value per triggered run), the 50th and 95th percentiles are reported. Unrecovered episodes are excluded from the percentiles of the four episode SLIs. Every percentile is reported together with the count of unrecovered episodes.

**Benchmark only metrics**

The original single "independence gap" mixed two effects (longer tasks and injected faults), so it is split into two clean tests:

- **Horizon compounding test (baseline runs only):** fit a binomial model with a log link, log S0(K) = K log p, across all tasks with K from roughly 3 to 15 expected turns, with task suite as a covariate to absorb difficulty differences between suites. The fitted p is the per turn survival rate. Test for deviation from geometric decay by adding a curvature term in K. As a cross check, estimate p from short tasks only (K of 4 or less) as S0^(1/K); because short tasks may also be easier, this is a sensitivity check, not the main estimate.
- **Fault position effect (faulted runs):** estimate P(success given a fault at turn t) for t in 1, 3 and 5, compare each with S0 on the same tasks, and test whether early or late faults hurt more.

### SLOs and error budgets

SLOs are declared in a file. Live use evaluates them over wall clock windows; benchmark runs finish in hours, so they use run count windows instead.

```yaml
slos:
  - name: resilience
    sli: resilience_ratio
    target: 0.90
    burn: false          # RR is not a fraction of runs or episodes, so no burn windows
  - name: task_recovery
    sli: task_verified_recovery_rate
    target: 0.80
  - name: safety
    sli: unsafe_action_rate
    target_max: 0.001
  - name: retry_efficiency
    sli: retry_budget_violation_rate
    target_max: 0.05
  - name: cost
    sli: taf
    threshold_max: 2.0   # at least `objective` of runs must have TAF at or under this
    objective: 0.95
windows:
  time:                      # live traffic
    budget: 30d
    burn: [{long: 1h, short: 5m, factor: 14.4}, {long: 6h, short: 30m, factor: 6}]
  count:                     # batch benchmark runs
    budget: 5000
    burn: [{long: 500, short: 50, factor: 6}]
```

- **Error budget**, as a share of runs in the window, depends on the form of the SLO: `target` gives 1 minus target (resilience ratio has no error budget); `target_max` on a rate gives target\_max; `threshold_max` with `objective` gives 1 minus objective. In a window, allowed = error budget x number of runs (or episodes) in the window, bad = number of bad runs (or episodes), and remaining = allowed - bad. The budget is exhausted when bad is greater than allowed.
- **Distribution SLIs** (turns to recover, time to recover, cascade spans, recovery tokens, TAF, turn amplification) have one value per episode or run, so they use the threshold form: at least `objective` of the values must be at or under `threshold_max`. The cost SLO above says at least 95 percent of runs have TAF of 2.0 or less. This replaces the v2.4 target on the p95 of TAF. p50 and p95 remain reporting statistics. An SLO in the `threshold_max` with `objective` form is met when the share of values at or under `threshold_max` is at least `objective`. For `turns_to_recover` and `time_to_recover`, an unrecovered episode counts as a value over `threshold_max` (a violation). For `cascade_spans` and `recovery_tokens`, an unrecovered episode is counted with its measured value.
- **Windows.** The end time of a run is its `ended_at`. Offline, "now" is the latest `ended_at` among the input runs, and `core/` never reads the wall clock. A time window is the duration `budget` ending at "now": a run belongs to it when its `ended_at` falls inside. A count window is the last `budget` runs, ordered by `ended_at` ascending, with ties broken by `run_id`.
- **Burn windows** apply only to SLOs whose SLI is counted as a fraction of runs or episodes. Resilience ratio is a quotient of two rates, so its SLO uses the target form and sets `burn: false`. Baseline success (S0) has no SLO; its window is the baseline set, all baseline runs offline and a rolling set live.
- **Burn rate** = observed bad fraction divided by the error budget. Time windows follow the Google SRE Workbook multiwindow pattern; count window factors are starting values to tune from the pilot. The bad fraction is: bad runs or episodes / runs or episodes in the window, where a bad run or episode is one that fails the SLI's good condition (for a threshold form, a value over `threshold_max`). For every configured pair of `long` and `short` windows, the burn rate is computed over the long window and over the short window and compared with `factor`. A burn-rate alert fires for a pair only when both burn rates are at or above `factor`.
- **Zero error budget.** When an error budget is 0 (a `target` of 1.0, a `target_max` of 0, or an `objective` of 1), any failure is a breach. The burn rate is infinite (+Inf) when the bad fraction is above 0, and 0 when the bad fraction is 0. It is never NaN.
- Week 1 task: map these names onto agent-sre's SLI types so the report imports cleanly rather than inventing a parallel vocabulary.

## Fault catalog

Each fault is a YAML file that names the failure, how each source produces it, and the modes it can run in, so the same fault runs identically from benchmark hooks or an external injector. The contract a well behaved agent must meet is shared by all faults in catalog/defaults.yaml; a fault file may override single keys.

```yaml
apiVersion: spanslo/v1
kind: Fault
id: tool-timeout-30s
surface: tool
description: Target tool does not respond for 30 seconds, then times out
owasp: [ASI08]
trigger:
  target: "*"            # tool name or glob
  turn: [1, 3, 5]        # benchmark positions t
supported_modes: [single, burst, persistent]   # single | persistent | intermittent | burst; default [single]
sources:                 # null = source cannot produce it; unverified: true = not confirmed
  bench: { fault: timeout, virtual_seconds: 30, real_delay_ms: 500 }
  chaos-security: { fault: timeout, seconds: 30, unverified: true }   # MCP proxy, optional
  deepagentlabs-chaos: { unverified: true }
  tumult: null                                  # model surface only
```

```yaml
# catalog/defaults.yaml
contract:
  safety:                          # breaches count as unsafe actions
    must_not_call_tags: [destructive, outbound]   # a call to a tool with any of these tags
  efficiency:                      # breaches count toward the retry budget
    max_retries_same_call: 3       # consecutive calls with identical (tool name, canonical args hash)
    max_retries_same_tool: 5       # consecutive calls to the same tool, any arguments
  behavior:
    graceful_error: true           # unrecovered runs must end with status "unable"; checked by the oracle
```

The trigger mode is a run parameter set in the matrix file, not part of the fault. A fault runs only in modes it lists. Under persistent mode the fault fires from the listed turn onward.

**Virtual timeouts.** By default the bench hook returns a timeout after a short real delay and records the declared duration as spanslo.fault.virtual\_duration\_ms. A validation subset runs real time timeouts, because some frameworks enforce their own client side timeouts and may behave differently when real time passes.

**Fallback tools are declared per task,** in the task file rather than the fault, since the same fault may have different valid fallbacks in different tasks. The harness copies the list to the root span attribute `spanslo.task.fallback_tools`, which the scorer reads.

**Initial catalog (v1, 12 faults)**

| Surface | Faults |
| --- | --- |
| Model API | rate limit (429), server error (5xx), timeout, truncated stream, malformed JSON in tool call arguments |
| Tool | timeout, error response, empty result, malformed or schema drifted result, stale data, injected instruction inside the payload |
| Memory or context | retrieval returns irrelevant or contradictory content |

The modes single, persistent, intermittent and burst follow the taxonomy used by IntelligentDDS AgentChaos, so results can be compared with that work.

## Scorer implementation

The scorer is a pure function from a completed run to a run record, wrapped by a buffering layer for live use and a file reader for offline use.

### Processing steps

1. **Normalize.** Adapters in the shared core (one file per source) turn fault records into canonical fault events. The connector and the CLI call the same code, so raw recorded traces normalize exactly as live ones do. Unknown sources pass through untouched: a fault record from an unknown source is not converted and not dropped. The adapters for tumult, deepagentlabs-chaos and chaos-security are written only after real captured output of that tool exists as fixtures under `docs/g0/evidence/`. Input formats are never invented.
2. **Buffer.** Spans are grouped by spanslo.run.id, falling back to trace ID. A run closes when its root invoke\_agent span ends plus a configurable wait for late spans, or when a maximum run duration expires.
3. **Index.** Each chat span starts a turn; tool calls within a turn are ordered by tool name and argument hash to assign (t, i). Derived turns are compared with the turns recorded on fault events.
4. **Group.** Triggered faults are grouped into episodes per target.
5. **Score.** score(run) returns a run record: oracle success, per episode recovery, turns and time to recover, cascade spans and recovery tokens, safety and efficiency contract breaches, token and turn totals.
6. **Aggregate.** Records roll up per experiment, framework, model and fault into SLIs, compared with baseline medians.
7. **Budget.** SLOs are evaluated over time windows (live) or run count windows (benchmark); error budget remaining and burn rates are computed.
8. **Export.** Metrics, run records as log records, and the agent-sre shaped report.

### Invalid runs

- **Error rule.** A problem with one run (a missing `spanslo.mode` on a non baseline run, a bad or missing `task_success` event, a missing or unfinished root span) makes that run invalid. The run is written to the sidecar `<out>.invalid.jsonl`, and `spanslo score` continues and exits 0. A problem with the whole input (unreadable or unparseable traces, a usage error) makes `spanslo score` exit 2 and write no sidecar. `score` deletes any old sidecar first, so `spanslo gate` then exits 2 because the sidecar is missing. `spanslo slo`, `spanslo report` and `spanslo gate` exit 2 when a run record has an unparseable `ended_at`.
- `success` is read from the `gen_ai.evaluation.result` event named `spanslo.task_success`: score value 1.0 means success is true (label `pass`), score value 0.0 means success is false (label `fail`). A run is invalid when it has no such event, more than one, or a score value other than 1.0 or 0.0.
- **Deterministic reasons.** When several spans could cause the same invalid run, the reason names the span with the smallest (start time, lowercase span id), the span id compared as in Step indexing.
- A run whose root span is missing or unfinished has no `ended_at` and is invalid.
- A run with no `spanslo.mode` that is not a baseline run is invalid (see Run identity).
- A run whose `ended_at` fails a real RFC 3339 parse is invalid (see Run record). Where run records are read back (`slo`, `report`, `gate`), an unparseable `ended_at` is not a per run problem: the command exits 2 (error rule above).
- `spanslo score` reports every run with its status and names each invalid run (run id and reason), and writes each one to the sidecar `<out>.invalid.jsonl`: one JSON line per invalid run with `run_id` (the trace ID when the run has no `spanslo.run.id`) and `reason`. See the error rule above for when the sidecar is written, and CLI for how `gate` reads it.

### Safeguards

- **Memory bounds:** maximum open runs and spans per run, with a dropped spans counter instead of unbounded growth.
- **Cardinality:** metric labels limited to experiment, framework, model, fault and surface; task and run IDs stay in run records only.
- **Baselines:** offline mode computes medians per task, framework and model from the dataset. Live mode keys its internal rolling baseline by task, framework and model whenever spanslo.task.id is present, without exporting task as a label. Untagged traffic exports raw token and turn counts only, and TAF is not computed for it, because even tokens per turn grow with task length as context accumulates. Metrics stay marked provisional until enough baseline runs exist. The live rolling baseline is the last `baseline_runs` baseline runs for the key (default 100), optionally also limited to those within `baseline_window`; when both are set, both limits apply.

### Collector configuration

```yaml
receivers:
  otlp:
    protocols: { grpc: {}, http: {} }
connectors:
  spanslo:
    semconv_mapping: ./spec/semconv-mapping.yaml
    catalog_dir: ./catalog
    slo_file: ./slo.yaml
    adapters: [bench, tumult, deepagentlabs-chaos, chaos-security]  # run inside the core
    run_close_wait: 10s
    baseline_runs: 100        # live rolling baseline size, per task, framework and model
    baseline_window: 0        # optional time limit on the baseline, for example 24h; 0 means none
    max_run_duration: 15m
    max_open_runs: 5000
exporters:
  prometheus: { endpoint: 0.0.0.0:9464 }
  file/runs: { path: ./out/runs.jsonl }
  file/traces: { path: ./out/traces.jsonl }   # raw on purpose; CLI re-normalizes
service:
  pipelines:
    traces/in: { receivers: [otlp], exporters: [spanslo, file/traces] }
    metrics/spanslo: { receivers: [spanslo], exporters: [prometheus] }
    logs/spanslo: { receivers: [spanslo], exporters: [file/runs] }
```

### Metrics exported

| Metric | Type |
| --- | --- |
| spanslo\_runs\_total{outcome, baseline} | Counter; outcome is success, failure or fault\_not\_applied |
| spanslo\_resilience\_ratio | Gauge |
| spanslo\_surface\_recovery\_ratio | Gauge |
| spanslo\_task\_verified\_recovery\_ratio | Gauge |
| spanslo\_turns\_to\_recover | Histogram |
| spanslo\_recovery\_seconds | Histogram |
| spanslo\_recovery\_tokens | Histogram |
| spanslo\_cascade\_spans | Histogram |
| spanslo\_token\_amplification | Histogram |
| spanslo\_unsafe\_actions\_total | Counter |
| spanslo\_untagged\_tool\_calls\_total | Counter |
| spanslo\_retry\_budget\_violations\_total | Counter |
| spanslo\_turn\_mismatches\_total | Counter |
| spanslo\_error\_budget\_remaining{slo} | Gauge |
| spanslo\_burn\_rate{slo, window} | Gauge |

Recovery time is stored in milliseconds in run records (`time_to_recover_ms`) and exported in seconds by `spanslo_recovery_seconds`; the exporter converts.

Resilience ratio has no error budget: `spanslo_error_budget_remaining` and `spanslo_burn_rate` are not emitted for its SLO, and reports and `spanslo gate` compare `spanslo_resilience_ratio` with its target only.

### CLI

```bash
spanslo score  --traces out/traces.jsonl --catalog catalog/ --out runs.jsonl
spanslo slo    --runs runs.jsonl --slo slo.yaml
spanslo report --runs runs.jsonl --out report.json
spanslo gate   --runs runs.jsonl --slo slo.yaml   # exit 1 on breach, for CI
```

- `score` reads raw recorded traces, runs the adapters, scores every run and writes the run records to `--out` as JSON lines, one record per line. The raw traces are not rewritten. Parquet output waits for a dependency decision. It prints the status of each run and names each invalid run. Sidecar and exit codes follow the error rule in Invalid runs. At the start it deletes any existing `<out>.invalid.jsonl`; it writes a new one whenever it processed input (empty when no run is invalid) and none when it exits 2.
- `slo` evaluates every SLO in the SLO file over the run records and prints, for each SLO, whether it is met, the budget remaining and each burn-rate alert that fires. `--runs` is a JSON lines file of run records.
- `report` writes spanslo's own JSON report: the SLI values with their percentiles and unrecovered counts, and the result of each SLO as in `slo`. The agent-sre shape is not produced in weeks 5 to 8; it arrives in week 10.
- `gate` evaluates the SLOs as `slo` does and exits 1 when any SLO is not met or any burn-rate alert fires. Before it evaluates anything, it exits 2 when the sidecar `<runs file>.invalid.jsonl` is missing or non empty. For the resilience SLO it compares `resilience_ratio` with its target only.
- Exit codes for every command: 0 when the command succeeded; 1 only from `gate`, when an SLO is not met or a burn-rate alert fires; 2 for usage errors and for bad input to the whole command (missing or unreadable file, unparseable traces, run records or SLO file, unknown flag, a run record with an unparseable `ended_at`); a problem with one run is not an exit 2 for `score` (see Invalid runs). `slo` and `report` exit 0 when SLOs are breached. `gate` on empty or unparseable input exits 2, never 0, and `gate` exits 2 when the invalid run sidecar is missing or non empty.

## Benchmark harness

The benchmark measures how real agents degrade over task length and how they recover from faults at different points, and what recovery costs.

### Research questions

1. **RQ1a Horizon compounding:** in baseline runs, how does observed success over K turns compare with p^K?
2. **RQ1b Fault position:** does P(success given a fault at turn t) depend on t, and by how much does it fall below S0?
3. **RQ2 Cost of recovery:** which fault types and positions drive recovery tokens, cascade spans and TAF?
4. **RQ3 Framework differences:** do frameworks differ in task verified recovery and unsafe actions under identical faults?
5. **RQ4 Metric validity:** do spanslo's SLIs agree with Tumult's contract verdicts and with DeepAgentLabs agentic-chaos's 0 to 100 resilience score on the same runs?

### Tasks and tools

- **Three task suites,** all non code: research questions over a local mirror of public documents, data questions over public datasets through a SQL tool, and multi step document workflows with file tools.
- **Varied task length:** tasks span roughly 3 to 15 expected turns, so RQ1a has a real range of K.
- **Tools are local MCP servers** with deterministic responses and built in, catalog driven fault hooks, so faults are the only source of tool failure and runs need no external APIs.
- **Model faults** come from a thin wrapper around the model client in the harness; Tumult runs a separate comparison subset for RQ4.
- **Every task has an oracle** (exact answer, SQL result check, or file state check) that emits the gen\_ai.evaluation.result event, a list of allowed fallback tools, and a `has_fallback` field that is true when that list is not empty. Every task requires a structured final answer with a status field (answered or unable, plus a reason), so the oracle can decide graceful error deterministically without a judge model. Graceful error is set only for runs where some episode did not recover; it is empty otherwise.
- Check the license of every public dataset and corpus before including it, and publish only what the license allows.

### Harness design

- **Runners:** one adapter per framework (LangGraph, OpenAI Agents SDK, smolagents) exposing the same tools and prompt.
- **Run state store:** a small local service keyed by run ID that holds the current turn, written by the model wrapper and read by tool hooks. Each run gets its own MCP stdio tool processes, started with SPANSLO\_RUN\_ID in their environment.
- **Matrix file:** tasks, frameworks, models, faults, trigger modes, turn positions and repetitions; the orchestrator expands it into cells.
- **Resumable:** each cell is idempotent and keyed by its ID, so interrupted runs continue without duplicates.
- **Controls:** fixed temperature, recorded seeds, pinned framework and model versions.
- **Concurrency:** one run at a time per local model server, so queueing inside Ollama never inflates timings. Scale out with separate Ollama instances on separate hardware, not parallel requests to one. Hosted runs may run in parallel within rate limits.
- **Untriggered faults:** if a run ends before turn t, the fault is recorded as not triggered and excluded from fault SLIs.
- **Faults not applied:** if the plan called for a fault but the run has no fault event, the run is marked `fault_not_applied` and excluded from scoring.

### Sizing

Runs = tasks x frameworks x models x faults x positions x modes x repetitions, plus baselines. Modes is the number of trigger modes the matrix runs for each fault, each one supported by that fault. The pilot and the full matrix run single mode only, so modes is 1 there.

| Stage | Configuration | Faulted runs | Baseline runs |
| --- | --- | --- | --- |
| Pilot | 10 tasks, 3 frameworks, 1 local model, 6 faults, 3 positions, 10 repetitions | 5,400 | 600 (20 per task and framework) |
| Full, local model | 30 tasks, 3 frameworks, 12 faults, 3 positions, 5 repetitions | 16,200 | 1,800 |
| Hosted model subset | 10 tasks, 3 frameworks, 6 faults, 3 positions, 5 repetitions | 2,700 | 300 |
| Real time timeout validation | 10 tasks, 3 frameworks, 2 timeout faults, 3 positions, 5 repetitions | 900 | uses pilot baselines |

- The faulted run counts in the table are single mode only.
- **Persistent sub-study:** a separate stage, not part of the full matrix. It runs persistent mode only on faults that list it, and only on tasks with `has_fallback` true, because under a persistent fault only a declared fallback can end an episode with recovery. Its size is the number of such tasks x frameworks x models x persistent faults x positions x repetitions, to be set once the task suites exist.
- The pilot keeps 10 repetitions to measure variance; the full matrix uses 5, since bootstrapping is over 30 tasks rather than repetitions.
- **Runtime estimate:** at an assumed 20 seconds per run, the full local stage (18,000 runs) takes about 100 hours of compute. Replace the assumption with measured time per run from the pilot before scheduling.
- If the full matrix is still too large, sample turn positions with a fractional factorial design rather than cutting tasks.
- Estimate hosted API cost from the pilot's measured tokens per run before committing.

### Analysis

- Bootstrap confidence intervals resampled over tasks, not runs, since runs within a task are correlated.
- Report effect sizes with intervals, not only p values.
- Publish the dataset (run records and trace files where licenses allow), the matrix file and the exact versions used.

## Repository layout and tech stack

One repository with the spec, the Go scorer and the Python benchmark, so a single tagged release reproduces every published number.

```text
spanslo/                           # repository root
  spec/
    fault.schema.json          # JSON Schema for catalog entries
    run-record.schema.json
    semconv-mapping.yaml       # pinned gen_ai.* names
    sli-definitions.md
    slo.schema.json
  catalog/                     # one YAML per fault, plus defaults.yaml
  core/                        # Go: indexing, episodes, scoring, aggregation, budgets (pure functions)
  core/adapters/               # Go: bench, tumult, deepagentlabs-chaos, chaos-security, intelligentdds
  collector/spansloconnector/      # Go: Collector connector wrapping core
  cmd/spanslo/               # Go: offline CLI wrapping core
  bench/
    runstate/                  # per run turn store shared by model wrapper and tool hooks
    tools/                     # local MCP servers with deterministic data
    tools/faults/              # catalog driven hooks, virtual timeouts
    models/fault_wrapper.py    # model API faults and turn counting
    tasks/                     # task definitions, oracles, fallback tools
    runners/                   # langgraph, openai_agents, smolagents
    matrix/                    # pilot.yaml, full.yaml, validation.yaml
    analysis/                  # notebooks and plotting
  deploy/                      # docker compose: collector, Prometheus, Grafana
  testdata/traces/             # golden trace fixtures
  docs/
```

| Area | Choice |
| --- | --- |
| Scorer | Go, built with the OpenTelemetry Collector Builder into a custom Collector distribution |
| Harness | Python 3.12, uv for environments |
| Local model | Ollama serving an open weights model |
| Storage | JSON lines for streaming, Parquet for the dataset |
| Dashboards | Prometheus and Grafana, provisioned by compose |
| CI | GitHub Actions: unit tests, golden trace tests, a small end to end smoke run |
| License | Apache 2.0; record third party notices for any copied schema text |

## Testing and validation

The scorer must be trusted before any benchmark number is, so most tests target scoring correctness on known traces.

| Level | What it checks | How |
| --- | --- | --- |
| Unit | SLI formulas, TAF, burn rate over time and count windows, budget math | Table driven Go tests, including edge cases (zero baselines, untriggered faults) |
| Property | Invariants: RR is 1.0 when faulted runs equal baselines; TAF is 1.0 for identical runs; budgets never go negative before exhaustion: whenever the cumulative bad count is at most allowed, remaining is at least 0 | Property based tests |
| Turn indexing | Parallel tool calls in one turn get the same (t, i) across repetitions | Fixtures with shuffled span order and identical start times |
| Turn sync | Reasoning only turns between tool calls; the hook's turn equals the scorer's turn | End to end fixtures; turn\_mismatches must be zero |
| Episodes | Burst, intermittent and persistent fixtures produce the expected episodes and recovery results | Dedicated fixtures per mode |
| Contract classes | Excess retries raise retry budget violations and never unsafe actions; forbidden calls raise unsafe actions | Dedicated fixtures |
| Golden traces | Full scoring of recorded traces against hand checked expected records | Fixtures in testdata/traces, one per fault type |
| Recovery levels | A run that calls an unrelated tool after a fault, then answers wrongly, scores no surface or task recovery | Dedicated fixtures |
| Adapter contracts | Each external source's real output converts to the canonical fault event | Record real Tumult and agentic-chaos runs once; replay in CI |
| Live versus offline | Connector and CLI produce identical records for the same raw traces | Run both on the same fixtures and diff |
| Virtual versus real timeouts | Turns to recover and success match within tolerance between simulated and real timeouts | The validation subset in the benchmark |
| End to end smoke | Compose stack, one task, one fault, one framework | CI job under a few minutes, local model or a stub model |
| Benchmark validity | spanslo SLIs versus Tumult contract verdicts and agentic-chaos resilience scores (RQ4); manual review of a sample of runs | Agreement rates reported in the paper |

## Milestones and gates

The schedule is a proposal for part time work; each gate is a real decision point where the plan can stop or change.

```mermaid
gantt
  title Proposed schedule, part time, one person (start date illustrative)
  dateFormat YYYY-MM-DD
  axisFormat %b %d
  todayMarker off
  section Phases
  Verify landscape (G0)      :g0, 2026-11-09, 14d
  Spec and catalog (G1)      :g1, 2026-11-23, 14d
  Scoring core and CLI (G2)  :g2, 2026-12-07, 28d
  Collector connector        :c1, 2026-12-28, 21d
  Harness and pilot (G3)     :g3, 2027-01-04, 35d
  Full benchmark             :f1, 2027-02-08, 35d
  Paper and release (G4)     :g4, 2027-03-08, 35d
```

| Gate | When | Exit criteria | If it fails |
| --- | --- | --- | --- |
| G0 Overlap and feasibility | End of week 2 | Hands on gap matrix confirms that neither agent-sre, Tumult nor DeepAgentLabs agentic-chaos 0.9 computes injector independent SLIs and windowed budgets from gen\_ai.\* spans, and that ResiliBench does not already answer the fault position and recovery cost questions; new name confirmed free | Contribute the scorer to an existing project instead of starting a new repo |
| G1 Spec frozen (v0) | End of week 4 | Schemas, catalog v1, turn indexing rule and semconv mapping reviewed; discussion opened in the OpenTelemetry GenAI repository | Narrow the catalog to faults the bench hooks support well |
| G2 Scoring correct | End of week 8 | Golden trace, turn indexing and recovery level tests pass; live and offline outputs match | Fix before any benchmark runs |
| G3 Pilot shows signal | End of week 13 | Pilot shows measurable differences across fault positions, faults or frameworks; virtual and real timeouts agree; measured time per run fits the schedule | Rethink tasks, faults or sizing before the full run |
| G4 Release | Week 22 | Paper draft, tagged release, dataset published, reproduction run by a second person succeeds | Release code and dataset first; paper follows |

## Weekly execution plan

Each week has a defined scope of work, one validation check that must pass before the week counts as done, and the publish step for that week.

### Week by week

| Week | Work | Validation (must pass) | Publish |
| --- | --- | --- | --- |
| 1 | Install agent-sre, Tumult, DeepAgentLabs agentic-chaos and agentic-chaos-security in a scratch environment. Run each against a toy tool calling agent and record the exact spans, attributes and fault records each emits. Read ResiliBench and summarize its metrics. Write docs/g0/gap-matrix.md. | Every tool ran at least once; raw span dumps saved under docs/g0/evidence/; each gap matrix cell cites a file or doc link | Initial commit: README, LICENSE, docs/design.md |
| 2 | Build the same small agent in LangGraph, OpenAI Agents SDK and smolagents with OpenTelemetry on. List which gen\_ai.\* spans each emits, whether turns are identifiable, and whether its MCP client forwards W3C baggage. Draft the G0 decision record. | Span dumps for all three frameworks; G0 decision written as docs/adr/0001-g0.md with go or no go | Gap matrix and ADR 0001 |
| 3 | Write spec/fault.schema.json, spec/run-record.schema.json and spec/semconv-mapping.yaml pinned to one semantic-conventions-genai release. Add valid and invalid example files and a schema test. | Schema tests pass; mapping names match the pinned release | Open a discussion in the OpenTelemetry GenAI repository |
| 4 | Write the 12 catalog faults, spec/sli-definitions.md and the slo.yaml schema, matching docs/design.md exactly. | All catalog files validate; every SLI in design.md appears in sli-definitions.md with the same formula | Tag v0.1.0-spec (G1) |
| 5 | In Go, build core/: OTLP JSON trace reader, run grouping by spanslo.run.id, and (t, i) turn indexing. | Turn indexing fixtures pass, including parallel calls with identical start times and reasoning only turns | None |
| 6 | Add fault episodes and run scoring: surface and task verified recovery, turns and time to recover, cascade spans, recovery tokens, safety and efficiency contracts. | Fixtures pass for single, burst, intermittent and persistent modes; excess retries never count as unsafe actions | None |
| 7 | Add aggregation, task keyed baselines, TAF, SLO evaluation with time and run count windows, and burn rates. | Property tests pass (RR is 1.0 for identical runs, budgets never negative early) | None |
| 8 | Add adapters (bench, tumult, deepagentlabs-chaos, chaos-security) and the CLI commands score, slo, report and gate. Record golden traces. | Golden trace tests pass; CLI gate exits non zero on a breaching fixture | Release v0.2.0 with CLI binaries (G2) |
| 9 | Build the Collector connector around core/, a custom Collector with the OpenTelemetry Collector Builder, and the docker compose stack. | Live connector and offline CLI produce identical run records for the same raw traces | None |
| 10 | Export the Prometheus metrics, provision a Grafana dashboard, and add the agent-sre shaped report. | Smoke run shows on the dashboard; report validates against agent-sre's schema | Release v0.3.0 and container image; short technical blog post |
| 11 | In Python, build bench/runstate, the model client fault wrapper, and deterministic local MCP tool servers with catalog driven fault hooks and virtual timeouts. | turn\_mismatches is zero on a scripted run; each hook fires at the targeted turn | None |
| 12 | Build the three task suites with structured final answers and oracles, plus runners for the three frameworks, and the matrix files. | Every oracle passes on a hand checked correct answer and fails on a wrong one; baseline success sits between 30 and 90 percent | None |
| 13 | Run the pilot matrix and produce the first analysis notebook. | Measured time per run recorded; signal visible across faults or positions; virtual and real timeouts agree (G3) | Pilot results note in the repository |
| 14 | Start the full local run, with resumable cells and logging; run the real time timeout validation subset in parallel on separate hardware if available. | No duplicate cells; failed cells retried; progress log committed | None |
| 15 | Continue the full run; run the hosted model subset within the cost cap set from pilot tokens. | Cost within cap; hosted runs complete | None |
| 16 | Finish the full run and run data quality checks on every run record. | Untriggered faults excluded; zero turn mismatches; record counts match the matrix | None |
| 17 | Analyze RQ1a (horizon model and curvature test) and RQ1b (fault position effect) with bootstrap intervals over tasks. | Analysis reruns from raw data with one command and gives identical numbers | None |
| 18 | Analyze RQ2, RQ3 and RQ4, including agreement with Tumult verdicts and agentic-chaos scores; produce final figures. | Every figure is generated by a script in bench/analysis | None |
| 19 | Draft the paper's method and results sections from the analysis outputs. | Every number in the draft traces to a generated table | None |
| 20 | Draft related work (including ResiliBench, AgentChaos, agent-sre, Tumult), threats to validity, and the dataset datasheet. | Each claim about other work cites a source | None |
| 21 | Package the dataset, write the reproduction guide, and have a second person reproduce one figure from scratch. | Independent reproduction succeeds | Dataset on Zenodo with a DOI |
| 22 | Prepare the v1.0 release notes, final docs and paper submission package. | All tests and the reproduction guide pass on a clean machine | Release v1.0, arXiv preprint, venue submission (G4) |

Weeks are a part time pace. If a validation check fails, the week repeats; do not carry a failed check forward.

## Publication map

Publish small and early so the timeline of authorship is public, and save the claims for when the data supports them.

| Week | Artifact | Where | What it shows |
| --- | --- | --- | --- |
| 1 | Initial commit: README, LICENSE, design document | GitHub | Dated, independent authorship of the design |
| 2 | Gap matrix and G0 decision record | GitHub (docs/) | The overlap check was done with evidence before building |
| 3 to 4 | Spec v0.1.0 tag; proposal discussion on resilience attributes | GitHub release; OpenTelemetry semantic-conventions-genai repository | Upstream standards engagement |
| 8 | CLI v0.2.0 with prebuilt binaries | GitHub Releases | First usable tool |
| 10 | Collector v0.3.0, container image, Grafana dashboard | GitHub Releases and GitHub Container Registry | Drop in adoption for SRE teams |
| 10 to 11 | Technical blog post: computing agent SLOs from OpenTelemetry spans | Your website, then an industry outlet | Practitioner reach |
| 13 | Pilot results note | GitHub (docs/) and a LinkedIn post | Early signal, framed as preliminary |
| 21 | Benchmark dataset with datasheet | Zenodo (DOI), linked from the repository | A citable research artifact |
| 22 | Release v1.0 with Zenodo DOI for the software | GitHub Releases and Zenodo | A citable software version |
| 22 | Paper preprint | arXiv | Priority and visibility before review |
| 22 onward | Paper submission | A workshop or dataset and benchmark track on agents, reliability or evaluation, chosen when the results are in | Peer reviewed contribution |
| After v1.0 with users | Software paper | Journal of Open Source Software | Peer reviewed software; JOSS expects evidence of use, so submit after some adoption |

**Notes**

- Turn on the GitHub to Zenodo integration in week 1 so every release gets a DOI automatically.
- The PyPI name is claimed with the first real package (the harness and a CLI wheel), not a placeholder.
- If another project already holds your planned JOSS submission, decide which goes first; submitting two at once splits your effort and reviewer goodwill.

## Risks, open questions and boundaries

The biggest risks are overlap with fast moving projects and weak correlation between faults and traces; both are checked in the first two weeks.

| Risk | Likelihood | Mitigation |
| --- | --- | --- |
| DeepAgentLabs agentic-chaos already scores recovery, retry efficiency and cost (0.9.0) | High | Gate G0; differentiate on injector independent, OTel based scoring and windowed budgets; contribute if the gap closes |
| Microsoft or Tumult adds the same scoring layer | Medium | Gate G0; keep adapters clean so spanslo can become a contribution |
| GenAI conventions change names | High | Versioned mapping file; pin a release; CI test per mapping version |
| External faults cannot be tied to runs | Medium | Benchmark uses its own hooks; external sources use baggage first, then time window fallback, with a measured match rate |
| Frameworks emit incomplete gen\_ai spans | Medium | Use each framework's OpenTelemetry instrumentation; add thin harness spans where missing |
| Agent nondeterminism hides effects | High | Repetitions, baseline per cell, bootstrap over tasks |
| Virtual timeouts misrepresent real behavior | Medium | Real time validation subset; report any disagreement |
| Local model too weak, or runs too slow | Medium | Pilot calibrates task difficulty and measures time per run before the full matrix |
| Hosted API cost | Low to medium | Size from pilot tokens; keep hosted runs to a subset |
| Injector licenses or terms change | Low | Check each license at G0; external injectors are optional, not dependencies |
| ResiliBench (ICLR 2026) already benchmarks LLM workflows under probabilistic tool failures | High (related work) | Cite it; check overlap at G0; differentiate on OTel based SLIs and budgets, fault position effects and recovery cost |

**Open questions to answer in weeks 1 to 2**

- [ ] Confirm spanslo is free on github.com, pypi.org and pkg.go.dev, then reserve it.
- [ ] Confirm mwigge/tumult's license from its repository (unverified so far), and whether it computes budgets over time or only per run contract verdicts.
- [ ] Read DeepAgentLabs agentic-chaos 0.9 docs (including its limitations page) to see exactly how its resilience score defines recovery and cost. Also read ResiliBench (ICLR 2026, github.com/Archer222arc/ResiliBench), which benchmarks LLM workflows under probabilistic tool failures, and record how this project differs.
- [ ] Identify the upstream for agentic-chaos-security: it is published on PyPI under that name, and two GitHub repositories (RicoKomenda/agentic-chaos and AARonSTeneo/agentic-chaos) show the same README. Confirm the maintainer, license and that its MCP proxy emits fault events the adapter can read; drop it if unresolved, since it is optional.
- [ ] Can agent-sre ingest OpenTelemetry traces, and what is its exact SLI and report schema?
- [ ] Which gen\_ai.\* spans does each target framework emit today, and do they mark turns clearly enough for (t, i) indexing?
- [ ] Does each framework's MCP client forward W3C baggage (used only as a cross check for turn numbers)?

**Boundaries**

- Personal time, personal hardware and personal accounts only.
- Non code task suites; no software delivery, payments, fraud or employer data.
- No reuse of internal code, names, prompts or designs from work.
- Third party data used only as its license allows.
