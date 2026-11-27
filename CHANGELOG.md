# Changelog

All notable changes are recorded here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versions are not stable before 0.2.0.

## [Unreleased]

### Added
- `report` has `groups` (per experiment, framework, model and fault) and `baselines` (per task, framework and model).

### Changed
- Report percentile keys are snake case (`turns_to_recover`, `p50`, `p95`, `unrecovered`). Undefined values are `null`. Before, they were Go field names, with an `OK` flag.

## [0.2.0] - 2026-11-26

First scoring release. Not a benchmark release.

### Added
- Names mapping read from the pinned semantic convention file, with a test that forbids hardcoded `gen_ai.*` literals.
- RFC 8785 canonical JSON and argument hashing in the core.
- OTLP JSON reader and grouping of spans into runs.
- Turn and step indexing, and the turn mismatch count.
- Trace fixtures for indexing edge cases, and a purity check for the core.
- Hand written Go run record types with a schema drift test (ADR 0004).

- Fault episodes, recovery rules, SLIs, percentiles, SLO evaluation and burn rate alerts.
- Offline CLI: `score`, `slo`, `report`, `gate`, with documented exit codes.
- Twelve golden trace fixtures scored through the real CLI.
- Tag triggered release workflow: five binaries and `SHA256SUMS.txt`, published as a pre-release.
- Semantic convention mapping version 9: names the fault event `spanslo.fault`.

### Known limits
- Duplicate JSON object keys keep the last value. RFC 8785 says to reject them (open question OQ-1).
- `report` pools all runs. No grouping by experiment, framework, model or fault (R-034, R-037).
- Adapters for external fault injectors are blocked until real fixtures are captured (T-036 to T-039).
- The fault event name is also a constant in core. A test checks it equals the mapping (OQ-2).

## [0.1.1-spec]

### Added
- Spec frozen: JSON Schemas for fault, run record, SLO and tool annotations, the SLI definitions and the semantic convention mapping.
- ADR 0002 pins the GenAI semantic conventions to a commit.

## [0.1.0-spec]

### Added
- First spec draft and the week 1 research record (`docs/g0/`).
