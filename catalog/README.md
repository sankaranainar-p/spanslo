# catalog

One YAML file per fault in the v1 fault catalog (12 faults), validated by
[spec/fault.schema.json](../spec/fault.schema.json).

- `defaults.yaml` holds the contract shared by all faults. A fault's effective contract is the
  defaults with the fault's own `contract:` keys laid over them, key by key. No fault overrides it
  yet.
- `trigger.turn` lists the benchmark positions. The trigger mode is not part of the fault. It is a
  run parameter chosen in the matrix file and must be one of the fault's `supported_modes`
  (default `single`). Under `persistent`, the fault fires from the listed turn onward.
- Entries under `sources.bench` are v0 defaults (durations, status codes, ages, fractions). They
  can change before the first benchmark run without changing the fault id.
- For the other sources, `null` means the source cannot produce the fault. `unverified: true` means
  the source may be able to, but no evidence in [docs/g0/gap-matrix.md](../docs/g0/gap-matrix.md)
  confirms it, or confirms the parameter names.
- Tumult is treated as a model surface source only, and its proxy faults the first model call
  only (turn 1), per the G0 evidence. Its model entries are `unverified`, and it cannot target turns
  3 or 5.
- Tags in `must_not_call_tags` are attached to tools, not to faults. They are read from MCP tool
  annotations: `destructive` from `destructiveHint` and `outbound` from `openWorldHint`
  (see `tool_tags` in [spec/semconv-mapping.yaml](../spec/semconv-mapping.yaml)).

Design: [docs/design.md](../docs/design.md), section "Fault catalog".
