# Decision: pin the GenAI semantic conventions to a commit

- **Status:** Proposed
- **Date:** 2026-11-11
- **Deciders:** Sankaranainar Parmasivan
- **Inputs:** [../../spec/semconv-mapping.yaml](../../spec/semconv-mapping.yaml), week 3 in [../design.md](../design.md)

## Question

spanslo reads `gen_ai.*` span names and attributes. Those conventions are in development and still
change. How do we pin them so scoring results stay reproducible?

## Decision

Pin to a full commit SHA of `open-telemetry/semantic-conventions-genai`, recorded in
`spec/semconv-mapping.yaml` together with the commit date, schema URL and base semconv version.

- Pinned commit: `e6658f14c9d9ea0760825f029cb82e7673f48fab` (2026-11-06).
- Code reads every name from the mapping file. No `gen_ai.*` string is hardcoded elsewhere.
- Every name in the mapping is checked against the pinned commit by
  `bench/tests/test_spec_schemas.py` when `SPANSLO_NETWORK_TESTS=1`.
- `mapping_version` bumps on any change to a name, an operation value or the pin.

## Why a commit and not a tag

- When pinned, the repository had no GitHub releases and no tags. A tag was not an option.
- A commit SHA is immutable, so a result can be traced to exact names.

## Costs

- A SHA says nothing about stability or order. Readers cannot tell how far behind we are.
- Upgrades are manual. Someone must diff the registry and bump `mapping_version`.

## Revisit

- Re-pin to a `vX.Y.Z-dev` tag as soon as one exists, and record that in a new ADR.
- Re-check the pin at each gate (G1 to G4) and before any release tag.
- If a pinned name is renamed upstream, change the mapping and the version. Do not edit old run
  records. Records carry the `mapping_version` they were scored under.
