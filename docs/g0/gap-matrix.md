# G0 gap matrix

Week 1 hands-on overlap check, run 2026-11-06 on one machine (macOS arm64). Evidence is raw JSON
and logs under `~/scratch/g0/out/` (kept outside the repo). Paths below are relative
to that directory. The toy agent is `~/scratch/g0/agent/agent.py`: plain OpenAI client against
local Ollama (`qwen3.8:27b`, temperature 0, seed 0), two deterministic tools
(`get_unit_price`, `convert_usd_to_eur`), hand-written `invoke_agent`/`chat`/`execute_tool`
spans. All runs used `env -i` with no API keys exported. No-fault baseline: success, 6 spans
(`baseline-agent.json`).

## Matrix

| | agent-sre | Tumult | DeepAgentLabs agentic-chaos | agentic-chaos-security |
| --- | --- | --- | --- | --- |
| **Version** | 3.7.0 (PyPI; repo tag v4.1.0 is the toolkit) | v2.22.0 (Rust binary, sha256 verified) | 1.0.0 (2026-11-05; design says 0.9) | 1.0.0 (wheel identical to git tag v1.0.0) |
| **License** | MIT | Apache-2.0 (`LICENSE`; GitHub API) | MIT | Apache-2.0 |
| **Emits `gen_ai.*` spans?** | **No.** Re-exports whatever it is fed as `agent.sre.custom.gen_ai.*`; own namespace is `agent.*` / `agent.sre.*`. Zero `gen_ai` strings in 3.7.0 source. `sre/sre-otel-spans.json` | **Not observed.** Code defines `gen_ai.*` constants (`tumult-otel/src/agentic.rs:47-65`) but the spans captured over OTLP/gRPC carry only `resilience.agent.*`, `tumult.*`, `http.*`. Docs claim `gen_ai.operation.name=invoke_agent`; not seen. `tumult/otlp-spans.jsonl` | **No.** No OTel code at all in source | **No.** Emits `chaos.*` spans (`chaos.experiment`, `chaos.run`) with span events. Its own roadmap lists GenAI-trace probes as unchecked. `cs/chaos-otel-spans.json` |
| **Reads `gen_ai.*` spans as input?** | No ingest path; I had to hand-translate agent spans into its `TraceCapture` (`run_sre.py`) | No (proxy sits on HTTP, injects `traceparent` upstream) | No | No |
| **Fault record format** | `FaultInjectionEvent` {fault_type, target, timestamp, applied, details}; recorded by the caller (`exp.inject_fault`), it does not hook the call. `sre/sre-fault-injection-events.json` | Journal (`.toon` / JSONL: scenario, faults[{fault_type, applied, latency_ms}], contracts) plus span events `fault`/`contract` with `resilience.agent.fault.type`, `.applied`. No turn or run id. `tumult/otlp-spans.jsonl`, `tumult/proxy-journal-*.jsonl` | `ChaosEvent` {id, fault_type, step_id, step_name, timestamp, outcome, message, detail}. `dal/chaos-report-*.json` | Trace event {kind:"fault", name, data:{point, target, params}}. No timestamp (OTel export stamps export time, per `integrations/otel.py`), no turn, no run id. Same shape from the MCP proxy. `cs/native-traces-*.json`, `cs-mcp/mcp-trace.json` |
| **Recovery metric** | Surface-level at best. `calculate_resilience` = experiment success rate / baseline rate; accepts `recovery_time_ms` and `cost_increase_percent` but ignores both (`chaos/engine.py`). Success flag supplied by caller | Contract-based verdicts on fixtures; trajectory pack has a `recovery` subscore. Against my agent via proxy: only per-request contract strings (`valid_json=fail`, `retry_budget=pass`), no task check. `tumult/proxy-journal-*.jsonl` | **Surface.** "A later call to the same step succeeded" or "nothing escaped and not retried" (`docs/limitations.md`). Demonstrated: model 429 on turn 1, agent produced no answer, score **100** (`fallback: 1`). `dal/chaos-report-model_ratelimit.json` vs `dal/agent-run-model_ratelimit.json` | Task-verifiable if the user writes a probe: `output_contains("27.0")` caught a failed task (verdict WEAKNESS). Probes are per-experiment pass rates with CI; no episode or recovery-point concept. `cs/result-*.json` |
| **Turns or time based** | Time: wall-clock windows (SLO 30d default, burn alert window 24h) | Step/request based (trajectory "fault @ step 0"; proxy per request); wall-clock `latency_ms` | Wall-clock latency ratio vs baseline; call-log order for recovery | Run-count based (runs, pass rate); `completes_within(seconds)` probe |
| **SLO / error budget** | **Yes.** `SLO`, `ErrorBudget`, `TaskSuccessRate` SLI, burn rate, `agent.sre.error_budget.remaining`, `agent.sre.burn_rate` metrics. Single-window burn (`alerts()` fixed 24h); no run-count windows, no multiwindow pairs. `sre/sre-slo.json`, `sre/sre-otel-metrics.json` | No budgets or burn rates. "Guards" are probes with debounce that can query an external SLO | No budgets. Score history/trend file only | No budgets |
| **Gap spanslo fills** | Budgets exist but are fed by caller-recorded booleans, not derived from spans; no `gen_ai.*` input; success-ratio "resilience" ignores recovery time and cost; no fault-position or episode model | `gen_ai.*` is documented but unobserved; no independent SLIs from spans, no windowed budgets; fixtures dominate its local paths | Recovery is surface-level and task-agnostic (score 100 on a failed task); no OTel; no spans to score from | Closest on task checks and OTel, but `chaos.*` attributes, no turn index, no timestamps on native events, no recovery speed or cost SLIs, no budgets |

## ResiliBench (ICLR 2026)

Sources: full paper PDF pages 1-5 (read directly), poster, ICLR abstract page; repo
`github.com/Archer222arc/ResiliBench` (supplementary README only; the repo has no license file,
and the task/tool JSON was not in my shallow clone). Paper appendices (B.3 simulator, B.4
scoring) were **not** read.

- **Task:** 5,040 generated tasks, 30 simulated tools (6 categories x 5), 5 task types, 4 prompt
  variants each (baseline, CoT, MDP-optimal workflow, flawed workflow with 7 perturbation types).
- **Faults:** probabilistic simulator, base success rate 0.8, five error types (INVALID_INPUT,
  OPERATION_FAILED, TIMEOUT, CALCULATION_ERROR, OVERFLOW), limit of 10 dialogue turns. Fault
  *position* is not controlled; failures are drawn per call.
- **Metrics:** three-level outcome `full_success` / `partial_success` / `failure`. Full success
  needs all four of: required-tool coverage 100%, correct sequence, output generation, explicit
  completion signal. Partial needs at least two conditions. Headline numbers are average success
  by prompt type (baseline 51.4%, CoT 50.8%, MDP-optimal 62.1%; flawed 54.3% per abstract) and
  the optimal-vs-flawed drop in percentage points (GPT-4o-mini -5.5, Gemini-2.5-Flash -40.1).
- **Not present (as far as read):** recovery speed, recovery cost (tokens/turns/time), fault
  position effect, OpenTelemetry/`gen_ai.*` spans, SLOs, error budgets, framework comparison.
- **Models:** GPT-4o-mini, O3, Gemini-2.5-Flash, GPT-5-mini, Llama-3.3-70B, Qwen2.5-32B,
  DeepSeek-V3.
- **Gate G0 reading:** the position and recovery-cost questions (RQ1b, RQ2) are not answered by
  what I read; confirm against appendices B.3/B.4 before closing G0.

## Findings that bear on the design

1. **Nobody derives SLIs from `gen_ai.*` spans.** The one budget implementation (agent-sre) is
   fed hand-recorded booleans; the one tool defining `gen_ai.*` (Tumult) did not emit it on the
   paths I could run.
2. **Surface versus task-verified recovery is a real gap, with a concrete demo:** DAL scored
   100 on a run whose task failed. Caveat: my agent catches the model error and stops, which is
   what DAL's "nothing escaped" rule rewards; a different agent could score differently.
3. **No source gives a canonical turn index or run id on its fault record.** All four need
   adapter-side correlation, as the design assumes. cs's native events have no timestamps.
4. **Tumult's local paths run on its own fixtures;** only `agentic proxy` touched my agent, and
   it faults the first model call only (one journal line per request).

## Design/doc discrepancies to resolve (not fixed here)

- Design says agentic-chaos 0.9; current is 1.0.0 (0.10, 0.11 and 1.0 shipped 2026-11-03 to 11-05).
- PyPI `tumult` is an unrelated zsh utility; the real project is `mwigge/tumult` (GitHub, Rust,
  Apache-2.0, not on PyPI).
- agentic-chaos-security upstream resolved to `RicoKomenda/agentic-chaos` (Apache-2.0, v1.0.0
  tag, matches the PyPI wheel). `AARonSTeneo/agentic-chaos` is a GitHub fork. Registers a pytest
  plugin entry point on install. Its MCP proxy does emit readable fault events (`cs-mcp/`).
- Week-1 validation text says evidence goes in `docs/g0/evidence/`; this run kept it in
  `~/scratch/g0/out/` outside the repo, so those cells cite out-of-repo paths.

## Not run, and why

- `tumult agentic run-live`: shells out to `claude -p` and needs Anthropic credentials; none
  were exported.
- Tumult's `tumultd`, `tumult-net-proxyd`, Kubernetes/SSH/DB plugins: out of scope for agent faults.
- agentic-chaos `hub pull` (remote recipes), `--plugins`, agent-sre `delivery`/`k8s`/vendor
  integrations (Datadog, PagerDuty, etc.): need network services or credentials.
- agentic-chaos-security judge probes: need an API key.
- Tumult OTLP capture used a local throwaway gRPC receiver (separate venv); its logs export
  returned `Unimplemented` because the receiver only handles traces.
- Single runs per scenario at temperature 0; no variance estimate. Nothing here is a benchmark
  number.
