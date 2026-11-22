# Decision: new spanslo attributes and a run end time for scoring

- **Status:** Proposed
- **Date:** 2026-11-15
- **Deciders:** Sankaranainar Parmasivan
- **Inputs:** [../../spec/semconv-mapping.yaml](../../spec/semconv-mapping.yaml), [../../spec.md](../../spec.md) (R-012, R-021, R-022), [0002-semconv-pin.md](0002-semconv-pin.md)

## Question

The scorer reads only files of spans. Two inputs it needs are not on any span today: the fallback
tools a task declares (design.md says they live in the task file), and the MCP annotations of each
tool. The run record also has no time, so windows cannot order runs. Where does the scorer read
these from?

## Decision

Add groups of `spanslo.*` attributes to `spec/semconv-mapping.yaml`, under `spanslo_attributes`.

- `spanslo.task.fallback_tools`: a string array on the root `invoke_agent` span, written by the
  harness. Absent or empty means the task has no fallback.
- `spanslo.tool.destructive_hint` and `spanslo.tool.open_world_hint`: booleans on `execute_tool`
  spans, carrying the MCP annotations `destructiveHint` and `openWorldHint`. An absent attribute
  means the tool is untagged.
- `spanslo.task.turns_expected`: an integer of at least 1 on the root `invoke_agent` span, written by the
  harness from the task file. Absent, not an integer, or below 1 means the run record's
  `task_turns_expected` is null (design v2.11). Added in mapping_version 7.
- `mapping_version` goes from 5 to 6 for the first group, and from 6 to 7 for `spanslo.task.turns_expected`, as ADR 0002 requires for any change to a name.
- Run records gain a required field `ended_at` in `spec/run-record.schema.json`: the end time of the
  root `invoke_agent` span, RFC 3339 date-time in UTC with a trailing `Z`. It orders runs and assigns
  them to time and count windows. The scorer derives it from the span, so it adds no attribute. The scorer parses it with a real RFC 3339 parse, because the schema pattern is only a first filter and does not reject dates like Feb 30. A run whose root span is missing or unfinished has no `ended_at` and is invalid, and `slo`, `report` and `gate` exit 2 on a run record with an unparseable `ended_at` (design v2.7, "Invalid runs").

These are not `gen_ai.*` names, so the network test against the pinned registry does not cover them.
`bench/tests/test_spec_schemas.py` checks that the keys exist.

## Why

- The CLI takes only traces and the catalog. Putting the data on spans keeps the boundary at files.
- Only the two hints behind `must_not_call_tags` are needed to tag a call. The other two MCP hints
  stay in the tool manifests under `bench/tools`.
- The names are proposed here. design.md and `spec/tool-annotations.schema.json` defined none before
  design v2.6, which documents them; v2.7 adds the invalid run rules.
- Window ordering needs a time, and the scorer must not read the wall clock. The root span end time is
  already in the input, so no clock is needed.

## Costs

- The harness and every tool hook must write these attributes. External tools that do not write them
  are untagged and have no fallbacks.
- `spec/` was frozen at G1 (tag `v0.1.0-spec`). This change needs a new spec tag.
- `ended_at` is required, so every run record and example written before this change is invalid.
  No run records exist yet outside `spec/examples/`, which were updated.

## Revisit

- Rename the attributes if the OpenTelemetry GenAI conventions add equivalents.
