# Decision: hand written run record types

- **Status:** Accepted
- **Date:** 2026-11-16
- **Deciders:** Sankaranainar Parmasivan
- **Inputs:** [../../spec/run-record.schema.json](../../spec/run-record.schema.json), [../../CLAUDE.md](../../CLAUDE.md)

## Context

CLAUDE.md said Go types are generated from the JSON Schemas in `spec/`, never hand edited.
Generating Go types needs a generator tool, and no such dependency has been approved. The
only approved Go dependency is the YAML parser (`go.yaml.in/yaml/v3`). Weeks 5 to 8 need a
Go run record type now.

## Decision

- The Go run record type is hand written, with JSON tags that mirror
  `spec/run-record.schema.json`.
- A Python test in `bench/tests/` runs `spanslo score` on the fixtures and validates every
  emitted record against the schema with `jsonschema`. This test is the drift guard.
- The schema stays the source of truth. If the struct and the schema disagree, the struct is
  wrong.
- CLAUDE.md and constitution.md are updated to say so.

## Revisit

- When a schema to Go generator dependency is approved, replace the hand written type with
  generated code, keep the Python test, and record that in a new ADR.
