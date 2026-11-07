# Framework matrix

Week 2 hands-on check of what LangGraph, the OpenAI Agents SDK and smolagents emit as
OpenTelemetry spans, whether turn and step indexes can be rebuilt from those spans alone, and
whether MCP calls carry W3C trace context. Run 2026-11-06 on one macOS arm64 machine. Raw spans,
MCP wire logs and Ollama request logs are under `~/scratch/g1/out/runs/<framework>-<transport>-<instrumentor>/`
(outside this repo). Analysis scripts are in `~/scratch/g1/common/`; the combined results are
`~/scratch/g1/out/analysis.json`.

## Setup

- **Task:** the week 1 toy task (3 widgets, price in USD, convert to EUR), two MCP tools
  (`get_unit_price`, `convert_usd_to_eur`) served by one `mcp_tools.py`, over stdio and over
  Streamable HTTP. Model: Ollama 0.40.15, `qwen3.8:27b`, temperature 0, via an OpenAI-compatible
  endpoint behind a logging proxy. 34 runs started and all succeeded (answer 27.0), each with 3
  model calls and 2 tool calls. Two further runs never started because the instrumentor failed
  to load (Table 2).
- **Environment:** `env -i`, no API keys, Python 3.12.13, one venv per framework.
- **Spans:** exported to a local OTLP-JSON file (one `ExportTraceServiceRequest` per line). The
  harness adds a root span `g1.harness.run` and sets baggage `spanslo.run.id` and `spanslo.turn`
  so propagation can be observed.
- **Ground truth:** the logging proxy counts real model requests with timestamps; a stdio shim and
  an ASGI wrapper log what actually arrives at the MCP server, including `_meta` and HTTP headers.
- **One run per cell.** Counts were identical across cells, but there is no repetition and no
  variance estimate.

### Pinned versions

| Package | LangGraph venv | Agents SDK venv | smolagents venv |
| --- | --- | --- | --- |
| framework | langgraph 1.2.20, langgraph-prebuilt 1.1.1, langchain-core 1.7.6, langchain-openai 1.7.4, langchain-mcp-adapters 0.3.2 | openai-agents 0.24.0 | smolagents 1.26.0, mcpadapt 0.1.20 |
| mcp | 1.30.0 | 2.3.0 | 1.30.0 (pinned below 2, see notes) |
| openai | 3.46.0 | 3.46.0 | 3.46.0 |
| OTel API / SDK | 1.45.2 / 1.45.2 | same | same |
| OTel instrumentation / semconv | 0.66b2 / 0.66b2 | same | same |
| opentelemetry-util-genai | 1.1b0 (pinned, see notes) | same | same |
| instrumentors | as named in the tables | as named | as named |

Full `pip freeze` output for each venv is in `~/scratch/g1/out/env/`.

## Rule used to rebuild turn t and step i from spans alone

1. A **model span** is any span with `gen_ai.operation.name` in `chat`, `text_completion`,
   `generate_content`; or `openinference.span.kind = LLM`; or `llm.request.type` in `chat`,
   `completion`.
2. A **tool span** is any span with `gen_ai.operation.name = execute_tool`, or
   `openinference.span.kind = TOOL`, or `traceloop.span.kind = tool`. smolagents' built-in
   `final_answer` tool is excluded because it is not an MCP call.
3. **t** is the 1-based rank of model spans by start time within the trace.
4. A tool span belongs to the turn **t of the latest model span that ended before the tool span
   started**.
5. **i** is the 0-based rank of a tool span inside its turn, ordering by tool name and then by the
   SHA-256 of the canonical-JSON arguments (the design's rule). This needs a tool name attribute
   and an arguments attribute on the tool span.

A cell is marked **t = yes** when the number of model spans equals the number of real model
requests seen by the proxy. It is marked **i = yes** when tool spans equal real MCP tool calls,
name and argument attributes exist, and the derived turn of every tool span equals the real turn
(the number of model requests started before that MCP call arrived).

Limits of this check: each turn made at most one tool call, so the (name, argument hash)
tie-break was never exercised, and parallel tool calls were not tested.

## Table 1: spans by framework and instrumentor (stdio)

"Other namespaces" counts distinct attribute keys per namespace. `contrib` is the OpenTelemetry
contrib GenAI instrumentation for the OpenAI client (`opentelemetry-instrumentation-openai-v2`);
`llmetry` is OpenLLMetry (Traceloop); `oi` is OpenInference. The `-latest` variant sets
`OTEL_SEMCONV_STABILITY_OPT_IN=gen_ai_latest_experimental`; it produced the same attribute set as
the default in these runs.

| Framework | Instrumentor (version) | Span names (x = count) | `gen_ai.*` attrs | Other namespaces | t from spans | i from spans |
|---|---|---|---|---|---|---|
| LangGraph | `contrib-openai-v2` (opentelemetry-instrumentation-openai-v2 2.4b0) | g1.harness.run, chat qwen3.8:27b x3 | 10 (incl. operation.name, request.model, usage.input_tokens, usage.output_tokens) | openai.* (1) | yes | no: no tool spans |
| LangGraph | `contrib-openai-v2-latest` (opentelemetry-instrumentation-openai-v2 2.4b0) | g1.harness.run, chat qwen3.8:27b x3 | 10 (incl. operation.name, request.model, usage.input_tokens, usage.output_tokens) | openai.* (1) | yes | no: no tool spans |
| LangGraph | `llmetry-langchain` (opentelemetry-instrumentation-langchain 0.63.0) | g1.harness.run, create_agent *, invoke_agent *, LangGraph.workflow, execute_task * x17, ChatOpenAI.chat x3, execute_tool * x2 | 29 (incl. operation.name, request.model, usage.input_tokens, usage.output_tokens, tool.name, tool.call.arguments) | traceloop.* (18) | yes | yes |
| LangGraph | `llmetry-openai` (opentelemetry-instrumentation-openai 0.63.0) | g1.harness.run, openai.chat x3 | 18 (incl. operation.name, request.model, usage.input_tokens, usage.output_tokens) |  | yes | no: no tool spans |
| LangGraph | `native-langsmith` (langsmith 0.14.17) | g1.harness.run, LangGraph, agent x3, call_model x3, RunnableSequence x3, Prompt x3, ChatOpenAI x3, should_continue x3, tools x2, get_unit_price, convert_usd_to_eur | 12 (incl. operation.name, request.model, usage.input_tokens, usage.output_tokens, tool.name) | langsmith.* (26) | yes | no: tool args missing |
| LangGraph | none (framework/MCP SDK only) | g1.harness.run | 0 |  | no: no model spans | no: no model spans |
| LangGraph | `oi-langchain` (openinference-instrumentation-langchain 0.1.82) | g1.harness.run, LangGraph, agent x3, call_model x3, RunnableSequence x3, Prompt x3, ChatOpenAI x3, should_continue x3, tools x2, get_unit_price, convert_usd_to_eur | 0 | input.* (2), llm.* (36), metadata.* (1), openinference.* (1), output.* (2), tool.* (2) | yes | yes |
| LangGraph | `oi-openai` (openinference-instrumentation-openai 0.1.69) | g1.harness.run, ChatCompletion x3 | 0 | input.* (2), llm.* (33), openinference.* (1), output.* (2) | yes | no: no tool spans |
| OpenAI Agents SDK | `contrib-openai-v2` (opentelemetry-instrumentation-openai-v2 2.4b0) | g1.harness.run, MCP send * x6, chat qwen3.8:27b x3 | 9 (incl. operation.name, request.model, usage.input_tokens, usage.output_tokens) | jsonrpc.* (1), mcp.* (1), openai.* (1) | yes | no: no tool spans |
| OpenAI Agents SDK | `contrib-openai-v2-latest` (opentelemetry-instrumentation-openai-v2 2.4b0) | g1.harness.run, MCP send * x6, chat qwen3.8:27b x3 | 9 (incl. operation.name, request.model, usage.input_tokens, usage.output_tokens) | jsonrpc.* (1), mcp.* (1), openai.* (1) | yes | no: no tool spans |
| OpenAI Agents SDK | `llmetry-openai` (opentelemetry-instrumentation-openai 0.63.0) | g1.harness.run, MCP send * x6, openai.chat x3 | 23 (incl. operation.name, request.model, usage.input_tokens, usage.output_tokens) | jsonrpc.* (1), mcp.* (1) | yes | no: no tool spans |
| OpenAI Agents SDK | `llmetry-openai-agents` (opentelemetry-instrumentation-openai-agents 0.63.0) | g1.harness.run, MCP send * x6, Agent Workflow, toy.agent, openai.response x3, get_unit_price.tool, convert_usd_to_eur.tool | 9 (incl. operation.name, request.model, tool.name, tool.call.arguments) | jsonrpc.* (1), mcp.* (1), traceloop.* (2) | yes | yes |
| OpenAI Agents SDK | none (framework/MCP SDK only) | g1.harness.run, MCP send * x6 | 0 | jsonrpc.* (1), mcp.* (1) | no: no model spans | no: no model spans |
| OpenAI Agents SDK | `oi-openai` (openinference-instrumentation-openai 0.1.69) | g1.harness.run, MCP send * x6, ChatCompletion x3 | 0 | input.* (2), jsonrpc.* (1), llm.* (35), mcp.* (1), openinference.* (1), output.* (2) | yes | no: no tool spans |
| OpenAI Agents SDK | `oi-openai-agents` (openinference-instrumentation-openai-agents 2.7.1) | g1.harness.run, MCP send * x6, Agent workflow x2, toy, turn x3, mcp_tools x3, generation x3, get_unit_price, convert_usd_to_eur | 0 | agent.* (1), graph.* (1), input.* (2), jsonrpc.* (1), llm.* (30), mcp.* (1), openinference.* (1), output.* (2), tool.* (3) | yes | yes |
| smolagents | `contrib-openai-v2` (opentelemetry-instrumentation-openai-v2 2.4b0) | g1.harness.run, chat qwen3.8:27b x3 | 10 (incl. operation.name, request.model, usage.input_tokens, usage.output_tokens) | openai.* (1) | yes | no: no tool spans |
| smolagents | `contrib-openai-v2-latest` (opentelemetry-instrumentation-openai-v2 2.4b0) | g1.harness.run, chat qwen3.8:27b x3 | 10 (incl. operation.name, request.model, usage.input_tokens, usage.output_tokens) | openai.* (1) | yes | no: no tool spans |
| smolagents | `llmetry-openai` (opentelemetry-instrumentation-openai 0.63.0) | g1.harness.run, openai.chat x3 | 17 (incl. operation.name, request.model, usage.input_tokens, usage.output_tokens) |  | yes | no: no tool spans |
| smolagents | none (framework/MCP SDK only) | g1.harness.run | 0 |  | no: no model spans | no: no model spans |
| smolagents | `oi-openai` (openinference-instrumentation-openai 0.1.69) | g1.harness.run, ChatCompletion x3 | 0 | input.* (2), llm.* (33), openinference.* (1), output.* (2) | yes | no: no tool spans |
| smolagents | `oi-smolagents` (openinference-instrumentation-smolagents 0.1.46) | g1.harness.run, ToolCallingAgent.run, Step * x3, OpenAIModel.generate x3, MCPAdaptTool x2, FinalAnswerTool | 0 | input.* (2), llm.* (35), openinference.* (1), output.* (2), smolagents.* (2), tool.* (3) | yes | yes |

Native OpenTelemetry: none of LangGraph, `langchain-core`, the Agents SDK or smolagents contains
OpenTelemetry code (searched the installed packages). The one exception found is the `langsmith`
SDK, whose OTel mode (`LANGSMITH_OTEL_ENABLED`, `LANGSMITH_OTEL_ONLY`) is the `native-langsmith` row;
no LangSmith API key was set and the run reported no errors. The MCP Python SDK 2.x does emit
spans itself (`MCP send *` rows, attributes `mcp.method.name`, `jsonrpc.request.id`), so
"none" rows for the Agents SDK still contain MCP client spans but no model or tool spans.

Agents SDK default export: the SDK started with 1 trace processor (the default exporter);
`set_trace_processors([])` was called before instrumenting and left 0 (instrumentor rows then
have their own processor). Evidence: `agents_sdk_trace_processors_*` in each `result.json`.

### `gen_ai.*` attribute sets, by instrumentor

Names without the `gen_ai.` prefix. Sets differ by framework only where noted.

- **contrib openai-v2:** `operation.name`, `provider.name`, `request.model`, `request.temperature`,
  `response.finish_reasons`, `response.id`, `response.model`, `usage.input_tokens`,
  `usage.output_tokens` (message content is off by default). Model spans only.
- **llmetry openai:** 17 to 23 attributes, including messages, tool definitions,
  `usage.total_tokens` and `request.*`. Model spans only. In its three runs and in the
  OpenLLMetry LangChain run, a `traceparent` header was sent to the model endpoint (3 of 3
  requests each); no other combination sent one.
- **llmetry langchain:** 29 attributes on model, tool and task spans, including
  `tool.name`, `tool.call.arguments`, `usage.*`, `workflow.nodes`.
- **llmetry openai-agents:** 9 attributes on agent, response and tool spans (`tool.name`,
  `tool.call.arguments`, `tool.call.result`); no `usage.*` on the model spans in these runs.
- **langsmith OTel:** 12 attributes, including legacy `gen_ai.prompt`, `gen_ai.completion`,
  `gen_ai.system`, `gen_ai.serialized.name`; tool spans have `tool.name` but no arguments.
- **OpenInference (all four):** no `gen_ai.*` attributes. It uses `openinference.span.kind`,
  `llm.*`, `input.value`, `output.value`, `tool.*`.

### Tool argument fidelity

Whether the tool argument attribute equals the arguments that crossed the MCP wire:

| Instrumentor | Argument attribute | Matches wire arguments |
| --- | --- | --- |
| llmetry openai-agents, oi openai-agents | `gen_ai.tool.call.arguments`, `input.value` | 2 of 2 |
| oi langchain | `input.value` | 1 of 2 (single-argument tool is a bare string, `widget`) |
| llmetry langchain | `gen_ai.tool.call.arguments` | 0 of 2 (wrapper object with `input_str`, `tags`, `metadata`; Python-repr string) |
| oi smolagents | `input.value` | 0 of 2 (wrapped as `{"args": [], "kwargs": {...}}`) |
| langsmith OTel | none | n/a |

The argument hash in the (t, i) rule therefore needs per-instrumentor unwrapping.

Other notes on `t` and `i`:

- smolagents with OpenInference also emits a `FinalAnswerTool` span, giving 3 tool spans for 2
  MCP calls; without excluding it the derived turns are [1, 2, 3] against real [1, 2].
- Instrumentors that only wrap the OpenAI client give `t` but no tool spans, so `i` cannot be
  rebuilt from them. Instrumentors that wrap the MCP client (`llmetry-mcp`, `oi-mcp`) give tool
  spans but no model spans, so `t` cannot be rebuilt from them. A combination is needed for both.
  Combinations of two instrumentors were not run.
- Each framework also emits its own step counter (`langgraph_step` in OpenLLMetry's LangChain
  spans, `Step N` and `turn` span names in smolagents and OpenInference's Agents SDK spans);
  these were not used.

## Table 2: W3C trace context on MCP calls

Counts are over the `tools/call` requests that reached the server (2 per run). Header and env
columns show `traceparent`/`baggage`. Evidence is the server-side wire log, not the client's
view.

| Framework | MCP SDK | Transport | MCP instrumentor | `_meta.traceparent` | `_meta.baggage` | HTTP header `traceparent`/`baggage` | env `TRACEPARENT`/`BAGGAGE` | same trace id as caller | `_meta` parent span |
|---|---|---|---|---|---|---|---|---|---|
| LangGraph | mcp 1.30.0 | http | `llmetry-mcp` (0.63.0) | 0/2 | 0/2 | 0/0 of 2 | 0/0 | - | - |
| LangGraph | mcp 1.30.0 | http | none | 0/2 | 0/2 | 0/0 of 2 | 0/0 | - | - |
| LangGraph | mcp 1.30.0 | http | `oi-mcp` (2.1.1) | 2/2 | 2/2 | 0/0 of 2 | 0/0 | yes | g1.harness.run |
| LangGraph | mcp 1.30.0 | stdio | `llmetry-mcp` (0.63.0) | 2/2 | 2/2 | 0/0 of 2 | 0/0 | yes | ResponseStreamWriter |
| LangGraph | mcp 1.30.0 | stdio | none | 0/2 | 0/2 | 0/0 of 2 | 0/0 | - | - |
| LangGraph | mcp 1.30.0 | stdio | `oi-mcp` (2.1.1) | 2/2 | 2/2 | 0/0 of 2 | 0/0 | yes | g1.harness.run |
| OpenAI Agents SDK | mcp 2.3.0 | http | `llmetry-mcp` (0.63.0) | FAILED to instrument | | | | | |
| OpenAI Agents SDK | mcp 2.3.0 | http | none | 2/2 | 2/2 | 0/0 of 2 | 0/0 | yes | MCP send tools/call convert_usd_to_eur, MCP send tools/call get_unit_price |
| OpenAI Agents SDK | mcp 2.3.0 | http | `oi-mcp` (2.1.1) | 2/2 | 2/2 | 0/0 of 2 | 0/0 | yes | MCP send tools/call convert_usd_to_eur, MCP send tools/call get_unit_price |
| OpenAI Agents SDK | mcp 2.3.0 | stdio | `llmetry-mcp` (0.63.0) | FAILED to instrument | | | | | |
| OpenAI Agents SDK | mcp 2.3.0 | stdio | none | 2/2 | 2/2 | 0/0 of 2 | 0/0 | yes | MCP send tools/call convert_usd_to_eur, MCP send tools/call get_unit_price |
| OpenAI Agents SDK | mcp 2.3.0 | stdio | `oi-mcp` (2.1.1) | 2/2 | 2/2 | 0/0 of 2 | 0/0 | yes | MCP send tools/call convert_usd_to_eur, MCP send tools/call get_unit_price |
| smolagents | mcp 1.30.0 | http | `llmetry-mcp` (0.63.0) | 2/2 | 2/2 | 0/0 of 2 | 0/0 | yes | ResponseStreamWriter |
| smolagents | mcp 1.30.0 | http | none | 0/2 | 0/2 | 0/0 of 2 | 0/0 | - | - |
| smolagents | mcp 1.30.0 | http | `oi-mcp` (2.1.1) | 2/2 | 2/2 | 0/0 of 2 | 0/0 | yes | g1.harness.run |
| smolagents | mcp 1.30.0 | stdio | `llmetry-mcp` (0.63.0) | 2/2 | 2/2 | 0/0 of 2 | 0/0 | yes | ResponseStreamWriter |
| smolagents | mcp 1.30.0 | stdio | none | 0/2 | 0/2 | 0/0 of 2 | 0/0 | - | - |
| smolagents | mcp 1.30.0 | stdio | `oi-mcp` (2.1.1) | 2/2 | 2/2 | 0/0 of 2 | 0/0 | yes | g1.harness.run |

Findings:

- Over **both transports**, trace context reaches the server only inside `params._meta`, as
  `traceparent` and `baggage` (no `tracestate` was present). **No HTTP `traceparent` or `baggage`
  header** was seen on Streamable HTTP requests (other headers such as `mcp-protocol-version`
  were logged, so header capture worked), and **no `TRACEPARENT` or `BAGGAGE` environment
  variable** was set on the stdio server process.
- **MCP Python SDK 2.3.0 (Agents SDK venv) injects `traceparent` and `baggage` itself**, with no
  instrumentor, on every request including `server/discover` and `tools/list`. The trace id
  matches the caller and the parent span is the SDK's own `MCP send tools/call <tool>` client
  span.
- **MCP Python SDK 1.30.0 (LangGraph and smolagents venvs) injects nothing** on its own.
  OpenInference's MCP instrumentor added both fields in 4 of 4 runs, parented to whatever span
  was current (`g1.harness.run`). OpenLLMetry's MCP instrumentor added them in 3 of 4 runs (not on
  LangGraph over HTTP), parented to its own `ResponseStreamWriter` span.
- OpenLLMetry's MCP instrumentor (0.63.0) failed to instrument under MCP 2.3.0
  (`PathResolutionError: unable to resolve attribute 'streamablehttp_client'`).
- Baggage reached the server wherever trace context did. The baggage values are the harness
  values (`spanslo.run.id`, `spanslo.turn=0`); no framework updates a turn value.
- LangGraph's MCP adapter opened a new session per operation: 3 server process spawns on stdio for
  one list and two calls. The Agents SDK and smolagents used 1.

## Compatibility notes found while building

- `mcpadapt` 0.1.20 (used by smolagents' `ToolCollection.from_mcp`) declares `mcp>=1.10.1`
  without an upper bound; the resolver chose mcp 2.3.0 and the import failed
  (`streamablehttp_client` renamed). The smolagents venv was pinned to `mcp[ws]<2`.
- `opentelemetry-instrumentation-openai-v2` 2.4b0 needs `opentelemetry-util-genai` older than
  1.2b0; the resolver installed 1.3b0 and import failed. Pinned to 1.1b0 in all three venvs.
- The same instrumentor imports `httpx` without declaring it; openai 3.x uses `httpx2`. `httpx`
  0.28.1 was added to the Agents SDK venv.
- `langgraph.prebuilt.create_react_agent` is deprecated in LangGraph 1.x in favour of
  `langchain.agents.create_agent`. The deprecated function was used; `langchain` was not installed.

## Not covered

- Parallel tool calls, streaming model responses, tasks with errors or injected faults.
- Two instrumentors active at once (duplicate or nested spans).
- Native Ollama client paths; only the OpenAI-compatible endpoint was used.
- Whether any run contacted external hosts: no key was set and no errors were seen, but traffic
  was not captured at the network level.
- Repetition: one run per cell.
